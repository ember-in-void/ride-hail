package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"ridehail/internal/driver/adapters/in/transport"
	messaging "ridehail/internal/driver/adapters/out/amqp"
	"ridehail/internal/driver/adapters/out/repo"
	"ridehail/internal/driver/application/usecase"
	"ridehail/internal/shared/auth"
	"ridehail/internal/shared/config"
	db_conn "ridehail/internal/shared/db"
	"ridehail/internal/shared/logger"
	"ridehail/internal/shared/mq"
)

// Run запускает Driver Service
func Run(ctx context.Context, cfg config.Config, log *logger.Logger) {
	log.Info(logger.Entry{Action: "driver_service_starting", Message: "initializing driver service"})

	// 1. Инициализация PostgreSQL
	dbPool, err := db_conn.NewPool(ctx, cfg.Database, log)
	if err != nil {
		log.Fatal(logger.Entry{
			Action:  "db_connection_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
	}
	defer db_conn.Close(dbPool, log)

	// Применяем миграции (shared migrations уже применены ride service)
	if err := db_conn.Migrate(ctx, dbPool, log); err != nil {
		log.Error(logger.Entry{
			Action:  "db_migration_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		// Не падаем если миграции уже применены
	}

	// 2. Инициализация RabbitMQ
	mqConn, err := mq.NewRabbitMQ(ctx, cfg.RabbitMQ, log)
	if err != nil {
		log.Fatal(logger.Entry{
			Action:  "rabbitmq_connection_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
	}
	defer mqConn.Close()

	// Создаем топологию (идемпотентно)
	if err := mq.SetupTopology(ctx, mqConn, log); err != nil {
		log.Error(logger.Entry{
			Action:  "rabbitmq_topology_setup_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		// Не падаем если топология уже создана
	}

	// 3. Инициализация репозиториев
	driverRepo := repo.NewDriverPgRepository(dbPool)
	locationRepo := repo.NewLocationRepository(dbPool)
	rideRepo := repo.NewRidePgRepository(dbPool)

	// 4. Инициализация MessagePublisher
	msgPublisher := messaging.NewMessagePublisher(mqConn, log)

	// 5. Инициализация use cases
	driverService := usecase.NewDriverService(
		driverRepo,
		locationRepo,
		rideRepo,
		msgPublisher,
		log,
	)

	// 6. Инициализация JWT сервиса для аутентификации
	jwtService := auth.NewJWTService(cfg.JWT)

	// 7. Инициализация HTTP handlers
	driverHandler := transport.NewDriverHandler(driverService, log)

	// 8. Создаем HTTP router с middleware
	mux := http.NewServeMux()

	// Health check endpoint (без аутентификации)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","service":"driver"}`))
	})

	// Создаем submux для защищенных endpoints
	protectedMux := http.NewServeMux()
	protectedMux.HandleFunc("POST /drivers/{driver_id}/online", driverHandler.HandleGoOnline)
	protectedMux.HandleFunc("POST /drivers/{driver_id}/offline", driverHandler.HandleGoOffline)
	protectedMux.HandleFunc("POST /drivers/{driver_id}/location", driverHandler.HandleUpdateLocation)
	protectedMux.HandleFunc("POST /drivers/{driver_id}/start", driverHandler.HandleStartRide)
	protectedMux.HandleFunc("POST /drivers/{driver_id}/complete", driverHandler.HandleCompleteRide)

	// Применяем middleware только к защищенным endpoints
	protectedHandler := transport.AuthMiddleware(jwtService, log)(protectedMux)

	// Объединяем в финальный handler
	finalMux := http.NewServeMux()
	finalMux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","service":"driver"}`))
	})
	finalMux.Handle("/drivers/", protectedHandler)

	// Применяем общие middleware (logging, request ID)
	handler := transport.LoggingMiddleware(log)(
		transport.RequestIDMiddleware(log)(finalMux),
	)

	// 9. HTTP Server
	addr := fmt.Sprintf(":%d", cfg.Services.DriverLocationServicePort)
	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Запускаем HTTP сервер в горутине
	go func() {
		log.Info(logger.Entry{
			Action:  "http_server_starting",
			Message: fmt.Sprintf("listening on %s", addr),
		})
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(logger.Entry{
				Action:  "http_server_failed",
				Message: err.Error(),
				Error:   &logger.ErrObj{Msg: err.Error()},
			})
		}
	}()

	// Ожидаем завершения контекста
	<-ctx.Done()
	log.Info(logger.Entry{Action: "driver_service_stopping", Message: "shutting down driver service"})

	// Graceful shutdown HTTP сервера
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error(logger.Entry{
			Action:  "http_server_shutdown_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
	} else {
		log.Info(logger.Entry{Action: "http_server_stopped", Message: "http server stopped gracefully"})
	}

	log.Info(logger.Entry{Action: "driver_service_stopped", Message: "driver service stopped"})
}
