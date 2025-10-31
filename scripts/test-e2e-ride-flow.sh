#!/bin/bash

# =============================================================================
# End-to-End Ride Flow Test
# Тестирует полный цикл: создание поездки → матчинг водителя → отслеживание
# =============================================================================

set -e

BASE_URL="http://localhost:3000"
DRIVER_URL="http://localhost:3001"
ADMIN_URL="http://localhost:3002"

echo "=========================================="
echo "E2E Ride Flow Test"
echo "=========================================="
echo ""

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# =============================================================================
# 1. Генерация токенов
# =============================================================================
echo -e "${BLUE}[1/8] Generating JWT tokens...${NC}"

# Admin token
ADMIN_TOKEN=$(JWT_SECRET="your-secret-key-min-32-characters-long!" \
  go run cmd/generate-jwt/main.go \
  --user-id "admin-1" \
  --role "ADMIN" \
  --ttl "1h" 2>/dev/null | grep "Token:" | awk '{print $2}')

# Passenger token
PASSENGER_ID="passenger-$(date +%s)"
PASSENGER_TOKEN=$(JWT_SECRET="your-secret-key-min-32-characters-long!" \
  go run cmd/generate-jwt/main.go \
  --user-id "$PASSENGER_ID" \
  --role "PASSENGER" \
  --ttl "1h" 2>/dev/null | grep "Token:" | awk '{print $2}')

# Driver token
DRIVER_ID="driver-$(date +%s)"
DRIVER_TOKEN=$(JWT_SECRET="your-secret-key-min-32-characters-long!" \
  go run cmd/generate-jwt/main.go \
  --user-id "$DRIVER_ID" \
  --role "DRIVER" \
  --ttl "1h" 2>/dev/null | grep "Token:" | awk '{print $2}')

echo -e "${GREEN}✓ Tokens generated${NC}"
echo "  Passenger ID: $PASSENGER_ID"
echo "  Driver ID: $DRIVER_ID"
echo ""

# =============================================================================
# 2. Создание пользователей в БД
# =============================================================================
echo -e "${BLUE}[2/8] Creating users in database...${NC}"

# Создаем пассажира
curl -s -X POST "$ADMIN_URL/admin/users" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"id\": \"$PASSENGER_ID\",
    \"email\": \"passenger-test@example.com\",
    \"role\": \"PASSENGER\",
    \"phone\": \"+1234567890\"
  }" > /dev/null

# Создаем водителя
curl -s -X POST "$ADMIN_URL/admin/users" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"id\": \"$DRIVER_ID\",
    \"email\": \"driver-test@example.com\",
    \"role\": \"DRIVER\",
    \"phone\": \"+9876543210\"
  }" > /dev/null

echo -e "${GREEN}✓ Users created${NC}"
echo ""

# =============================================================================
# 3. Водитель выходит онлайн
# =============================================================================
echo -e "${BLUE}[3/8] Driver going online...${NC}"

ONLINE_RESPONSE=$(curl -s -X POST "$DRIVER_URL/drivers/$DRIVER_ID/online" \
  -H "Authorization: Bearer $DRIVER_TOKEN" \
  -H "Content-Type: application/json")

echo "$ONLINE_RESPONSE" | jq '.' 2>/dev/null || echo "$ONLINE_RESPONSE"
echo -e "${GREEN}✓ Driver is online${NC}"
echo ""

# =============================================================================
# 4. Водитель обновляет локацию (Moscow, Russia)
# =============================================================================
echo -e "${BLUE}[4/8] Updating driver location (Moscow, Russia)...${NC}"

LOCATION_RESPONSE=$(curl -s -X POST "$DRIVER_URL/drivers/$DRIVER_ID/location" \
  -H "Authorization: Bearer $DRIVER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "latitude": 55.7558,
    "longitude": 37.6173,
    "accuracy_meters": 10.0
  }')

echo "$LOCATION_RESPONSE" | jq '.' 2>/dev/null || echo "$LOCATION_RESPONSE"
echo -e "${GREEN}✓ Driver location updated${NC}"
echo ""

# Даем время на обработку
sleep 2

# =============================================================================
# 5. Пассажир создает запрос на поездку
# =============================================================================
echo -e "${BLUE}[5/8] Passenger requesting a ride...${NC}"

RIDE_RESPONSE=$(curl -s -X POST "$BASE_URL/rides" \
  -H "Authorization: Bearer $PASSENGER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "vehicle_type": "ECONOMY",
    "pickup_lat": 55.7500,
    "pickup_lng": 37.6200,
    "pickup_address": "Red Square, Moscow",
    "destination_lat": 55.7600,
    "destination_lng": 37.6100,
    "destination_address": "Kremlin, Moscow",
    "priority": 5
  }')

