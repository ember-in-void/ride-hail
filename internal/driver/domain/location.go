package domain

import "time"

type Coordinates struct {
	ID              string    `json:"id" db:"id"`
	EntityID        string    `json:"entity_id" db:"entity_id"`
	EntityType      string    `json:"entity_type" db:"entity_type"` // driver | passenger
	Address         string    `json:"address" db:"address"`
	Latitude        float64   `json:"latitude" db:"latitude"`
	Longitude       float64   `json:"longitude" db:"longitude"`
	FareAmount      *float64  `json:"fare_amount,omitempty" db:"fare_amount"`
	DistanceKM      *float64  `json:"distance_km,omitempty" db:"distance_km"`
	DurationMinutes *int      `json:"duration_minutes,omitempty" db:"duration_minutes"`
	IsCurrent       bool      `json:"is_current" db:"is_current"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}
