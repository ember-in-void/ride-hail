# ride-hail

## treestructure

ridehail/
│
├── cmd/                         # Точки входа для сервисов
│   ├── ride-service/            # Ride Service (оркестратор поездок)
│   │   └── main.go
│   ├── driver-service/          # Driver & Location Service
│   │   └── main.go
│   └── admin-service/           # Admin Service (метрики, аналитика)
│       └── main.go
│
├── internal/                    # Внутренняя бизнес-логика
│   ├── ride/                    # Модуль Ride Service
│   │   ├── service.go           # Бизнес-логика (создание/отмена поездки, matching)
│   │   ├── handlers.go          # HTTP endpoints (POST /rides, cancel)
│   │   ├── ws.go                # WebSocket менеджер (уведомления пассажирам)
│   │   ├── consumer.go          # MQ consumer (driver responses)
│   │   └── producer.go          # MQ producer (ride requests, status updates)
│   │
│   ├── driver/                  # Модуль Driver & Location Service
│   │   ├── service.go           # Логика работы с водителями (online/offline, matching)
│   │   ├── handlers.go          # HTTP endpoints (online, offline, location, start/complete)
│   │   ├── ws.go                # WebSocket менеджер (уведомления водителям)
│   │   ├── consumer.go          # MQ consumer (ride requests → matching)
│   │   └── producer.go          # MQ producer (driver responses, location updates)
│   │
│   ├── admin/                   # Модуль Admin Service
│   │   ├── service.go           # Бизнес-логика (агрегации, отчёты)
│   │   ├── handlers.go          # HTTP endpoints (/admin/overview, /admin/rides/active)
│   │   └── queries.go           # SQL-запросы для аналитики
│   │
│   ├── db/                      # Работа с PostgreSQL
│   │   ├── connection.go        # Инициализация pgxpool
│   │   ├── ride_repo.go         # Репозиторий поездок
│   │   ├── driver_repo.go       # Репозиторий водителей
│   │   ├── location_repo.go     # Репозиторий координат/истории
│   │   └── migrations/          # SQL-миграции (DDL из README)
│   │
│   ├── mq/                      # Работа с RabbitMQ
│   │   ├── connection.go        # Подключение к RabbitMQ
│   │   ├── publisher.go         # Обёртка для публикации сообщений
│   │   ├── consumer.go          # Обёртка для консьюмеров
│   │   └── topology.go          # Объявления exchanges/queues/bindings
│   │
│   ├── ws/                      # Общий WebSocket менеджер
│   │   ├── hub.go               # Hub (пул соединений)
│   │   ├── client.go            # Клиент WebSocket
│   │   └── message.go           # Формат сообщений
│   │
│   ├── model/                   # Общие модели домена
│   │   ├── ride.go              # Ride (ID, passenger, driver, status...)
│   │   ├── driver.go            # Driver (ID, статус, рейтинг...)
│   │   ├── user.go              # User (ID, роль, статус)
│   │   ├── events.go            # RideEventType, DriverStatus
│   │   └── location.go          # Location/Coordinate
│   │
│   ├── config/                  # Конфигурации
│   │   └── config.go            # Чтение env/config.yaml
│   │
│   └── logger/                  # Логирование
│       └── logger.go            # JSON-логирование (zap/zerolog)
│
├── api/                         # Спецификации API (OpenAPI/Swagger, JSON Schema)
│   ├── ride.yaml
│   ├── driver.yaml
│   └── admin.yaml
│
├── deployments/                 # Инфраструктурные файлы
│   ├── docker-compose.yaml      # Локальный запуск (Postgres, RabbitMQ, сервисы)
│   ├── Dockerfile.ride
│   ├── Dockerfile.driver
│   ├── Dockerfile.admin
│   └── k8s/                     # Манифесты для Kubernetes (если понадобится)
│
├── scripts/                     # Утилиты/скрипты
│   ├── migrate.sh               # Запуск миграций
│   └── seed.sh                  # Тестовые данные
│
├── tests/                       # Интеграционные тесты
│   ├── ride_test.go
│   ├── driver_test.go
│   └── admin_test.go
│
├── go.mod
└── README.md
