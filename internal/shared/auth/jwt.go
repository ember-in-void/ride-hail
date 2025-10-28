package auth

import (
	"fmt"
	"time"

	"ridehail/internal/shared/config"

	"github.com/golang-jwt/jwt/v5"
)

// Claims представляет JWT claims для нашей системы
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"` // PASSENGER | DRIVER | ADMIN
	jwt.RegisteredClaims
}

// JWTService работает с JWT токенами
type JWTService struct {
	secret        []byte
	expiryMinutes int
}

// NewJWTService создает новый сервис для работы с JWT
func NewJWTService(cfg config.JWTConfig) *JWTService {
	return &JWTService{
		secret:        []byte(cfg.Secret),
		expiryMinutes: cfg.ExpiryMinutes,
	}
}

// GenerateToken создает новый JWT токен для пользователя
func (s *JWTService) GenerateToken(userID, email, role string) (string, error) {
	now := time.Now()
	expiresAt := now.Add(time.Duration(s.expiryMinutes) * time.Minute)

	claims := &Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "ride-hail",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

// ValidateToken проверяет токен и возвращает claims
func (s *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Проверяем метод подписи
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("invalid claims type")
	}

	return claims, nil
}

// ExtractUserID извлекает user_id из токена (упрощенный метод для WebSocket)
func (s *JWTService) ExtractUserID(tokenString string) (userID, role string, err error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return "", "", err
	}
	return claims.UserID, claims.Role, nil
}

// RefreshToken обновляет токен (генерирует новый с обновленным expiry)
func (s *JWTService) RefreshToken(tokenString string) (string, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}

	// Генерируем новый токен с теми же данными
	return s.GenerateToken(claims.UserID, claims.Email, claims.Role)
}
