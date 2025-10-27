package domain

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

type User struct {
	ID           string    `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	Role         string    `json:"role" db:"role"`
	Status       string    `json:"status" db:"status"`
	PasswordHash string    `json:"-" db:"password_hash"`
	Attrs        any       `json:"attrs" db:"attrs"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type Event struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data"`
}
