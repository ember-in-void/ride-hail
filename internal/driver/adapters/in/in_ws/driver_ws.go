package in_ws

import (
	"context"
	"encoding/json"
	"fmt"

	"ridehail/internal/driver/application/ports/out"
	"ridehail/internal/shared/logger"
	"ridehail/internal/shared/ws"
)

// Hub — управление WebSocket соединениями водителей и обработка ride responses
type Hub struct {
	wsHub          *ws.Hub
	eventPublisher out.EventPublisher
	rideRepo       out.RideRepository
	log            *logger.Logger
}

func NewHub(wsHub *ws.Hub, eventPublisher out.EventPublisher, rideRepo out.RideRepository, log *logger.Logger) *Hub {
	return &Hub{
		wsHub:          wsHub,
		eventPublisher: eventPublisher,
		rideRepo:       rideRepo,
		log:            log,
	}
}

// Start запускает обработку входящих сообщений от водителей
func (h *Hub) Start(ctx context.Context) {
	h.log.Info(logger.Entry{
		Action:  "driver_ws_hub_started",
		Message: "driver websocket hub started processing messages",
	})

	for {
		select {
		case <-ctx.Done():
			h.log.Info(logger.Entry{
				Action:  "driver_ws_hub_stopped",
				Message: "driver websocket hub stopped",
			})
			return
		case msg := <-h.wsHub.Broadcast:
			h.handleIncomingMessage(ctx, msg)
		}
	}
}

func (h *Hub) handleIncomingMessage(ctx context.Context, msg ws.Message) {
	var payload map[string]any
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		h.log.Warn(logger.Entry{
			Action:  "driver_ws_invalid_message",
			Message: "failed to unmarshal message",
			Error:   &logger.ErrObj{Msg: err.Error()},
			Additional: map[string]any{
				"driver_id": msg.UserID,
			},
		})
		return
	}

	msgType, ok := payload["type"].(string)
	if !ok {
		h.log.Warn(logger.Entry{
			Action:  "driver_ws_missing_type",
			Message: "message type missing",
			Additional: map[string]any{
				"driver_id": msg.UserID,
			},
		})
		return
	}

	switch msgType {
	case "ride_response":
		h.handleRideResponse(ctx, msg.UserID, payload)
	case "location_update":
		h.handleLocationUpdate(ctx, msg.UserID, payload)
	default:
		h.log.Debug(logger.Entry{
			Action:  "driver_ws_unknown_message_type",
			Message: fmt.Sprintf("unknown message type: %s", msgType),
			Additional: map[string]any{
				"driver_id": msg.UserID,
				"type":      msgType,
			},
		})
	}
}

func (h *Hub) handleRideResponse(ctx context.Context, driverID string, payload map[string]any) {
	rideID, ok := payload["ride_id"].(string)
	if !ok {
		h.log.Warn(logger.Entry{
			Action:  "ride_response_missing_ride_id",
			Message: "ride_id missing in ride_response",
			Additional: map[string]any{
				"driver_id": driverID,
			},
		})
		return
	}

	accepted, ok := payload["accepted"].(bool)
	if !ok {
		h.log.Warn(logger.Entry{
			Action:  "ride_response_missing_accepted",
			Message: "accepted field missing in ride_response",
			Additional: map[string]any{
				"driver_id": driverID,
				"ride_id":   rideID,
			},
		})
		return
	}

	h.log.Info(logger.Entry{
		Action:  "ride_response_received",
		Message: "driver responded to ride offer",
		RideID:  rideID,
		Additional: map[string]any{
			"driver_id": driverID,
			"accepted":  accepted,
		},
	})

	// Если водитель принял поездку — обновляем ride и публикуем событие
	if accepted {
		// Обновляем ride.driver_id и ride.status = MATCHED
		if err := h.rideRepo.UpdateRideDriver(ctx, rideID, driverID); err != nil {
			h.log.Error(logger.Entry{
				Action:  "ride_response_update_failed",
				Message: err.Error(),
				Error:   &logger.ErrObj{Msg: err.Error()},
				RideID:  rideID,
				Additional: map[string]any{
					"driver_id": driverID,
				},
			})
			return
		}

		// Получаем driver info для отправки в Ride Service
		driverInfo := map[string]any{
			"driver_id": driverID,
			"rating":    4.8, // TODO: получать из driver repository
			"vehicle": map[string]string{
				"make":  "Toyota",
				"model": "Camry",
				"color": "White",
				"plate": "KZ 123 ABC",
			},
		}

		// Публикуем driver.response.{ride_id} в RabbitMQ
		if err := h.eventPublisher.PublishDriverResponse(ctx, rideID, driverID, true, driverInfo); err != nil {
			h.log.Error(logger.Entry{
				Action:  "ride_response_publish_failed",
				Message: err.Error(),
				Error:   &logger.ErrObj{Msg: err.Error()},
				RideID:  rideID,
			})
		}
	} else {
		// Водитель отклонил — просто публикуем событие
		if err := h.eventPublisher.PublishDriverResponse(ctx, rideID, driverID, false, nil); err != nil {
			h.log.Error(logger.Entry{
				Action:  "ride_response_publish_failed",
				Message: err.Error(),
				Error:   &logger.ErrObj{Msg: err.Error()},
				RideID:  rideID,
			})
		}
	}
}

func (h *Hub) handleLocationUpdate(ctx context.Context, driverID string, payload map[string]any) {
	// WebSocket location updates обрабатываются аналогично HTTP POST /location
	// Можно переиспользовать UpdateLocationUseCase, но для простоты пока логируем
	h.log.Debug(logger.Entry{
		Action:  "location_update_via_ws",
		Message: "driver sent location update via websocket",
		Additional: map[string]any{
			"driver_id": driverID,
		},
	})
}
