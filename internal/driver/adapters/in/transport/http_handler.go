package transport

import (
	"encoding/json"
	"io"
	"net/http"

	in "ridehail/internal/driver/application/ports/in"
	"ridehail/internal/shared/logger"
)

const maxRequestBodySize = 1 << 20 // 1 MB

type Handler struct {
	goOnlineUC       in.GoOnlineUseCase
	goOfflineUC      in.GoOfflineUseCase
	updateLocationUC in.UpdateLocationUseCase
	startRideUC      in.StartRideUseCase
	completeRideUC   in.CompleteRideUseCase
	log              *logger.Logger
}

func NewHandler(
	goOnlineUC in.GoOnlineUseCase,
	goOfflineUC in.GoOfflineUseCase,
	updateLocationUC in.UpdateLocationUseCase,
	startRideUC in.StartRideUseCase,
	completeRideUC in.CompleteRideUseCase,
	log *logger.Logger,
) *Handler {
	return &Handler{
		goOnlineUC:       goOnlineUC,
		goOfflineUC:      goOfflineUC,
		updateLocationUC: updateLocationUC,
		startRideUC:      startRideUC,
		completeRideUC:   completeRideUC,
		log:              log,
	}
}

// Health — liveness probe
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

// GoOnline — POST /drivers/{driver_id}/online
func (h *Handler) GoOnline(w http.ResponseWriter, r *http.Request) {
	driverID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "driver_id not found in context")
		return
	}

	var req GoOnlineRequest
	if err := readJSON(r, &req); err != nil {
		h.log.Warn(logger.Entry{
			Action:  "go_online_invalid_request",
			Message: err.Error(),
		})
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	output, err := h.goOnlineUC.Execute(r.Context(), in.GoOnlineInput{
		DriverID:  driverID,
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
	})
	if err != nil {
		h.log.Error(logger.Entry{
			Action:  "go_online_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
			Additional: map[string]any{
				"driver_id": driverID,
			},
		})
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, output)
}

// GoOffline — POST /drivers/{driver_id}/offline
func (h *Handler) GoOffline(w http.ResponseWriter, r *http.Request) {
	driverID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "driver_id not found in context")
		return
	}

	output, err := h.goOfflineUC.Execute(r.Context(), in.GoOfflineInput{
		DriverID: driverID,
	})
	if err != nil {
		h.log.Error(logger.Entry{
			Action:  "go_offline_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
			Additional: map[string]any{
				"driver_id": driverID,
			},
		})
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, output)
}

// UpdateLocation — POST /drivers/{driver_id}/location
func (h *Handler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	driverID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "driver_id not found in context")
		return
	}

	var req UpdateLocationRequest
	if err := readJSON(r, &req); err != nil {
		h.log.Warn(logger.Entry{
			Action:  "update_location_invalid_request",
			Message: err.Error(),
		})
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	output, err := h.updateLocationUC.Execute(r.Context(), in.UpdateLocationInput{
		DriverID:       driverID,
		Latitude:       req.Latitude,
		Longitude:      req.Longitude,
		AccuracyMeters: req.AccuracyMeters,
		SpeedKmh:       req.SpeedKmh,
		HeadingDegrees: req.HeadingDegrees,
	})
	if err != nil {
		h.log.Error(logger.Entry{
			Action:  "update_location_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
			Additional: map[string]any{
				"driver_id": driverID,
			},
		})
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, output)
}

// StartRide — POST /drivers/{driver_id}/start
func (h *Handler) StartRide(w http.ResponseWriter, r *http.Request) {
	driverID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "driver_id not found in context")
		return
	}

	var req StartRideRequest
	if err := readJSON(r, &req); err != nil {
		h.log.Warn(logger.Entry{
			Action:  "start_ride_invalid_request",
			Message: err.Error(),
		})
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	output, err := h.startRideUC.Execute(r.Context(), in.StartRideInput{
		DriverID:  driverID,
		RideID:    req.RideID,
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
	})
	if err != nil {
		h.log.Error(logger.Entry{
			Action:  "start_ride_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
			RideID:  req.RideID,
		})
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, output)
}

// CompleteRide — POST /drivers/{driver_id}/complete
func (h *Handler) CompleteRide(w http.ResponseWriter, r *http.Request) {
	driverID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "driver_id not found in context")
		return
	}

	var req CompleteRideRequest
	if err := readJSON(r, &req); err != nil {
		h.log.Warn(logger.Entry{
			Action:  "complete_ride_invalid_request",
			Message: err.Error(),
		})
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	output, err := h.completeRideUC.Execute(r.Context(), in.CompleteRideInput{
		DriverID:              driverID,
		RideID:                req.RideID,
		FinalLatitude:         req.FinalLatitude,
		FinalLongitude:        req.FinalLongitude,
		ActualDistanceKm:      req.ActualDistanceKm,
		ActualDurationMinutes: req.ActualDurationMinutes,
	})
	if err != nil {
		h.log.Error(logger.Entry{
			Action:  "complete_ride_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
			RideID:  req.RideID,
		})
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, output)
}

// Helper functions

func readJSON(r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, maxRequestBodySize)
	defer r.Body.Close()

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return err
	}

	// Убеждаемся, что тело содержит только один JSON объект
	if dec.More() {
		return io.ErrUnexpectedEOF
	}

	return nil
}

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = writeJSON(w, data)
}

func writeJSON(w http.ResponseWriter, data any) error {
	return json.NewEncoder(w).Encode(data)
}
