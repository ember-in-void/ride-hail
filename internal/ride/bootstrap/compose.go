package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"ridehail/internal/ride/adapter/in/transport"
	"ridehail/internal/ride/adapter/out/out_amqp"
	"ridehail/internal/ride/adapter/out/out_ws"
	"ridehail/internal/ride/adapter/out/repo"
	"ridehail/internal/ride/application/usecase"
	"ridehail/internal/shared/auth"
	"ridehail/internal/shared/config"
	db_conn "ridehail/internal/shared/db"
	"ridehail/internal/shared/logger"
	"ridehail/internal/shared/mq"
	"ridehail/internal/shared/user"
	"ridehail/internal/shared/ws"
)

// Run запускает Ride Service
func Run(ctx context.Context, cfg config.Config, log *logger.Logger) {
	log.Info(logger.Entry{Action: "ride_service_starting", Message: "initializing ride service"})

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

	// Применяем миграции
	if err := db_conn.Migrate(ctx, dbPool, log); err != nil {
		log.Fatal(logger.Entry{
			Action:  "db_migration_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
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

	// Создаем топологию RabbitMQ
	if err := mq.SetupTopology(ctx, mqConn, log); err != nil {
		log.Fatal(logger.Entry{
			Action:  "rabbitmq_topology_setup_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
	}

	// 3. Инициализация WebSocket Hub
	jwtService := auth.NewJWTService(cfg.JWT)
	wsHub := ws.NewHub(jwtService.ExtractUserID, log)
	go wsHub.Run(ctx)

	// 4. Создаем репозитории (Adapter OUT)
	rideRepo := repo.NewRidePgRepository(dbPool, log)
	coordRepo := repo.NewCoordinatePgRepository(dbPool, log)
	userRepo := user.NewPgRepository(dbPool, log) // NEW: User repository

	// 5. Создаем publishers/notifiers (Adapter OUT)
	eventPublisher := out_amqp.NewRideEventPublisher(mqConn, log)
	rideNotifier := out_ws.NewWsRideNotifier(wsHub, log)

	// 6. Создаем use cases (Application)
	requestRideUC := usecase.NewRequestRideService(
		rideRepo,
		coordRepo,
		eventPublisher,
		rideNotifier,
		log,
	)

	// 7. Создаем HTTP handler (Adapter IN)
	httpHandler := transport.NewHTTPHandler(requestRideUC, log)

	// 8. Настраиваем HTTP сервер
	mux := http.NewServeMux()

	// Middleware для JWT + проверка пользователя в БД
	authMiddleware := transport.JWTMiddleware(jwtService, userRepo, log)

	// Регистрируем маршруты
	httpHandler.RegisterRoutes(mux, authMiddleware)

	// WebSocket endpoint
	mux.HandleFunc("/ws", wsHub.ServeWS)

	// HTTP сервер
	addr := fmt.Sprintf(":%d", cfg.Services.RideServicePort)
	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
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
	log.Info(logger.Entry{Action: "ride_service_stopping", Message: "shutting down ride service"})

	// Завершаем работу HTTP сервера
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

	log.Info(logger.Entry{Action: "ride_service_stopped", Message: "ride service stopped"})
}
