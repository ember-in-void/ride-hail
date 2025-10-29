#!/bin/bash

set -e

ADMIN_API_URL="http://localhost:3004"
RIDE_API_URL="http://localhost:3000"

echo "=== E2E Test: Admin → Ride Integration ==="
echo ""

# 1. Проверяем health обоих сервисов
echo "1. Checking services health..."
ADMIN_HEALTH=$(curl -s "$ADMIN_API_URL/health" | jq -r '.service')
RIDE_HEALTH=$(curl -s "$RIDE_API_URL/health" | jq -r '.status')

if [ "$ADMIN_HEALTH" != "admin" ] || [ "$RIDE_HEALTH" != "ok" ]; then
    echo "❌ Services not ready"
    echo "Admin: $ADMIN_HEALTH, Ride: $RIDE_HEALTH"
    exit 1
fi

echo "✅ Both services are healthy"
echo ""

# 2. Генерируем admin токен
echo "2. Generating ADMIN token..."
ADMIN_TOKEN=$(go run cmd/generate-jwt/main.go \
  -user=00000000-0000-0000-0000-000000000001 \
  -email=admin@ridehail.com \
  -role=ADMIN \
  2>/dev/null | grep '^eyJ' | head -n1 | xargs)

if [ -z "$ADMIN_TOKEN" ]; then
    echo "❌ Failed to generate admin token"
    exit 1
fi

echo "✅ Admin token generated"
echo ""

# 3. Создаем passenger через Admin API
echo "3. Creating PASSENGER via Admin API..."
PASSENGER_EMAIL="e2e-passenger-$(date +%s)@example.com"
PASSENGER_PASSWORD="TestPass123"

PASSENGER_RESPONSE=$(curl -s -X POST "$ADMIN_API_URL/admin/users" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "'"$PASSENGER_EMAIL"'",
    "password": "'"$PASSENGER_PASSWORD"'",
    "role": "PASSENGER",
    "attrs": {
      "name": "E2E Test User",
      "phone": "+1234567890"
    }
  }')

echo "$PASSENGER_RESPONSE" | jq .

PASSENGER_ID=$(echo "$PASSENGER_RESPONSE" | jq -r '.user_id // .UserID // empty')

if [ -z "$PASSENGER_ID" ] || [ "$PASSENGER_ID" == "null" ]; then
    echo "❌ Failed to create passenger"
    exit 1
fi

echo "✅ Created passenger: $PASSENGER_ID"
echo ""

# 4. Генерируем JWT для passenger
echo "4. Generating JWT token for passenger..."
PASSENGER_TOKEN=$(go run cmd/generate-jwt/main.go \
  -user="$PASSENGER_ID" \
  -email="$PASSENGER_EMAIL" \
  -role=PASSENGER \
  2>/dev/null | grep '^eyJ' | head -n1 | xargs)

if [ -z "$PASSENGER_TOKEN" ]; then
    echo "❌ Failed to generate passenger token"
    exit 1
fi

echo "✅ Passenger token generated"
echo ""

# 5. Создаем ride через Ride API
echo "5. Creating ride as passenger..."
RIDE_RESPONSE=$(curl -s -X POST "$RIDE_API_URL/rides" \
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
    "priority": 1
  }')

echo "$RIDE_RESPONSE" | jq .

RIDE_ID=$(echo "$RIDE_RESPONSE" | jq -r '.ride_id // empty')

if [ -z "$RIDE_ID" ] || [ "$RIDE_ID" == "null" ]; then
    echo "❌ Failed to create ride"
    exit 1
fi

echo "✅ Created ride: $RIDE_ID"
echo ""

# 6. Тест: попытка создать ride с несуществующим user_id
echo "6. Testing with non-existent user (should fail with 401)..."
FAKE_TOKEN=$(go run cmd/generate-jwt/main.go \
  -user=99999999-9999-9999-9999-999999999999 \
  -email=fake@example.com \
  -role=PASSENGER \
  2>/dev/null | grep '^eyJ' | head -n1 | xargs)

FAKE_RESPONSE=$(curl -s -X POST "$RIDE_API_URL/rides" \
  -H "Authorization: Bearer $FAKE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "vehicle_type": "ECONOMY",
    "pickup_address": "Test",
    "destination_address": "Test"
  }')

echo "$FAKE_RESPONSE" | jq .

FAKE_ERROR=$(echo "$FAKE_RESPONSE" | jq -r '.error // empty')
if [ "$FAKE_ERROR" != "user not found" ]; then
    echo "❌ Expected 'user not found' error, got: $FAKE_ERROR"
    exit 1
fi

echo "✅ Correctly rejected non-existent user"
echo ""

# 7. Тест: попытка DRIVER создать ride (должен быть 403)
echo "7. Creating DRIVER and testing ride creation (should fail with 403)..."
DRIVER_EMAIL="e2e-driver-$(date +%s)@example.com"

DRIVER_RESPONSE=$(curl -s -X POST "$ADMIN_API_URL/admin/users" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "'"$DRIVER_EMAIL"'",
    "password": "DriverPass123",
    "role": "DRIVER"
  }')

DRIVER_ID=$(echo "$DRIVER_RESPONSE" | jq -r '.user_id // .UserID // empty')

if [ -z "$DRIVER_ID" ]; then
    echo "❌ Failed to create driver"
    exit 1
fi

DRIVER_TOKEN=$(go run cmd/generate-jwt/main.go \
  -user="$DRIVER_ID" \
  -email="$DRIVER_EMAIL" \
  -role=DRIVER \
  2>/dev/null | grep '^eyJ' | head -n1 | xargs)

DRIVER_RIDE_RESPONSE=$(curl -s -X POST "$RIDE_API_URL/rides" \
  -H "Authorization: Bearer $DRIVER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "vehicle_type": "ECONOMY",
    "pickup_address": "Test",
    "destination_address": "Test"
  }')

echo "$DRIVER_RIDE_RESPONSE" | jq .

DRIVER_ERROR=$(echo "$DRIVER_RIDE_RESPONSE" | jq -r '.error // empty')
if [ "$DRIVER_ERROR" != "insufficient permissions" ]; then
    echo "❌ Expected 'insufficient permissions', got: $DRIVER_ERROR"
    exit 1
fi

echo "✅ Correctly rejected DRIVER role"
echo ""

# 8. Сводка
echo "=== E2E Test Summary ==="
echo "✅ Admin service: operational"
echo "✅ Ride service: operational"
echo "✅ Created passenger: $PASSENGER_ID"
echo "✅ Created ride: $RIDE_ID"
echo "✅ Non-existent user validation: working"
echo "✅ Role-based access control: working"
echo ""
echo "🎉 All E2E tests passed!"