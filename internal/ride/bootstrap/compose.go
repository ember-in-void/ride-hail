// ============================================================================
// BOOTSTRAP (Compose Root)
// ============================================================================
//
// 📦 НАЗНАЧЕНИЕ:
// Этот файл — "точка сборки" всего Ride Service. Здесь мы:
// 1. Создаем все зависимости (БД, RabbitMQ, WebSocket)
// 2. Собираем Use Cases с их зависимостями
// 3. Связываем адаптеры с Use Cases
// 4. Запускаем HTTP сервер и фоновые процессы
//
// 💡 ПРИНЦИП: Dependency Injection Container
// Все зависимости создаются в одном месте, затем передаются в конструкторы.
// Это позволяет легко заменить реализацию (например, подменить PostgreSQL
// на In-Memory репозиторий для тестов).
//
// 🏗️ АРХИТЕКТУРА:
//
//   Инфраструктура → Адаптеры → Use Cases → Domain
//   (PostgreSQL)     (Repository)  (Business Logic)  (Entities)
//        ↓               ↓              ↓               ↓
//   NewPool()    NewRidePgRepo()  NewRequestRide()  Ride{}
//        ↓               ↓              ↓
//   RabbitMQ      AMQP Consumer   HandleDriverResponse
//        ↓               ↓
//   WebSocket     WS Handler
//
// 📚 СЛОИ (создаются в таком порядке):
// 1. ИНФРАСТРУКТУРА: PostgreSQL, RabbitMQ, JWT
// 2. REPOSITORIES: Реализации интерфейсов для БД
// 3. USE CASES: Бизнес-логика
// 4. ADAPTERS: HTTP, WebSocket, AMQP
// 5. SERVER: Запуск всех компонентов
//
// ============================================================================

package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"time"

	inamqp "ridehail/internal/ride/adapter/in/in_amqp"
	"ridehail/internal/ride/adapter/in/in_ws"
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
)

