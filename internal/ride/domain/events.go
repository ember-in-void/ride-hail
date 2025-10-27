package domain

import "time"

// RideEvent — событие, которое отправляется в RabbitMQ.
type RideEvent struct {
	ID        string    `json:"id" db:"id"`
	RideID    string    `json:"ride_id" db:"ride_id"`
	EventType string    `json:"event_type" db:"event_type"`
	EventData any       `json:"event_data" db:"event_data"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
