# 🚗 Driver Service Testing Guide

## Быстрый старт

### 1. Запустите все сервисы
```bash
cd deployments
docker-compose up -d
```

### 2. Создайте тестового водителя
```bash
./scripts/setup-test-driver.sh
```

Скрипт создаст водителя и выведет его ID. Сохраните этот ID!

### 3. Запустите тесты
```bash
# Экспортируйте ID созданного водителя
export DRIVER_ID="YOUR-DRIVER-ID-HERE"

# Запустите полное тестирование API
./scripts/test-driver-api.sh
```

## 📋 Доступные скрипты

### `setup-test-driver.sh`
Создает тестового водителя через Admin API.
```bash
./scripts/setup-test-driver.sh
# С кастомным email:
DRIVER_EMAIL="mydriver@test.com" ./scripts/setup-test-driver.sh
```

### `generate-driver-token.sh`
Генерирует JWT токен для водителя.
```bash
./scripts/generate-driver-token.sh [DRIVER_ID] [EMAIL]
# Пример:
./scripts/generate-driver-token.sh "e0b3bb3e-c7ce-46d8-9f8a-f2cf84d81ddf" "testdriver@ridehail.com"
```

### `test-driver-api.sh` ⭐
Полное автоматическое тестирование всех эндпоинтов:
- ✅ Health check
- ✅ Go online
- ✅ Update location  
- ✅ Rate limit (3 секунды)
- ✅ Go offline
- ✅ Негативные тесты (invalid token, wrong ID, invalid coordinates)

```bash
export DRIVER_ID="your-driver-id"
./scripts/test-driver-api.sh
```

### `test-driver-workflow.sh`
Тестирует полный workflow водителя:
1. Выход онлайн
2. Обновление локации
3. Начало поездки
4. Обновление локации во время поездки
5. Завершение поездки
6. Выход офлайн

```bash
export DRIVER_ID="your-driver-id"
export RIDE_ID="your-ride-id"  # Опционально, для тестов с поездкой
./scripts/test-driver-workflow.sh
```

### `driver-api-helpers.sh`
Интерактивные функции для ручного тестирования:
```bash
source ./scripts/driver-api-helpers.sh

# Примеры:
health
online
online 43.238949 76.889709
location 43.240000 76.890000 50 180
offline
start_ride <ride-id>
complete_ride <ride-id> 43.222015 76.851511 5.2 15
```

## 🌐 API Эндпоинты

### Health Check (без авторизации)
```bash
curl http://localhost:3001/health
```

### Go Online
```bash
curl -X POST http://localhost:3001/drivers/{driver_id}/online \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "latitude": 43.238949,
    "longitude": 76.889709
  }'
```

### Update Location
```bash
curl -X POST http://localhost:3001/drivers/{driver_id}/location \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "latitude": 43.240000,
    "longitude": 76.890000,
    "speed_kmh": 45.5,
    "heading_degrees": 180.0
  }'
```

### Go Offline
```bash
curl -X POST http://localhost:3001/drivers/{driver_id}/offline \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Start Ride
```bash
curl -X POST http://localhost:3001/drivers/{driver_id}/start \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "ride_id": "ride-uuid",
    "latitude": 43.241000,
    "longitude": 76.891000
  }'
```

### Complete Ride
```bash
curl -X POST http://localhost:3001/drivers/{driver_id}/complete \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "ride_id": "ride-uuid",
    "final_latitude": 43.222015,
    "final_longitude": 76.851511,
    "actual_distance_km": 5.2,
    "actual_duration_minutes": 15
  }'
```

## 🔧 Переменные окружения

```bash
# Driver service URL
export DRIVER_SERVICE_URL="http://localhost:3001"

# Driver credentials
export DRIVER_ID="e0b3bb3e-c7ce-46d8-9f8a-f2cf84d81ddf"
export DRIVER_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

# Admin service (для создания пользователей)
export ADMIN_SERVICE_URL="http://localhost:3004"

# Ride ID (для тестов с поездками)
export RIDE_ID="99999999-9999-9999-9999-999999999999"
```

## ✅ Тестовые сценарии

### Базовый сценарий
```bash
# 1. Создать водителя
./scripts/setup-test-driver.sh

# 2. Запустить тесты
export DRIVER_ID="<ID из вывода скрипта>"
./scripts/test-driver-api.sh
```

### Полный workflow
```bash
# 1. Создать водителя и сохранить ID
export DRIVER_ID=$(./scripts/setup-test-driver.sh | grep "Driver ID:" | awk '{print $3}')

# 2. Запустить workflow тест
./scripts/test-driver-workflow.sh
```

### Ручное тестирование
```bash
# 1. Загрузить helper функции
source ./scripts/driver-api-helpers.sh

# 2. Использовать команды
health
online 43.238949 76.889709
sleep 4
location 43.240000 76.890000
sleep 4
offline
```

## 🐛 Troubleshooting

### "driver not found"
Убедитесь что:
1. Водитель создан через `setup-test-driver.sh`
2. Используете правильный DRIVER_ID
3. Admin service запущен и доступен

### "invalid or expired token"
Сгенерируйте новый токен:
```bash
./scripts/generate-driver-token.sh $DRIVER_ID "your-email@example.com"
export DRIVER_TOKEN="<новый токен>"
```

### "rate limit exceeded"
Подождите 3 секунды между обновлениями локации:
```bash
location 43.25 76.90
sleep 3
location 43.26 76.91
```

### "driver cannot go online: invalid status or not verified"
Водитель должен быть verified. Пересоздайте через `setup-test-driver.sh` с обновленной версией admin service.

## 📊 Ожидаемые результаты

При успешном прохождении `test-driver-api.sh`:
```
✅ PASSED: Health Check
✅ PASSED: Go Online
✅ PASSED: Update Location
✅ PASSED: Rate limit works correctly
✅ PASSED: Go Offline
✅ PASSED: Invalid token rejected
✅ PASSED: ID mismatch detected
✅ PASSED: Invalid coordinates rejected

========================================
✅ Driver Service API Tests Completed
========================================
```

## 🚀 Что дальше?

1. Интеграция с Ride Service для реальных поездок
2. WebSocket тестирование
3. Нагрузочное тестирование
4. E2E тесты полного flow: passenger request → driver accept → ride → complete
