package transport

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"time"

	"ridehail/internal/shared/logger"
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

func NewHTTPServer(svc ServiceInterface, port int, log *logger.Logger) *Server {
	router := newRouter(svc, log)
	addr := ":" + strconv.Itoa(port)

	httpServer := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return &Server{
		addr:   addr,
		router: router,
		svc:    svc,
		logger: log,
		server: httpServer,
	}
}

// Serve запускает HTTP-сервер и блокируется до его остановки.
func (s *Server) Serve() error {
	s.logger.Info(logger.Entry{
		Action:  "http_server_start",
		Message: "HTTP server listening",
		Additional: map[string]any{
			"addr": s.addr,
		},
	})

	err := s.server.ListenAndServe()
	if err == http.ErrServerClosed {
		// Штатное завершение через Shutdown; финальный лог будет в Shutdown
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

	// На практике сюда не дойдём, но оставим на всякий
	s.logger.Info(logger.Entry{
		Action:  "http_server_exit",
		Message: "HTTP server exited without error",
		Additional: map[string]any{
			"addr": s.addr,
		},
	})
	return nil
}

// Shutdown мягко останавливает сервер с логированием статуса.
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

func NewHandler(svc ServiceInterface, log *logger.Logger) *Handler {
	return &Handler{
		svc:    svc,
		logger: log,
	}
}

// Health — простой liveness-проб: возвращает 200 OK и "OK".
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "OK")
}
