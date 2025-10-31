// ============================================================================
// WEBSOCKET HUB - Менеджер всех WebSocket соединений
// ============================================================================
//
// 📡 НАЗНАЧЕНИЕ:
// WebSocket Hub — это "центральная станция" для управления WebSocket соединениями.
// Представьте себе диспетчерскую службу такси: она знает всех водителей и
// пассажиров, которые сейчас онлайн, и может отправить им сообщения.
//
// 🎯 ОСНОВНЫЕ ЗАДАЧИ:
// 1. Регистрация новых клиентов (когда пользователь подключается)
// 2. Отключение клиентов (когда соединение разрывается)
// 3. Отправка сообщений конкретному пользователю (по userID)
// 4. Broadcast сообщений всем подключенным клиентам
// 5. Поддержание соединения активным (ping/pong)
//
// 🔐 БЕЗОПАСНОСТЬ:
// - Клиент ДОЛЖЕН аутентифицироваться в течение 5 секунд после подключения
// - Аутентификация происходит через JWT токен
// - Без валидного токена соединение закрывается
//
// 💡 ПРИМЕР ИСПОЛЬЗОВАНИЯ:
//
//   // Создаем Hub
//   hub := ws.NewHub(authFunc, logger)
//   hub.SetMessageHandler(myMessageHandler)
//   go hub.Run(ctx)
//
//   // Отправляем сообщение пассажиру с ID = "uuid-123"
//   hub.SendToUser("uuid-123", map[string]interface{}{
//     "type": "ride_matched",
//     "driver_id": "driver-456",
//   })
//
// 🏗️ АРХИТЕКТУРА:
//
//   Пассажир (браузер/приложение)
//        │
//        │ WebSocket handshake
//        ▼
//   HTTP Handler (ServeWS)
//        │
//        ├─► Upgrade HTTP → WebSocket
//        ├─► Создать Client{...}
//        └─► hub.register ← Client
//                │
//                ▼
//           Hub.Run() [goroutine]
//                │
//                ├─► Регистрирует клиента в map
//                ├─► Запускает client.readPump()
//                └─► Запускает client.writePump()
//
//   Когда приходит уведомление:
//        hub.SendToUser(userID, msg)
//             │
//             ├─► Ищет клиента по userID
//             └─► client.send ← JSON(msg)
//                     │
//                     ▼
//                client.writePump() отправляет в WebSocket
//
// ============================================================================

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

// ============================================================================
// КОНСТАНТЫ КОНФИГУРАЦИИ
// ============================================================================

const (
	// authTimeout — максимальное время ожидания аутентификации
	// После подключения клиент ДОЛЖЕН отправить токен в течение 5 секунд,
	// иначе соединение будет разорвано.
	authTimeout = 5 * time.Second

	// pingInterval — как часто сервер отправляет ping клиенту
	// Это нужно чтобы проверить, что соединение живое.
	pingInterval = 30 * time.Second

	// pongWait — максимальное время ожидания pong от клиента
	// Если клиент не ответил за 60 секунд, соединение считается мертвым.
	pongWait = 60 * time.Second

	// maxMessageSize — максимальный размер сообщения (8 KB)
	// Защита от слишком больших сообщений.
	maxMessageSize = 8192

	// writeWait — таймаут на отправку сообщения
	// Если не удалось отправить за 10 секунд, соединение разрывается.
	writeWait = 10 * time.Second
)

// ============================================================================
// WEBSOCKET UPGRADER
// ============================================================================
// upgrader конвертирует обычный HTTP запрос в WebSocket соединение

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// ⚠️ В PRODUCTION здесь должна быть проверка origin!
		// Пример: return r.Header.Get("Origin") == "https://myapp.com"
		// Сейчас разрешены все origins (только для разработки!)
		return true
	},
}

// ============================================================================
// ТИПЫ ФУНКЦИЙ
// ============================================================================

// AuthFunc — функция для валидации JWT токена
// Принимает: строку токена
// Возвращает: userID, role, error
//
// ПРИМЕР:
//
//	func myAuthFunc(token string) (string, string, error) {
//	  claims, err := jwt.Parse(token)
//	  if err != nil { return "", "", err }
//	  return claims.UserID, claims.Role, nil
//	}
type AuthFunc func(token string) (userID, role string, err error)

// MessageHandler — функция обработки входящих сообщений от клиента
// Вызывается когда клиент отправляет сообщение серверу.
//
// ПАРАМЕТРЫ:
// - client: откуда пришло сообщение
// - messageType: тип сообщения (например "ping", "chat_message")
// - data: JSON данные сообщения
//
// ПРИМЕР:
//
//	func myHandler(client *Client, msgType string, data json.RawMessage) error {
//	  switch msgType {
//	  case "ping":
//	    client.Send(map[string]string{"type": "pong"})
//	  }
//	  return nil
//	}
type MessageHandler func(client *Client, messageType string, data json.RawMessage) error

// ============================================================================
// CLIENT - Одно WebSocket соединение
// ============================================================================

// Client представляет одно WebSocket соединение с клиентом.
//
// ПОЛЯ:
// - ID: уникальный ID соединения (UUID)
// - UserID: ID пользователя из JWT токена
// - Role: роль пользователя ("passenger", "driver", "admin")
// - conn: низкоуровневое WebSocket соединение
// - send: канал для отправки сообщений клиенту
// - hub: ссылка на Hub для регистрации/отключения
// - log: logger для записи событий
type Client struct {
	ID     string          // Уникальный ID соединения
	UserID string          // ID пользователя (из JWT)
	Role   string          // Роль пользователя
	conn   *websocket.Conn // WebSocket соединение
	send   chan []byte     // Канал для исходящих сообщений
	hub    *Hub            // Ссылка на Hub
	log    *logger.Logger  // Logger
}

