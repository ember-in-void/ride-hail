package out

import "context"

// RideRepository определяет операции с поездками в БД
type RideRepository interface {
	// FindByID находит поездку по ID
	FindByID(ctx context.Context, rideID string) (*Ride, error)

	// UpdateRideDriver обновляет водителя для поездки и меняет статус на MATCHED
	UpdateRideDriver(ctx context.Context, rideID, driverID string) error

	// UpdateRideStatus обновляет статус поездки
	UpdateRideStatus(ctx context.Context, rideID, status string) error

	// UpdateFinalFare обновляет финальную стоимость поездки
	UpdateFinalFare(ctx context.Context, rideID string, finalFare float64) error
}

// Ride — упрощенная модель поездки для driver service
type Ride struct {
	ID                      string   `json:"id" db:"id"`
	RideNumber              string   `json:"ride_number" db:"ride_number"`
	PassengerID             string   `json:"passenger_id" db:"passenger_id"`
	DriverID                *string  `json:"driver_id,omitempty" db:"driver_id"`
	VehicleType             string   `json:"vehicle_type" db:"vehicle_type"`
	Status                  string   `json:"status" db:"status"`
	PickupCoordinateID      *string  `json:"pickup_coordinate_id,omitempty" db:"pickup_coordinate_id"`
	DestinationCoordinateID *string  `json:"destination_coordinate_id,omitempty" db:"destination_coordinate_id"`
	EstimatedFare           *float64 `json:"estimated_fare,omitempty" db:"estimated_fare"`
	FinalFare               *float64 `json:"final_fare,omitempty" db:"final_fare"`
}
