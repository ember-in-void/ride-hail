package transport

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"ridehail/internal/shared/auth"
	"ridehail/internal/shared/logger"
	"ridehail/internal/shared/user"
)

type contextKey string

const (
	// Контекстные ключи для хранения данных пользователя
	ContextKeyUserID    contextKey = "user_id"
	ContextKeyUserEmail contextKey = "user_email"
	ContextKeyUserRole  contextKey = "user_role"
)

// JWTMiddleware создает middleware для валидации JWT токенов + проверки пользователя в БД
func JWTMiddleware(jwtService *auth.JWTService, userRepo user.Repository, log *logger.Logger) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Извлекаем токен из заголовка Authorization
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				log.Warn(logger.Entry{
					Action:  "jwt_auth_missing_header",
					Message: "missing authorization header",
				})
				respondUnauthorized(w, "missing authorization header")
				return
			}

			// Проверяем формат "Bearer <token>"
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				log.Warn(logger.Entry{
					Action:  "jwt_auth_invalid_format",
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
					Action:  "jwt_validation_failed",
					Message: err.Error(),
					Error:   &logger.ErrObj{Msg: err.Error()},
				})
				respondUnauthorized(w, "invalid or expired token")
				return
			}

			// Проверяем существование пользователя в БД
			userEntity, err := userRepo.FindByID(ctx, claims.UserID)
			if err != nil {
				if errors.Is(err, user.ErrUserNotFound) {
					log.Warn(logger.Entry{
						Action:  "user_not_found",
						Message: "user does not exist in database",
						Additional: map[string]interface{}{
							"user_id": claims.UserID,
						},
					})
					respondUnauthorized(w, "user not found")
					return
				}

				// Ошибка БД
				log.Error(logger.Entry{
					Action:  "user_lookup_failed",
					Message: err.Error(),
					Error:   &logger.ErrObj{Msg: err.Error()},
					Additional: map[string]interface{}{
						"user_id": claims.UserID,
					},
				})
				respondInternalError(w, "internal server error")
				return
			}

			// Проверяем статус пользователя
			if !userEntity.IsActive() {
				log.Warn(logger.Entry{
					Action:  "user_inactive",
					Message: "user is not active",
					Additional: map[string]interface{}{
						"user_id": claims.UserID,
						"status":  userEntity.Status,
					},
				})

				if userEntity.Status == "BANNED" {
					respondForbidden(w, "user is banned")
				} else {
					respondForbidden(w, "user is inactive")
				}
				return
			}

			// Проверяем роль (для ride-сервиса разрешены PASSENGER и ADMIN)
			if !userEntity.HasRole("PASSENGER") && !userEntity.HasRole("ADMIN") {
				log.Warn(logger.Entry{
					Action:  "user_invalid_role",
					Message: "user does not have required role",
					Additional: map[string]interface{}{
						"user_id": claims.UserID,
						"role":    userEntity.Role,
					},
				})
				respondForbidden(w, "insufficient permissions")
				return
			}

			log.Info(logger.Entry{
				Action:  "user_authenticated",
				Message: "user authenticated successfully",
				Additional: map[string]interface{}{
					"user_id": claims.UserID,
					"role":    claims.Role,
					"status":  userEntity.Status,
				},
			})

			// Добавляем данные пользователя в контекст
			ctx = context.WithValue(ctx, ContextKeyUserID, claims.UserID)
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

// respondInternalError отправляет 500 ответ
func respondInternalError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	_, _ = w.Write([]byte(`{"error":"` + message + `"}`))
}
