package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"ridehail/internal/ride/application/usecase"
	"ridehail/internal/ride/domain"
	"ridehail/internal/shared/logger"
)

type Handler struct {
	svc    usecase.ServiceInterface
	logger *logger.Logger
}

type Server struct {
	addr   string
	router *http.ServeMux
	svc    usecase.ServiceInterface
	logger *logger.Logger
	server *http.Server
}

func NewHTTPServer(svc usecase.ServiceInterface, port int, logger *logger.Logger) *Server {
	router := newRouter(svc, logger)
	addr := ":" + strconv.Itoa(port)

	httpServer := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	return &Server{
		addr:   addr,
		router: router,
		svc:    svc,
		logger: logger,
		server: httpServer,
	}
}

func (s *Server) Serve() error {
	err := s.server.ListenAndServe()
	if err != nil {
		s.logger.Error("HTTP server failed", map[string]any{
			"error":  err.Error(),
			"status": "stopped",
		})
		return err
	}
	s.logger.Info("Starting HTTP server", map[string]any{
		"addr":   s.addr,
		"status": "running",
	})
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	if err := s.server.Shutdown(ctx); err != nil {
		s.logger.Error("HTTP server shutdown failed", map[string]any{
			"error":  err.Error(),
			"status": "failed",
		})
		return err
	}
	s.logger.Info("HTTP server stopped", map[string]any{
		"status": "stopped",
	})
	return nil
}

func newRouter(svc usecase.ServiceInterface, logger *logger.Logger) *http.ServeMux {
	router := http.NewServeMux()
	Routes(router, svc, logger)
	return router
}

func NewHandler(svc usecase.ServiceInterface, logger *logger.Logger) *Handler {
	return &Handler{
		svc:    svc,
		logger: logger,
	}
}

func Routes(router *http.ServeMux, svc usecase.ServiceInterface, logger *logger.Logger) {
	h := NewHandler(svc, logger)
	router.HandleFunc("POST /rides", h.CreateRide)
	router.HandleFunc("POST /rides/{ride_id}/cancel", h.CancelRide)

	logger.Info("Ride routes registered", nil)
}

func (h *Handler) CreateRide(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 2)

	var req domain.RideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON payload", http.StatusBadRequest)
		h.logger.Error("Error while Deconding JSON", map[string]any{
			"error":  err.Error(),
			"status": "bad_request",
		})
		return
	}

	if req.PickupLat == 0 || req.DestinationLat == 0 {
		http.Error(w, "missing required fields", http.StatusBadRequest)
		h.logger.Error("missing required fields", map[string]any{
			"error":           "missing required fields",
			"pickup_lat":      req.PickupLat,
			"destination_lat": req.DestinationLat,
			"status":          "bad_request",
		})
		return
	}
	h.logger.Info("request", map[string]any{
		"request": req,
	})

	// парсим JSON
	// вызываем h.svc.CreateRide()
	// возвращаем JSON и код 201
}

func (h *Handler) CancelRide(w http.ResponseWriter, r *http.Request) {
	// ride_id := r.PathValue("ride_id")

	// достаём ride_id из URL
	// вызываем h.svc.CancelRide()
	// возвращаем 200 OK
}
