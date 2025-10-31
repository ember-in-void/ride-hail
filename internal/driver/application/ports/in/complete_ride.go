package in

import "context"

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

// CompleteRideUseCase — use-case для завершения поездки
type CompleteRideUseCase interface {
	Execute(ctx context.Context, input CompleteRideInput) (*CompleteRideOutput, error)
}