// ============================================================================
// ГЛАВНАЯ ФУНКЦИЯ ЗАПУСКА RIDE SERVICE
// ============================================================================
//
// Run запускает Ride Service со всеми его компонентами.
//
// ЧТО ПРОИСХОДИТ ВНУТРИ:
// 1. Инициализация инфраструктуры (БД, RabbitMQ)
// 2. Создание всех Use Cases
// 3. Запуск AMQP consumers (в фоне)
// 4. Запуск WebSocket hub (в фоне)
// 5. Запуск HTTP сервера (блокирующий)
func Run(ctx context.Context, cfg config.Config, log *logger.Logger) {
	log.Info(logger.Entry{Action: "ride_service_starting", Message: "initializing ride service"})

	// ========================================================================
	// СЛОЙ 1: ИНФРАСТРУКТУРА
	// ========================================================================
	// Здесь создаем "низкоуровневые" компоненты: БД, очереди, JWT.
	// Они не знают о бизнес-логике — это просто технические инструменты.

	// 1. Инициализация PostgreSQL + PostGIS
	// Пул соединений для работы с базой данных.
	// Используем pgxpool для эффективного управления соединениями.
	dbPool, err := db_conn.NewPool(ctx, cfg.Database, log)
	if err != nil {
		log.Fatal(logger.Entry{
			Action:  "db_connection_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
	}
	defer db_conn.Close(dbPool, log)

	// Применяем миграции (создаем таблицы, индексы, extensions)
	if err := db_conn.Migrate(ctx, dbPool, log); err != nil {
		log.Fatal(logger.Entry{
			Action:  "db_migration_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
	}

	// 2. Инициализация RabbitMQ
	// Подключаемся к брокеру сообщений для асинхронной коммуникации.
	mqConn, err := mq.NewRabbitMQ(ctx, cfg.RabbitMQ, log)
	if err != nil {
		log.Fatal(logger.Entry{
			Action:  "rabbitmq_connection_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
	}
	defer mqConn.Close()

	// Создаем топологию RabbitMQ (exchanges, queues, bindings)
	// Это как создание "почтовых ящиков" и "маршрутов" для сообщений.
	if err := mq.SetupTopology(ctx, mqConn, log); err != nil {
		log.Fatal(logger.Entry{
			Action:  "rabbitmq_topology_setup_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
	}

	// 3. Инициализация JWT Service
	// Сервис для проверки токенов аутентификации.
	jwtService := auth.NewJWTService(cfg.JWT)

	// ========================================================================
	// СЛОЙ 2: WEBSOCKET HUB (для real-time уведомлений)
	// ========================================================================
	// WebSocket позволяет отправлять уведомления пассажирам в реальном времени
	// (например, "Водитель найден!", "Водитель прибыл на место").

	// Создаем WebSocket handler для пассажиров
	passengerWS := in_ws.NewPassengerWSHandler(jwtService, log)
	wsHub := passengerWS.GetHub()

	// Запускаем Hub в отдельной горутине
	// Hub управляет всеми WebSocket соединениями: регистрирует новые,
	// удаляет отключенные, отправляет сообщения.
	go wsHub.Run(ctx)

	// ========================================================================
	// СЛОЙ 3: REPOSITORIES (Адаптеры для работы с БД)
	// ========================================================================
	// Repositories — это "переводчики" между бизнес-логикой и БД.
	// Они реализуют интерфейсы, определенные в Use Cases.

	rideRepo := repo.NewRidePgRepository(dbPool, log)        // CRUD для rides
	coordRepo := repo.NewCoordinatePgRepository(dbPool, log) // CRUD для coordinates
	userRepo := user.NewPgRepository(dbPool, log)            // CRUD для users

	// ========================================================================
	// СЛОЙ 4: PUBLISHERS / NOTIFIERS (Адаптеры для отправки данных)
	// ========================================================================
	// Эти компоненты отправляют события и уведомления наружу:
	// - eventPublisher → отправляет события в RabbitMQ
	// - rideNotifier → отправляет уведомления через WebSocket

	eventPublisher := out_amqp.NewRideEventPublisher(mqConn, log) // Publish в RabbitMQ
	rideNotifier := out_ws.NewWsRideNotifier(wsHub, log)          // Send через WebSocket

	// ========================================================================
	// СЛОЙ 5: USE CASES (Бизнес-логика)
	// ========================================================================
	// Use Cases — это "мозг" приложения. Здесь описаны все бизнес-правила.
	// Они НЕ знают о деталях БД, HTTP или RabbitMQ — только об интерфейсах.

	// Use Case 1: Создание новой поездки пассажиром
	requestRideUC := usecase.NewRequestRideService(
		rideRepo,       // Для сохранения поездки в БД
		coordRepo,      // Для сохранения координат
		eventPublisher, // Для отправки события "ride_requested" водителям
		rideNotifier,   // Для уведомления пассажира (опционально)
		log,
	)

	// Use Case 2: Обработка ответа водителя (принял/отклонил поездку)
	handleDriverResponseUC := usecase.NewHandleDriverResponseService(
		rideRepo, // Для обновления поездки (назначение водителя)
		log,
	)

	// ========================================================================
	// СЛОЙ 6: CONSUMERS (Входящие адаптеры для RabbitMQ)
	// ========================================================================
	// Consumers "слушают" очереди RabbitMQ и вызывают Use Cases.

	// Consumer 1: Получает обновления местоположения от водителей
	// Маршрут: Driver App → Driver Service → RabbitMQ → Location Consumer → WebSocket Hub → Passenger App
	locationConsumer := inamqp.NewLocationConsumer(mqConn, passengerWS, log)
	go func() {
		if err := locationConsumer.Start(ctx); err != nil {
			log.Error(logger.Entry{
				Action:  "location_consumer_failed",
				Message: err.Error(),
				Error:   &logger.ErrObj{Msg: err.Error()},
			})
		}
	}()

	// Consumer 2: Получает ответы водителей (accept/reject)
	// Маршрут: Driver App → Driver Service → RabbitMQ → Driver Response Consumer → Use Case → PostgreSQL
	driverResponseConsumer := inamqp.NewDriverResponseConsumer(mqConn, handleDriverResponseUC, passengerWS, log)
	go func() {
		if err := driverResponseConsumer.Start(ctx); err != nil {
			log.Error(logger.Entry{
				Action:  "driver_response_consumer_failed",
				Message: err.Error(),
				Error:   &logger.ErrObj{Msg: err.Error()},
			})
		}
	}()

	// ========================================================================
	// СЛОЙ 7: HTTP HANDLER (Входящий адаптер для REST API)
	// ========================================================================
	// HTTP handler обрабатывает REST запросы и вызывает Use Cases.

	httpHandler := transport.NewHTTPHandler(requestRideUC, log)

	// ========================================================================
	// СЛОЙ 8: HTTP СЕРВЕР (Настройка и запуск)
	// ========================================================================

	mux := http.NewServeMux()

	// Middleware для проверки JWT токена + загрузки пользователя из БД
	// Без валидного токена запросы не пройдут дальше.
	authMiddleware := transport.JWTMiddleware(jwtService, userRepo, log)

	// Регистрируем маршруты REST API
	// POST /api/v1/rides/request — создать поездку
	httpHandler.RegisterRoutes(mux, authMiddleware)

	// WebSocket endpoint для пассажиров
	// Пассажиры подключаются сюда для получения real-time уведомлений
	mux.HandleFunc("/ws", passengerWS.ServeWS)

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
