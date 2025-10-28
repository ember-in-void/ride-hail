package out

import (
	"context"
)

// RideEventData — данные события поездки
type RideEventData struct {
	RideID         string                 `json:"ride_id"`
	PassengerID    string                 `json:"passenger_id"`
	DriverID       *string                `json:"driver_id,omitempty"`
	Status         string                 `json:"status"`
	VehicleType    string                 `json:"vehicle_type"`
	AdditionalData map[string]interface{} `json:"additional_data,omitempty"`
}

// EventPublisher — интерфейс для публикации событий в RabbitMQ
type EventPublisher interface {
	// PublishRideEvent публикует событие поездки
	// eventType: RIDE_REQUESTED | DRIVER_MATCHED | RIDE_STARTED | RIDE_COMPLETED | RIDE_CANCELLED
	PublishRideEvent(ctx context.Context, eventType string, data RideEventData) error
}
