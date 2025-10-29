package transport

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"ridehail/internal/ride/application/ports/in"
	"ridehail/internal/ride/domain"
	"ridehail/internal/shared/logger"
)

const maxBodySize = 1 << 20 // 1MB

// HTTPHandler обрабатывает HTTP запросы для Ride Service
type HTTPHandler struct {
	requestRideUC in.RequestRideUseCase
	log           *logger.Logger
}

// NewHTTPHandler создает новый HTTP handler
func NewHTTPHandler(requestRideUC in.RequestRideUseCase, log *logger.Logger) *HTTPHandler {
	return &HTTPHandler{
		requestRideUC: requestRideUC,
		log:           log,
	}
}

// RegisterRoutes регистрирует все HTTP маршруты
func (h *HTTPHandler) RegisterRoutes(mux *http.ServeMux, authMiddleware func(http.HandlerFunc) http.HandlerFunc) {
	// liveness
	mux.HandleFunc("GET /health", h.handleHealth)

	// ride request
	mux.HandleFunc("POST /rides", authMiddleware(h.handleRequestRide))
}

// handleHealth обрабатывает health check
func (h *HTTPHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// RequestRideHTTPRequest — HTTP DTO для запроса поездки
type RequestRideHTTPRequest struct {
	VehicleType   string  `json:"vehicle_type"`
	PickupLat     float64 `json:"pickup_lat"`
	PickupLng     float64 `json:"pickup_lng"`
	PickupAddress string  `json:"pickup_address"`
	DestLat       float64 `json:"destination_lat"`
	DestLng       float64 `json:"destination_lng"`
	DestAddress   string  `json:"destination_address"`
	Priority      int     `json:"priority,omitempty"`
}

// handleRequestRide обрабатывает POST /rides
func (h *HTTPHandler) handleRequestRide(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Извлекаем user_id из контекста (добавлен JWT middleware)
	userID, ok := ctx.Value("user_id").(string)
	if !ok || userID == "" {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Ограничиваем размер тела запроса
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	// Парсим JSON
	var req RequestRideHTTPRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		if errors.Is(err, io.EOF) {
			h.respondError(w, http.StatusBadRequest, "empty request body")
			return
		}
		h.log.Error(logger.Entry{
			Action:  "parse_request_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		h.respondError(w, http.StatusBadRequest, "invalid request format")
		return
	}

	// Валидация обязательных полей
	if req.VehicleType == "" {
		h.respondError(w, http.StatusBadRequest, "vehicle_type is required")
		return
	}
	if req.PickupAddress == "" {
		h.respondError(w, http.StatusBadRequest, "pickup_address is required")
		return
	}
	if req.DestAddress == "" {
		h.respondError(w, http.StatusBadRequest, "destination_address is required")
		return
	}

	// Маппинг HTTP DTO → Use Case Input
	input := in.RequestRideInput{
		PassengerID:   userID,
		VehicleType:   req.VehicleType,
		PickupLat:     req.PickupLat,
		PickupLng:     req.PickupLng,
		PickupAddress: req.PickupAddress,
		DestLat:       req.DestLat,
		DestLng:       req.DestLng,
		DestAddress:   req.DestAddress,
		Priority:      req.Priority,
	}

	output, err := h.requestRideUC.Execute(ctx, input)
	if err != nil {
		h.handleUseCaseError(w, err)
		return
	}

	h.respondJSON(w, http.StatusCreated, output)
}

// handleUseCaseError обрабатывает ошибки use case
func (h *HTTPHandler) handleUseCaseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidCoordinates):
		h.respondError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrInvalidVehicleType):
		h.respondError(w, http.StatusBadRequest, "invalid vehicle type")
	case errors.Is(err, domain.ErrUnauthorized):
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
	default:
		h.log.Error(logger.Entry{
			Action:  "usecase_error",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		h.respondError(w, http.StatusInternalServerError, "internal server error")
	}
}

// respondJSON отправляет JSON ответ
func (h *HTTPHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.log.Error(logger.Entry{
			Action:  "encode_response_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
	}
}

// respondError отправляет JSON с ошибкой
func (h *HTTPHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}
