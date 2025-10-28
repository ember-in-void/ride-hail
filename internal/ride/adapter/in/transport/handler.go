package transport

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"ridehail/internal/ride/domain"
	"ridehail/internal/shared/logger"

	"ridehail/internal/ride/application/usecase"
)

type Server struct {
	addr   string
	router *http.ServeMux
	svc    usecase.ServiceInterface
	logger *logger.Logger
	server *http.Server
}
type Handler struct {
	svc    usecase.ServiceInterface
	logger *logger.Logger
}

func NewHandler(svc usecase.ServiceInterface, log *logger.Logger) *Handler {
	return &Handler{svc: svc, logger: log}
}

func NewHTTPServer(svc usecase.ServiceInterface, port int, log *logger.Logger) *Server {
	mux := newRouter(svc, log)
	addr := ":" + strconv.Itoa(port)

	httpServer := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return &Server{
		addr:   addr,
		router: mux,
		svc:    svc,
		logger: log,
		server: httpServer,
	}
}

func (s *Server) Serve() error {
	s.logger.Info(logger.Entry{
		Action:  "http_server_start",
		Message: "HTTP server listening",
		Additional: map[string]any{
			"addr": s.addr,
		},
	})

	err := s.server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		// штатное завершение, финальный лог в Shutdown
		return nil
	}
	if err != nil {
		s.logger.Error(logger.Entry{
			Action:  "http_server_failed",
			Message: "HTTP server terminated with error",
			Error:   &logger.ErrObj{Msg: err.Error()},
			Additional: map[string]any{
				"addr":   s.addr,
				"status": "stopped",
			},
		})
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info(logger.Entry{
		Action:  "http_server_shutdown_begin",
		Message: "HTTP server shutdown initiated",
		Additional: map[string]any{
			"addr": s.addr,
		},
	})

	if err := s.server.Shutdown(ctx); err != nil {
		s.logger.Error(logger.Entry{
			Action:  "http_server_shutdown_failed",
			Message: "HTTP server shutdown failed",
			Error:   &logger.ErrObj{Msg: err.Error()},
			Additional: map[string]any{
				"addr":   s.addr,
				"status": "failed",
			},
		})
		return err
	}

	s.logger.Info(logger.Entry{
		Action:  "http_server_shutdown_complete",
		Message: "HTTP server stopped",
		Additional: map[string]any{
			"addr":   s.addr,
			"status": "stopped",
		},
	})
	return nil
}

// Health — простой liveness-проб
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "OK")
}

const maxBodyBytes = 1 << 20 // 1 MiB

func (h *Handler) CreateRide(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	// ограничиваем размер тела
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	defer r.Body.Close()

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req domain.RideRequest
	if err := dec.Decode(&req); err != nil {
		h.logger.Error(logger.Entry{
			Action:  "create_ride_decode_failed",
			Message: "Invalid JSON payload",
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		writeJSONError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON payload")
		return
	}

	// минимальная валидация
	if req.PickupLat == 0 || req.DestinationLat == 0 {
		h.logger.Error(logger.Entry{
			Action:  "create_ride_validation_failed",
			Message: "Missing required fields",
			Additional: map[string]any{
				"pickup_lat":      req.PickupLat,
				"destination_lat": req.DestinationLat,
			},
		})
		writeJSONError(w, http.StatusBadRequest, "validation_error", "Missing required fields")
		return
	}

	h.logger.Info(logger.Entry{
		Action:  "create_ride_request_received",
		Message: "Create ride request accepted",
		Additional: map[string]any{
			"pickup_lat":      req.PickupLat,
			"pickup_lng":      req.PickupLng,
			"destination_lat": req.DestinationLat,
			"destination_lng": req.DestinationLng,
		},
	})

	// TODO: интеграция с бизнес-логикой
	// out, err := h.svc.CreateRide(r.Context(), req)
	// if err != nil {
	// 	h.logger.Error(logger.Entry{
	// 		Action:  "create_ride_failed",
	// 		Message: "Create ride use case failed",
	// 		Error:   &logger.ErrObj{Msg: err.Error()},
	// 	})
	// 	writeJSONError(w, http.StatusInternalServerError, "create_failed", "Failed to create ride")
	// 	return
	// }

	// Пока логики нет: возвращаем 202 Accepted c echo-запросом (заменить на 201 с out при внедрении use case)
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":  "accepted",
		"message": "Ride creation queued",
		"request": req,
		// "data": out, // раскомментировать при интеграции
	})
}

func (h *Handler) CancelRide(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	rideID := r.PathValue("ride_id")
	if rideID == "" {
		writeJSONError(w, http.StatusBadRequest, "validation_error", "ride_id is required")
		return
	}

	h.logger.Info(logger.Entry{
		Action:  "cancel_ride_request_received",
		Message: "Cancel ride request accepted",
		Additional: map[string]any{
			"ride_id": rideID,
		},
	})

	// TODO: интеграция с бизнес-логикой
	// if err := h.svc.CancelRide(r.Context(), rideID); err != nil {
	// 	h.logger.Error(logger.Entry{
	// 		Action:  "cancel_ride_failed",
	// 		Message: "Cancel ride use case failed",
	// 		Error:   &logger.ErrObj{Msg: err.Error()},
	// 		Additional: map[string]any{"ride_id": rideID},
	// 	})
	// 	writeJSONError(w, http.StatusInternalServerError, "cancel_failed", "Failed to cancel ride")
	// 	return
	// }

	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":  "ok",
		"message": "Ride cancellation queued",
		"ride_id": rideID,
	})
}

func methodNotAllowed(w http.ResponseWriter) {
	w.WriteHeader(http.StatusMethodNotAllowed)
	_, _ = w.Write([]byte("method not allowed"))
}

type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func writeJSONError(w http.ResponseWriter, code int, errCode, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(errorResponse{Error: errCode, Message: msg})
}
