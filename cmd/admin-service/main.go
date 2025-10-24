package main

import (
	"log"

	"ridehail/internal/logger"
)

func main() {
	logger, err := logger.NewLogger("admin-service", "info", "./admin_service_logs/")
	if err != nil {
		log.Fatalln("failed to create logger:", err)
	}
	defer logger.Close()

	logger.Info("Admin service Logger", map[string]any{
		"status": "Initialized",
	})
}
