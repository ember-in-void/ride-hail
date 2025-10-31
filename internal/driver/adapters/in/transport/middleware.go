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
	contextKeyUserID contextKey = "user_id"
	contextKeyRole   contextKey = "role"
)

// JWTMiddleware проверяет JWT токен и role=DRIVER
func JWTMiddleware(jwtService *auth.JWTService, log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				log.Warn(logger.Entry{
					Action:  "jwt_middleware_missing_token",
					Message: "authorization header missing",
				})
				respondError(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				log.Warn(logger.Entry{
					Action:  "jwt_middleware_invalid_format",
					Message: "invalid authorization header format",
				})
				respondError(w, http.StatusUnauthorized, "invalid authorization header format")
				return
			}

			token := parts[1]

			claims, err := jwtService.ValidateToken(token)
			if err != nil {
				log.Warn(logger.Entry{
					Action:  "jwt_middleware_invalid_token",
					Message: err.Error(),
					Error:   &logger.ErrObj{Msg: err.Error()},
				})
				respondError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			// Проверяем роль DRIVER
			if claims.Role != "DRIVER" {
				log.Warn(logger.Entry{
					Action:  "jwt_middleware_forbidden_role",
					Message: "user does not have DRIVER role",
					Additional: map[string]any{
						"user_id": claims.UserID,
						"role":    claims.Role,
					},
				})
				respondError(w, http.StatusForbidden, "access denied: DRIVER role required")
				return
			}

			// Добавляем user_id и role в контекст
			ctx := context.WithValue(r.Context(), contextKeyUserID, claims.UserID)
			ctx = context.WithValue(ctx, contextKeyRole, claims.Role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserIDFromContext извлекает user_id из контекста
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(contextKeyUserID).(string)
	return userID, ok
}

func respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = writeJSON(w, ErrorResponse{
		Error:   http.StatusText(status),
		Message: message,
	})
}
