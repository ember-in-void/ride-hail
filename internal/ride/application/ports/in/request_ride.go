package in

import "context"

// RequestRideInput — входные данные для создания поездки
type RequestRideInput struct {
	PassengerID   string  `json:"passenger_id"`
	VehicleType   string  `json:"vehicle_type"`
	PickupLat     float64 `json:"pickup_lat"`
	PickupLng     float64 `json:"pickup_lng"`
	PickupAddress string  `json:"pickup_address"`
	DestLat       float64 `json:"destination_lat"`
	DestLng       float64 `json:"destination_lng"`
	DestAddress   string  `json:"destination_address"`
	Priority      int     `json:"priority"` // 1-10, по умолчанию 1
}

// RequestRideOutput — результат создания поездки
type RequestRideOutput struct {
	RideID        string  `json:"ride_id"`
	RideNumber    string  `json:"ride_number"`
	Status        string  `json:"status"`
	EstimatedFare float64 `json:"estimated_fare"`
	PickupAddress string  `json:"pickup_address"`
	DestAddress   string  `json:"destination_address"`
}

// RequestRideUseCase — интерфейс use-case для запроса поездки
type RequestRideUseCase interface {
	Execute(ctx context.Context, input RequestRideInput) (*RequestRideOutput, error)
}
