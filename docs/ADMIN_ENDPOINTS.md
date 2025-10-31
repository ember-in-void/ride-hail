# Admin Service Endpoints

## Overview

Admin Service предоставляет API для мониторинга и управления системой ride-hailing.

**Base URL:** `http://localhost:3002`

**Authentication:** Все endpoints требуют JWT токен с ролью `ADMIN`

---

## Endpoints

### 1. GET /admin/overview

Получить системный обзор с метриками и статистикой.

#### Request

```bash
curl -X GET http://localhost:3002/admin/overview \
  -H "Authorization: Bearer YOUR_ADMIN_JWT_TOKEN"
```

#### Response (200 OK)

```json
{
  "timestamp": "2025-10-31T12:00:00Z",
  "metrics": {
    "active_rides": 45,
    "available_drivers": 123,
    "busy_drivers": 45,
    "total_rides_today": 892,
    "total_revenue_today": 1234567.5,
    "average_wait_time_minutes": 4.2,
    "average_ride_duration_minutes": 18.5,
    "cancellation_rate": 0.05
  },
  "driver_distribution": {
    "ECONOMY": 89,
    "PREMIUM": 28,
    "XL": 6
  },
  "hotspots": [
    {
      "location": "Almaty Airport",
      "active_rides": 12,
      "waiting_drivers": 34
    },
    {
      "location": "Mega Alma-Ata",
      "active_rides": 8,
      "waiting_drivers": 15
    }
  ]
}
```

#### Metrics Explanation

| Metric | Description |
|--------|-------------|
| `active_rides` | Количество поездок в статусах MATCHED, EN_ROUTE, ARRIVED, IN_PROGRESS |
| `available_drivers` | Количество водителей со статусом AVAILABLE |
| `busy_drivers` | Количество водителей со статусом BUSY |
| `total_rides_today` | Общее количество поездок за сегодня |
| `total_revenue_today` | Общая выручка за завершенные поездки сегодня |
| `average_wait_time_minutes` | Среднее время ожидания (от requested_at до matched_at) |
| `average_ride_duration_minutes` | Средняя продолжительность поездки (от started_at до completed_at) |
| `cancellation_rate` | Доля отмененных поездок от общего количества за сегодня |

---

### 2. GET /admin/rides/active

Получить список активных поездок с пагинацией.

#### Request Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `page` | integer | No | 1 | Номер страницы (начиная с 1) |
| `page_size` | integer | No | 20 | Количество записей на странице (макс 100) |

#### Request

```bash
curl -X GET "http://localhost:3002/admin/rides/active?page=1&page_size=20" \
  -H "Authorization: Bearer YOUR_ADMIN_JWT_TOKEN"
```

#### Response (200 OK)

```json
{
  "rides": [
    {
      "ride_id": "550e8400-e29b-41d4-a716-446655440000",
      "ride_number": "RIDE_20241216_001",
      "status": "IN_PROGRESS",
      "passenger_id": "880e8400-e29b-41d4-a716-446655440003",
      "driver_id": "660e8400-e29b-41d4-a716-446655440001",
      "pickup_address": "Almaty Central Park",
      "destination_address": "Kok-Tobe Hill",
      "started_at": "2024-12-16T10:30:00Z",
      "estimated_completion": "2024-12-16T10:45:00Z",
      "current_driver_location": {
        "latitude": 43.23,
        "longitude": 76.87
      },
      "distance_completed_km": 2.3,
      "distance_remaining_km": 2.9
    }
  ],
  "total_count": 45,
  "page": 1,
  "page_size": 20
}
```

#### Active Ride Statuses

Возвращаются поездки со следующими статусами:
- `MATCHED` - водитель назначен
- `EN_ROUTE` - водитель едет к точке pickup
- `ARRIVED` - водитель прибыл на точку pickup
- `IN_PROGRESS` - поездка в процессе

---

### 3. POST /admin/users

Создать нового пользователя (пассажира, водителя или администратора).

#### Request

```bash
curl -X POST http://localhost:3002/admin/users \
  -H "Authorization: Bearer YOUR_ADMIN_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "driver@example.com",
    "password": "securepassword123",
    "role": "DRIVER",
    "status": "ACTIVE",
    "attrs": {
      "license_number": "DL123456",
      "vehicle_type": "ECONOMY",
      "vehicle_make": "Toyota",
      "vehicle_model": "Camry",
      "vehicle_color": "White",
      "vehicle_plate": "KZ 123 ABC",
      "vehicle_year": 2020
    }
  }'
```

#### Response (201 Created)

```json
{
  "id": "660e8400-e29b-41d4-a716-446655440001",
  "email": "driver@example.com",
  "role": "DRIVER",
  "status": "ACTIVE",
  "created_at": "2025-10-31T12:00:00Z"
}
```

#### Roles

| Role | Description |
|------|-------------|
| `PASSENGER` | Пассажир (может создавать поездки) |
| `DRIVER` | Водитель (может принимать поездки) |
| `ADMIN` | Администратор (полный доступ к системе) |

