# Admin API Documentation

## Overview

Admin Service предоставляет REST API для управления пользователями системы ride-hail.

**Базовый URL:** `http://localhost:3004`

---

## Authentication

Все endpoints (кроме `/health`) требуют JWT токен с ролью `ADMIN`.

**Header:**
```
Authorization: Bearer <JWT_TOKEN>
```

**Получение токена для дефолтного админа:**
```bash
./scripts/generate-admin-token.sh
```

Или вручную:
```bash
go run cmd/generate-jwt/main.go \
  -user=admin-001 \
  -email=admin@ridehail.com \
  -role=ADMIN
```

---

## Default Admin Credentials

При первом запуске системы автоматически создаётся дефолтный админ:

- **Email:** `admin@ridehail.com`
- **Password:** `admin123`
- **Role:** `ADMIN`
- **Status:** `ACTIVE`

**⚠️ ВАЖНО:** Смените пароль в production!

---

## Endpoints

### 1. Health Check

**GET** `/health`

Проверка состояния сервиса (liveness probe).

**Response:**
```json
{
  "status": "ok",
  "service": "admin"
}
```

---

### 2. Create User

**POST** `/admin/users`

Создание нового пользователя (PASSENGER или DRIVER).

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "SecurePassword123",
  "role": "PASSENGER",
  "status": "ACTIVE",
  "attrs": {
    "phone": "+1234567890",
    "name": "John Doe"
  }
}
```

**Fields:**
- `email` (required) — уникальный email
- `password` (required) — минимум 8 символов
- `role` (required) — `PASSENGER` или `DRIVER`
- `status` (optional) — `ACTIVE` (default), `INACTIVE`, `BANNED`
- `attrs` (optional) — произвольные атрибуты (JSONB)

**Response (201 Created):**
```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440001",
  "email": "user@example.com",
  "role": "PASSENGER",
  "status": "ACTIVE",
  "created_at": "2025-01-29T20:02:19Z"
}
```

**Errors:**
- `400 Bad Request` — невалидные данные (короткий пароль, некорректный email/role)
- `401 Unauthorized` — отсутствует или невалидный токен
- `403 Forbidden` — токен не имеет роли ADMIN
- `409 Conflict` — пользователь с таким email уже существует

---

### 3. List Users

**GET** `/admin/users`

Получение списка пользователей с фильтрацией и пагинацией.

**Query Parameters:**
- `role` (optional) — фильтр по роли (`PASSENGER`, `DRIVER`, `ADMIN`)
- `status` (optional) — фильтр по статусу (`ACTIVE`, `INACTIVE`, `BANNED`)
- `limit` (optional) — количество записей (default: 50, max: 100)
- `offset` (optional) — смещение для пагинации (default: 0)

**Example:**
```bash
GET /admin/users?role=PASSENGER&status=ACTIVE&limit=10&offset=0
```

**Response (200 OK):**
```json
{
  "users": [
    {
      "user_id": "550e8400-e29b-41d4-a716-446655440001",
      "email": "passenger1@example.com",
      "role": "PASSENGER",
      "status": "ACTIVE",
      "attrs": {
        "phone": "+1234567890"
      },
      "created_at": "2025-01-29T19:00:00Z",
      "updated_at": "2025-01-29T19:00:00Z"
    }
  ],
  "total_count": 1,
  "limit": 10,
  "offset": 0
}
```

**Errors:**
- `401 Unauthorized` — отсутствует или невалидный токен
- `403 Forbidden` — токен не имеет роли ADMIN

---

## Examples

### Создание пассажира

```bash
curl -X POST http://localhost:3004/admin/users \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john.doe@example.com",
    "password": "MySecurePass123",
    "role": "PASSENGER",
    "attrs": {
      "name": "John Doe",
      "phone": "+1234567890",
      "preferred_vehicle": "ECONOMY"
    }
  }'
```

### Создание водителя

```bash
curl -X POST http://localhost:3004/admin/users \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "driver@example.com",
    "password": "DriverPass456",
    "role": "DRIVER",
    "attrs": {
      "license_number": "DRV12345",
      "vehicle_model": "Toyota Camry",
      "vehicle_year": 2022
    }
  }'
```

### Получение всех водителей

```bash
curl -X GET "http://localhost:3004/admin/users?role=DRIVER&limit=20" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

---

## Testing

Запустите полный тест Admin API:

```bash
chmod +x scripts/test-admin-api.sh
./scripts/test-admin-api.sh
```

Тест выполнит:
1. Health check
2. Генерацию admin токена
3. Создание PASSENGER
4. Создание DRIVER
5. Получение списка пользователей с фильтрами
6. Проверку отказа доступа без токена
7. Проверку отказа доступа с PASSENGER токеном
8. Валидацию (короткий пароль, дубликат email)

---

## Security Notes

1. **JWT токен должен иметь роль ADMIN** — иначе будет `403 Forbidden`
2. **Пароли хешируются через bcrypt** (cost=10) перед сохранением
3. **Email уникален** — дубликаты вернут `409 Conflict`
4. **Валидация email** через regex: `^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`
5. **Минимальная длина пароля**: 8 символов

---

## Production Checklist

- [ ] Смените пароль дефолтного админа
- [ ] Используйте сильные пароли (16+ символов, спецсимволы)
- [ ] Настройте JWT_SECRET через environment variable
- [ ] Включите HTTPS (reverse proxy: nginx/traefik)
- [ ] Ограничьте доступ к Admin API по IP/network policy
- [ ] Настройте rate limiting (например, через nginx)
- [ ] Включите audit logging для всех операций с пользователями

---

## Roadmap

Planned features:
- [ ] `PATCH /admin/users/:id` — обновление пользователя
- [ ] `DELETE /admin/users/:id` — удаление/бан пользователя
- [ ] `GET /admin/users/:id` — получение одного пользователя
- [ ] `POST /admin/users/:id/change-password` — смена пароля
- [ ] WebSocket для real-time мониторинга