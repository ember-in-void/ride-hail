package inamqp

import (
	"context"
	"encoding/json"
	"fmt"

	"ridehail/internal/ride/adapter/in/in_ws"
	"ridehail/internal/shared/logger"
	"ridehail/internal/shared/mq"

	amqp "github.com/rabbitmq/amqp091-go"
)

// LocationUpdateMessage структура обновления локации водителя
type LocationUpdateMessage struct {
	DriverID  string  `json:"driver_id"`
	RideID    string  `json:"ride_id,omitempty"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Heading   float64 `json:"heading,omitempty"`
	Speed     float64 `json:"speed,omitempty"`
	Accuracy  float64 `json:"accuracy,omitempty"`
	Timestamp string  `json:"timestamp"`
}

// LocationConsumer обрабатывает обновления локации и отправляет их пассажирам
type LocationConsumer struct {
	mqConn      *mq.RabbitMQ
	passengerWS *in_ws.PassengerWSHandler
	log         *logger.Logger
}

// NewLocationConsumer создает новый consumer для location updates
func NewLocationConsumer(
	mqConn *mq.RabbitMQ,
	passengerWS *in_ws.PassengerWSHandler,
	log *logger.Logger,
) *LocationConsumer {
	return &LocationConsumer{
		mqConn:      mqConn,
		passengerWS: passengerWS,
		log:         log,
	}
}

// Start запускает consumer для location_fanout exchange
func (c *LocationConsumer) Start(ctx context.Context) error {
	ch := c.mqConn.Channel()
	if ch == nil {
		return fmt.Errorf("failed to get RabbitMQ channel")
	}

	// Объявляем временную очередь для этого сервиса
	queue, err := ch.QueueDeclare(
		"ride_service_locations", // name
		false,                    // durable
		true,                     // auto-delete
		false,                    // exclusive
		false,                    // no-wait
		nil,                      // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Привязываем очередь к location_fanout exchange
	err = ch.QueueBind(
		queue.Name,        // queue name
		"",                // routing key (игнорируется для fanout)
		"location_fanout", // exchange
		false,             // no-wait
		nil,               // arguments
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
		Action:  "location_consumer_started",
		Message: fmt.Sprintf("listening on location_fanout (queue: %s)", queue.Name),
	})

	// Обработка сообщений
	for {
		select {
		case <-ctx.Done():
			c.log.Info(logger.Entry{Action: "location_consumer_stopping", Message: "context cancelled"})
			return ctx.Err()

		case msg, ok := <-msgs:
			if !ok {
				c.log.Warn(logger.Entry{Action: "location_consumer_channel_closed", Message: "message channel closed"})
				return fmt.Errorf("message channel closed")
			}

			if err := c.handleLocationUpdate(ctx, msg); err != nil {
				c.log.Error(logger.Entry{
					Action:  "handle_location_update_failed",
					Message: err.Error(),
					Error:   &logger.ErrObj{Msg: err.Error()},
				})
				_ = msg.Nack(false, true)
			} else {
				_ = msg.Ack(false)
			}
		}
	}
}

// handleLocationUpdate обрабатывает обновление локации и отправляет пассажиру
func (c *LocationConsumer) handleLocationUpdate(ctx context.Context, msg amqp.Delivery) error {
	var locationUpdate LocationUpdateMessage
	if err := json.Unmarshal(msg.Body, &locationUpdate); err != nil {
		return fmt.Errorf("failed to parse location update: %w", err)
	}

	c.log.Debug(logger.Entry{
		Action: "location_update_received",
		Message: fmt.Sprintf("driver=%s, ride=%s, lat=%f, lng=%f",
			locationUpdate.DriverID, locationUpdate.RideID,
			locationUpdate.Latitude, locationUpdate.Longitude),
	})

	// Если есть ride_id, отправляем обновление пассажиру через WebSocket
	if locationUpdate.RideID != "" {
		locationData := map[string]interface{}{
			"type":      "driver_location",
			"ride_id":   locationUpdate.RideID,
			"driver_id": locationUpdate.DriverID,
			"latitude":  locationUpdate.Latitude,
			"longitude": locationUpdate.Longitude,
			"heading":   locationUpdate.Heading,
			"speed":     locationUpdate.Speed,
			"timestamp": locationUpdate.Timestamp,
		}

		c.log.Debug(logger.Entry{
			Action:  "sending_location_to_passenger",
			Message: fmt.Sprintf("ride_id=%s", locationUpdate.RideID),
		})

		// TODO: Получить passenger_id из ride_id и отправить конкретному пассажиру
		// c.passengerWS.SendDriverLocationUpdate(passengerID, locationData)
		_ = locationData // suppress unused warning
	}

	return nil
}
