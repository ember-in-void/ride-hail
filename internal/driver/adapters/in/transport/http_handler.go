package transport

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"ridehail/internal/driver/application/ports/in"
	"ridehail/internal/shared/logger"
)

// DriverHandler обрабатывает HTTP запросы к API водителей
type DriverHandler struct {
	driverUseCase in.DriverUseCase
	log           *logger.Logger
}

// NewDriverHandler создает новый хендлер для водителей
func NewDriverHandler(driverUseCase in.DriverUseCase, log *logger.Logger) *DriverHandler {
	return &DriverHandler{
		driverUseCase: driverUseCase,
		log:           log,
	}
}

// HandleGoOnline обрабатывает POST /drivers/{driver_id}/online
func (h *DriverHandler) HandleGoOnline(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Извлекаем driver_id из URL path
	driverIDFromURL := extractDriverID(r.URL.Path)
	if driverIDFromURL == "" {
		h.log.Error(logger.Entry{
			Action:  "go_online_missing_driver_id",
			Message: "driver_id not found in URL",
		})
		writeJSONError(w, "driver_id is required", http.StatusBadRequest)
		return
	}

	// Извлекаем user_id из JWT токена
	userIDFromToken := GetUserID(ctx)
	role := GetRole(ctx)

	// Проверяем роль
	if role != "DRIVER" {
		h.log.Error(logger.Entry{
			Action:  "go_online_invalid_role",
			Message: fmt.Sprintf("expected role DRIVER, got %s", role),
		})
		writeJSONError(w, "only drivers can go online", http.StatusForbidden)
		return
	}

	// Проверяем, что driver_id из URL совпадает с user_id из токена (безопасность)
	if driverIDFromURL != userIDFromToken {
		h.log.Error(logger.Entry{
			Action:  "go_online_id_mismatch",
			Message: fmt.Sprintf("driver_id from URL (%s) != user_id from token (%s)", driverIDFromURL, userIDFromToken),
		})
		writeJSONError(w, "driver_id mismatch", http.StatusForbidden)
		return
	}

	// Декодируем тело запроса
	var req GoOnlineRequest
	body := http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB limit
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		h.log.Error(logger.Entry{
			Action:  "go_online_decode_failed",
			Message: err.Error(),
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
		writeJSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Вызываем use-case
	output, err := h.driverUseCase.GoOnline(ctx, in.GoOnlineInput{
		DriverID:  driverIDFromURL,
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
	})
	if err != nil {
		h.log.Error(logger.Entry{
			Action:  "go_online_usecase_failed",
			Message: err.Error(),
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Возвращаем успешный ответ
	writeJSON(w, GoOnlineResponse{
		Status:    output.Status,
		SessionID: output.SessionID,
		Message:   output.Message,
	}, http.StatusOK)
}

// HandleGoOffline обрабатывает POST /drivers/{driver_id}/offline
func (h *DriverHandler) HandleGoOffline(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Извлекаем driver_id из URL path
	driverIDFromURL := extractDriverID(r.URL.Path)
	if driverIDFromURL == "" {
		h.log.Error(logger.Entry{
			Action:  "go_offline_missing_driver_id",
			Message: "driver_id not found in URL",
		})
		writeJSONError(w, "driver_id is required", http.StatusBadRequest)
		return
	}

	// Извлекаем user_id из JWT токена
	userIDFromToken := GetUserID(ctx)
	role := GetRole(ctx)

	// Проверяем роль
	if role != "DRIVER" {
		h.log.Error(logger.Entry{
			Action:  "go_offline_invalid_role",
			Message: fmt.Sprintf("expected role DRIVER, got %s", role),
		})
		writeJSONError(w, "only drivers can go offline", http.StatusForbidden)
		return
	}

	// Проверяем, что driver_id из URL совпадает с user_id из токена
	if driverIDFromURL != userIDFromToken {
		h.log.Error(logger.Entry{
			Action:  "go_offline_id_mismatch",
			Message: fmt.Sprintf("driver_id from URL (%s) != user_id from token (%s)", driverIDFromURL, userIDFromToken),
		})
		writeJSONError(w, "driver_id mismatch", http.StatusForbidden)
		return
	}

	// Вызываем use-case
	output, err := h.driverUseCase.GoOffline(ctx, in.GoOfflineInput{
		DriverID: driverIDFromURL,
	})
	if err != nil {
		h.log.Error(logger.Entry{
			Action:  "go_offline_usecase_failed",
			Message: err.Error(),
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Возвращаем успешный ответ
	writeJSON(w, GoOfflineResponse{
		Status:    output.Status,
		SessionID: output.SessionID,
		SessionSummary: SessionSummaryResponse{
			DurationHours:  output.SessionSummary.DurationHours,
			RidesCompleted: output.SessionSummary.RidesCompleted,
			Earnings:       output.SessionSummary.Earnings,
		},
		Message: output.Message,
	}, http.StatusOK)
}

// HandleUpdateLocation обрабатывает POST /drivers/{driver_id}/location
func (h *DriverHandler) HandleUpdateLocation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Извлекаем driver_id из URL path
	driverIDFromURL := extractDriverID(r.URL.Path)
	if driverIDFromURL == "" {
		h.log.Error(logger.Entry{
			Action:  "update_location_missing_driver_id",
			Message: "driver_id not found in URL",
		})
		writeJSONError(w, "driver_id is required", http.StatusBadRequest)
		return
	}

	// Извлекаем user_id из JWT токена
	userIDFromToken := GetUserID(ctx)
	role := GetRole(ctx)

	// Проверяем роль
	if role != "DRIVER" {
		h.log.Error(logger.Entry{
			Action:  "update_location_invalid_role",
			Message: fmt.Sprintf("expected role DRIVER, got %s", role),
		})
		writeJSONError(w, "only drivers can update location", http.StatusForbidden)
		return
	}

	// Проверяем, что driver_id из URL совпадает с user_id из токена
	if driverIDFromURL != userIDFromToken {
		h.log.Error(logger.Entry{
			Action:  "update_location_id_mismatch",
			Message: fmt.Sprintf("driver_id from URL (%s) != user_id from token (%s)", driverIDFromURL, userIDFromToken),
		})
		writeJSONError(w, "driver_id mismatch", http.StatusForbidden)
		return
	}

	// Декодируем тело запроса
	var req UpdateLocationRequest
	body := http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB limit
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		h.log.Error(logger.Entry{
			Action:  "update_location_decode_failed",
			Message: err.Error(),
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
		writeJSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Вызываем use-case
	output, err := h.driverUseCase.UpdateLocation(ctx, in.UpdateLocationInput{
		DriverID:       driverIDFromURL,
		Latitude:       req.Latitude,
		Longitude:      req.Longitude,
		AccuracyMeters: req.AccuracyMeters,
		SpeedKmh:       req.SpeedKmh,
		HeadingDegrees: req.HeadingDegrees,
	})
	if err != nil {
		h.log.Error(logger.Entry{
			Action:  "update_location_usecase_failed",
			Message: err.Error(),
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Возвращаем успешный ответ
	writeJSON(w, UpdateLocationResponse{
		CoordinateID: output.CoordinateID,
		UpdatedAt:    output.UpdatedAt,
	}, http.StatusOK)
}

// HandleStartRide обрабатывает POST /drivers/{driver_id}/start
func (h *DriverHandler) HandleStartRide(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Извлекаем driver_id из URL path
	driverIDFromURL := extractDriverID(r.URL.Path)
	if driverIDFromURL == "" {
		h.log.Error(logger.Entry{
			Action:  "start_ride_missing_driver_id",
			Message: "driver_id not found in URL",
		})
		writeJSONError(w, "driver_id is required", http.StatusBadRequest)
		return
	}

	// Извлекаем user_id из JWT токена
	userIDFromToken := GetUserID(ctx)
	role := GetRole(ctx)

	// Проверяем роль
	if role != "DRIVER" {
		h.log.Error(logger.Entry{
			Action:  "start_ride_invalid_role",
			Message: fmt.Sprintf("expected role DRIVER, got %s", role),
		})
		writeJSONError(w, "only drivers can start rides", http.StatusForbidden)
		return
	}

	// Проверяем, что driver_id из URL совпадает с user_id из токена
	if driverIDFromURL != userIDFromToken {
		h.log.Error(logger.Entry{
			Action:  "start_ride_id_mismatch",
			Message: fmt.Sprintf("driver_id from URL (%s) != user_id from token (%s)", driverIDFromURL, userIDFromToken),
		})
		writeJSONError(w, "driver_id mismatch", http.StatusForbidden)
		return
	}

	// Декодируем тело запроса
	var req StartRideRequest
	body := http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB limit
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		h.log.Error(logger.Entry{
			Action:  "start_ride_decode_failed",
			Message: err.Error(),
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
		writeJSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Вызываем use-case
	output, err := h.driverUseCase.StartRide(ctx, in.StartRideInput{
		DriverID:  driverIDFromURL,
		RideID:    req.RideID,
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
	})
	if err != nil {
		h.log.Error(logger.Entry{
			Action:  "start_ride_usecase_failed",
			Message: err.Error(),
			RideID:  req.RideID,
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Возвращаем успешный ответ
	writeJSON(w, StartRideResponse{
		RideID:    output.RideID,
		Status:    output.Status,
		StartedAt: output.StartedAt,
		Message:   output.Message,
	}, http.StatusOK)
}

// HandleCompleteRide обрабатывает POST /drivers/{driver_id}/complete
func (h *DriverHandler) HandleCompleteRide(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Извлекаем driver_id из URL path
	driverIDFromURL := extractDriverID(r.URL.Path)
	if driverIDFromURL == "" {
		h.log.Error(logger.Entry{
			Action:  "complete_ride_missing_driver_id",
			Message: "driver_id not found in URL",
		})
		writeJSONError(w, "driver_id is required", http.StatusBadRequest)
		return
	}

	// Извлекаем user_id из JWT токена
	userIDFromToken := GetUserID(ctx)
	role := GetRole(ctx)

	// Проверяем роль
	if role != "DRIVER" {
		h.log.Error(logger.Entry{
			Action:  "complete_ride_invalid_role",
			Message: fmt.Sprintf("expected role DRIVER, got %s", role),
		})
		writeJSONError(w, "only drivers can complete rides", http.StatusForbidden)
		return
	}

	// Проверяем, что driver_id из URL совпадает с user_id из токена
	if driverIDFromURL != userIDFromToken {
		h.log.Error(logger.Entry{
			Action:  "complete_ride_id_mismatch",
			Message: fmt.Sprintf("driver_id from URL (%s) != user_id from token (%s)", driverIDFromURL, userIDFromToken),
		})
		writeJSONError(w, "driver_id mismatch", http.StatusForbidden)
		return
	}

	// Декодируем тело запроса
	var req CompleteRideRequest
	body := http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB limit
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		h.log.Error(logger.Entry{
			Action:  "complete_ride_decode_failed",
			Message: err.Error(),
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
		writeJSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Вызываем use-case
	output, err := h.driverUseCase.CompleteRide(ctx, in.CompleteRideInput{
		DriverID:              driverIDFromURL,
		RideID:                req.RideID,
		FinalLatitude:         req.FinalLatitude,
		FinalLongitude:        req.FinalLongitude,
		ActualDistanceKm:      req.ActualDistanceKm,
		ActualDurationMinutes: req.ActualDurationMinutes,
	})
	if err != nil {
		h.log.Error(logger.Entry{
			Action:  "complete_ride_usecase_failed",
			Message: err.Error(),
			RideID:  req.RideID,
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Возвращаем успешный ответ
	writeJSON(w, CompleteRideResponse{
		RideID:         output.RideID,
		Status:         output.Status,
		CompletedAt:    output.CompletedAt,
		DriverEarnings: output.DriverEarnings,
		Message:        output.Message,
	}, http.StatusOK)
}

// extractDriverID извлекает driver_id из пути /drivers/{driver_id}/online
func extractDriverID(path string) string {
	// Ожидаем формат: /drivers/{driver_id}/online
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 2 && parts[0] == "drivers" {
		return parts[1]
	}
	return ""
}

// writeJSON отправляет JSON ответ
func writeJSON(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Fallback если не удалось закодировать
		io.WriteString(w, `{"error":"internal_server_error"}`)
	}
}

// writeJSONError отправляет JSON ошибку
func writeJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
		Code:    statusCode,
	})
}
