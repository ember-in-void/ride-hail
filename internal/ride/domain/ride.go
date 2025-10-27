package domain

import "time"

// Ride представляет основную сущность поездки.
type Ride struct {
	ID                      string     `json:"id" db:"id"`
	RideNumber              string     `json:"ride_number" db:"ride_number"`
	PassengerID             string     `json:"passenger_id" db:"passenger_id"`
	DriverID                *string    `json:"driver_id,omitempty" db:"driver_id"`
	VehicleType             string     `json:"vehicle_type" db:"vehicle_type"`
	Status                  string     `json:"status" db:"status"`
	Priority                int        `json:"priority" db:"priority"`
	RequestedAt             time.Time  `json:"requested_at" db:"requested_at"`
	MatchedAt               *time.Time `json:"matched_at,omitempty" db:"matched_at"`
	ArrivedAt               *time.Time `json:"arrived_at,omitempty" db:"arrived_at"`
	StartedAt               *time.Time `json:"started_at,omitempty" db:"started_at"`
	CompletedAt             *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	CancelledAt             *time.Time `json:"cancelled_at,omitempty" db:"cancelled_at"`
	CancellationReason      *string    `json:"cancellation_reason,omitempty" db:"cancellation_reason"`
	EstimatedFare           *float64   `json:"estimated_fare,omitempty" db:"estimated_fare"`
	FinalFare               *float64   `json:"final_fare,omitempty" db:"final_fare"`
	PickupCoordinateID      string     `json:"pickup_coordinate_id" db:"pickup_coordinate_id"`
	DestinationCoordinateID string     `json:"destination_coordinate_id" db:"destination_coordinate_id"`
	CreatedAt               time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at" db:"updated_at"`
}

// RideRequest — DTO для создания поездки через HTTP.
type RideRequest struct {
	PickupLat      float64 `json:"pickup_lat"`
	PickupLng      float64 `json:"pickup_lng"`
	DestinationLat float64 `json:"destination_lat"`
	DestinationLng float64 `json:"destination_lng"`
}
