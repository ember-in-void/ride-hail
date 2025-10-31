package out

import (
	"context"
)

// Ride — минимальная модель поездки (read-only для Driver Service)
type Ride struct {
	ID                      string
	RideNumber              string
	PassengerID             string
	DriverID                *string
	VehicleType             string
	Status                  string
	PickupCoordinateID      string
	DestinationCoordinateID string
	EstimatedFare           float64
	FinalFare               *float64
}

// RideRepository — репозиторий для чтения информации о поездках
type RideRepository interface {
	// FindByID находит поездку по ID
	FindByID(ctx context.Context, rideID string) (*Ride, error)

	// UpdateRideDriver назначает водителя на поездку (MATCHED)
	UpdateRideDriver(ctx context.Context, rideID, driverID string) error

	// UpdateRideStatus обновляет статус поездки
	UpdateRideStatus(ctx context.Context, rideID, status string) error

	// UpdateFinalFare обновляет финальную стоимость поездки
	UpdateFinalFare(ctx context.Context, rideID string, finalFare float64) error
}
