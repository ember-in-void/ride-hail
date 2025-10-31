package inamqp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ridehail/internal/ride/adapter/in/in_ws"
	"ridehail/internal/ride/application/ports/in"
	"ridehail/internal/shared/logger"
	"ridehail/internal/shared/mq"

	amqp "github.com/rabbitmq/amqp091-go"
)

// DriverResponseMessage структура ответа водителя
type DriverResponseMessage struct {
	RideID                  string  `json:"ride_id"`
	DriverID                string  `json:"driver_id"`
	Accepted                bool    `json:"accepted"`
	EstimatedArrivalMinutes int     `json:"estimated_arrival_minutes,omitempty"`
	DriverLocation          *LocDTO `json:"driver_location,omitempty"`
	DriverInfo              *DrvDTO `json:"driver_info,omitempty"`
	CorrelationID           string  `json:"correlation_id,omitempty"`
}

// LocDTO координаты
type LocDTO struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// DrvDTO информация о водителе
type DrvDTO struct {
	Name    string      `json:"name"`
	Rating  float64     `json:"rating"`
	Vehicle *VehicleDTO `json:"vehicle,omitempty"`
}

// VehicleDTO информация о транспорте
type VehicleDTO struct {
	Make  string `json:"make"`
	Model string `json:"model"`
	Color string `json:"color"`
	Plate string `json:"plate"`
}

// DriverResponseConsumer обрабатывает ответы водителей на ride requests
type DriverResponseConsumer struct {
	mqConn                      *mq.RabbitMQ
	handleDriverResponseUseCase in.HandleDriverResponseUseCase
	passengerWS                 *in_ws.PassengerWSHandler
	log                         *logger.Logger
}

// NewDriverResponseConsumer создает новый consumer
func NewDriverResponseConsumer(
	mqConn *mq.RabbitMQ,
	handleDriverResponseUseCase in.HandleDriverResponseUseCase,
	passengerWS *in_ws.PassengerWSHandler,
	log *logger.Logger,
) *DriverResponseConsumer {
	return &DriverResponseConsumer{
		mqConn:                      mqConn,
		handleDriverResponseUseCase: handleDriverResponseUseCase,
		passengerWS:                 passengerWS,
		log:                         log,
	}
}

// Start запускает consumer для driver.response.*
func (c *DriverResponseConsumer) Start(ctx context.Context) error {
	ch := c.mqConn.Channel()
	if ch == nil {
		return fmt.Errorf("failed to get RabbitMQ channel")
	}

	// Объявляем очередь для driver responses
	queueName := "ride_service_driver_responses"
	queue, err := ch.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // auto-delete
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Привязываем к driver_topic с routing key pattern driver.response.*
	err = ch.QueueBind(
		queue.Name,          // queue name
		"driver.response.*", // routing key pattern
		"driver_topic",      // exchange
		false,               // no-wait
		nil,                 // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue: %w", err)
	}

	// Подписываемся на сообщения
	msgs, err := ch.Consume(
		queue.Name, // queue
		"",         // consumer tag
		false,      // auto-ack
		false,      // exclusive
		false,      // no-local
		false,      // no-wait
		nil,        // args
	)
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	c.log.Info(logger.Entry{
		Action:  "driver_response_consumer_started",
		Message: fmt.Sprintf("listening on driver_topic (queue: %s, pattern: driver.response.*)", queueName),
	})

	// Обработка сообщений
	for {
		select {
		case <-ctx.Done():
			c.log.Info(logger.Entry{Action: "driver_response_consumer_stopping", Message: "context cancelled"})
			return ctx.Err()

		case msg, ok := <-msgs:
			if !ok {
				c.log.Warn(logger.Entry{Action: "driver_response_consumer_channel_closed", Message: "message channel closed"})
				return fmt.Errorf("message channel closed")
			}

			if err := c.handleDriverResponse(ctx, msg); err != nil {
				c.log.Error(logger.Entry{
					Action:  "handle_driver_response_failed",
					Message: err.Error(),
					Error:   &logger.ErrObj{Msg: err.Error()},
				})
				// Nack with requeue для повторной попытки
				_ = msg.Nack(false, true)
			} else {
				// Ack успешно обработанное сообщение
				_ = msg.Ack(false)
			}
		}
	}
}

// handleDriverResponse обрабатывает ответ водителя
func (c *DriverResponseConsumer) handleDriverResponse(ctx context.Context, msg amqp.Delivery) error {
	var response DriverResponseMessage
	if err := json.Unmarshal(msg.Body, &response); err != nil {
		return fmt.Errorf("failed to parse driver response: %w", err)
	}

	c.log.Info(logger.Entry{
		Action:  "driver_response_received",
		Message: fmt.Sprintf("ride=%s, driver=%s, accepted=%t", response.RideID, response.DriverID, response.Accepted),
		RideID:  response.RideID,
		Additional: map[string]any{
			"driver_id":   response.DriverID,
			"accepted":    response.Accepted,
			"routing_key": msg.RoutingKey,
		},
	})

	// Извлекаем ride_id из routing key (driver.response.{ride_id})
	parts := strings.Split(msg.RoutingKey, ".")
	if len(parts) != 3 {
		return fmt.Errorf("invalid routing key format: %s", msg.RoutingKey)
	}
	rideID := parts[2]

	// Вызываем use case для обработки ответа водителя
	useCaseInput := in.HandleDriverResponseInput{
		RideID:                  rideID,
		DriverID:                response.DriverID,
		Accepted:                response.Accepted,
		EstimatedArrivalMinutes: response.EstimatedArrivalMinutes,
	}

	if response.DriverLocation != nil {
		useCaseInput.DriverLocationLat = response.DriverLocation.Lat
		useCaseInput.DriverLocationLng = response.DriverLocation.Lng
	}

	output, err := c.handleDriverResponseUseCase.Execute(ctx, useCaseInput)
	if err != nil {
		c.log.Error(logger.Entry{
			Action:  "handle_driver_response_usecase_failed",
			Message: err.Error(),
			RideID:  rideID,
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		return fmt.Errorf("execute use case: %w", err)
	}

	if output.DriverAssigned {
		// Водитель назначен - отправляем уведомление пассажиру через WebSocket
		c.log.Info(logger.Entry{
			Action:  "sending_ride_matched_notification",
			Message: fmt.Sprintf("driver matched for ride %s", rideID),
			RideID:  rideID,
			Additional: map[string]any{
				"driver_id":    response.DriverID,
				"passenger_id": output.PassengerID,
			},
		})

		// TODO: Отправить WebSocket уведомление пассажиру
		// notification := map[string]any{
		//   "type": "ride_matched",
		//   "ride_id": rideID,
		//   "driver_id": response.DriverID,
		//   "estimated_arrival_minutes": response.EstimatedArrivalMinutes,
		//   "driver_info": response.DriverInfo,
		//   "driver_location": response.DriverLocation,
		// }
		// c.passengerWS.SendToPassenger(output.PassengerID, notification)
	}

	return nil
}
