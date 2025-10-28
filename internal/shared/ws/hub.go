package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"ridehail/internal/shared/logger"

	"github.com/gorilla/websocket"
)

const (
	// Таймаут для аутентификации после подключения
	authTimeout = 5 * time.Second

	// Интервалы ping/pong для keep-alive
	pingInterval = 30 * time.Second
	pongWait     = 60 * time.Second

	// Лимиты сообщений
	maxMessageSize = 8192
	writeWait      = 10 * time.Second
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// В проде здесь должна быть проверка origin
		return true
	},
}

// AuthFunc — функция валидации JWT токена и извлечения user_id, role
type AuthFunc func(token string) (userID, role string, err error)

// Client представляет одно WebSocket соединение
type Client struct {
	ID     string
	UserID string
	Role   string
	conn   *websocket.Conn
	send   chan []byte
	hub    *Hub
	log    *logger.Logger
}

// Hub управляет всеми WebSocket соединениями
type Hub struct {
	clients    map[string]*Client // key = client.ID
	mu         sync.RWMutex
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	authFunc   AuthFunc
	log        *logger.Logger
}

// NewHub создает новый WebSocket хаб
func NewHub(authFunc AuthFunc, log *logger.Logger) *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		register:   make(chan *Client, 10),
		unregister: make(chan *Client, 10),
		broadcast:  make(chan []byte, 256),
		authFunc:   authFunc,
		log:        log,
	}
}

// Run запускает главный цикл хаба
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			h.log.Info(logger.Entry{Action: "hub_stopped", Message: "websocket hub stopped"})
			return

		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.ID] = client
			h.mu.Unlock()
			h.log.Info(logger.Entry{
				Action:  "client_registered",
				Message: client.ID,
				Additional: map[string]any{
					"user_id": client.UserID,
					"role":    client.Role,
				},
			})

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				close(client.send)
			}
			h.mu.Unlock()
			h.log.Info(logger.Entry{
				Action:  "client_unregistered",
				Message: client.ID,
			})

		case message := <-h.broadcast:
			h.mu.RLock()
			for _, client := range h.clients {
				select {
				case client.send <- message:
				default:
					// Канал переполнен, закрываем клиента
					close(client.send)
					delete(h.clients, client.ID)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast отправляет сообщение всем подключенным клиентам
func (h *Hub) Broadcast(message []byte) {
	select {
	case h.broadcast <- message:
	default:
		h.log.Error(logger.Entry{
			Action:  "broadcast_dropped",
			Message: "broadcast channel full",
		})
	}
}

// SendToUser отправляет сообщение конкретному пользователю
func (h *Hub) SendToUser(userID string, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, client := range h.clients {
		if client.UserID == userID {
			select {
			case client.send <- message:
			default:
				h.log.Error(logger.Entry{
					Action:  "send_to_user_failed",
					Message: userID,
				})
			}
		}
	}
}

// ServeWS обрабатывает HTTP запрос на WebSocket соединение
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Error(logger.Entry{
			Action:  "ws_upgrade_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		return
	}

	clientID := fmt.Sprintf("ws_%d", time.Now().UnixNano())

	client := &Client{
		ID:   clientID,
		conn: conn,
		send: make(chan []byte, 256),
		hub:  h,
		log:  h.log,
	}

	// Устанавливаем дедлайн для аутентификации
	authDeadline := time.Now().Add(authTimeout)
	_ = conn.SetReadDeadline(authDeadline)

	// Ожидаем первое сообщение с JWT токеном
	var authMsg struct {
		Token string `json:"token"`
	}

	if err := conn.ReadJSON(&authMsg); err != nil {
		_ = conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseProtocolError, "auth timeout"))
		_ = conn.Close()
		h.log.Error(logger.Entry{
			Action:  "ws_auth_failed",
			Message: "no auth message received",
		})
		return
	}

	// Валидируем токен
	userID, role, err := h.authFunc(authMsg.Token)
	if err != nil {
		_ = conn.WriteJSON(map[string]string{"error": "invalid token"})
		_ = conn.Close()
		h.log.Error(logger.Entry{
			Action:  "ws_auth_invalid_token",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		return
	}

	client.UserID = userID
	client.Role = role

	// Снимаем дедлайн, ставим нормальный pong wait
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// Регистрируем клиента
	h.register <- client

	// Отправляем подтверждение аутентификации
	_ = conn.WriteJSON(map[string]string{"status": "authenticated", "user_id": userID})

	// Запускаем горутины для чтения и записи
	go client.writePump()
	go client.readPump()
}

// readPump читает сообщения от клиента
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		_ = c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.log.Error(logger.Entry{
					Action:  "ws_read_error",
					Message: c.ID,
					Error:   &logger.ErrObj{Msg: err.Error()},
				})
			}
			break
		}

		// Обработка входящих сообщений (пока просто логируем)
		c.log.Debug(logger.Entry{
			Action:  "ws_message_received",
			Message: string(message),
			Additional: map[string]any{
				"client_id": c.ID,
				"user_id":   c.UserID,
			},
		})
	}
}

// writePump отправляет сообщения клиенту
func (c *Client) writePump() {
	ticker := time.NewTicker(pingInterval)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub закрыл канал
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// BroadcastJSON отправляет JSON всем клиентам
func (h *Hub) BroadcastJSON(data interface{}) error {
	msg, err := json.Marshal(data)
	if err != nil {
		return err
	}
	h.Broadcast(msg)
	return nil
}

// SendToUserJSON отправляет JSON конкретному пользователю
func (h *Hub) SendToUserJSON(userID string, data interface{}) error {
	msg, err := json.Marshal(data)
	if err != nil {
		return err
	}
	h.SendToUser(userID, msg)
	return nil
}
