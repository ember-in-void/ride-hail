package transport

import (
	"net/http"

	"ridehail/internal/driver/application/usecase"
	"ridehail/internal/shared/logger"
)

func newRouter(svc usecase.ServiceInterface, log *logger.Logger) *http.ServeMux {
	h := NewHandler(svc, log)
	router := http.NewServeMux()

	// liveness
	router.HandleFunc("GET /health", h.Health)

	// business
	router.HandleFunc("POST /drivers/{driver_id}/online", h.DriverOnline)
	router.HandleFunc("POST /drivers/{driver_id}/offline", h.DriverOffline)
	router.HandleFunc("POST /drivers/{driver_id}/location", h.DriverLocation)
	router.HandleFunc("POST /drivers/{driver_id}/start", h.DriverStart)
	router.HandleFunc("POST /drivers/{driver_id}/complete", h.DriverComplete)

	log.Info(logger.Entry{
		Action:  "http_routes_registered",
		Message: "Driver routes registered",
	})

	return router
}
