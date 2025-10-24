package main

import (
	"log"

	"ridehail/internal/logger"
)

func main() {
	logger, err := logger.NewLogger("driver-service", "info", "./driver_service_logs/")
	if err != nil {
		log.Fatalln("failed to create logger:", err)
	}
	defer logger.Close()

	logger.Info("Driver service Logger", map[string]any{
		"status": "Initialized",
	})
}
