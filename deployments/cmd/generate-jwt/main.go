package main

import (
	"flag"
	"fmt"
	"os"

	"ridehail/internal/shared/auth"
	"ridehail/internal/shared/config"
)

func main() {
	userID := flag.String("user", "550e8400-e29b-41d4-a716-446655440000", "User ID (UUID)")
	email := flag.String("email", "test@example.com", "Email address")
	role := flag.String("role", "PASSENGER", "Role (PASSENGER|DRIVER|ADMIN)")
	flag.Parse()

	// Загружаем конфигурацию
	cfg := config.Load()

	// Создаем JWT сервис
	jwtService := auth.NewJWTService(cfg.JWT)

	// Генерируем токен
	token, err := jwtService.GenerateToken(*userID, *email, *role)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating JWT token: %v\n", err)
		os.Exit(1)
	}

	// Выводим токен
	fmt.Printf("\n✅ JWT Token generated successfully!\n\n")
	fmt.Printf("User ID:   %s\n", *userID)
	fmt.Printf("Email:     %s\n", *email)
	fmt.Printf("Role:      %s\n", *role)
	fmt.Printf("\nToken:\n%s\n", token)
	fmt.Printf("\n📋 Copy this for API requests:\n")
	fmt.Printf("Authorization: Bearer %s\n", token)
	fmt.Printf("\n💡 Example curl:\n")
	fmt.Printf("curl -X POST http://localhost:3000/rides \\\n")
	fmt.Printf("  -H 'Authorization: Bearer %s' \\\n", token)
	fmt.Printf("  -H 'Content-Type: application/json' \\\n")
	fmt.Printf("  -d '{\n")
	fmt.Printf("    \"vehicle_type\": \"ECONOMY\",\n")
	fmt.Printf("    \"pickup_lat\": 55.7558,\n")
	fmt.Printf("    \"pickup_lng\": 37.6173,\n")
	fmt.Printf("    \"pickup_address\": \"Red Square, Moscow\",\n")
	fmt.Printf("    \"destination_lat\": 55.7522,\n")
	fmt.Printf("    \"destination_lng\": 37.6156,\n")
	fmt.Printf("    \"destination_address\": \"Kremlin, Moscow\"\n")
	fmt.Printf("  }'\n\n")
}
