package out

import (
	"context"
)

// RideOffer — предложение поездки для водителя
type RideOffer struct {
	OfferID                      string
	RideID                       string
	RideNumber                   string
	PickupLatitude               float64
	PickupLongitude              float64
	PickupAddress                string
	DestinationLatitude          float64
	DestinationLongitude         float64
	DestinationAddress           string
	EstimatedFare                float64
	DriverEarnings               float64
	DistanceToPickupKm           float64
	EstimatedRideDurationMinutes int
	ExpiresAt                    string
}

// RideDetails — детали поездки после принятия
type RideDetails struct {
	RideID          string
	PassengerName   string
	PassengerPhone  string
	PickupLatitude  float64
	PickupLongitude float64
	PickupAddress   string
	PickupNotes     string
}

// DriverNotifier — отправка уведомлений водителям через WebSocket
type DriverNotifier interface {
	// SendRideOffer отправляет предложение поездки водителю
	SendRideOffer(ctx context.Context, driverID string, offer RideOffer) error

	// SendRideDetails отправляет детали поездки после принятия
	SendRideDetails(ctx context.Context, driverID string, details RideDetails) error

	// IsDriverConnected проверяет, подключен ли водитель к WebSocket
	IsDriverConnected(driverID string) bool
}
