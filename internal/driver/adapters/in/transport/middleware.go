package transport

import (
	"context"
	"net/http"
	"strings"

	"ridehail/internal/shared/auth"
	"ridehail/internal/shared/logger"
	"ridehail/internal/shared/utils"
)

type contextKey string

const (
	contextKeyUserID    contextKey = "user_id"
	contextKeyRole      contextKey = "role"
	contextKeyRequestID contextKey = "request_id"
)

// AuthMiddleware проверяет JWT токен и извлекает user_id
func AuthMiddleware(jwtService *auth.JWTService, log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				log.Error(logger.Entry{
					Action:  "auth_missing_header",
					Message: "Authorization header is missing",
				})
				writeJSONError(w, "missing authorization header", http.StatusUnauthorized)
				return
			}

			// Ожидаем формат "Bearer <token>"
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				log.Error(logger.Entry{
					Action:  "auth_invalid_format",
					Message: "Authorization header format must be Bearer {token}",
				})
				writeJSONError(w, "invalid authorization header format", http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]
			claims, err := jwtService.ValidateToken(tokenString)
			if err != nil {
				log.Error(logger.Entry{
					Action:  "auth_invalid_token",
					Message: err.Error(),
					Error: &logger.ErrObj{
						Msg: err.Error(),
					},
				})
				writeJSONError(w, "invalid or expired token", http.StatusUnauthorized)
				return
			}

			// Добавляем user_id и role в контекст
			ctx := context.WithValue(r.Context(), contextKeyUserID, claims.UserID)
			ctx = context.WithValue(ctx, contextKeyRole, claims.Role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequestIDMiddleware добавляет request_id в контекст
func RequestIDMiddleware(log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = utils.NewUUID()
			}

			ctx := context.WithValue(r.Context(), contextKeyRequestID, requestID)
			w.Header().Set("X-Request-ID", requestID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// LoggingMiddleware логирует HTTP запросы
func LoggingMiddleware(log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := getRequestID(r.Context())

			log.Info(logger.Entry{
				Action:    "http_request",
				Message:   r.Method + " " + r.URL.Path,
				RequestID: requestID,
			})

			next.ServeHTTP(w, r)
		})
	}
}

// GetUserID извлекает user_id из контекста
func GetUserID(ctx context.Context) string {
	if userID, ok := ctx.Value(contextKeyUserID).(string); ok {
		return userID
	}
	return ""
}

// GetRole извлекает role из контекста
func GetRole(ctx context.Context) string {
	if role, ok := ctx.Value(contextKeyRole).(string); ok {
		return role
	}
	return ""
}

// getRequestID извлекает request_id из контекста
func getRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(contextKeyRequestID).(string); ok {
		return requestID
	}
	return ""
}
