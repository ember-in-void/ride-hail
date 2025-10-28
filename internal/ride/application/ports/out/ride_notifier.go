package out

import (
	"context"
)

// RideNotification — уведомление о поездке через WebSocket
type RideNotification struct {
	Type    string                 `json:"type"` // ride_requested | ride_matched | etc
	RideID  string                 `json:"ride_id"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

// RideNotifier — интерфейс для отправки WebSocket уведомлений
type RideNotifier interface {
	// NotifyPassenger отправляет уведомление пассажиру
	NotifyPassenger(ctx context.Context, passengerID string, notification RideNotification) error

	// NotifyDriver отправляет уведомление водителю
	NotifyDriver(ctx context.Context, driverID string, notification RideNotification) error

	// BroadcastRideUpdate отправляет обновление всем заинтересованным (админка)
	BroadcastRideUpdate(ctx context.Context, notification RideNotification) error
}
