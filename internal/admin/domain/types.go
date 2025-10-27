package domain

import "time"

type RideSummary struct {
	RideID    string    `json:"ride_id" db:"ride_id"`
	Status    string    `json:"status" db:"status"`
	DriverID  string    `json:"driver_id" db:"driver_id"`
	UserID    string    `json:"user_id" db:"user_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type Stats struct {
	TotalRides     int `json:"total_rides"`
	CompletedRides int `json:"completed_rides"`
	CancelledRides int `json:"cancelled_rides"`
}
