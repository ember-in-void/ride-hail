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
	ContextKeyUserID    contextKey = "user_id"
	ContextKeyUserEmail contextKey = "user_email"
	ContextKeyUserRole  contextKey = "user_role"
)

// AdminAuthMiddleware создает middleware для проверки JWT + роль ADMIN
func AdminAuthMiddleware(jwtService *auth.JWTService, log *logger.Logger) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Извлекаем токен из заголовка Authorization
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				log.Warn(logger.Entry{
					Action:  "admin_auth_missing_header",
					Message: "missing authorization header",
				})
				respondUnauthorized(w, "missing authorization header")
				return
			}

			// Проверяем формат "Bearer <token>"
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				log.Warn(logger.Entry{
					Action:  "admin_auth_invalid_format",
					Message: "invalid authorization header format",
				})
				respondUnauthorized(w, "invalid authorization header format")
				return
			}

			token := parts[1]

			// Валидируем токен
			claims, err := jwtService.ValidateToken(token)
			if err != nil {
				log.Error(logger.Entry{
					Action:  "admin_jwt_validation_failed",
					Message: err.Error(),
					Error:   &logger.ErrObj{Msg: err.Error()},
				})
				respondUnauthorized(w, "invalid or expired token")
				return
			}

			// Проверяем роль ADMIN
			if claims.Role != "ADMIN" {
				log.Warn(logger.Entry{
					Action:  "admin_auth_forbidden",
					Message: "insufficient permissions",
					Additional: map[string]interface{}{
						"user_id": claims.UserID,
						"role":    claims.Role,
					},
				})
				respondForbidden(w, "admin role required")
				return
			}

			log.Info(logger.Entry{
				Action:  "admin_auth_success",
				Message: "admin authenticated",
				Additional: map[string]interface{}{
					"user_id": claims.UserID,
					"role":    claims.Role,
				},
			})

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

// respondForbidden отправляет 403 ответ
func respondForbidden(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_, _ = w.Write([]byte(`{"error":"` + message + `"}`))
}
