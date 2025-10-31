package in

import (
	"context"
	"time"
)

// GetActiveRidesInput — входные данные для получения активных поездок
type GetActiveRidesInput struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// GetActiveRidesOutput — выходные данные активных поездок
type GetActiveRidesOutput struct {
	Rides      []ActiveRideInfo `json:"rides"`
	TotalCount int              `json:"total_count"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
}

// ActiveRideInfo — информация об активной поездке
type ActiveRideInfo struct {
	RideID                string        `json:"ride_id"`
	RideNumber            string        `json:"ride_number"`
	Status                string        `json:"status"`
	PassengerID           string        `json:"passenger_id"`
	DriverID              *string       `json:"driver_id,omitempty"`
	PickupAddress         string        `json:"pickup_address"`
	DestinationAddress    string        `json:"destination_address"`
	StartedAt             *time.Time    `json:"started_at,omitempty"`
	EstimatedCompletion   *time.Time    `json:"estimated_completion,omitempty"`
	CurrentDriverLocation *LocationInfo `json:"current_driver_location,omitempty"`
	DistanceCompletedKm   *float64      `json:"distance_completed_km,omitempty"`
	DistanceRemainingKm   *float64      `json:"distance_remaining_km,omitempty"`
}

// LocationInfo — информация о локации
type LocationInfo struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// GetActiveRidesUseCase — use case для получения активных поездок
type GetActiveRidesUseCase interface {
	Execute(ctx context.Context, input GetActiveRidesInput) (*GetActiveRidesOutput, error)
}
