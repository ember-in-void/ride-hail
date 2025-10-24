package model

import "time"

// DriverLocation — текущее состояние водителя.
type DriverLocation struct {
	DriverID  string    `json:"driver_id" db:"driver_id"`
	Latitude  float64   `json:"latitude" db:"latitude"`
	Longitude float64   `json:"longitude" db:"longitude"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// LocationUpdate — DTO от WebSocket или HTTP клиента.
type LocationUpdate struct {
	DriverID  string  `json:"driver_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
