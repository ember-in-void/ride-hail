package in

import (
	"context"
)

// DriverUseCase определяет бизнес-логику управления водителем
type DriverUseCase interface {
	GoOnline(ctx context.Context, input GoOnlineInput) (GoOnlineOutput, error)
	GoOffline(ctx context.Context, input GoOfflineInput) (GoOfflineOutput, error)
	UpdateLocation(ctx context.Context, input UpdateLocationInput) (UpdateLocationOutput, error)
	StartRide(ctx context.Context, input StartRideInput) (StartRideOutput, error)
	CompleteRide(ctx context.Context, input CompleteRideInput) (CompleteRideOutput, error)
}

// GoOnlineInput — входные данные для перехода в онлайн
type GoOnlineInput struct {
	DriverID  string  `json:"driver_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// GoOnlineOutput — результат перехода в онлайн
type GoOnlineOutput struct {
	Status    string `json:"status"`
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

// GoOfflineInput — входные данные для перехода в офлайн
type GoOfflineInput struct {
	DriverID string `json:"driver_id"`
}

// GoOfflineOutput — результат перехода в офлайн
type GoOfflineOutput struct {
	Status         string               `json:"status"`
	SessionID      string               `json:"session_id"`
	SessionSummary SessionSummaryOutput `json:"session_summary"`
	Message        string               `json:"message"`
}

// SessionSummaryOutput — сводка по сессии
type SessionSummaryOutput struct {
	DurationHours  float64 `json:"duration_hours"`
	RidesCompleted int     `json:"rides_completed"`
	Earnings       float64 `json:"earnings"`
}

// UpdateLocationInput — входные данные для обновления локации
type UpdateLocationInput struct {
	DriverID       string  `json:"driver_id"`
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	AccuracyMeters float64 `json:"accuracy_meters,omitempty"`
	SpeedKmh       float64 `json:"speed_kmh,omitempty"`
	HeadingDegrees float64 `json:"heading_degrees,omitempty"`
}

// UpdateLocationOutput — результат обновления локации
type UpdateLocationOutput struct {
	CoordinateID string `json:"coordinate_id"`
	UpdatedAt    string `json:"updated_at"`
}

// StartRideInput — входные данные для старта поездки
type StartRideInput struct {
	DriverID  string  `json:"driver_id"`
	RideID    string  `json:"ride_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// StartRideOutput — результат старта поездки
type StartRideOutput struct {
	RideID    string `json:"ride_id"`
	Status    string `json:"status"`
	StartedAt string `json:"started_at"`
	Message   string `json:"message"`
}

// CompleteRideInput — входные данные для завершения поездки
type CompleteRideInput struct {
	DriverID              string  `json:"driver_id"`
	RideID                string  `json:"ride_id"`
	FinalLatitude         float64 `json:"final_latitude"`
	FinalLongitude        float64 `json:"final_longitude"`
	ActualDistanceKm      float64 `json:"actual_distance_km"`
	ActualDurationMinutes int     `json:"actual_duration_minutes"`
}

// CompleteRideOutput — результат завершения поездки
type CompleteRideOutput struct {
	RideID         string  `json:"ride_id"`
	Status         string  `json:"status"`
	CompletedAt    string  `json:"completed_at"`
	DriverEarnings float64 `json:"driver_earnings"`
	Message        string  `json:"message"`
}