**Note:** При создании пользователя с ролью `DRIVER` автоматически создается запись в таблице `drivers`.

---

### 4. GET /admin/users

Получить список пользователей с фильтрацией.

#### Request Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `role` | string | No | - | Фильтр по роли (PASSENGER, DRIVER, ADMIN) |
| `status` | string | No | - | Фильтр по статусу (ACTIVE, INACTIVE, BANNED) |
| `limit` | integer | No | 50 | Максимальное количество записей |
| `offset` | integer | No | 0 | Смещение для пагинации |

#### Request

```bash
curl -X GET "http://localhost:3002/admin/users?role=DRIVER&status=ACTIVE&limit=10&offset=0" \
  -H "Authorization: Bearer YOUR_ADMIN_JWT_TOKEN"
```

#### Response (200 OK)

```json
{
  "users": [
    {
      "id": "660e8400-e29b-41d4-a716-446655440001",
      "email": "driver1@example.com",
      "role": "DRIVER",
      "status": "ACTIVE",
      "created_at": "2025-10-30T10:00:00Z",
      "updated_at": "2025-10-30T10:00:00Z"
    }
  ],
  "total_count": 156,
  "limit": 10,
  "offset": 0
}
```

---

### 5. GET /health

Health check endpoint (без аутентификации).

#### Request

```bash
curl -X GET http://localhost:3002/health
```

#### Response (200 OK)

```json
{
  "status": "ok",
  "service": "admin"
}
```

---

## Authentication

### Генерация Admin токена

```bash
go run cmd/generate-jwt/main.go \
  --user-id "admin-1" \
  --role "ADMIN" \
  --ttl "24h"
```

### Использование токена

```bash
# Сохранить токен в переменную
export ADMIN_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

# Использовать в запросах
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  http://localhost:3002/admin/overview
```

---

## Error Responses

### 400 Bad Request

```json
{
  "error": "invalid request format"
}
```

### 401 Unauthorized

```json
{
  "error": "unauthorized"
}
```

### 403 Forbidden

```json
{
  "error": "insufficient permissions"
}
```

### 500 Internal Server Error

```json
{
  "error": "internal server error"
}
```

---

## SQL Implementation Details

### GetSystemMetrics Query

Использует CTE (Common Table Expressions) для эффективного вычисления метрик:

```sql
WITH ride_stats AS (
  SELECT
    COUNT(*) FILTER (WHERE status IN (...)) as active_rides,
    AVG(EXTRACT(EPOCH FROM (matched_at - requested_at))/60) as avg_wait_time,
    ...
),
driver_stats AS (
  SELECT
    COUNT(*) FILTER (WHERE status = 'AVAILABLE') as available_drivers,
    ...
)
SELECT ... FROM ride_stats CROSS JOIN driver_stats
```

**Оптимизации:**
- FILTER для условной агрегации (более эффективно чем WHERE)
- CTE для разделения логики и кеширования подзапросов
- Использование индексов на полях status и created_at

### GetActiveRides Query

Использует LATERAL JOIN для получения последней локации водителя:

```sql
SELECT
  r.id,
  r.ride_number,
  ...
  driver_c.latitude,
  driver_c.longitude
FROM rides r
LEFT JOIN LATERAL (
  SELECT latitude, longitude
  FROM coordinates
  WHERE entity_id = r.driver_id
    AND entity_type = 'driver'
    AND is_current = true
  LIMIT 1
) driver_c ON true
WHERE r.status IN ('MATCHED', 'EN_ROUTE', 'ARRIVED', 'IN_PROGRESS')
```

**Преимущества:**
- LATERAL JOIN позволяет использовать значения из внешнего запроса
- Индекс на (entity_id, entity_type, is_current) ускоряет поиск
- LIMIT 1 гарантирует одну запись на водителя

---

## Usage Examples

### Мониторинг системы

```bash
#!/bin/bash

# Получить admin токен
ADMIN_TOKEN=$(go run cmd/generate-jwt/main.go --user-id "admin-1" --role "ADMIN" --ttl "24h" | grep "JWT:" | cut -d' ' -f2)

# Проверить метрики
echo "=== System Overview ==="
curl -s -H "Authorization: Bearer $ADMIN_TOKEN" \
  http://localhost:3002/admin/overview | jq '.metrics'

# Проверить активные поездки
echo ""
echo "=== Active Rides ==="
curl -s -H "Authorization: Bearer $ADMIN_TOKEN" \
  "http://localhost:3002/admin/rides/active?page=1&page_size=5" | jq '.total_count'

# Проверить водителей
echo ""
echo "=== Driver Stats ==="
curl -s -H "Authorization: Bearer $ADMIN_TOKEN" \
  "http://localhost:3002/admin/users?role=DRIVER&status=ACTIVE" | jq '.total_count'
```

---

## Notes

- Все timestamps в формате ISO 8601 (UTC)
- Пагинация поддерживается через параметры page/page_size
- Hotspots рассчитываются на основе поездок за последний час
- Distance fields могут быть null для поездок в начальных статусах
- Current driver location обновляется в реальном времени через WebSocket

