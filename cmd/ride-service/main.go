package main

import (
	"log"

	"ridehail/internal/config"
	"ridehail/internal/logger"
)

func main() {
	logger, err := logger.NewLogger("ride-service", "info", "./ride_service_logs/")
	if err != nil {
		log.Fatalln("failed to create logger:", err)
	}
	defer logger.Close()

	logger.Info("Ride service Logger", map[string]any{
		"status": "Initialized",
	})

	cfgDB, err := config.LoadDatabaseConfig("./config/db.yaml")
	if err != nil {
		logger.Error("LoadDatabaseConfig", map[string]any{
			"error":  err.Error(),
			"status": "failed",
		})
		return
	}

	cfgMQ, err := config.LoadRabbitConfig("./config/rabbitmq.yaml")
	if err != nil {
		logger.Error("LoadRabbitConfig", map[string]any{
			"error":  err.Error(),
			"status": "failed",
		})
		return
	}

	cfgServices, err := config.LoadServicesConfig("./config/services.yaml")
	if err != nil {
		logger.Error("LoadServicesConfig", map[string]any{
			"error":  err.Error(),
			"status": "failed",
		})
		return
	}

	cfgWS, err := config.LoadWSConfig("./config/ws.yaml")
	if err != nil {
		logger.Error("LoadWSConfig", map[string]any{
			"error":  err.Error(),
			"status": "failed",
		})
		return
	}

	cfgJWT, err := config.LoadJWTConfig("./config/jwt.yaml")
	if err != nil {
		logger.Error("LoadJWTConfig", map[string]any{
			"error":  err.Error(),
			"status": "failed",
		})
		return
	}

	logger.Info("Configurations loaded", map[string]any{
		"db_host":  cfgDB.Host,
		"db_port":  cfgDB.Port,
		"db_name":  cfgDB.Name,
		"mq_host":  cfgMQ.Host,
		"mq_port":  cfgMQ.Port,
		"ws_port":  cfgWS.Port,
		"JWT_exp":  cfgJWT.ExpiryMinutes,
		"Services": cfgServices,
		"status":   "success",
	})
}
