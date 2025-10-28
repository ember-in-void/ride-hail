package out_amqp

import (
	"context"
	"encoding/json"
	"fmt"

	"ridehail/internal/ride/application/ports/out"
	"ridehail/internal/shared/logger"
	"ridehail/internal/shared/mq"
)

// RideEventPublisher публикует события поездок в RabbitMQ
type RideEventPublisher struct {
	mq  *mq.RabbitMQ
	log *logger.Logger
}

// NewRideEventPublisher создает новый publisher
func NewRideEventPublisher(mqConn *mq.RabbitMQ, log *logger.Logger) *RideEventPublisher {
	return &RideEventPublisher{
		mq:  mqConn,
		log: log,
	}
}

// PublishRideEvent публикует событие поездки в RabbitMQ
func (p *RideEventPublisher) PublishRideEvent(ctx context.Context, eventType string, data out.RideEventData) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal event data: %w", err)
	}

	// Определяем routing key на основе типа события
	routingKey := getRoutingKey(eventType)

	// Публикуем в ride_topic exchange
	if err := p.mq.Publish(ctx, "ride_topic", routingKey, payload); err != nil {
		p.log.Error(logger.Entry{
			Action:  "publish_ride_event_failed",
			Message: err.Error(),
			RideID:  data.RideID,
			Error:   &logger.ErrObj{Msg: err.Error()},
			Additional: map[string]any{
				"event_type":  eventType,
				"routing_key": routingKey,
			},
		})
		return fmt.Errorf("publish to rabbitmq: %w", err)
	}

	p.log.Debug(logger.Entry{
		Action:  "ride_event_published",
		Message: eventType,
		RideID:  data.RideID,
		Additional: map[string]any{
			"routing_key": routingKey,
		},
	})

	return nil
}

// getRoutingKey возвращает routing key для события
func getRoutingKey(eventType string) string {
	switch eventType {
	case "RIDE_REQUESTED":
		return "ride.requested"
	case "DRIVER_MATCHED":
		return "ride.matched"
	case "RIDE_STARTED":
		return "ride.started"
	case "RIDE_COMPLETED":
		return "ride.completed"
	case "RIDE_CANCELLED":
		return "ride.cancelled"
	default:
		return "ride.event"
	}
}
