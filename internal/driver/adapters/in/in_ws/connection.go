package in_ws

import (
	"net/http"

	"ridehail/internal/shared/auth"
	"ridehail/internal/shared/logger"
	"ridehail/internal/shared/ws"
)

// ConnectionHandler — обработчик WebSocket подключений для водителей
type ConnectionHandler struct {
	hub        *ws.Hub
	jwtService *auth.JWTService
	log        *logger.Logger
}

func NewConnectionHandler(hub *ws.Hub, jwtService *auth.JWTService, log *logger.Logger) *ConnectionHandler {
	return &ConnectionHandler{
		hub:        hub,
		jwtService: jwtService,
		log:        log,
	}
}

// HandleWebSocket обрабатывает WebSocket подключение водителя
// URL: /ws/drivers/{driver_id}
func (h *ConnectionHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// WebSocket upgrade и аутентификация выполняются в shared/ws.Hub
	// Здесь просто передаем в hub
	h.hub.HandleConnection(w, r, h.jwtService, "DRIVER")
}
