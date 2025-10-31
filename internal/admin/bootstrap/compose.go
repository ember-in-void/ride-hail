package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"ridehail/internal/admin/adapters/in/transport"
	"ridehail/internal/admin/adapters/out/repo"
	"ridehail/internal/admin/application/usecase"
	"ridehail/internal/shared/auth"
	"ridehail/internal/shared/config"
	db_conn "ridehail/internal/shared/db"
	"ridehail/internal/shared/logger"
)

// Run запускает Admin Service
func Run(ctx context.Context, cfg config.Config, log *logger.Logger) {
	log.Info(logger.Entry{Action: "admin_service_starting", Message: "initializing admin service"})

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

	// Применяем миграции (идемпотентно)
	if err := db_conn.Migrate(ctx, dbPool, log); err != nil {
		log.Error(logger.Entry{
			Action:  "db_migration_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		// Не падаем если миграции уже применены
	}

	// 2. Инициализация JWT сервиса
	jwtService := auth.NewJWTService(cfg.JWT)

	// 3. Создаем репозитории (Adapter OUT)
	userRepo := repo.NewUserPgRepository(dbPool, log)

	// 4. Создаем use cases (Application)
	createUserUC := usecase.NewCreateUserService(userRepo, log)
	listUsersUC := usecase.NewListUsersService(userRepo, log)
	getOverviewUC := usecase.NewGetOverviewService(userRepo, log)
	getActiveRidesUC := usecase.NewGetActiveRidesService(userRepo, log)

	// 5. Создаем HTTP handler (Adapter IN)
	httpHandler := transport.NewHTTPHandler(createUserUC, listUsersUC, getOverviewUC, getActiveRidesUC, log)

	// 6. Настраиваем HTTP сервер
	mux := http.NewServeMux()

	// Middleware для ADMIN аутентификации
	adminAuthMiddleware := transport.AdminAuthMiddleware(jwtService, log)

	// Регистрируем маршруты
	httpHandler.RegisterRoutes(mux, adminAuthMiddleware)

	// HTTP сервер
	addr := fmt.Sprintf(":%d", cfg.Services.AdminServicePort)
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
	log.Info(logger.Entry{Action: "admin_service_stopping", Message: "shutting down admin service"})

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

	log.Info(logger.Entry{Action: "admin_service_stopped", Message: "admin service stopped"})
}
