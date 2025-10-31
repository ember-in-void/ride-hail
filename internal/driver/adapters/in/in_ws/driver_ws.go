package in_ws

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	messaging "ridehail/internal/driver/adapters/out/amqp"
	"ridehail/internal/driver/application/ports/out"
	"ridehail/internal/shared/auth"
	"ridehail/internal/shared/logger"
	"ridehail/internal/shared/ws"
)

// DriverWSHandler обрабатывает WebSocket соединения для водителей
type DriverWSHandler struct {
	hub          *ws.Hub
	jwtSvc       *auth.JWTService
	msgPublisher *messaging.MessagePublisher
	log          *logger.Logger
}

// NewDriverWSHandler создает новый handler для водителей
func NewDriverWSHandler(
	jwtSvc *auth.JWTService,
	msgPublisher *messaging.MessagePublisher,
	log *logger.Logger,
) *DriverWSHandler {
	// Создаем auth функцию для валидации токенов
	authFunc := func(token string) (userID, role string, err error) {
		claims, err := jwtSvc.ValidateToken(token)
		if err != nil {
			return "", "", err
		}

		// Проверяем, что пользователь - DRIVER
		if claims.Role != "DRIVER" {
			return "", "", fmt.Errorf("invalid role: %s (expected DRIVER)", claims.Role)
		}

		return claims.UserID, claims.Role, nil
	}

	hub := ws.NewHub(authFunc, log)

	handler := &DriverWSHandler{
		hub:          hub,
		jwtSvc:       jwtSvc,
		msgPublisher: msgPublisher,
		log:          log,
	}

	// Устанавливаем обработчик входящих сообщений
	hub.SetMessageHandler(handler.handleMessage)

	return handler
}

// GetHub возвращает WebSocket hub
func (h *DriverWSHandler) GetHub() *ws.Hub {
	return h.hub
}

// ServeWS обрабатывает WebSocket соединение для водителя
func (h *DriverWSHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
	h.hub.ServeWS(w, r)
}

// RideResponseMessage структура ответа водителя на оффер
type RideResponseMessage struct {
	OfferID         string        `json:"offer_id"`
	RideID          string        `json:"ride_id"`
	Accepted        bool          `json:"accepted"`
	CurrentLocation *LocationData `json:"current_location,omitempty"`
}

// LocationUpdateMessage структура обновления локации от водителя
type LocationUpdateMessage struct {
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	AccuracyMeters float64 `json:"accuracy_meters,omitempty"`
	SpeedKmh       float64 `json:"speed_kmh,omitempty"`
	HeadingDegrees float64 `json:"heading_degrees,omitempty"`
}

// LocationData данные локации
type LocationData struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// handleMessage обрабатывает входящие сообщения от водителей
func (h *DriverWSHandler) handleMessage(client *ws.Client, msgType string, data json.RawMessage) error {
	h.log.Info(logger.Entry{
		Action:  "driver_ws_message",
		Message: msgType,
		Additional: map[string]any{
			"driver_id": client.UserID,
			"msg_type":  msgType,
		},
	})

	switch msgType {
	case "ping":
		// Ответ на ping
		return h.hub.SendTypedMessage(client.UserID, "pong", map[string]string{
			"status": "ok",
		})

	case "ride_response":
		// Обработка ответа водителя на оффер
		var resp RideResponseMessage
		if err := json.Unmarshal(data, &resp); err != nil {
			return fmt.Errorf("invalid ride_response format: %w", err)
		}

		h.log.Info(logger.Entry{
			Action:  "driver_ride_response",
			Message: resp.RideID,
			Additional: map[string]any{
				"driver_id": client.UserID,
				"ride_id":   resp.RideID,
				"accepted":  resp.Accepted,
			},
		})

		// Публикуем ответ в RabbitMQ driver.response.{ride_id}
		dto := &out.DriverResponseDTO{
			RideID:   resp.RideID,
			DriverID: client.UserID,
			Accepted: resp.Accepted,
		}

		// Добавляем текущую локацию, если она была передана
		if resp.CurrentLocation != nil {
			dto.DriverLocation = out.LocationDTO{
				Lat: resp.CurrentLocation.Latitude,
				Lng: resp.CurrentLocation.Longitude,
			}
		}

		if err := h.msgPublisher.PublishDriverResponse(context.Background(), dto); err != nil {
			h.log.Error(logger.Entry{
				Action:  "publish_driver_response_failed",
				Message: err.Error(),
				Additional: map[string]any{
					"ride_id":   resp.RideID,
					"driver_id": client.UserID,
				},
				Error: &logger.ErrObj{Msg: err.Error()},
			})
			return fmt.Errorf("failed to publish driver response: %w", err)
		}

		h.log.Info(logger.Entry{
			Action:  "driver_response_published",
			Message: fmt.Sprintf("published driver.response.%s", resp.RideID),
			Additional: map[string]any{
				"ride_id":   resp.RideID,
				"driver_id": client.UserID,
				"accepted":  resp.Accepted,
			},
		})

	case "location_update":
		// Обработка обновления локации
		var loc LocationUpdateMessage
		if err := json.Unmarshal(data, &loc); err != nil {
			return fmt.Errorf("invalid location_update format: %w", err)
		}

		h.log.Debug(logger.Entry{
			Action:  "driver_location_update",
			Message: client.UserID,
			Additional: map[string]any{
				"driver_id": client.UserID,
				"latitude":  loc.Latitude,
				"longitude": loc.Longitude,
			},
		})

		// TODO: Здесь должна быть логика обновления локации в БД и публикации в RabbitMQ
		// Пока просто логируем

	default:
		h.log.Warn(logger.Entry{
			Action:  "driver_ws_unknown_message_type",
			Message: msgType,
			Additional: map[string]any{
				"driver_id": client.UserID,
			},
		})
	}

	return nil
}

// SendRideOffer отправляет оффер поездки водителю
func (h *DriverWSHandler) SendRideOffer(driverID string, offer map[string]interface{}) error {
	return h.hub.SendTypedMessage(driverID, "ride_offer", offer)
}

// SendRideDetails отправляет детали поездки водителю после принятия
func (h *DriverWSHandler) SendRideDetails(driverID string, details map[string]interface{}) error {
	return h.hub.SendTypedMessage(driverID, "ride_details", details)
}

// SendRideStatusUpdate отправляет обновление статуса поездки водителю
func (h *DriverWSHandler) SendRideStatusUpdate(driverID, rideID, status, message string) error {
	data := map[string]interface{}{
		"ride_id": rideID,
		"status":  status,
		"message": message,
	}

	return h.hub.SendTypedMessage(driverID, "ride_status_update", data)
}

// IsDriverConnected проверяет, подключен ли водитель
func (h *DriverWSHandler) IsDriverConnected(driverID string) bool {
	return h.hub.IsUserConnected(driverID)
}

// GetConnectedDrivers возвращает список подключенных водителей
func (h *DriverWSHandler) GetConnectedDrivers() []string {
	return h.hub.GetClientsByRole("DRIVER")
}
