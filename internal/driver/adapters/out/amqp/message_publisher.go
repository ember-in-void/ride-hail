package messaging

import (
	"context"
	"encoding/json"
	"fmt"

	"ridehail/internal/driver/application/ports/out"
	"ridehail/internal/shared/logger"
	"ridehail/internal/shared/mq"
)

// MessagePublisher реализует публикацию событий в RabbitMQ
type MessagePublisher struct {
	mq  *mq.RabbitMQ
	log *logger.Logger
}

// NewMessagePublisher создает новый publisher для RabbitMQ
func NewMessagePublisher(mq *mq.RabbitMQ, log *logger.Logger) *MessagePublisher {
	return &MessagePublisher{
		mq:  mq,
		log: log,
	}
}

// PublishDriverResponse публикует ответ водителя на запрос поездки
// Routing key: driver.response.{ride_id}
func (p *MessagePublisher) PublishDriverResponse(ctx context.Context, dto *out.DriverResponseDTO) error {
	body, err := json.Marshal(dto)
	if err != nil {
		return fmt.Errorf("marshal driver response: %w", err)
	}

	routingKey := fmt.Sprintf("driver.response.%s", dto.RideID)

	if err := p.mq.Publish(ctx, "driver_topic", routingKey, body); err != nil {
		p.log.Error(logger.Entry{
			Action:  "publish_driver_response_failed",
			Message: err.Error(),
			RideID:  dto.RideID,
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
		return fmt.Errorf("publish to driver_topic: %w", err)
	}

	p.log.Debug(logger.Entry{
		Action:  "driver_response_published",
		Message: fmt.Sprintf("ride_id=%s, driver_id=%s, accepted=%t", dto.RideID, dto.DriverID, dto.Accepted),
		RideID:  dto.RideID,
	})

	return nil
}

// PublishDriverStatus публикует изменение статуса водителя
// Routing key: driver.status.{driver_id}
func (p *MessagePublisher) PublishDriverStatus(ctx context.Context, dto *out.DriverStatusDTO) error {
	body, err := json.Marshal(dto)
	if err != nil {
		return fmt.Errorf("marshal driver status: %w", err)
	}

	routingKey := fmt.Sprintf("driver.status.%s", dto.DriverID)

	if err := p.mq.Publish(ctx, "driver_topic", routingKey, body); err != nil {
		p.log.Error(logger.Entry{
			Action:  "publish_driver_status_failed",
			Message: err.Error(),
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
		return fmt.Errorf("publish to driver_topic: %w", err)
	}

	p.log.Debug(logger.Entry{
		Action:  "driver_status_published",
		Message: fmt.Sprintf("driver_id=%s, status=%s", dto.DriverID, dto.Status),
	})

	return nil
}

// PublishLocationUpdate публикует обновление локации водителя
// Exchange: location_fanout (fanout type)
func (p *MessagePublisher) PublishLocationUpdate(ctx context.Context, dto *out.LocationUpdateDTO) error {
	body, err := json.Marshal(dto)
	if err != nil {
		return fmt.Errorf("marshal location update: %w", err)
	}

	// Fanout exchange не использует routing key, но передаем пустую строку
	if err := p.mq.Publish(ctx, "location_fanout", "", body); err != nil {
		p.log.Error(logger.Entry{
			Action:  "publish_location_update_failed",
			Message: err.Error(),
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
		return fmt.Errorf("publish to location_fanout: %w", err)
	}

	p.log.Debug(logger.Entry{
		Action:  "location_update_published",
		Message: fmt.Sprintf("driver_id=%s, lat=%.6f, lng=%.6f", dto.DriverID, dto.Location.Lat, dto.Location.Lng),
	})

	return nil
}