// ============================================================================
// HUB - Менеджер всех соединений
// ============================================================================

// Hub управляет всеми активными WebSocket соединениями.
//
// ВНУТРЕННЯЯ СТРУКТУРА:
// - clients: map[clientID]*Client — все активные клиенты
// - mu: мьютекс для thread-safe доступа к clients
// - register: канал для регистрации новых клиентов
// - unregister: канал для отключения клиентов
// - broadcast: канал для отправки сообщений всем
// - authFunc: функция проверки JWT токена
// - messageHandler: функция обработки входящих сообщений
//
// РАБОТА С КЛИЕНТАМИ:
//
//	hub.clients["client-123"] = &Client{...}
//
// ПОТОКОБЕЗОПАСНОСТЬ:
// Весь доступ к hub.clients защищен мьютексом (mu.Lock/Unlock)
type Hub struct {
	clients        map[string]*Client // Все активные клиенты
	mu             sync.RWMutex       // Защита от concurrent access
	register       chan *Client       // Канал регистрации
	unregister     chan *Client       // Канал отключения
	broadcast      chan []byte        // Канал broadcast сообщений
	authFunc       AuthFunc           // Функция аутентификации
	messageHandler MessageHandler     // Обработчик сообщений
	log            *logger.Logger     // Logger
}

// ============================================================================
// КОНСТРУКТОР HUB
// ============================================================================

// NewHub создает новый WebSocket Hub.
//
// ПАРАМЕТРЫ:
// - authFunc: функция для валидации JWT токенов
// - log: logger для записи событий
//
// ВАЖНО: После создания Hub НЕ забудьте:
// 1. Установить MessageHandler (если нужна обработка входящих сообщений)
// 2. Запустить hub.Run(ctx) в горутине
//
// ПРИМЕР:
//
//	hub := ws.NewHub(myAuthFunc, logger)
//	hub.SetMessageHandler(myHandler)
//	go hub.Run(ctx)
func NewHub(authFunc AuthFunc, log *logger.Logger) *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		register:   make(chan *Client, 10), // Буфер на 10 клиентов
		unregister: make(chan *Client, 10), // Буфер на 10 клиентов
		broadcast:  make(chan []byte, 256), // Буфер на 256 сообщений
		authFunc:   authFunc,
		log:        log,
	}
}

// ============================================================================
// УСТАНОВКА ОБРАБОТЧИКА СООБЩЕНИЙ
// ============================================================================

// SetMessageHandler устанавливает обработчик входящих сообщений от клиентов.
//
// КОГДА ВЫЗЫВАЕТСЯ:
// Каждый раз, когда клиент отправляет сообщение серверу, вызывается handler.
//
// ПРИМЕР:
//
//	hub.SetMessageHandler(func(client *Client, msgType string, data json.RawMessage) error {
//	  log.Info("Received message", msgType, "from", client.UserID)
//	  return nil
//	})
func (h *Hub) SetMessageHandler(handler MessageHandler) {
	h.messageHandler = handler
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

// SendToRole отправляет сообщение всем пользователям с определенной ролью
func (h *Hub) SendToRole(role string, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, client := range h.clients {
		if client.Role == role {
			select {
			case client.send <- message:
			default:
				h.log.Error(logger.Entry{
					Action:  "send_to_role_failed",
					Message: role,
					Additional: map[string]any{
						"client_id": client.ID,
					},
				})
			}
		}
	}
}

// GetClientsByRole возвращает список user_id для клиентов с определенной ролью
func (h *Hub) GetClientsByRole(role string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var userIDs []string
	for _, client := range h.clients {
		if client.Role == role {
			userIDs = append(userIDs, client.UserID)
		}
	}
	return userIDs
}

// GetClient возвращает клиента по user_id
func (h *Hub) GetClient(userID string) *Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, client := range h.clients {
		if client.UserID == userID {
			return client
		}
	}
	return nil
}

// IsUserConnected проверяет, подключен ли пользователь
func (h *Hub) IsUserConnected(userID string) bool {
	return h.GetClient(userID) != nil
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

		// Парсим входящее сообщение
		var msg struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data,omitempty"`
		}

		if err := json.Unmarshal(message, &msg); err != nil {
			c.log.Error(logger.Entry{
				Action:  "ws_parse_message_error",
				Message: err.Error(),
				Error:   &logger.ErrObj{Msg: err.Error()},
				Additional: map[string]any{
					"client_id": c.ID,
					"raw":       string(message),
				},
			})
			continue
		}

		// Вызываем обработчик сообщений, если установлен
		if c.hub.messageHandler != nil {
			if err := c.hub.messageHandler(c, msg.Type, msg.Data); err != nil {
				c.log.Error(logger.Entry{
					Action:  "ws_handle_message_error",
					Message: err.Error(),
					Error:   &logger.ErrObj{Msg: err.Error()},
					Additional: map[string]any{
						"client_id": c.ID,
						"msg_type":  msg.Type,
					},
				})
			}
		}
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

// SendToRoleJSON отправляет JSON всем пользователям с ролью
func (h *Hub) SendToRoleJSON(role string, data interface{}) error {
	msg, err := json.Marshal(data)
	if err != nil {
		return err
	}
	h.SendToRole(role, msg)
	return nil
}

// SendTypedMessage отправляет сообщение с типом конкретному пользователю
func (h *Hub) SendTypedMessage(userID, msgType string, data interface{}) error {
	message := map[string]interface{}{
		"type": msgType,
		"data": data,
	}
	return h.SendToUserJSON(userID, message)
}
