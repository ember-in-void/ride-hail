package in_amqp

import (
	"context"
	"encoding/json"
	"fmt"

	"ridehail/internal/driver/adapters/in/in_ws"
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

// LocationUpdateConsumer обрабатывает обновления локации водителей
type LocationUpdateConsumer struct {
	mqConn      *mq.RabbitMQ
	passengerWS *in_ws.DriverWSHandler // для отправки обновлений пассажирам
	log         *logger.Logger
}

// NewLocationUpdateConsumer создает новый consumer для location updates
func NewLocationUpdateConsumer(
	mqConn *mq.RabbitMQ,
	passengerWS *in_ws.DriverWSHandler,
	log *logger.Logger,
) *LocationUpdateConsumer {
	return &LocationUpdateConsumer{
		mqConn:      mqConn,
		passengerWS: passengerWS,
		log:         log,
	}
}

// Start запускает consumer для location_fanout exchange
func (c *LocationUpdateConsumer) Start(ctx context.Context) error {
	ch := c.mqConn.Channel()
	if ch == nil {
		return fmt.Errorf("failed to get RabbitMQ channel")
	}

	// Объявляем временную эксклюзивную очередь (удалится при отключении)
	queue, err := ch.QueueDeclare(
		"",    // name - пустое, RabbitMQ сгенерирует уникальное
		false, // durable
		true,  // auto-delete
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
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
		true,       // exclusive
		false,      // no-local
		false,      // no-wait
		nil,        // args
	)
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	c.log.Info(logger.Entry{
		Action:  "location_consumer_started",
		Message: fmt.Sprintf("listening on location_fanout exchange (queue: %s)", queue.Name),
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
				// Nack with requeue
				_ = msg.Nack(false, true)
			} else {
				// Ack успешно обработанное сообщение
				_ = msg.Ack(false)
			}
		}
	}
}

// handleLocationUpdate обрабатывает одно сообщение location update
func (c *LocationUpdateConsumer) handleLocationUpdate(ctx context.Context, msg amqp.Delivery) error {
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
	// В текущей реализации у нас нет прямого доступа к PassengerWSHandler
	// Это будет реализовано через отдельный consumer в Ride Service
	// который подписан на location_fanout и отправляет обновления своим пассажирам

	// TODO: В будущем можно добавить публикацию в ride.location.{ride_id}
	// для более целевой доставки обновлений конкретным пассажирам

	return nil
}
