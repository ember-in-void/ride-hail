# Ride Hail — Монорепозиторий, Микросервисы, Hex архитектура

## Принципы
- Monorepo: все сервисы в одном репозитории, один бинарник, разные процессы (-service=ride|driver|admin).
- Hex (Ports & Adapters) в каждом сервисе:
  - domain: сущности, инварианты, события, порты (интерфейсы).
  - application: use-case'ы, координация домена, без инфраструктуры.
  - adapters: in (HTTP/WS/AMQP вход), out (PG/AMQP/WS выход).
  - bootstrap: сборка зависимостей сервиса.
- Shared слой: только инфраструктура (config, logging, db, mq, ws, auth), без бизнес-логики.

## Репозиторий (общее дерево)

.
├── go.mod
├── main.go                        # единый бинарник: -service=ride|driver|admin|all
├── README.md
├── deployments/
│   └── docker-compose.yml         # Postgres + RabbitMQ + 3 процесса одного бинаря
├── build/
│   └── Makefile                   # gofumpt, vet, build, run-*
└── internal/
    ├── shared/                    # общая инфраструктура (без бизнес-логики)
    │   ├── config/
    │   │   └── config.go          # env: DB_*, RABBITMQ_*, WS_PORT, *_SERVICE_PORT, JWT_*_SECRET
    │   ├── logging/
    │   │   └── logger.go          # JSON-лог по схеме из ТЗ
    │   ├── db/
    │   │   ├── db.go              # pgxpool подключение
    │   │   ├── migrator.go        # go:embed раннер миграций
    │   │   └── migrations/
    │   │       └── 001_init.sql   # полная схема из ТЗ + outbox + extensions (pgcrypto, postgis)
    │   ├── mq/
    │   │   ├── rabbitmq.go        # подключение, QoS, переподключение
    │   │   └── topology.go        # exchanges/queues/bindings (ride_topic, driver_topic, location_fanout)
    │   ├── ws/
    │   │   └── hub.go             # WS-хаб: auth в 5 сек, ping/pong, broadcast
    │   └── auth/
    │       └── jwt.go             # JWT и роли (PASSENGER/DRIVER/ADMIN)
    │
    ├── ride/                      # Ride Service (оркестратор поездок)
    │   ├── domain/
    │   │   ├── ride.go            # сущности, статусы, инварианты
    │   │   ├── events.go          # доменные события
    │   │   └── errors.go
    │   ├── application/
    │   │   ├── ports/
    │   │   │   ├── in/
    │   │   │   │   ├── create_ride.go
    │   │   │   │   ├── cancel_ride.go
    │   │   │   │   └── handle_driver_response.go
    │   │   │   └── out/
    │   │   │       ├── ride_repository.go
    │   │   │       ├── event_store.go
    │   │   │       ├── outbox.go
    │   │   │       └── passenger_notifier.go
    │   │   └── usecase/
    │   │       ├── create_ride.go           # расчёт тарифа, запись, outbox: ride.request.*
    │   │       ├── cancel_ride.go           # CANCELLED → event + outbox: ride.status.CANCELLED
    │   │       └── handle_driver_response.go# MATCHED → WS пассажиру + outbox: ride.status.MATCHED
    │   ├── adapters/
    │   │   ├── in/
    │   │   │   ├── http/
    │   │   │   │   ├── handler.go           # POST /rides; POST /rides/{id}/cancel
    │   │   │   │   └── routes.go
    │   │   │   ├── ws/
    │   │   │   │   └── passenger_ws.go      # ws://.../ws/passengers/{passenger_id}
    │   │   │   └── amqp/
    │   │   │       ├── driver_response_consumer.go   # driver.response.{ride_id}
    │   │   │       └── location_consumer.go          # (fanout) location_updates_ride → WS пассажиру
    │   │   └── out/
    │   │       ├── pg/
    │   │       │   ├── ride_repository.go
    │   │       │   ├── event_store.go
    │   │       │   └── outbox_repo.go
    │   │       ├── amqp/
    │   │       │   ├── ride_publisher.go    # ride.request.*, ride.status.*
    │   │       │   └── outbox_worker.go     # читает outbox → публикует в MQ
    │   │       └── ws/
    │   │           └── passenger_notifier.go
    │   └── bootstrap/
    │       └── compose.go         # сборка: cfg/logger/db/mq/ws, HTTP, консьюмеры, outbox worker
    │
    ├── driver/                    # Driver & Location Service
    │   ├── domain/
    │   │   ├── driver.go
    │   │   ├── location.go
    │   │   └── errors.go
    │   ├── application/
    │   │   ├── ports/
    │   │   │   ├── in/
    │   │   │   │   ├── go_online.go
    │   │   │   │   ├── go_offline.go
    │   │   │   │   ├── update_location.go
    │   │   │   │   ├── handle_ride_request.go
    │   │   │   │   └── ride_lifecycle.go
    │   │   │   └── out/
    │   │   │       ├── driver_repository.go
    │   │   │       ├── location_repository.go
    │   │   │       ├── driver_response_publisher.go
    │   │   │       ├── driver_status_publisher.go
    │   │   │       ├── location_fanout_publisher.go
    │   │   │       └── driver_notifier.go
    │   │   └── usecase/
    │   │       ├── go_online.go
    │   │       ├── go_offline.go
    │   │       ├── update_location.go       # rate limit, history, broadcast
    │   │       ├── handle_ride_request.go   # consume ride.request.* → WS ride_offer → publish driver.response
    │   │       └── ride_lifecycle.go        # start/complete → driver.status + location_fanout
    │   ├── adapters/
    │   │   ├── in/
    │   │   │   ├── http/
    │   │   │   │   ├── handler.go           # /drivers/{id}/online|offline|location|start|complete
    │   │   │   │   └── routes.go
    │   │   │   ├── ws/
    │   │   │   │   └── driver_ws.go         # ride_response, location_update
    │   │   │   └── amqp/
    │   │   │       ├── ride_request_consumer.go # ride.request.*
    │   │   │       └── ride_status_consumer.go  # ride.status.*
    │   │   └── out/
    │   │       ├── pg/
    │   │       │   ├── driver_repository.go
    │   │       │   └── location_repository.go
    │   │       ├── amqp/
    │   │       │   ├── driver_response_publisher.go
    │   │       │   ├── driver_status_publisher.go
    │   │       │   └── location_fanout_publisher.go
    │   │       └── ws/
    │   │           └── driver_notifier.go
    │   └── bootstrap/
    │       └── compose.go
    │
    └── admin/                     # Admin Dashboard API
        ├── domain/
        │   └── types.go
        ├── application/
        │   ├── ports/
        │   │   ├── in/
        │   │   │   ├── get_overview.go
        │   │   │   └── get_active_rides.go
        │   │   └── out/
        │   │       └── stats_repository.go
        │   └── usecase/
        │       ├── get_overview.go
        │       └── get_active_rides.go
        ├── adapters/
        │   ├── in/
        │   │   └── http/
        │   │       ├── handler.go           # GET /admin/overview, /admin/rides/active
        │   │       └── routes.go
        │   └── out/
        │       └── pg/
        │           └── stats_repository.go
        └── bootstrap/
            └── compose.go

## Где что живёт
- shared/*: инфраструктура (DB, MQ, WS, конфиг, логирование, JWT). Не импортируется из domain/application.
- <service>/domain: только бизнес-модель, инварианты, Без зависимостей на shared.
- <service>/application: use-case'ы, оркестрация домена через порты, порты
- <service>/adapters/in: входные адаптеры (HTTP/WS/AMQP) — вызывают use-case'ы.
- <service>/adapters/out: реализации портов (PG/AMQP/WS) — используются use-case'ами.
- <service>/bootstrap: сборка зависимостей и запуск сервиса.

## Запуск одним бинарником
- go build -o ride-hail-system .
- ./ride-hail-system -service=ride   # порт 3000
- ./ride-hail-system -service=driver # порт 3001
- ./ride-hail-system -service=admin  # порт 3004
