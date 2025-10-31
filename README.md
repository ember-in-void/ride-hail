# 🚗 Ride-Hailing System

> Полнофункциональная микросервисная система вызова такси с real-time коммуникацией, геопространственным матчингом и event-driven архитектурой.

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://golang.org)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-336791?style=flat&logo=postgresql)](https://postgresql.org)
[![RabbitMQ](https://img.shields.io/badge/RabbitMQ-3.13-FF6600?style=flat&logo=rabbitmq)](https://rabbitmq.com)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

## 📋 Содержание

- [Обзор](#-обзор)
- [Архитектура](#️-архитектура)
- [Быстрый старт](#-быстрый-старт)
- [Проверка работы](#-проверка-работы)
- [API Documentation](#-api-documentation)
- [WebSocket](#-websocket)
- [RabbitMQ](#-rabbitmq)
- [База данных](#️-база-данных)
- [Тестирование](#-тестирование)
- [Troubleshooting](#-troubleshooting)

---

## 🌟 Обзор

**Ride-Hailing System** — это production-ready backend для системы вызова такси, реализованный с использованием современных архитектурных паттернов и технологий.

### Ключевые возможности

✨ **Real-time коммуникация**
- WebSocket соединения для пассажиров и водителей
- Мгновенные уведомления о матчинге
- Live отслеживание локации водителя

🗺️ **Геопространственный матчинг**
- PostGIS для поиска ближайших водителей
- Radius search (5km) с ST_DWithin
- Оптимизация через GIST индексы

📨 **Event-Driven Architecture**
- RabbitMQ для асинхронной коммуникации
- Topic и Fanout exchanges
- Автоматический retry и error handling

🏗️ **Clean Architecture**
- Hexagonal Pattern (Ports & Adapters)
- SOLID principles
- Полная независимость от frameworks

### Технологический стек

**Backend:**
- Go 1.24+
- PostgreSQL 16 + PostGIS
- RabbitMQ 3.13
- gorilla/websocket
- pgx/v5
- golang-jwt/jwt/v5

**Infrastructure:**
- Docker & Docker Compose
- Health checks
- Graceful shutdown

---

## 🏗️ Архитектура

### Микросервисы

```
┌─────────────────────────────────────────────────────────┐
│                   RIDE-HAILING SYSTEM                   │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │
│  │ Ride Service │  │Driver Service│  │Admin Service │ │
│  │   :3000      │  │    :3001     │  │    :3002     │ │
│  │              │  │              │  │              │ │
│  │ • Rides      │  │ • Drivers    │  │ • Users      │ │
│  │ • Passengers │  │ • Location   │  │ • Overview   │ │
│  │ • WebSocket  │  │ • Matching   │  │ • Analytics  │ │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘ │
│         │                 │                  │         │
│         └─────────────────┴──────────────────┘         │
│                           │                            │
│         ┌─────────────────┴─────────────────┐         │
│         │                                   │         │
│    ┌────▼─────┐                      ┌─────▼──────┐  │
│    │PostgreSQL│                      │  RabbitMQ  │  │
│    │  :5432   │                      │   :5672    │  │
│    │ + PostGIS│                      │ Management │  │
│    └──────────┘                      │   :15672   │  │
│                                      └────────────┘  │
└─────────────────────────────────────────────────────────┘
```

### Компоненты системы

#### 1. **Ride Service** (порт 3000)
- Управление поездками
- WebSocket для пассажиров
- RabbitMQ consumers (location updates, driver responses)
- REST API для создания поездок

#### 2. **Driver Service** (порт 3001)
- Управление водителями
- WebSocket для водителей
- PostGIS matching algorithm
- RabbitMQ consumers (ride requests)
- Location tracking

#### 3. **Admin Service** (порт 3002)
- Административная панель
- Управление пользователями
- Статистика и аналитика

#### 4. **PostgreSQL + PostGIS**
- Основная база данных
- Геопространственные запросы
- Хранение rides, drivers, coordinates

#### 5. **RabbitMQ**
- Message broker
- 3 exchanges: ride_topic, driver_topic, location_fanout
- Event-driven коммуникация между сервисами

---

## 🚀 Быстрый старт

### Предварительные требования

- Go 1.24+ ([установка](https://golang.org/dl/))
- Docker и Docker Compose ([установка](https://docs.docker.com/get-docker/))
- jq для тестовых скриптов (опционально)

### Шаг 1: Клонирование репозитория

```bash
git clone https://github.com/ember-in-void/ride-hail.git
cd ride-hail
```

### Шаг 2: Запуск инфраструктуры

```bash
cd deployments
docker compose up -d
```

Это запустит:
- ✅ PostgreSQL на порту 5432
- ✅ RabbitMQ на портах 5672 (AMQP) и 15672 (Management UI)

Проверка статуса:
```bash
docker compose ps

# Должны быть запущены:
# ridehail-postgres
# ridehail-rabbitmq
```

### Шаг 3: Сборка проекта

```bash
cd ..  # вернуться в корень проекта
go build -o bin/ridehail ./main.go
```

Проверка сборки:
```bash
ls -lh bin/ridehail
# Должен быть создан исполняемый файл ~16MB
```

### Шаг 4: Запуск сервисов

Откройте **3 терминала** и запустите каждый сервис:

**Terminal 1 - Ride Service:**
```bash
./bin/ridehail
# Запустится на порту 3000
```

**Terminal 2 - Driver Service:**
```bash
SERVICE_MODE=driver ./bin/ridehail
# Запустится на порту 3001
```

**Terminal 3 - Admin Service:**
```bash
SERVICE_MODE=admin ./bin/ridehail
# Запустится на порту 3002
```

### Шаг 5: Проверка работы

```bash
# Проверка здоровья всех сервисов
curl http://localhost:3000/health  # Ride Service
curl http://localhost:3001/health  # Driver Service
curl http://localhost:3004/health  # Admin Service

# Ожидаемый ответ от каждого:
# {"status":"ok","service":"ride"}
# {"status":"ok","service":"driver"}
# {"status":"ok","service":"admin"}
```

---

## ✅ Проверка работы

### 1. Проверка инфраструктуры

#### PostgreSQL
```bash
# Подключение к базе данных
docker exec -it ridehail-postgres psql -U ridehail_user -d ridehail_db

# Проверка PostGIS
ridehail_db=# SELECT PostGIS_version();
# Должна показать версию PostGIS

# Проверка таблиц
ridehail_db=# \dt
# Список таблиц: users, drivers, rides, coordinates, location_history

# Проверка индексов
ridehail_db=# \di
# Должен быть idx_coordinates_geography (GIST)

# Выход
ridehail_db=# \q
```

#### RabbitMQ
```bash
# Открыть Management UI в браузере
# http://localhost:15672
# Login: guest / Password: guest

# Проверить exchanges:
# - ride_topic (type: topic)
# - driver_topic (type: topic)
# - location_fanout (type: fanout)

# Проверить queues:
# - driver_matching
# - ride_service_driver_responses
# - ride_service_locations
```

### 2. Проверка WebSocket соединений

```bash
# Запустить автоматический тест
chmod +x scripts/test-websocket.sh
./scripts/test-websocket.sh
```

Ожидаемый результат:
```
========================================
Testing WebSocket Connections
========================================

[1/2] Testing Ride Service WebSocket...
  ✓ Connection successful (HTTP 101 Switching Protocols)

[2/2] Testing Driver Service WebSocket...
  ✓ Connection successful (HTTP 101 Switching Protocols)

========================================
✅ All WebSocket tests passed!
========================================
```

### 3. Проверка Driver API

```bash
# Запустить полный тест Driver Service
chmod +x scripts/test-driver-api.sh
./scripts/test-driver-api.sh
```

Этот тест проверяет:
1. ✅ Создание водителя через Admin API
2. ✅ GoOnline endpoint
3. ✅ UpdateLocation с PostGIS
4. ✅ Location публикация в RabbitMQ
5. ✅ GoOffline endpoint

### 4. End-to-End тест полного флоу

```bash
# Запустить E2E тест
chmod +x scripts/test-e2e-ride-flow.sh
./scripts/test-e2e-ride-flow.sh
```

Этот тест проверяет полный цикл:
1. ✅ Генерация JWT токенов для пассажира и водителя
2. ✅ Создание пользователей в БД
3. ✅ Водитель выходит онлайн
4. ✅ Водитель обновляет локацию (Moscow: 55.7558, 37.6173)
5. ✅ Пассажир создает поездку (Red Square → Kremlin)
6. ✅ Ride Service публикует в RabbitMQ
7. → Driver Service находит водителя с PostGIS (5km radius)
8. → Driver получает offer через WebSocket
9. → Driver отвечает через WebSocket
10. → Ride Service получает ответ
11. → Passenger получает уведомление

### 5. 🎬 Demo: Полный цикл поездки (красивый вывод)

**Новый красивый demo-скрипт с цветным выводом и детальным логированием!**

```bash
# Запустить красивую демонстрацию полного цикла
chmod +x scripts/demo-full-ride-cycle.sh
./scripts/demo-full-ride-cycle.sh
```

**Что показывает demo:**

```
🚗 RIDE-HAILING SYSTEM - FULL CYCLE DEMONSTRATION 🚗

STEP 0:  ✓ Проверка доступности всех сервисов
STEP 1:  ✓ Генерация тестовых UUID и данных
STEP 2:  ✓ Создание JWT токенов (ADMIN, PASSENGER, DRIVER)
STEP 3:  👤 Создание пассажира и 🚗 водителя через Admin API
STEP 4:  🚗 Водитель выходит онлайн (статус → AVAILABLE)
STEP 5:  📍 Обновление локации водителя (Almaty Central Park)
STEP 6:  👤 Пассажир создает поездку (Central Park → Kok-Tobe)
         🚀 RabbitMQ: ride.request.ECONOMY → driver_matching queue
         📊 PostGIS: ST_DWithin(5km) - поиск водителей
STEP 7:  🚗 Водитель получает и принимает предложение
         🚀 RabbitMQ: driver.response → ride_service_driver_responses
STEP 8:  ⏱ Водитель начинает поездку (статус → IN_PROGRESS)
STEP 9:  📍 Симуляция движения с обновлением локации:
         • 43.235, 76.885 - Moving towards destination (25.5 km/h)
         • 43.230, 76.870 - Halfway there (35.2 km/h)
         • 43.225, 76.860 - Almost arrived (28.7 km/h)
         • 43.222, 76.851 - Arriving at destination (15.3 km/h)
STEP 10: 💰 Водитель завершает поездку
         Distance: 5.2 km | Duration: 18 min
STEP 11: 📊 Проверка Admin Dashboard (метрики и активные поездки)

✓ ВСЕ ЭТАПЫ УСПЕШНО ВЫПОЛНЕНЫ!
```

**Особенности demo-скрипта:**
- 🎨 Красивый цветной вывод с эмодзи
- 📝 Детальное логирование каждого шага
- ⚡ Автоматическая проверка доступности сервисов
- 🔍 Вывод всех созданных UUID для отладки
- 📊 Финальная таблица с результатами тестирования
- 🎯 Симуляция реального движения водителя
- ✅ Проверка всех компонентов системы

**Проверяемые компоненты:**
- JWT Authentication (3 роли)
- Admin Service (создание пользователей, метрики)
- Driver Service (lifecycle, локация, PostGIS)
- Ride Service (создание поездок, RabbitMQ)
- RabbitMQ (3 exchanges, все queues)
- PostGIS (ST_DWithin геопоиск в радиусе 5km)
- WebSocket simulation (ride offers & responses)

---

## 📡 API Documentation

### Ride Service (http://localhost:3000)

### Ride Service (http://localhost:3000)

#### Endpoints

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| GET | `/health` | Health check | No |
| POST | `/rides` | Создать поездку | JWT (PASSENGER/ADMIN) |
| GET | `/ws` | WebSocket для пассажиров | JWT |

#### POST /rides - Создание поездки

**Request:**
```bash
curl -X POST http://localhost:3000/rides \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "vehicle_type": "ECONOMY",
    "pickup_lat": 55.7558,
    "pickup_lng": 37.6173,
    "pickup_address": "Red Square, Moscow",
    "destination_lat": 55.7522,
    "destination_lng": 37.6156,
    "destination_address": "Kremlin, Moscow",
    "priority": 5
  }'
```

**Response (201 Created):**
```json
{
  "ride_id": "abc-123-def-456",
  "ride_number": "R-20251031-001",
  "status": "PENDING",
  "estimated_fare": 250.50,
  "pickup_address": "Red Square, Moscow",
  "destination_address": "Kremlin, Moscow"
}
```

### Driver Service (http://localhost:3001)

#### Endpoints

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| GET | `/health` | Health check | No |
| POST | `/drivers/{id}/online` | Выход онлайн | JWT (DRIVER) |
| POST | `/drivers/{id}/offline` | Выход оффлайн | JWT (DRIVER) |
| POST | `/drivers/{id}/location` | Обновить локацию | JWT (DRIVER) |
| POST | `/drivers/{id}/start` | Начать поездку | JWT (DRIVER) |
| POST | `/drivers/{id}/complete` | Завершить поездку | JWT (DRIVER) |
| GET | `/ws` | WebSocket для водителей | JWT |

#### POST /drivers/{id}/online - Выход онлайн

**Request:**
```bash
curl -X POST http://localhost:3001/drivers/driver-123/online \
  -H "Authorization: Bearer YOUR_DRIVER_JWT_TOKEN" \
  -H "Content-Type: application/json"
```

**Response (200 OK):**
```json
{
  "driver_id": "driver-123",
  "status": "AVAILABLE",
  "is_online": true,
  "timestamp": "2025-10-31T12:00:00Z"
}
```

#### POST /drivers/{id}/location - Обновление локации

**Request:**
```bash
curl -X POST http://localhost:3001/drivers/driver-123/location \
  -H "Authorization: Bearer YOUR_DRIVER_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "latitude": 55.7558,
    "longitude": 37.6173,
    "accuracy_meters": 10.0,
    "speed_kmh": 45.5,
    "heading_degrees": 180
  }'
```

**Response (200 OK):**
```json
{
  "success": true,
  "coordinate_id": "coord-789",
  "message": "Location updated successfully"
}
```

**Что происходит:**
1. Локация сохраняется в PostgreSQL с PostGIS
2. Публикуется в RabbitMQ exchange `location_fanout`
3. Все подписчики (Ride Service) получают обновление
4. Пассажиры получают уведомление через WebSocket

### Admin Service (http://localhost:3004)

#### Endpoints

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| GET | `/health` | Health check | No |
| POST | `/admin/users` | Создать пользователя | JWT (ADMIN) |
| GET | `/admin/users` | Список пользователей | JWT (ADMIN) |
| GET | `/admin/overview` | Обзор системы | JWT (ADMIN) |
| GET | `/admin/rides/active` | Активные поездки | JWT (ADMIN) |

#### POST /admin/users - Создание пользователя

**Request:**
```bash
curl -X POST http://localhost:3004/admin/users \
  -H "Authorization: Bearer YOUR_ADMIN_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "user-123",
    "email": "driver@example.com",
    "role": "DRIVER",
    "phone": "+79991234567"
  }'
```

**Response (201 Created):**
```json
{
  "id": "user-123",
  "email": "driver@example.com",
  "role": "DRIVER",
  "created_at": "2025-10-31T12:00:00Z"
}
```

---

## 🔐 JWT Authentication

### Генерация токенов

```bash
# Пассажир
go run cmd/generate-jwt/main.go \
  --user-id "passenger-123" \
  --role "PASSENGER" \
  --ttl "24h"

# Водитель
go run cmd/generate-jwt/main.go \
  --user-id "driver-456" \
  --role "DRIVER" \
  --ttl "24h"

# Администратор
go run cmd/generate-jwt/main.go \
  --user-id "admin-1" \
  --role "ADMIN" \
  --ttl "24h"
```

### Использование токена

```bash
# Сохранить токен
export TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

# Использовать в запросах
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:3000/rides
```

### Роли и доступ

| Роль | Доступ |
|------|--------|
| **PASSENGER** | Ride Service (создание поездок, WebSocket) |
| **DRIVER** | Driver Service (управление статусом, локацией, WebSocket) |
| **ADMIN** | Admin Service (управление пользователями, аналитика) |

---

## 🔌 WebSocket

### Ride Service WebSocket (Пассажиры)

**Подключение:**
```
ws://localhost:3000/ws?token=YOUR_JWT_TOKEN
```

**Входящие сообщения (от сервера):**

1. **Ride Status Update**
```json
{
  "type": "ride_status",
  "ride_id": "abc-123",
  "status": "DRIVER_ASSIGNED",
  "driver_id": "driver-456"
}
```

2. **Driver Location Update**
```json
{
  "type": "driver_location",
  "ride_id": "abc-123",
  "latitude": 55.7558,
  "longitude": 37.6173,
  "heading": 180,
  "speed": 45.5
}
```

3. **Match Notification**
```json
{
  "type": "ride_matched",
  "ride_id": "abc-123",
  "driver_id": "driver-456",
  "estimated_arrival_minutes": 5,
  "driver_info": {
    "name": "John Doe",
    "rating": 4.8,
    "vehicle": {
      "make": "Toyota",
      "model": "Camry",
      "color": "Black",
      "plate": "A123BC77"
    }
  }
}
```

### Driver Service WebSocket (Водители)

**Подключение:**
```
ws://localhost:3001/ws?token=YOUR_DRIVER_JWT_TOKEN
```

**Входящие сообщения (от сервера):**

1. **Ride Offer**
```json
{
  "type": "ride_offer",
  "ride_id": "abc-123",
  "pickup_location": {
    "lat": 55.7558,
    "lng": 37.6173,
    "address": "Red Square"
  },
  "destination_location": {
    "lat": 55.7522,
    "lng": 37.6156,
    "address": "Kremlin"
  },
  "estimated_fare": 250.50,
  "distance_km": 2.5
}
```

**Исходящие сообщения (от клиента):**

1. **Accept Ride**
```json
{
  "type": "ride_response",
  "ride_id": "abc-123",
  "accepted": true,
  "current_location": {
    "latitude": 55.7600,
    "longitude": 37.6200
  }
}
```

2. **Location Update**
```json
{
  "type": "location_update",
  "latitude": 55.7558,
  "longitude": 37.6173,
  "accuracy_meters": 10.0,
  "speed_kmh": 45.5,
  "heading_degrees": 180
}
```

### Тестирование WebSocket

```bash
# Автоматический тест
./scripts/test-websocket.sh

# Ручное тестирование с websocat
# Установка: cargo install websocat

# Пассажир
websocat "ws://localhost:3000/ws?token=$PASSENGER_TOKEN"

# Водитель
websocat "ws://localhost:3001/ws?token=$DRIVER_TOKEN"
```

---

## 📨 RabbitMQ

### Топология

```
Exchanges:
├─ ride_topic (topic)
│  ├─ Routing: ride.request.*
│  └─ Queue: driver_matching
│
├─ driver_topic (topic)
│  ├─ Routing: driver.response.*
│  └─ Queue: ride_service_driver_responses
│
└─ location_fanout (fanout)
   ├─ Queue: ride_service_locations
   └─ Queue: driver_service_locations (опционально)
```

### Потоки сообщений

#### 1. Ride Request Flow
```
POST /rides
    ↓
Ride Service → ride_topic (ride.request.{ride_id})
    ↓
driver_matching queue
    ↓
Driver Service Consumer
    ↓
PostGIS: ST_DWithin(5km) + ST_Distance
    ↓
WebSocket → Driver (ride offer)
```

#### 2. Driver Response Flow
```
WebSocket ← Driver (ride_response)
    ↓
Driver Service
    ↓
driver_topic (driver.response.{ride_id})
    ↓
ride_service_driver_responses queue
    ↓
Ride Service Consumer
    ↓
WebSocket → Passenger (match notification)
```

#### 3. Location Update Flow
```
POST /drivers/{id}/location
    ↓
Driver Service
    ↓
location_fanout (broadcast)
    ↓
├─ ride_service_locations queue
│  ↓
│  Ride Service Consumer
│  ↓
│  WebSocket → Passenger
└─ (другие подписчики)
```

### Проверка RabbitMQ

```bash
# Открыть Management UI
# http://localhost:15672 (guest/guest)

# Проверить exchanges
curl -u guest:guest http://localhost:15672/api/exchanges

# Проверить queues
curl -u guest:guest http://localhost:15672/api/queues

# Проверить bindings
curl -u guest:guest http://localhost:15672/api/bindings
```

---

## 🗄️ База данных

### Schema Overview

```sql
-- Users (все типы: PASSENGER, DRIVER, ADMIN)
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    role VARCHAR(50) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Drivers
CREATE TABLE drivers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    vehicle_info JSONB,
    rating DECIMAL(3,2),
    status VARCHAR(50) DEFAULT 'OFFLINE',
    is_online BOOLEAN DEFAULT false
);

-- Rides
CREATE TABLE rides (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ride_number VARCHAR(50) UNIQUE,
    passenger_id UUID REFERENCES users(id),
    driver_id UUID REFERENCES drivers(id),
    status VARCHAR(50) NOT NULL,
    vehicle_type VARCHAR(50),
    pickup_location GEOGRAPHY(POINT, 4326),
    destination_location GEOGRAPHY(POINT, 4326),
    pickup_address TEXT,
    destination_address TEXT,
    estimated_fare DECIMAL(10,2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Driver Coordinates (PostGIS)
CREATE TABLE driver_coordinates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_id UUID REFERENCES drivers(id),
    location GEOGRAPHY(POINT, 4326),
    accuracy_meters DECIMAL(10,2),
    speed_kmh DECIMAL(10,2),
    heading_degrees DECIMAL(5,2),
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Spatial Index
CREATE INDEX idx_driver_coordinates_location 
ON driver_coordinates USING GIST (location);
```

### PostGIS Query Examples

#### 1. Найти водителей в радиусе 5 км

```sql
SELECT 
    d.id,
    d.user_id,
    dc.location,
    ST_Distance(
        dc.location,
        ST_SetSRID(ST_MakePoint(37.6173, 55.7558), 4326)::geography
    ) as distance_meters
FROM drivers d
INNER JOIN LATERAL (
    SELECT location
    FROM driver_coordinates
    WHERE driver_id = d.id
    ORDER BY recorded_at DESC
    LIMIT 1
) dc ON true
WHERE d.is_online = true
  AND d.status = 'AVAILABLE'
  AND d.vehicle_info->>'type' = 'ECONOMY'
  AND ST_DWithin(
    dc.location,
    ST_SetSRID(ST_MakePoint(37.6173, 55.7558), 4326)::geography,
    5000  -- 5 км в метрах
  )
ORDER BY distance_meters ASC
LIMIT 10;
```

**Что происходит:**
- `ST_DWithin` - быстрая проверка попадания в радиус (использует spatial index)
- `ST_Distance` - точное вычисление расстояния для сортировки
- `LATERAL JOIN` - получение последней координаты для каждого водителя
- `GEOGRAPHY` - автоматический учет кривизны Земли

#### 2. История поездок с расстоянием

```sql
SELECT 
    r.ride_number,
    r.pickup_address,
    r.destination_address,
    r.estimated_fare,
    ST_Distance(
        r.pickup_location,
        r.destination_location
    ) / 1000.0 as distance_km,
    r.created_at
FROM rides r
WHERE r.passenger_id = 'passenger-123'
ORDER BY r.created_at DESC
LIMIT 10;
```

#### 3. Активные водители на карте (GeoJSON)

```sql
SELECT jsonb_build_object(
    'type', 'FeatureCollection',
    'features', jsonb_agg(
        jsonb_build_object(
            'type', 'Feature',
            'geometry', ST_AsGeoJSON(dc.location)::jsonb,
            'properties', jsonb_build_object(
                'driver_id', d.id,
                'status', d.status,
                'vehicle_type', d.vehicle_info->>'type',
                'rating', d.rating
            )
        )
    )
) as geojson
FROM drivers d
INNER JOIN LATERAL (
    SELECT location
    FROM driver_coordinates
    WHERE driver_id = d.id
    ORDER BY recorded_at DESC
    LIMIT 1
) dc ON true
WHERE d.is_online = true;
```

### Database Maintenance

```bash
# Подключение к PostgreSQL
docker exec -it ride-postgres psql -U postgres -d ridehail

# Проверка расширений
SELECT * FROM pg_extension WHERE extname IN ('uuid-ossp', 'postgis');

# Статистика таблиц
SELECT 
    schemaname,
    tablename,
    n_live_tup as rows,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_stat_user_tables
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

# Проверка spatial index
SELECT 
    indexname, 
    indexdef 
FROM pg_indexes 
WHERE tablename = 'driver_coordinates';

# Анализ производительности GIST index
EXPLAIN ANALYZE
SELECT *
FROM driver_coordinates
WHERE ST_DWithin(
    location,
    ST_SetSRID(ST_MakePoint(37.6173, 55.7558), 4326)::geography,
    5000
);
```

---

## 🧪 Тестирование

### 1. Infrastructure Tests

```bash
# PostgreSQL
docker exec ride-postgres pg_isready -U postgres

# PostgreSQL + PostGIS
docker exec -it ride-postgres psql -U postgres -d ridehail \
  -c "SELECT PostGIS_Version();"

# RabbitMQ
curl -u guest:guest http://localhost:15672/api/overview

# Exchanges и queues
curl -u guest:guest http://localhost:15672/api/exchanges | jq
curl -u guest:guest http://localhost:15672/api/queues | jq
```

### 2. Health Checks

```bash
# Все сервисы
curl http://localhost:3000/health  # Ride Service
curl http://localhost:3001/health  # Driver Service
curl http://localhost:3004/health  # Admin Service
```

### 3. Unit Tests

```bash
# Запуск всех тестов
go test ./... -v

# Тесты с покрытием
go test ./... -cover -coverprofile=coverage.out

# HTML отчет
go tool cover -html=coverage.out -o coverage.html

# Тесты конкретного модуля
go test ./internal/ride/... -v
go test ./internal/driver/... -v
go test ./internal/admin/... -v
```

### 4. Integration Tests

```bash
# Полный E2E тест
./scripts/test-e2e-ride-flow.sh

# Тест с подробным выводом
bash -x ./scripts/test-e2e-ride-flow.sh

# Тест driver API
./scripts/test-driver-flow.sh

# Тест admin API
./scripts/test-admin-api.sh
```

### 5. Manual Testing Workflow

#### Шаг 1: Создать пользователей

```bash
# Генерация admin токена
ADMIN_TOKEN=$(go run cmd/generate-jwt/main.go \
  --user-id "admin-1" \
  --role "ADMIN" \
  --ttl "24h" | grep "JWT:" | cut -d' ' -f2)

# Создать пассажира
curl -X POST http://localhost:3004/admin/users \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "passenger-test-123",
    "email": "passenger@test.com",
    "role": "PASSENGER",
    "phone": "+79991234567"
  }'

# Создать водителя
curl -X POST http://localhost:3004/admin/users \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "driver-test-456",
    "email": "driver@test.com",
    "role": "DRIVER",
    "phone": "+79997654321"
  }'
```

#### Шаг 2: Водитель выходит онлайн

```bash
# Генерация driver токена
DRIVER_TOKEN=$(go run cmd/generate-jwt/main.go \
  --user-id "driver-test-456" \
  --role "DRIVER" \
  --ttl "24h" | grep "JWT:" | cut -d' ' -f2)

# Выход онлайн
curl -X POST http://localhost:3001/drivers/driver-test-456/online \
  -H "Authorization: Bearer $DRIVER_TOKEN"

# Обновление локации (Москва, центр)
curl -X POST http://localhost:3001/drivers/driver-test-456/location \
  -H "Authorization: Bearer $DRIVER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "latitude": 55.7558,
    "longitude": 37.6173,
    "accuracy_meters": 10.0,
    "speed_kmh": 0.0,
    "heading_degrees": 0
  }'
```

#### Шаг 3: Пассажир создает поездку

```bash
# Генерация passenger токена
PASSENGER_TOKEN=$(go run cmd/generate-jwt/main.go \
  --user-id "passenger-test-123" \
  --role "PASSENGER" \
  --ttl "24h" | grep "JWT:" | cut -d' ' -f2)

# Создание поездки
RIDE_RESPONSE=$(curl -X POST http://localhost:3000/rides \
  -H "Authorization: Bearer $PASSENGER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "vehicle_type": "ECONOMY",
    "pickup_lat": 55.7558,
    "pickup_lng": 37.6173,
    "pickup_address": "Red Square, Moscow",
    "destination_lat": 55.7522,
    "destination_lng": 37.6156,
    "destination_address": "Kremlin, Moscow",
    "priority": 5
  }')

echo $RIDE_RESPONSE | jq
RIDE_ID=$(echo $RIDE_RESPONSE | jq -r '.ride_id')
```

#### Шаг 4: Проверка RabbitMQ

```bash
# Проверить сообщения в очереди driver_matching
curl -u guest:guest \
  "http://localhost:15672/api/queues/%2F/driver_matching" | jq

# Получить сообщение (non-destructive peek)
curl -u guest:guest \
  -X POST "http://localhost:15672/api/queues/%2F/driver_matching/get" \
  -H "Content-Type: application/json" \
  -d '{"count":1,"ackmode":"ack_requeue_true","encoding":"auto"}' | jq
```

#### Шаг 5: WebSocket тестирование

```bash
# Установить websocat (если нет)
# cargo install websocat

# Подключиться как водитель
websocat "ws://localhost:3001/ws?token=$DRIVER_TOKEN"

# В другом терминале - подключиться как пассажир
websocat "ws://localhost:3000/ws?token=$PASSENGER_TOKEN"

# Создать поездку и наблюдать за событиями в обоих WebSocket
```

### 6. Performance Testing

```bash
# Установить Apache Bench
sudo apt-get install apache2-utils

# Тест health endpoint
ab -n 1000 -c 10 http://localhost:3000/health

# Тест создания поездок (с JWT)
ab -n 100 -c 5 \
  -H "Authorization: Bearer $PASSENGER_TOKEN" \
  -p ride-payload.json \
  -T application/json \
  http://localhost:3000/rides
```

### 7. Load Testing с k6

```javascript
// load-test.js
import http from 'k6/http';
import { check, sleep } from 'k6';

export let options = {
  stages: [
    { duration: '30s', target: 10 },
    { duration: '1m', target: 50 },
    { duration: '30s', target: 0 },
  ],
};

export default function () {
  let token = 'YOUR_JWT_TOKEN';
  
  let payload = JSON.stringify({
    vehicle_type: 'ECONOMY',
    pickup_lat: 55.7558,
    pickup_lng: 37.6173,
    pickup_address: 'Red Square',
    destination_lat: 55.7522,
    destination_lng: 37.6156,
    destination_address: 'Kremlin',
  });

  let params = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`,
    },
  };

  let res = http.post('http://localhost:3000/rides', payload, params);
  
  check(res, {
    'status is 201': (r) => r.status === 201,
    'ride_id present': (r) => JSON.parse(r.body).ride_id !== undefined,
  });

  sleep(1);
}
```

```bash
# Запуск k6
k6 run load-test.js
```

---

## 📊 Monitoring

### Metrics Endpoints

```bash
# Health checks
curl http://localhost:3000/health | jq
curl http://localhost:3001/health | jq
curl http://localhost:3004/health | jq

# RabbitMQ metrics
curl -u guest:guest http://localhost:15672/api/overview | jq '.queue_totals'

# Database stats
docker exec -it ride-postgres psql -U postgres -d ridehail \
  -c "SELECT COUNT(*) FROM rides WHERE status = 'PENDING';"

docker exec -it ride-postgres psql -U postgres -d ridehail \
  -c "SELECT COUNT(*) FROM drivers WHERE is_online = true;"
```

### Logs

```bash
# Docker compose logs
docker-compose -f deployments/docker-compose.yml logs -f

# Specific service logs
docker-compose -f deployments/docker-compose.yml logs -f ride-postgres
docker-compose -f deployments/docker-compose.yml logs -f ride-rabbitmq

# Application logs (если запущено через go run)
# Логи пишутся в stdout
```

---

## 🚀 Deployment

### Production Build

```bash
# Сборка бинарника
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -ldflags="-w -s" \
  -o bin/ridehail-linux-amd64 \
  ./main.go

# Размер бинарника
ls -lh bin/ridehail-linux-amd64

# Upx compression (опционально)
upx --best --lzma bin/ridehail-linux-amd64
```

### Docker Build

```bash
# Build образа
docker build -f deployments/Dockerfile -t ridehail:latest .

# Проверка размера
docker images ridehail:latest

# Запуск контейнера
docker run -d \
  --name ridehail-app \
  -p 3000:3000 \
  -p 3001:3001 \
  -p 3002:3002 \
  -e DB_HOST=postgres \
  -e RABBITMQ_HOST=rabbitmq \
  ridehail:latest
```

### Environment Variables

```bash
# Database
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=ridehail

# RabbitMQ
export RABBITMQ_HOST=localhost
export RABBITMQ_PORT=5672
export RABBITMQ_USER=guest
export RABBITMQ_PASSWORD=guest

# JWT
export JWT_SECRET=your-super-secret-key-change-in-production
export JWT_ISSUER=ride-hail-system

# Service Ports
export RIDE_SERVICE_PORT=3000
export DRIVER_SERVICE_PORT=3001
export ADMIN_SERVICE_PORT=3002

# Logging
export LOG_LEVEL=info  # debug, info, warn, error
```

---

## 📝 Documentation

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

### Driver Service Testing ⭐

Полная документация: [TESTING_GUIDE.md](TESTING_GUIDE.md)

```bash
# 1. Создать тестового водителя
./scripts/setup-test-driver.sh

# 2. Запустить полное тестирование (8 тестов)
export DRIVER_ID="your-driver-id"
./scripts/test-driver-api.sh
```

Доступные скрипты:
- `setup-test-driver.sh` - создание тестового водителя
- `generate-driver-token.sh` - генерация JWT токена
- `test-driver-api.sh` - автоматическое тестирование API (8 тестов)
- `test-driver-workflow.sh` - полный workflow водителя
- `driver-api-helpers.sh` - интерактивные функции

### Интеграционные тесты

```bash
# Запустить сервисы
make docker-up

# Выполнить тесты
./scripts/integration-test.sh
```

## 📊 Мониторинг

### Metrics Endpoints

```bash
# Health checks
curl http://localhost:3000/health | jq
curl http://localhost:3001/health | jq
curl http://localhost:3004/health | jq

# RabbitMQ metrics
curl -u guest:guest http://localhost:15672/api/overview | jq '.queue_totals'

# Database stats
docker exec -it ride-postgres psql -U postgres -d ridehail \
  -c "SELECT COUNT(*) FROM rides WHERE status = 'PENDING';"

docker exec -it ride-postgres psql -U postgres -d ridehail \
  -c "SELECT COUNT(*) FROM drivers WHERE is_online = true;"
```

### Logs

```bash
# Docker compose logs
docker-compose -f deployments/docker-compose.yml logs -f

# Specific service logs
docker-compose -f deployments/docker-compose.yml logs -f ride-postgres
docker-compose -f deployments/docker-compose.yml logs -f ride-rabbitmq

# Application logs (если запущено через go run)
# Логи пишутся в stdout с JSON structured logging
```

---

## 🚀 Deployment

### Production Build

```bash
# Сборка бинарника
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -ldflags="-w -s" \
  -o bin/ridehail-linux-amd64 \
  ./main.go

# Размер бинарника
ls -lh bin/ridehail-linux-amd64

# Upx compression (опционально)
upx --best --lzma bin/ridehail-linux-amd64
```

### Docker Build

```bash
# Build образа
docker build -f deployments/Dockerfile -t ridehail:latest .

# Проверка размера
docker images ridehail:latest

# Запуск контейнера
docker run -d \
  --name ridehail-app \
  -p 3000:3000 \
  -p 3001:3001 \
  -p 3002:3002 \
  -e DB_HOST=postgres \
  -e RABBITMQ_HOST=rabbitmq \
  ridehail:latest
```

### Environment Variables

```bash
# Database
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=ridehail

# RabbitMQ
export RABBITMQ_HOST=localhost
export RABBITMQ_PORT=5672
export RABBITMQ_USER=guest
export RABBITMQ_PASSWORD=guest

# JWT
export JWT_SECRET=your-super-secret-key-change-in-production
export JWT_ISSUER=ride-hail-system

# Service Ports
export RIDE_SERVICE_PORT=3000
export DRIVER_SERVICE_PORT=3001
export ADMIN_SERVICE_PORT=3002

# Logging
export LOG_LEVEL=info  # debug, info, warn, error
```

---

## 📝 Documentation

### Architecture Documents

- **[IMPLEMENTATION_GUIDE.md](docs/IMPLEMENTATION_GUIDE.md)** - Полное руководство по архитектуре
  - Детальное описание каждого компонента
  - Диаграммы потоков данных
  - PostGIS query examples
  - RabbitMQ топология с примерами
  - WebSocket протоколы

- **[PROJECT_COMPLETION.md](PROJECT_COMPLETION.md)** - Отчет о завершении проекта
  - Чеклисты всех компонентов
  - Технические метрики
  - Результаты тестирования
  - Список достижений

- **[FINAL_SUMMARY.md](FINAL_SUMMARY.md)** - Краткий обзор проекта
  - Quick start guide
  - Ключевые метрики
  - Основные достижения
  - Навигация по документации

### API Documentation

- **[docs/admin_api.md](docs/admin_api.md)** - Admin Service API reference
  - Все endpoints с примерами
  - Модели данных
  - Примеры curl запросов

- **[docs/INTEGRATION.md](docs/INTEGRATION.md)** - Руководство по интеграции
  - WebSocket протоколы
  - RabbitMQ message formats
  - JWT authentication flow

### Code Examples

```bash
# Примеры использования в директории scripts/
./scripts/test-e2e-ride-flow.sh      # E2E тест полного flow
./scripts/test-admin-api.sh           # Тестирование Admin API
./scripts/generate-admin-token.sh     # Генерация admin токена
```

---

## 🏗️ Architecture Patterns

### Clean Architecture (Hexagonal)

```
internal/
├─ ride/
│  ├─ domain/           # Бизнес-логика, entities
│  ├─ application/      # Use cases, ports
│  │  ├─ ports/
│  │  │  ├─ in/        # Входящие порты (интерфейсы use cases)
│  │  │  └─ out/       # Исходящие порты (репозитории)
│  │  └─ usecase/      # Реализация use cases
│  ├─ adapter/         # Адаптеры
│  │  ├─ in/           # Входящие адаптеры
│  │  │  ├─ transport/ # HTTP handlers
│  │  │  ├─ in_ws/     # WebSocket handlers
│  │  │  └─ in_amqp/   # RabbitMQ consumers
│  │  └─ out/          # Исходящие адаптеры (DB, MQ producers)
│  └─ bootstrap/       # Dependency injection
```

**Принципы:**
- ✅ **Dependency Inversion** - domain не зависит от внешних библиотек
- ✅ **Ports & Adapters** - четкие границы между слоями
- ✅ **Use Cases** - бизнес-логика изолирована
- ✅ **Testability** - легко мокать зависимости

### Event-Driven Architecture

**Асинхронная коммуникация через RabbitMQ:**

1. **Topic Exchange** - маршрутизация по routing key
   - `ride_topic`: `ride.request.*`
   - `driver_topic`: `driver.response.*`

2. **Fanout Exchange** - broadcast всем подписчикам
   - `location_fanout`: обновления локации водителя

3. **Dead Letter Queues** - обработка ошибок
   - Retry механизм с экспоненциальной задержкой
   - Мониторинг failed messages

**Преимущества:**
- 🔄 **Loose Coupling** - сервисы независимы
- 📈 **Scalability** - горизонтальное масштабирование
- 🛡️ **Resilience** - отказоустойчивость через очереди
- 📊 **Auditability** - все события логируются

### Geospatial Architecture (PostGIS)

**Оптимизация запросов:**

```sql
-- 1. Spatial Index (GIST)
CREATE INDEX idx_driver_coordinates_location 
ON driver_coordinates USING GIST (location);

-- 2. Two-step query optimization
-- Шаг 1: ST_DWithin (быстрая фильтрация по индексу)
-- Шаг 2: ST_Distance (точное расстояние для топ-N)

-- 3. LATERAL JOIN для latest location
SELECT d.*, dc.location
FROM drivers d
INNER JOIN LATERAL (
    SELECT location
    FROM driver_coordinates
    WHERE driver_id = d.id
    ORDER BY recorded_at DESC
    LIMIT 1
) dc ON true;
```

**Performance:**
- ⚡ GIST index: O(log n) search vs O(n) table scan
- 🎯 ST_DWithin использует bounding box для быстрой фильтрации
- 📍 GEOGRAPHY type автоматически учитывает кривизну Земли

---

## 🔧 Troubleshooting

### Common Issues

#### 1. RabbitMQ Connection Failed

**Симптомы:**
```
Failed to connect to RabbitMQ: dial tcp: connection refused
```

**Решение:**
```bash
# Проверить статус
docker-compose -f deployments/docker-compose.yml ps

# Перезапустить RabbitMQ
docker-compose -f deployments/docker-compose.yml restart ride-rabbitmq

# Проверить логи
docker-compose -f deployments/docker-compose.yml logs ride-rabbitmq

# Проверить порты
netstat -tlnp | grep 5672
```

#### 2. PostgreSQL Connection Failed

**Симптомы:**
```
Error connecting to database: connection refused
```

**Решение:**
```bash
# Проверить статус
docker exec ride-postgres pg_isready -U postgres

# Проверить соединение
docker exec -it ride-postgres psql -U postgres -d ridehail -c "\conninfo"

# Пересоздать БД (ОСТОРОЖНО!)
docker-compose -f deployments/docker-compose.yml down -v
docker-compose -f deployments/docker-compose.yml up -d
```

#### 3. PostGIS Extension Missing

**Симптомы:**
```
ERROR: type "geography" does not exist
```

**Решение:**
```bash
docker exec -it ride-postgres psql -U postgres -d ridehail \
  -c "CREATE EXTENSION IF NOT EXISTS postgis;"

# Проверить
docker exec -it ride-postgres psql -U postgres -d ridehail \
  -c "SELECT PostGIS_Version();"
```

#### 4. JWT Token Invalid

**Симптомы:**
```json
{"error": "unauthorized", "message": "invalid token"}
```

**Решение:**
```bash
# Проверить секрет в config/jwt.yaml
cat config/jwt.yaml

# Сгенерировать новый токен
go run cmd/generate-jwt/main.go \
  --user-id "test-123" \
  --role "PASSENGER" \
  --ttl "24h"

# Проверить токен
go run cmd/verify-jwt/main.go --token "YOUR_TOKEN"
```

#### 5. WebSocket Connection Failed

**Симптомы:**
```
WebSocket handshake failed: 401 Unauthorized
```

**Решение:**
```bash
# Проверить токен в URL
ws://localhost:3000/ws?token=YOUR_JWT_TOKEN

# Проверить роль (PASSENGER для /rides, DRIVER для /drivers)

# Тест подключения с curl
curl -i -N \
  -H "Connection: Upgrade" \
  -H "Upgrade: websocket" \
  -H "Sec-WebSocket-Version: 13" \
  -H "Sec-WebSocket-Key: SGVsbG8sIHdvcmxkIQ==" \
  "http://localhost:3000/ws?token=$TOKEN"
```

#### 6. Driver Matching Not Working

**Симптомы:**
- Ride создается, но водитель не получает уведомление

**Диагностика:**
```bash
# 1. Проверить, что водитель онлайн
docker exec -it ride-postgres psql -U postgres -d ridehail \
  -c "SELECT id, is_online, status FROM drivers;"

# 2. Проверить локацию водителя
docker exec -it ride-postgres psql -U postgres -d ridehail \
  -c "SELECT driver_id, ST_AsText(location), recorded_at 
      FROM driver_coordinates 
      ORDER BY recorded_at DESC 
      LIMIT 5;"

# 3. Проверить очередь driver_matching
curl -u guest:guest \
  http://localhost:15672/api/queues/%2F/driver_matching | jq

# 4. Проверить логи Driver Service
docker-compose -f deployments/docker-compose.yml logs driver-service

# 5. Тест PostGIS query вручную
docker exec -it ride-postgres psql -U postgres -d ridehail \
  -c "SELECT d.id, 
      ST_Distance(
        dc.location,
        ST_SetSRID(ST_MakePoint(37.6173, 55.7558), 4326)::geography
      ) as distance_meters
      FROM drivers d
      INNER JOIN LATERAL (
        SELECT location FROM driver_coordinates 
        WHERE driver_id = d.id 
        ORDER BY recorded_at DESC LIMIT 1
      ) dc ON true
      WHERE d.is_online = true
        AND ST_DWithin(
          dc.location,
          ST_SetSRID(ST_MakePoint(37.6173, 55.7558), 4326)::geography,
          5000
        )
      ORDER BY distance_meters ASC;"
```

#### 7. High Memory Usage

**Решение:**
```bash
# Проверить использование памяти
docker stats

# Ограничить память для контейнеров
# В docker-compose.yml добавить:
services:
  ride-postgres:
    deploy:
      resources:
        limits:
          memory: 512M
        reservations:
          memory: 256M

# Очистить неиспользуемые образы
docker system prune -a
```

#### 8. Docker Buildx Error

Если получаете ошибку `fork/exec .../docker-buildx: no such file or directory`:

```bash
# Используйте обычный docker build вместо buildx
docker build -f deployments/Dockerfile -t ride-hail .
```

#### 9. Порты заняты

```bash
# Проверить занятые порты
sudo lsof -i :3000
sudo lsof -i :5432

# Убить процесс на порту
sudo kill -9 $(sudo lsof -t -i:3000)

# Изменить порты в docker-compose.yml
```

#### 10. Проблемы с миграциями

```bash
# Пересоздать БД (удалит все данные!)
docker-compose -f deployments/docker-compose.yml down -v
docker-compose -f deployments/docker-compose.yml up -d

# Или вручную
docker exec -it ride-postgres psql -U postgres -c "DROP DATABASE IF EXISTS ridehail;"
docker exec -it ride-postgres psql -U postgres -c "CREATE DATABASE ridehail;"
```

### Debug Mode

```bash
# Запуск с debug логами
export LOG_LEVEL=debug
go run main.go

# Трассировка SQL запросов (PostgreSQL)
export DB_LOG_LEVEL=debug

# Трассировка RabbitMQ сообщений
export RABBITMQ_LOG_LEVEL=debug
```

---

## 🤝 Contributing

### Development Workflow

1. **Fork the repository**
2. **Create feature branch**
   ```bash
   git checkout -b feature/amazing-feature
   ```
3. **Make changes**
   - Следовать Clean Architecture
   - Добавить unit tests
   - Обновить документацию
4. **Run tests**
   ```bash
   go test ./... -v
   ./scripts/test-e2e-ride-flow.sh
   ```
5. **Commit changes**
   ```bash
   git commit -m "feat: add amazing feature"
   ```
6. **Push to branch**
   ```bash
   git push origin feature/amazing-feature
   ```
7. **Open Pull Request**

### Code Style

```bash
# Formatting
go fmt ./...

# Linting
golangci-lint run

# Imports organization
goimports -w .
```

### Commit Convention

```
feat: новая функция
fix: исправление бага
docs: изменения в документации
refactor: рефакторинг кода
test: добавление тестов
chore: обновление зависимостей, конфигурации
```

---

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details

---

## 👥 Authors

- **Adam** - Initial work and architecture

---

## 🙏 Acknowledgments

- **Go Community** - amazing ecosystem
- **PostGIS Team** - powerful geospatial extension
- **RabbitMQ Team** - reliable messaging
- **Clean Architecture** - Robert C. Martin
- **Hexagonal Architecture** - Alistair Cockburn

---

## 📞 Support

### Issues

Если вы обнаружили баг или хотите предложить улучшение:
1. Проверьте [Troubleshooting](#-troubleshooting)
2. Откройте issue на GitHub
3. Опишите проблему с примерами

### Questions

Для вопросов по проекту:
- Создайте discussion на GitHub
- Укажите версию Go, PostgreSQL, RabbitMQ
- Приложите логи и конфигурацию

---

## 🎯 Roadmap

### Version 2.0 (Planned)

- [ ] **Payment Integration**
  - Stripe/PayPal integration
  - Fare calculation algorithm
  - Transaction history

- [ ] **Advanced Matching**
  - Machine learning for ETA prediction
  - Traffic data integration
  - Dynamic pricing

- [ ] **Mobile Apps**
  - React Native passenger app
  - React Native driver app
  - Push notifications

- [ ] **Analytics Dashboard**
  - Real-time metrics (Grafana)
  - Business intelligence
  - Driver performance tracking

- [ ] **Advanced Features**
  - Ride scheduling
  - Ride sharing (pooling)
  - Multi-stop rides
  - Favorite locations

- [ ] **Operations**
  - Kubernetes deployment
  - CI/CD pipeline (GitHub Actions)
  - Automated testing
  - Load testing framework

---

**⭐ If you find this project useful, please star it on GitHub!**