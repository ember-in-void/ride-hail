package transport

import (
	"context"
	"net/http"
	"strings"

	"ridehail/internal/shared/auth"
	"ridehail/internal/shared/logger"
)

type contextKey string

const (
	// Контекстные ключи для хранения данных пользователя
	ContextKeyUserID    contextKey = "user_id"
	ContextKeyUserEmail contextKey = "user_email"
	ContextKeyUserRole  contextKey = "user_role"
)

// JWTMiddleware создает middleware для валидации JWT токенов
func JWTMiddleware(jwtService *auth.JWTService, log *logger.Logger) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Извлекаем токен из заголовка Authorization
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				respondUnauthorized(w, "missing authorization header")
				return
			}

			// Проверяем формат "Bearer <token>"
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				respondUnauthorized(w, "invalid authorization header format")
				return
			}

			token := parts[1]

			// Валидируем токен
			claims, err := jwtService.ValidateToken(token)
			if err != nil {
				log.Error(logger.Entry{
					Action:  "jwt_validation_failed",
					Message: err.Error(),
					Error:   &logger.ErrObj{Msg: err.Error()},
				})
				respondUnauthorized(w, "invalid or expired token")
				return
			}

			// Добавляем данные пользователя в контекст
			ctx := context.WithValue(r.Context(), ContextKeyUserID, claims.UserID)
			ctx = context.WithValue(ctx, ContextKeyUserEmail, claims.Email)
			ctx = context.WithValue(ctx, ContextKeyUserRole, claims.Role)

			// Передаем управление следующему обработчику
			next.ServeHTTP(w, r.WithContext(ctx))
		}
	}
}

// respondUnauthorized отправляет 401 ответ
func respondUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"` + message + `"}`))
}
