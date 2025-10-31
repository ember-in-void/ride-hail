package in

import "context"

// StartRideInput — входные данные для начала поездки
type StartRideInput struct {
	DriverID  string  `json:"driver_id"`
	RideID    string  `json:"ride_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// StartRideOutput — результат начала поездки
type StartRideOutput struct {
	RideID    string `json:"ride_id"`
	Status    string `json:"status"`
	StartedAt string `json:"started_at"`
	Message   string `json:"message"`
}

// StartRideUseCase — use-case для начала поездки
type StartRideUseCase interface {
	Execute(ctx context.Context, input StartRideInput) (*StartRideOutput, error)
}
