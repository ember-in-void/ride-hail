package in_ws

import (
	"encoding/json"
	"fmt"
	"net/http"

	"ridehail/internal/shared/auth"
	"ridehail/internal/shared/logger"
	"ridehail/internal/shared/ws"
)

// PassengerWSHandler обрабатывает WebSocket соединения для пассажиров
type PassengerWSHandler struct {
	hub    *ws.Hub
	jwtSvc *auth.JWTService
	log    *logger.Logger
}

// NewPassengerWSHandler создает новый handler для пассажиров
func NewPassengerWSHandler(jwtSvc *auth.JWTService, log *logger.Logger) *PassengerWSHandler {
	// Создаем auth функцию для валидации токенов
	authFunc := func(token string) (userID, role string, err error) {
		claims, err := jwtSvc.ValidateToken(token)
		if err != nil {
			return "", "", err
		}

		// Проверяем, что пользователь - PASSENGER или ADMIN
		if claims.Role != "PASSENGER" && claims.Role != "ADMIN" {
			return "", "", fmt.Errorf("invalid role: %s (expected PASSENGER or ADMIN)", claims.Role)
		}

		return claims.UserID, claims.Role, nil
	}

	hub := ws.NewHub(authFunc, log)

	handler := &PassengerWSHandler{
		hub:    hub,
		jwtSvc: jwtSvc,
		log:    log,
	}

	// Устанавливаем обработчик входящих сообщений
	hub.SetMessageHandler(handler.handleMessage)

	return handler
}

// GetHub возвращает WebSocket hub
func (h *PassengerWSHandler) GetHub() *ws.Hub {
	return h.hub
}

// ServeWS обрабатывает WebSocket соединение для пассажира
func (h *PassengerWSHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
	h.hub.ServeWS(w, r)
}

// handleMessage обрабатывает входящие сообщения от пассажиров
func (h *PassengerWSHandler) handleMessage(client *ws.Client, msgType string, data json.RawMessage) error {
	h.log.Info(logger.Entry{
		Action:  "passenger_ws_message",
		Message: msgType,
		Additional: map[string]any{
			"user_id":  client.UserID,
			"msg_type": msgType,
		},
	})

	switch msgType {
	case "ping":
		// Ответ на ping
		return h.hub.SendTypedMessage(client.UserID, "pong", map[string]string{
			"status": "ok",
		})

	default:
		h.log.Warn(logger.Entry{
			Action:  "passenger_ws_unknown_message_type",
			Message: msgType,
			Additional: map[string]any{
				"user_id": client.UserID,
			},
		})
	}

	return nil
}

// SendRideStatusUpdate отправляет обновление статуса поездки пассажиру
func (h *PassengerWSHandler) SendRideStatusUpdate(passengerID, rideID, status, message string, additionalData map[string]interface{}) error {
	data := map[string]interface{}{
		"ride_id": rideID,
		"status":  status,
		"message": message,
	}

	// Добавляем дополнительные данные
	for k, v := range additionalData {
		data[k] = v
	}

	return h.hub.SendTypedMessage(passengerID, "ride_status_update", data)
}

// SendDriverLocationUpdate отправляет обновление локации водителя пассажиру
func (h *PassengerWSHandler) SendDriverLocationUpdate(passengerID, rideID string, location map[string]interface{}) error {
	data := map[string]interface{}{
		"ride_id":         rideID,
		"driver_location": location,
	}

	return h.hub.SendTypedMessage(passengerID, "driver_location_update", data)
}

// SendMatchNotification отправляет уведомление о найденном водителе
func (h *PassengerWSHandler) SendMatchNotification(passengerID, rideID string, driverInfo map[string]interface{}) error {
	data := map[string]interface{}{
		"ride_id":     rideID,
		"driver_info": driverInfo,
		"status":      "MATCHED",
	}

	return h.hub.SendTypedMessage(passengerID, "ride_matched", data)
}
