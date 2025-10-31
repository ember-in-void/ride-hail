package bootstrap

import (
	"context"
	"fmt"
	"net/http"

	"ridehail/internal/driver/adapters/in/amqp"
	wsAdapter "ridehail/internal/driver/adapters/in/in_ws"
	httpAdapter "ridehail/internal/driver/adapters/in/transport"
	"ridehail/internal/driver/adapters/out/messaging"
	"ridehail/internal/driver/adapters/out/notification"
	"ridehail/internal/driver/adapters/out/persistence"
	"ridehail/internal/driver/application/usecase"
	"ridehail/internal/shared/auth"
	"ridehail/internal/shared/config"
	"ridehail/internal/shared/db"
	"ridehail/internal/shared/logger"
	"ridehail/internal/shared/mq"
	"ridehail/internal/shared/ws"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DriverService struct {
	httpServer *http.Server
	pool       *pgxpool.Pool
	mq         *mq.RabbitMQ
	wsHub      *ws.Hub
	log        *logger.Logger
}

func InitializeDriverService(ctx context.Context, cfg config.Config) (*DriverService, error) {
	log := logger.NewLogger("driver-service")

	log.Info(logger.Entry{
		Action:  "driver_service_initializing",
		Message: "initializing driver service",
	})

	// 1. Postgres
	pool, err := db.NewPool(ctx, cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("init postgres: %w", err)
	}

	// 2. RabbitMQ
	rabbitMQ, err := mq.NewRabbitMQ(ctx, cfg.RabbitMQ, log)
	if err != nil {
		return nil, fmt.Errorf("init rabbitmq: %w", err)
	}

	// 3. Setup RabbitMQ Topology
	if err := mq.SetupTopology(ctx, rabbitMQ, log); err != nil {
		return nil, fmt.Errorf("setup topology: %w", err)
	}

	// 4. WebSocket Hub
	wsHub := ws.NewHub(log)
	go wsHub.Run()

	// 5. JWT Service
	jwtService := auth.NewJWTService(cfg.JWT)

	// 6. Repositories (OUT Ports)
	driverRepo := persistence.NewDriverPgRepository(pool)
	sessionRepo := persistence.NewSessionPgRepository(pool)
	locationRepo := persistence.NewLocationPgRepository(pool)
	rideRepo := persistence.NewRidePgRepository(pool)

	// 7. Event Publisher
	eventPublisher := messaging.NewEventPublisher(rabbitMQ, log)

	// 8. Driver Notifier (WebSocket)
	driverNotifier := notification.NewDriverNotifier(wsHub, log)

	// 9. Use Cases (IN Ports)
	goOnlineUC := usecase.NewGoOnlineUseCase(driverRepo, sessionRepo, locationRepo, eventPublisher, log)
	goOfflineUC := usecase.NewGoOfflineUseCase(driverRepo, sessionRepo, eventPublisher, log)
	updateLocationUC := usecase.NewUpdateLocationUseCase(locationRepo, eventPublisher, log)
	startRideUC := usecase.NewStartRideUseCase(driverRepo, rideRepo, locationRepo, log)
	completeRideUC := usecase.NewCompleteRideUseCase(driverRepo, sessionRepo, rideRepo, locationRepo, log)

	// 10. HTTP Handler
	httpHandler := httpAdapter.NewHandler(goOnlineUC, goOfflineUC, updateLocationUC, startRideUC, completeRideUC, log)

	// 11. WebSocket Handler
	wsConnHandler := wsAdapter.NewConnectionHandler(wsHub, jwtService, log)
	wsHubHandler := wsAdapter.NewHub(wsHub, eventPublisher, rideRepo, log)

	// Запускаем WebSocket Hub для обработки ride responses
	go wsHubHandler.Start(ctx)

	// 12. AMQP Consumer (ride.requested)
	rideConsumer := amqp.NewRideConsumer(rabbitMQ, driverRepo, driverNotifier, eventPublisher, log)
	if err := rideConsumer.Start(ctx); err != nil {
		return nil, fmt.Errorf("start ride consumer: %w", err)
	}

	// 13. HTTP Router
	mux := http.NewServeMux()

	// Health endpoint
	mux.HandleFunc("/health", httpHandler.Health)

	// WebSocket endpoint
	mux.HandleFunc("/ws/drivers/", wsConnHandler.HandleWebSocket)

	// Protected endpoints (с JWT middleware)
	jwtMiddleware := httpAdapter.JWTMiddleware(jwtService, log)

	mux.Handle("/drivers/online", jwtMiddleware(http.HandlerFunc(httpHandler.GoOnline)))
	mux.Handle("/drivers/offline", jwtMiddleware(http.HandlerFunc(httpHandler.GoOffline)))
	mux.Handle("/drivers/location", jwtMiddleware(http.HandlerFunc(httpHandler.UpdateLocation)))
	mux.Handle("/drivers/start", jwtMiddleware(http.HandlerFunc(httpHandler.StartRide)))
	mux.Handle("/drivers/complete", jwtMiddleware(http.HandlerFunc(httpHandler.CompleteRide)))

	// 14. HTTP Server
	port := cfg.Services.DriverLocationServicePort
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	log.Info(logger.Entry{
		Action:  "driver_service_initialized",
		Message: fmt.Sprintf("driver service listening on port %d", port),
		Additional: map[string]any{
			"port": port,
		},
	})

	return &DriverService{
		httpServer: httpServer,
		pool:       pool,
		mq:         rabbitMQ,
		wsHub:      wsHub,
		log:        log,
	}, nil
}

func (s *DriverService) Start() error {
	s.log.Info(logger.Entry{
		Action:  "driver_service_starting",
		Message: "starting http server",
	})

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("http server error: %w", err)
	}

	return nil
}

func (s *DriverService) Shutdown(ctx context.Context) error {
	s.log.Info(logger.Entry{
		Action:  "driver_service_shutting_down",
		Message: "gracefully shutting down driver service",
	})

	// 1. Shutdown HTTP server
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.log.Error(logger.Entry{
			Action:  "http_server_shutdown_error",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
	}

	// 2. Close WebSocket Hub
	s.wsHub.Close()

	// 3. Close RabbitMQ
	s.mq.Close()

	// 4. Close Postgres
	s.pool.Close()

	s.log.Info(logger.Entry{
		Action:  "driver_service_stopped",
		Message: "driver service stopped successfully",
	})

	return nil
}
