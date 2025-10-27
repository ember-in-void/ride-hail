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
	router.HandleFunc("POST /drivers/{driver_id}/online", h.)
	router.HandleFunc("POST /drivers/{driver_id}/offline", h.)
	router.HandleFunc("POST /drivers/{driver_id}/location", h.)
	router.HandleFunc("POST /drivers/{driver_id}/start", h.)
	router.HandleFunc("POST /drivers/{driver_id}/complete", h.)

	logger.Info("Driver routes registered", nil)
}