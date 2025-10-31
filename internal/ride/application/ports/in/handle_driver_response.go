package in

import "context"

// HandleDriverResponseInput — входные данные для обработки ответа водителя
type HandleDriverResponseInput struct {
	RideID                  string
	DriverID                string
	Accepted                bool
	EstimatedArrivalMinutes int
	DriverLocationLat       float64
	DriverLocationLng       float64
}

// HandleDriverResponseOutput — результат обработки ответа водителя
type HandleDriverResponseOutput struct {
	RideID         string
	Status         string
	DriverAssigned bool
	PassengerID    string // для отправки WebSocket уведомления
}

// HandleDriverResponseUseCase — интерфейс use case обработки ответа водителя
type HandleDriverResponseUseCase interface {
	Execute(ctx context.Context, input HandleDriverResponseInput) (*HandleDriverResponseOutput, error)
}
