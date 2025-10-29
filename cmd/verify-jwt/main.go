package main

import (
	"flag"
	"fmt"
	"os"

	"ridehail/internal/shared/auth"
	"ridehail/internal/shared/config"
)

func main() {
	token := flag.String("token", "", "JWT token to verify")
	flag.Parse()

	if *token == "" {
		fmt.Fprintln(os.Stderr, "Error: -token flag is required")
		fmt.Fprintln(os.Stderr, "Usage: go run cmd/verify-jwt/main.go -token=<JWT_TOKEN>")
		os.Exit(1)
	}

	// Загружаем конфигурацию (тот же способ, что и в сервисе)
	cfg := config.Load()

	fmt.Printf("🔍 Verifying JWT token...\n\n")
	fmt.Printf("Config loaded from: %s\n", os.Getenv("CONFIG_DIR"))
	fmt.Printf("JWT Secret: %s\n", cfg.JWT.Secret)
	fmt.Printf("JWT Expiry: %d minutes\n\n", cfg.JWT.ExpiryMinutes)

	// Создаем JWT сервис с тем же секретом, что и в ride-сервисе
	jwtService := auth.NewJWTService(cfg.JWT)

	// Валидируем токен
	claims, err := jwtService.ValidateToken(*token)
	if err != nil {
		fmt.Printf("❌ Token validation FAILED: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Token is VALID!\n\n")
	fmt.Printf("Claims:\n")
	fmt.Printf("  User ID: %s\n", claims.UserID)
	fmt.Printf("  Email:   %s\n", claims.Email)
	fmt.Printf("  Role:    %s\n", claims.Role)
	fmt.Printf("  Issuer:  %s\n", claims.Issuer)
	fmt.Printf("  Issued At:  %s\n", claims.IssuedAt.Time)
	fmt.Printf("  Expires At: %s\n", claims.ExpiresAt.Time)
	fmt.Printf("  Not Before: %s\n", claims.NotBefore.Time)
}
