package out

import (
	"context"

	"ridehail/internal/driver/domain"
)

// MessagePublisher определяет публикацию событий в RabbitMQ
type MessagePublisher interface {
	// PublishDriverResponse публикует ответ водителя на запрос поездки
	PublishDriverResponse(ctx context.Context, dto *DriverResponseDTO) error

	// PublishDriverStatus публикует изменение статуса водителя
	PublishDriverStatus(ctx context.Context, dto *DriverStatusDTO) error

	// PublishLocationUpdate публикует обновление локации водителя
	PublishLocationUpdate(ctx context.Context, dto *LocationUpdateDTO) error
}

// DriverResponseDTO — ответ водителя на предложение поездки
type DriverResponseDTO struct {
	RideID                  string        `json:"ride_id"`
	DriverID                string        `json:"driver_id"`
	Accepted                bool          `json:"accepted"`
	EstimatedArrivalMinutes int           `json:"estimated_arrival_minutes,omitempty"`
	DriverLocation          LocationDTO   `json:"driver_location,omitempty"`
	DriverInfo              DriverInfoDTO `json:"driver_info,omitempty"`
	CorrelationID           string        `json:"correlation_id,omitempty"`
}

// DriverStatusDTO — изменение статуса водителя
type DriverStatusDTO struct {
	DriverID  string              `json:"driver_id"`
	Status    domain.DriverStatus `json:"status"`
	RideID    string              `json:"ride_id,omitempty"`
	Timestamp string              `json:"timestamp"`
}

// LocationUpdateDTO — обновление локации водителя
type LocationUpdateDTO struct {
	DriverID       string      `json:"driver_id"`
	RideID         string      `json:"ride_id,omitempty"`
	Location       LocationDTO `json:"location"`
	SpeedKmh       float64     `json:"speed_kmh,omitempty"`
	HeadingDegrees float64     `json:"heading_degrees,omitempty"`
	Timestamp      string      `json:"timestamp"`
}

// LocationDTO — координаты
type LocationDTO struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// DriverInfoDTO — информация о водителе для пассажира
type DriverInfoDTO struct {
	Name    string     `json:"name"`
	Rating  float64    `json:"rating"`
	Vehicle VehicleDTO `json:"vehicle"`
}

// VehicleDTO — информация о транспорте
type VehicleDTO struct {
	Make  string `json:"make"`
	Model string `json:"model"`
	Color string `json:"color"`
	Plate string `json:"plate"`
}
