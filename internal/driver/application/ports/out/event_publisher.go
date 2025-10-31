package out

import (
	"context"
)

// DriverEvent — событие от Driver Service
type DriverEvent struct {
	Type      string
	DriverID  string
	RideID    *string
	Status    *string
	Latitude  *float64
	Longitude *float64
	Data      map[string]any
}

// EventPublisher — публикация событий в RabbitMQ
type EventPublisher interface {
	// PublishDriverResponse публикует ответ водителя на ride offer
	// Exchange: driver_topic, RoutingKey: driver.response.{ride_id}
	PublishDriverResponse(ctx context.Context, rideID, driverID string, accepted bool, driverInfo map[string]any) error

	// PublishDriverStatusChanged публикует изменение статуса водителя
	// Exchange: driver_topic, RoutingKey: driver.status.{driver_id}
	PublishDriverStatusChanged(ctx context.Context, driverID, status string) error

	// PublishLocationUpdate публикует обновление локации
	// Exchange: location_fanout (fanout, без routing key)
	PublishLocationUpdate(ctx context.Context, driverID string, rideID *string, lat, lng, speed, heading float64) error
}
