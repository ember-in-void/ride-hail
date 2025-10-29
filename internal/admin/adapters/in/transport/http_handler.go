package transport

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"ridehail/internal/admin/application/ports/in"
	"ridehail/internal/admin/domain"
	"ridehail/internal/shared/logger"
)

const maxBodySize = 1 << 20 // 1MB

// HTTPHandler обрабатывает HTTP запросы для Admin Service
type HTTPHandler struct {
	createUserUC in.CreateUserUseCase
	listUsersUC  in.ListUsersUseCase
	log          *logger.Logger
}

// NewHTTPHandler создает новый HTTP handler
func NewHTTPHandler(
	createUserUC in.CreateUserUseCase,
	listUsersUC in.ListUsersUseCase,
	log *logger.Logger,
) *HTTPHandler {
	return &HTTPHandler{
		createUserUC: createUserUC,
		listUsersUC:  listUsersUC,
		log:          log,
	}
}

// RegisterRoutes регистрирует все HTTP маршруты
func (h *HTTPHandler) RegisterRoutes(mux *http.ServeMux, adminAuthMiddleware func(http.HandlerFunc) http.HandlerFunc) {
	// liveness probe (без аутентификации)
	mux.HandleFunc("GET /health", h.handleHealth)

	// admin endpoints (требуют ADMIN роль)
	mux.HandleFunc("POST /admin/users", adminAuthMiddleware(h.handleCreateUser))
	mux.HandleFunc("GET /admin/users", adminAuthMiddleware(h.handleListUsers))
}

// handleHealth обрабатывает health check
func (h *HTTPHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok","service":"admin"}`))
}

// CreateUserHTTPRequest — HTTP DTO для создания пользователя
type CreateUserHTTPRequest struct {
	Email    string                 `json:"email"`
	Password string                 `json:"password"`
	Role     string                 `json:"role"`
	Status   string                 `json:"status,omitempty"`
	Attrs    map[string]interface{} `json:"attrs,omitempty"`
}

// handleCreateUser обрабатывает POST /admin/users
func (h *HTTPHandler) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Ограничиваем размер тела
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	// Парсим JSON
	var req CreateUserHTTPRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		if errors.Is(err, io.EOF) {
			h.respondError(w, http.StatusBadRequest, "empty request body")
			return
		}
		h.log.Error(logger.Entry{
			Action:  "parse_create_user_request_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		h.respondError(w, http.StatusBadRequest, "invalid request format")
		return
	}

	// Валидация обязательных полей
	if req.Email == "" {
		h.respondError(w, http.StatusBadRequest, "email is required")
		return
	}
	if req.Password == "" {
		h.respondError(w, http.StatusBadRequest, "password is required")
		return
	}
	if req.Role == "" {
		h.respondError(w, http.StatusBadRequest, "role is required")
		return
	}

	// Маппинг HTTP DTO → Use Case Input
	input := in.CreateUserInput{
		Email:    req.Email,
		Password: req.Password,
		Role:     req.Role,
		Status:   req.Status,
		Attrs:    req.Attrs,
	}

	output, err := h.createUserUC.Execute(ctx, input)
	if err != nil {
		h.handleUseCaseError(w, err)
		return
	}

	h.respondJSON(w, http.StatusCreated, output)
}

// handleListUsers обрабатывает GET /admin/users
func (h *HTTPHandler) handleListUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Извлекаем query параметры
	query := r.URL.Query()
	role := query.Get("role")
	status := query.Get("status")

	limit := 50
	if limitStr := query.Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	offset := 0
	if offsetStr := query.Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Маппинг → Use Case Input
	input := in.ListUsersInput{
		Role:   role,
		Status: status,
		Limit:  limit,
		Offset: offset,
	}

	output, err := h.listUsersUC.Execute(ctx, input)
	if err != nil {
		h.handleUseCaseError(w, err)
		return
	}

	h.respondJSON(w, http.StatusOK, output)
}

// handleUseCaseError обрабатывает ошибки use case
func (h *HTTPHandler) handleUseCaseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrUserAlreadyExists):
		h.respondError(w, http.StatusConflict, "user already exists")
	case errors.Is(err, domain.ErrInvalidEmail):
		h.respondError(w, http.StatusBadRequest, "invalid email format")
	case errors.Is(err, domain.ErrInvalidRole):
		h.respondError(w, http.StatusBadRequest, "invalid role")
	case errors.Is(err, domain.ErrInvalidStatus):
		h.respondError(w, http.StatusBadRequest, "invalid status")
	case errors.Is(err, domain.ErrPasswordTooShort):
		h.respondError(w, http.StatusBadRequest, "password too short (minimum 8 characters)")
	case errors.Is(err, domain.ErrUnauthorized):
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
	default:
		h.log.Error(logger.Entry{
			Action:  "admin_usecase_error",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		h.respondError(w, http.StatusInternalServerError, "internal server error")
	}
}

// respondJSON отправляет JSON ответ
func (h *HTTPHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.log.Error(logger.Entry{
			Action:  "encode_admin_response_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
	}
}

// respondError отправляет JSON с ошибкой
func (h *HTTPHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}
