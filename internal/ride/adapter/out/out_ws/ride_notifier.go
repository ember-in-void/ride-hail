package out_ws

import (
	"context"

	"ridehail/internal/ride/application/ports/out"
	"ridehail/internal/shared/logger"
	"ridehail/internal/shared/ws"
)

// WsRideNotifier отправляет уведомления через WebSocket
type WsRideNotifier struct {
	hub *ws.Hub
	log *logger.Logger
}

// NewWsRideNotifier создает новый notifier
func NewWsRideNotifier(hub *ws.Hub, log *logger.Logger) *WsRideNotifier {
	return &WsRideNotifier{
		hub: hub,
		log: log,
	}
}

// NotifyPassenger отправляет уведомление пассажиру
func (n *WsRideNotifier) NotifyPassenger(ctx context.Context, passengerID string, notification out.RideNotification) error {
	if err := n.hub.SendToUserJSON(passengerID, notification); err != nil {
		n.log.Error(logger.Entry{
			Action:  "notify_passenger_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
			Additional: map[string]any{
				"passenger_id": passengerID,
				"ride_id":      notification.RideID,
			},
		})
		return err
	}

	n.log.Debug(logger.Entry{
		Action:  "passenger_notified",
		Message: notification.Type,
		RideID:  notification.RideID,
		Additional: map[string]any{
			"passenger_id": passengerID,
		},
	})

	return nil
}

// NotifyDriver отправляет уведомление водителю
func (n *WsRideNotifier) NotifyDriver(ctx context.Context, driverID string, notification out.RideNotification) error {
	if err := n.hub.SendToUserJSON(driverID, notification); err != nil {
		n.log.Error(logger.Entry{
			Action:  "notify_driver_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
			Additional: map[string]any{
				"driver_id": driverID,
				"ride_id":   notification.RideID,
			},
		})
		return err
	}

	n.log.Debug(logger.Entry{
		Action:  "driver_notified",
		Message: notification.Type,
		RideID:  notification.RideID,
		Additional: map[string]any{
			"driver_id": driverID,
		},
	})

	return nil
}

// BroadcastRideUpdate отправляет обновление всем (админка)
func (n *WsRideNotifier) BroadcastRideUpdate(ctx context.Context, notification out.RideNotification) error {
	if err := n.hub.BroadcastJSON(notification); err != nil {
		n.log.Error(logger.Entry{
			Action:  "broadcast_ride_update_failed",
			Message: err.Error(),
			RideID:  notification.RideID,
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		return err
	}

	n.log.Debug(logger.Entry{
		Action:  "ride_update_broadcasted",
		Message: notification.Type,
		RideID:  notification.RideID,
	})

	return nil
}
