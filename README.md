# Ride Hail System

Микросервисная система управления поездками на Go с Hexagonal Architecture.

## 🏗️ Архитектура

- **Ride Service** (порт 3000) — управление поездками
- **Driver Service** (порт 3001) — управление водителями и локацией
- **Admin Service** (порт 3004) — административная панель
- **PostgreSQL** с PostGIS — основная БД
- **RabbitMQ** — message broker
- **WebSocket** (порт 8080) — real-time уведомления

## 🚀 Быстрый старт

### Предварительные требования

- Go 1.24+
- Docker и Docker Compose
- Make (опционально)

### Локальная разработка

```bash
# Клонировать репозиторий
git clone https://github.com/ember-in-void/ride-hail.git
cd ride-hail

# Установить зависимости
go mod download

# Собрать проект
make build

# Запустить все сервисы локально
make run-all
```

### Запуск в Docker

```bash
# Собрать и запустить все сервисы
make docker-up-build

# Или вручную
cd deployments
docker compose up --build

# Просмотр логов
make docker-logs

# Остановка
make docker-down
```

## 📡 API Endpoints

### Ride Service (http://localhost:3000)

- `GET /health` — health check
- `POST /rides` — создать поездку (требует JWT)
- `GET /ws` — WebSocket соединение

### JWT Authentication

```bash
# Сгенерировать токен
make generate-jwt

# Или с параметрами
go run cmd/generate-jwt/main.go \
  -user=test-user-123 \
  -email=passenger@test.com \
  -role=PASSENGER
```

### Пример запроса

```bash
# Установить токен
export TOKEN="your-jwt-token-here"

# Создать поездку
curl -X POST http://localhost:3000/rides \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "vehicle_type": "ECONOMY",
    "pickup_lat": 55.7558,
    "pickup_lng": 37.6173,
    "pickup_address": "Red Square, Moscow",
    "destination_lat": 55.7522,
    "destination_lng": 37.6156,
    "destination_address": "Kremlin, Moscow"
  }'
```

## 🗄️ База данных

```bash
# Подключиться к PostgreSQL
make db-shell

# Или вручную
docker exec -it ridehail-postgres psql -U ridehail_user -d ridehail_db

# Проверить созданные поездки
SELECT * FROM rides ORDER BY created_at DESC LIMIT 5;
```

## 🐰 RabbitMQ

```bash
# Открыть Management UI
# http://localhost:15672
# Login: guest / guest

# Проверить очереди
# Exchanges: ride_topic, driver_topic, location_fanout
# Queues: ride.requested, ride.matched, ride.completed, etc.
```

## 🛠️ Полезные команды

```bash
# Показать все доступные команды
make help

# Запустить тесты
make test

# Линтер
make lint

# Очистить артефакты
make clean

# Пересобрать Docker образы
make docker-build

# Перезапустить сервисы
make docker-restart
```

## 📝 Структура проекта

```
ride-hail/
├── cmd/                      # CLI утилиты
│   └── generate-jwt/         # Генератор JWT токенов
├── config/                   # Конфигурационные файлы
│   ├── db.yaml
│   ├── mq.yaml
│   ├── service.yaml
│   ├── ws.yaml
│   └── jwt.yaml
├── deployments/              # Docker файлы
│   ├── Dockerfile
│   └── docker-compose.yml
├── internal/
│   ├── ride/                 # Ride Service
│   │   ├── domain/
│   │   ├── application/
│   │   ├── adapter/
│   │   └── bootstrap/
│   ├── driver/               # Driver Service
│   ├── admin/                # Admin Service
│   └── shared/               # Общая инфраструктура
│       ├── config/
│       ├── logger/
│       ├── db/
│       ├── mq/
│       ├── ws/
│       └── auth/
├── main.go                   # Точка входа
├── Makefile
└── README.md
```

## 🧪 Тестирование

### Unit тесты

```bash
make test
```

### Интеграционные тесты

```bash
# Запустить сервисы
make docker-up

# Выполнить тесты
./scripts/integration-test.sh
```

## 📊 Мониторинг

- **Logs**: JSON structured logging в stdout
- **Health checks**: `/health` endpoints
- **RabbitMQ**: http://localhost:15672

## 🔧 Troubleshooting

### Проблема с Docker Buildx

Если получаете ошибку `fork/exec .../docker-buildx: no such file or directory`:

```bash
# Используйте обычный docker build вместо buildx
docker build -f deployments/Dockerfile -t ride-hail .
```

### Порты заняты

```bash
# Проверить занятые порты
sudo lsof -i :3000
sudo lsof -i :5432

# Изменить порты в docker-compose.yml
```

### Проблемы с миграциями

```bash
# Пересоздать БД
make docker-down-volumes
make docker-up
```

## 📄 Лицензия

MIT