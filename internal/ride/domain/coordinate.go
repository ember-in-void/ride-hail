package domain

import (
	"fmt"
	"time"
)

// Coordinate представляет географическую точку
type Coordinate struct {
	ID              string    `json:"id" db:"id"`
	EntityID        string    `json:"entity_id" db:"entity_id"`
	EntityType      string    `json:"entity_type" db:"entity_type"` // driver | passenger
	Address         string    `json:"address" db:"address"`
	Latitude        float64   `json:"latitude" db:"latitude"`
	Longitude       float64   `json:"longitude" db:"longitude"`
	FareAmount      *float64  `json:"fare_amount,omitempty" db:"fare_amount"`
	DistanceKm      *float64  `json:"distance_km,omitempty" db:"distance_km"`
	DurationMinutes *int      `json:"duration_minutes,omitempty" db:"duration_minutes"`
	IsCurrent       bool      `json:"is_current" db:"is_current"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// ValidateCoordinates проверяет корректность координат
func ValidateCoordinates(lat, lng float64) error {
	if lat < -90 || lat > 90 {
		return fmt.Errorf("%w: latitude must be between -90 and 90", ErrInvalidCoordinates)
	}
	if lng < -180 || lng > 180 {
		return fmt.Errorf("%w: longitude must be between -180 and 180", ErrInvalidCoordinates)
	}
	return nil
}
