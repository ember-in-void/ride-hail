package transport

import (
	"net/http"

	"ridehail/internal/shared/logger"
)

func newRouter(svc ServiceInterface, logger *logger.Logger) *http.ServeMux {
	router := http.NewServeMux()
	Routes(router, svc, logger)
	return router
}

func Routes(router *http.ServeMux, svc ServiceInterface, logger *logger.Logger) {
	h := NewHandler(svc, logger)
	// router.HandleFunc("GET /admin/overview", h.)
	// router.HandleFunc("GET /admin/rides/active", h.)
	router.HandleFunc("GET /health", h.Health)
}