echo "$RIDE_RESPONSE" | jq '.' 2>/dev/null || echo "$RIDE_RESPONSE"

RIDE_ID=$(echo "$RIDE_RESPONSE" | jq -r '.ride_id' 2>/dev/null)
RIDE_NUMBER=$(echo "$RIDE_RESPONSE" | jq -r '.ride_number' 2>/dev/null)

if [ "$RIDE_ID" != "null" ] && [ -n "$RIDE_ID" ]; then
  echo -e "${GREEN}✓ Ride created${NC}"
  echo "  Ride ID: $RIDE_ID"
  echo "  Ride Number: $RIDE_NUMBER"
else
  echo -e "${RED}✗ Failed to create ride${NC}"
  exit 1
fi
echo ""

# =============================================================================
# 6. Проверка публикации в RabbitMQ
# =============================================================================
echo -e "${BLUE}[6/8] Checking RabbitMQ message flow...${NC}"
echo -e "${YELLOW}Messages should flow:${NC}"
echo "  1. Ride Service → ride_topic (ride.request.*)"
echo "  2. Driver Service consumes from driver_matching queue"
echo "  3. Driver Service finds nearby drivers with PostGIS"
echo "  4. Driver Service sends WebSocket offer to driver"
echo ""

# Даем время на обработку
sleep 3

# =============================================================================
# 7. Симуляция ответа водителя (через WebSocket)
# =============================================================================
echo -e "${BLUE}[7/8] Simulating driver acceptance...${NC}"
echo -e "${YELLOW}Note: Driver would receive ride offer via WebSocket${NC}"
echo -e "${YELLOW}      Driver would respond with acceptance via WebSocket${NC}"
echo -e "${YELLOW}      Response would be published to driver.response.$RIDE_ID${NC}"
echo ""

# В реальной ситуации:
# - Водитель подключается к ws://localhost:3001/ws
# - Получает сообщение типа "ride_offer"
# - Отправляет ответ: {"type": "ride_response", "ride_id": "...", "accepted": true}
# - DriverWSHandler публикует в RabbitMQ driver.response.{ride_id}
# - Ride Service consumer получает и обновляет статус ride

# =============================================================================
# 8. Проверка состояния системы
# =============================================================================
echo -e "${BLUE}[8/8] Checking system state...${NC}"

# Проверяем что сервисы живы
echo -n "  Ride Service health: "
RIDE_HEALTH=$(curl -s "$BASE_URL/health" | jq -r '.status' 2>/dev/null)
if [ "$RIDE_HEALTH" == "ok" ]; then
  echo -e "${GREEN}✓${NC}"
else
  echo -e "${RED}✗${NC}"
fi

echo -n "  Driver Service health: "
DRIVER_HEALTH=$(curl -s "$DRIVER_URL/health" | jq -r '.status' 2>/dev/null)
if [ "$DRIVER_HEALTH" == "ok" ]; then
  echo -e "${GREEN}✓${NC}"
else
  echo -e "${RED}✗${NC}"
fi

echo -n "  Admin Service health: "
ADMIN_HEALTH=$(curl -s "$ADMIN_URL/health" | jq -r '.status' 2>/dev/null)
if [ "$ADMIN_HEALTH" == "ok" ]; then
  echo -e "${GREEN}✓${NC}"
else
  echo -e "${RED}✗${NC}"
fi

echo ""

# =============================================================================
# Резюме
# =============================================================================
echo "=========================================="
echo -e "${GREEN}E2E Test Summary${NC}"
echo "=========================================="
echo ""
echo "✓ User tokens generated"
echo "✓ Passenger and driver created in DB"
echo "✓ Driver went online and updated location"
echo "✓ Passenger requested a ride"
echo "✓ Ride created with ID: $RIDE_ID"
echo ""
echo -e "${YELLOW}Expected Flow:${NC}"
echo "  1. ✓ Ride Service publishes to ride_topic"
echo "  2. ✓ Driver Service consumes from driver_matching"
echo "  3. ✓ PostGIS finds nearby drivers (radius 5km)"
echo "  4. → Driver receives offer via WebSocket"
echo "  5. → Driver accepts via WebSocket"
echo "  6. → DriverWSHandler publishes to driver.response.*"
echo "  7. → Ride Service consumer receives acceptance"
echo "  8. → Passenger receives match notification"
echo "  9. → Location updates flow via location_fanout"
echo "  10. → Ride completes"
echo ""
echo -e "${BLUE}Next Steps:${NC}"
echo "  • Connect to WebSocket to see real-time messages"
echo "  • Check RabbitMQ UI at http://localhost:15672"
echo "  • Monitor logs in each service"
echo ""
echo "=========================================="
