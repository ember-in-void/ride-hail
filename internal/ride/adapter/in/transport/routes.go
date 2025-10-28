package transport

import (
	"net/http"

	"ridehail/internal/ride/application/usecase"
	"ridehail/internal/shared/logger"
)

func newRouter(svc usecase.ServiceInterface, log *logger.Logger) *http.ServeMux {
	mux := http.NewServeMux()
	h := NewHandler(svc, log)

	// liveness
	mux.HandleFunc("GET /health", h.Health)

	// business
	mux.HandleFunc("POST /rides", h.CreateRide)
	mux.HandleFunc("POST /rides/{ride_id}/cancel", h.CancelRide)

	log.Info(logger.Entry{
		Action:  "http_routes_registered",
		Message: "Ride routes registered",
	})
	return mux
}
