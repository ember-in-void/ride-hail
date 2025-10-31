package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	out "ridehail/internal/driver/application/ports/out"
	"ridehail/internal/shared/logger"
	"ridehail/internal/shared/mq"
)

type eventPublisher struct {
	mq  *mq.RabbitMQ
	log *logger.Logger
}

func NewEventPublisher(mq *mq.RabbitMQ, log *logger.Logger) out.EventPublisher {
	return &eventPublisher{mq: mq, log: log}
}

func (p *eventPublisher) PublishDriverResponse(ctx context.Context, rideID, driverID string, accepted bool, driverInfo map[string]any) error {
	event := map[string]any{
		"ride_id":   rideID,
		"driver_id": driverID,
		"accepted":  accepted,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	if accepted && driverInfo != nil {
		event["driver_info"] = driverInfo
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal driver response: %w", err)
	}

	routingKey := fmt.Sprintf("driver.response.%s", rideID)

	if err := p.mq.Publish(ctx, "driver_topic", routingKey, body); err != nil {
		p.log.Error(logger.Entry{
			Action:  "publish_driver_response_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
			RideID:  rideID,
		})
		return fmt.Errorf("publish driver response: %w", err)
	}

	p.log.Debug(logger.Entry{
		Action:  "driver_response_published",
		Message: "driver response sent to ride service",
		RideID:  rideID,
		Additional: map[string]any{
			"driver_id": driverID,
			"accepted":  accepted,
		},
	})

	return nil
}

func (p *eventPublisher) PublishDriverStatusChanged(ctx context.Context, driverID, status string) error {
	event := map[string]any{
		"driver_id": driverID,
		"status":    status,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal status change: %w", err)
	}

	routingKey := fmt.Sprintf("driver.status.%s", driverID)

	if err := p.mq.Publish(ctx, "driver_topic", routingKey, body); err != nil {
		p.log.Error(logger.Entry{
			Action:  "publish_status_changed_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
			Additional: map[string]any{
				"driver_id": driverID,
			},
		})
		return fmt.Errorf("publish status change: %w", err)
	}

	p.log.Debug(logger.Entry{
		Action:  "driver_status_published",
		Message: "driver status change published",
		Additional: map[string]any{
			"driver_id": driverID,
			"status":    status,
		},
	})

	return nil
}

func (p *eventPublisher) PublishLocationUpdate(ctx context.Context, driverID string, rideID *string, lat, lng, speed, heading float64) error {
	event := map[string]any{
		"driver_id": driverID,
		"location": map[string]float64{
			"lat": lat,
			"lng": lng,
		},
		"speed_kmh":       speed,
		"heading_degrees": heading,
		"timestamp":       time.Now().UTC().Format(time.RFC3339),
	}

	if rideID != nil {
		event["ride_id"] = *rideID
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal location update: %w", err)
	}

	// Fanout exchange не требует routing key
	if err := p.mq.Publish(ctx, "location_fanout", "", body); err != nil {
		p.log.Error(logger.Entry{
			Action:  "publish_location_update_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
			Additional: map[string]any{
				"driver_id": driverID,
			},
		})
		return fmt.Errorf("publish location update: %w", err)
	}

	return nil
}
