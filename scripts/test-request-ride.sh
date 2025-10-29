#!/bin/bash

set -e

API_URL="http://localhost:3000"

echo "=== Testing POST /rides endpoint ==="
echo ""

# 1. Проверяем health
echo "1. Checking /health endpoint..."
curl -s "$API_URL/health" | jq .
echo ""

# 2. Генерируем JWT токен (нужен cmd/generate-jwt)
echo "2. Generating JWT token..."
TOKEN=$(go run cmd/generate-jwt/main.go -user=test-user-123 -email=test@example.com -role=PASSENGER 2>/dev/null | grep -v "^✅" | grep -v "^User" | grep -v "^Token:" | grep -v "^📋" | grep -v "^💡" | grep -v "^curl" | grep -v "^  -H" | grep -v "^  -d" | tr -d '\n' | xargs)
echo "Token: $TOKEN"
echo ""

# 3. Создаем поездку
echo "3. Creating ride with valid data..."
RESPONSE=$(curl -s -X POST "$API_URL/rides" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "vehicle_type": "ECONOMY",
    "pickup_lat": 55.7558,
    "pickup_lng": 37.6173,
    "pickup_address": "Red Square, Moscow",
    "destination_lat": 55.7522,
    "destination_lng": 37.6156,
    "destination_address": "Kremlin, Moscow",
    "priority": 1
  }')

echo "$RESPONSE" | jq .
RIDE_ID=$(echo "$RESPONSE" | jq -r '.ride_id')
echo ""
echo "Created ride_id: $RIDE_ID"
echo ""

# 4. Тест без токена (должна быть ошибка 401)
echo "4. Testing without token (should fail)..."
curl -s -X POST "$API_URL/rides" \
  -H "Content-Type: application/json" \
  -d '{
    "vehicle_type": "ECONOMY",
    "pickup_address": "Test",
    "destination_address": "Test"
  }' | jq .
echo ""

# 5. Тест с невалидным типом авто
echo "5. Testing with invalid vehicle type..."
curl -s -X POST "$API_URL/rides" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "vehicle_type": "INVALID_TYPE",
    "pickup_address": "Test",
    "destination_address": "Test"
  }' | jq .
echo ""

# 6. Тест с пустым телом
echo "6. Testing with empty body..."
curl -s -X POST "$API_URL/rides" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}' | jq .
echo ""

echo "=== All tests completed ==="