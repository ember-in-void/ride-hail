# Service Integration

## Overview

Система ride-hail состоит из трёх микросервисов в одном бинарнике:
- **Admin Service** (порт 3004) — управление пользователями
- **Ride Service** (порт 3000) — создание и управление поездками
- **Driver Service** (порт 3001) — управление водителями и геолокацией

---

## Admin → Ride Integration

### Архитектура

```
┌─────────────────┐         ┌──────────────────┐
│  Admin Service  │         │   Ride Service   │
│   (port 3004)   │         │   (port 3000)    │
└────────┬────────┘         └────────┬─────────┘
         │                           │
         │  1. POST /admin/users     │
         │  (создаёт пользователя)   │
         │                           │
         └───────────┬───────────────┘
                     │
                     ▼
         ┌───────────────────────┐
         │   PostgreSQL (users)  │
         │   - id (UUID)         │
         │   - email             │
         │   - role (PASSENGER)  │
         │   - status (ACTIVE)   │
         │   - password_hash     │
         └───────────────────────┘
                     │
                     │
         ┌───────────▼───────────┐
         │   Ride Service        │
         │   JWT Middleware      │
         │                       │
         │  2. Проверяет:        │
         │  - валидность токена  │
         │  - user exists in DB  │
         │  - user.status=ACTIVE │
         │  - user.role=PASSENGER│
         └───────────────────────┘
                     │
                     ▼
         3. POST /rides (success)
```

### Workflow

#### 1. Создание пользователя (Admin Service)

**Request:**
```bash
curl -X POST http://localhost:3004/admin/users \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePass123",
    "role": "PASSENGER"
  }'
```

**Response:**
```json
{
  "user_id": "173e0723-9dc5-4911-9a2e-47e67fe0b3d9",
  "email": "user@example.com",
  "role": "PASSENGER",
  "status": "ACTIVE",
  "created_at": "2025-10-29T22:13:21Z"
}
```

**БД (таблица users):**
```sql
INSERT INTO users (id, email, role, status, password_hash, ...)
VALUES ('173e0723-...', 'user@example.com', 'PASSENGER', 'ACTIVE', '$2a$10$...', ...);
```

---

#### 2. Создание поездки (Ride Service)

**Генерация JWT токена:**
```bash
TOKEN=$(go run cmd/generate-jwt/main.go \
  -user=173e0723-9dc5-4911-9a2e-47e67fe0b3d9 \
  -email=user@example.com \
  -role=PASSENGER \
  | grep '^eyJ' | head -n1)
```

**Request:**
```bash
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

**Middleware проверяет:**
1. ✅ JWT токен валиден (signature, expiry)
2. ✅ `SELECT * FROM users WHERE id = '173e0723-...'` — пользователь существует
3. ✅ `user.status = 'ACTIVE'` — пользователь активен
4. ✅ `user.role = 'PASSENGER'` — роль разрешена для /rides

**Response:**
```json
{
  "ride_id": "c4f9b0d1-e2f3-5a6b-0c9d-8e7f6a5b4c3d",
  "ride_number": "RIDE-20251029-123456",
  "status": "REQUESTED",
  "estimated_fare": 125.50,
  "pickup_address": "Red Square, Moscow",
  "dest_address": "Kremlin, Moscow"
}
```

---

### Validation Logic

#### JWT Middleware (`internal/ride/adapter/in/transport/middleware.go`)

```go
// 1. Валидация JWT токена
claims, err := jwtService.ValidateToken(token)
if err != nil {
    return 401 Unauthorized
}

// 2. Проверка существования пользователя в БД
user, err := userRepo.FindByID(ctx, claims.UserID)
if err == ErrUserNotFound {
    return 401 Unauthorized, "user not found"
}

// 3. Проверка статуса
if user.Status == "BANNED" {
    return 403 Forbidden, "user is banned"
}
if user.Status != "ACTIVE" {
    return 403 Forbidden, "user is inactive"
}

// 4. Проверка роли
if user.Role != "PASSENGER" && user.Role != "ADMIN" {
    return 403 Forbidden, "insufficient permissions"
}

// ✅ Все проверки пройдены
next.ServeHTTP(w, r.WithContext(ctx))
```

---

### Error Scenarios

| Scenario | HTTP Status | Error Message |
|----------|-------------|---------------|
| Токен отсутствует | 401 | `missing authorization header` |
| Невалидный токен | 401 | `invalid or expired token` |
| Пользователь не найден в БД | 401 | `user not found` |
| Пользователь забанен | 403 | `user is banned` |
| Пользователь неактивен | 403 | `user is inactive` |
| Роль DRIVER пытается создать ride | 403 | `insufficient permissions` |

---

## Shared Components

### User Repository (`internal/shared/user/`)

**Интерфейс:**
```go
type Repository interface {
    FindByID(ctx context.Context, userID string) (*User, error)
    Exists(ctx context.Context, userID string) (bool, error)
}
```

**Используется в:**
- ✅ `internal/ride/bootstrap/compose.go` — ride service
- 🔜 `internal/driver/bootstrap/compose.go` — driver service (future)

**Модель:**
```go
type User struct {
    ID        string
    Email     string
    Role      string // PASSENGER | DRIVER | ADMIN
    Status    string // ACTIVE | INACTIVE | BANNED
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

---

## Testing

### E2E Test

Запустить полный тест интеграции:

```bash
chmod +x scripts/test-full-flow.sh
./scripts/test-full-flow.sh
```

**Тест выполняет:**
1. Health check обоих сервисов
2. Создание пассажира через Admin API
3. Генерация JWT для пассажира
4. Создание поездки через Ride API
5. Проверка отказа для несуществующего пользователя
6. Проверка отказа для DRIVER роли

---

## Database Schema

### Table: users

```sql
CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    email varchar(100) UNIQUE NOT NULL,
    role text REFERENCES roles(value) NOT NULL,
    status text REFERENCES user_status(value) NOT NULL DEFAULT 'ACTIVE',
    password_hash text NOT NULL,
    attrs jsonb DEFAULT '{}'::jsonb
);

-- Roles: PASSENGER | DRIVER | ADMIN
-- Status: ACTIVE | INACTIVE | BANNED
```

---

## Performance Considerations

### Current Implementation

- **DB query per request**: `SELECT * FROM users WHERE id = ?`
- **No caching**: каждый HTTP запрос → 1 SELECT

### Future Optimizations

1. **Redis cache** (TTL 5 min):
   ```go
   user, err := cache.Get(userID)
   if err != nil {
       user, err = userRepo.FindByID(ctx, userID)
       cache.Set(userID, user, 5*time.Minute)
   }
   ```

2. **In-memory cache** (для low-traffic):
   ```go
   cache := lru.New(1000) // 1000 пользователей
   ```

3. **DB connection pooling**:
   - Уже реализовано через `pgxpool` (max_conns=25)

---

## Security Notes

1. **JWT Secret**: одинаковый для всех сервисов (`config/jwt.yaml`)
2. **Password hashing**: bcrypt cost=10
3. **User validation**: двухэтапная (JWT + DB lookup)
4. **Role-based access**: middleware проверяет роли per endpoint
5. **Banned users**: немедленная блокировка (403 Forbidden)

---

## Roadmap

- [ ] Кеширование user lookups (Redis/in-memory)
- [ ] Audit logging для Admin API операций
- [ ] WebSocket уведомления при изменении user.status
- [ ] Driver Service интеграция (аналогично Ride Service)
- [ ] Rate limiting per user_id