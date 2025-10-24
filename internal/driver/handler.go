package driver

import (
	"context"
	"net/http"
	"strconv"

	"ridehail/internal/logger"
)

type Handler struct {
	svc    ServiceInterface
	logger *logger.Logger
}

type Server struct {
	addr   string
	router *http.ServeMux
	svc    ServiceInterface
	logger *logger.Logger
	server *http.Server
}

func NewHTTPServer(svc ServiceInterface, port int, logger *logger.Logger) *Server {
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

func newRouter(svc ServiceInterface, logger *logger.Logger) *http.ServeMux {
	router := http.NewServeMux()
	Routes(router, svc, logger)
	return router
}

func NewHandler(svc ServiceInterface, logger *logger.Logger) *Handler {
	return &Handler{
		svc:    svc,
		logger: logger,
	}
}

func Routes(router *http.ServeMux, svc ServiceInterface, logger *logger.Logger) {
	h := NewHandler(svc, logger)
	router.HandleFunc("POST /drivers/{driver_id}/online", h.)
	router.HandleFunc("POST /drivers/{driver_id}/offline", h.)
	router.HandleFunc("POST /drivers/{driver_id}/location", h.)
	router.HandleFunc("POST /drivers/{driver_id}/start", h.)
	router.HandleFunc("POST /drivers/{driver_id}/complete", h.)

	logger.Info("Driver routes registered", nil)
}