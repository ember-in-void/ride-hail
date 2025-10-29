#!/bin/bash

set -e

ADMIN_API_URL="http://localhost:3004"

echo "=== Testing Admin API ==="
echo ""

# 1. Проверяем health
echo "1. Checking /health endpoint..."
curl -s "$ADMIN_API_URL/health" | jq .
echo ""

# 2. Генерируем JWT токен для ADMIN
echo "2. Generating ADMIN JWT token..."
ADMIN_TOKEN=$(go run cmd/generate-jwt/main.go \
  -user=admin-001 \
  -email=admin@ridehail.com \
  -role=ADMIN \
  2>/dev/null | grep '^eyJ' | head -n1 | xargs)

if [ -z "$ADMIN_TOKEN" ]; then
    echo "❌ Failed to generate admin token"
    exit 1
fi

echo "✅ Admin token generated (first 50 chars): ${ADMIN_TOKEN:0:50}..."
echo ""

# 3. Создаем PASSENGER через Admin API
echo "3. Creating PASSENGER user..."
PASSENGER_RESPONSE=$(curl -s -X POST "$ADMIN_API_URL/admin/users" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "passenger1@example.com",
    "password": "SecurePass123",
    "role": "PASSENGER",
    "attrs": {
      "phone": "+1234567890",
      "name": "John Doe"
    }
  }')

echo "$PASSENGER_RESPONSE" | jq .
PASSENGER_ID=$(echo "$PASSENGER_RESPONSE" | jq -r '.user_id // empty')

if [ -n "$PASSENGER_ID" ] && [ "$PASSENGER_ID" != "null" ]; then
    echo ""
    echo "✅ Created passenger: $PASSENGER_ID"
else
    echo ""
    echo "⚠️  Passenger creation failed or user already exists"
fi
echo ""

# 4. Создаем DRIVER через Admin API
echo "4. Creating DRIVER user..."
DRIVER_RESPONSE=$(curl -s -X POST "$ADMIN_API_URL/admin/users" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "driver1@example.com",
    "password": "SecurePass456",
    "role": "DRIVER",
    "attrs": {
      "license_number": "DRV12345",
      "vehicle_model": "Toyota Camry"
    }
  }')

echo "$DRIVER_RESPONSE" | jq .
DRIVER_ID=$(echo "$DRIVER_RESPONSE" | jq -r '.user_id // empty')

if [ -n "$DRIVER_ID" ] && [ "$DRIVER_ID" != "null" ]; then
    echo ""
    echo "✅ Created driver: $DRIVER_ID"
else
    echo ""
    echo "⚠️  Driver creation failed or user already exists"
fi
echo ""

# 5. Получаем список всех пользователей
echo "5. Listing all users..."
curl -s -X GET "$ADMIN_API_URL/admin/users?limit=10" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq .
echo ""

# 6. Фильтр: только PASSENGER
echo "6. Listing PASSENGER users only..."
curl -s -X GET "$ADMIN_API_URL/admin/users?role=PASSENGER&limit=10" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq .
echo ""

# 7. Фильтр: только DRIVER
echo "7. Listing DRIVER users only..."
curl -s -X GET "$ADMIN_API_URL/admin/users?role=DRIVER&limit=10" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq .
echo ""

# 8. Тест без токена (должен вернуть 401)
echo "8. Testing without token (should fail with 401)..."
curl -s -X GET "$ADMIN_API_URL/admin/users" | jq .
echo ""

# 9. Тест с PASSENGER токеном (должен вернуть 403 Forbidden)
echo "9. Testing with PASSENGER token (should fail with 403)..."
PASSENGER_TOKEN=$(go run cmd/generate-jwt/main.go \
  -user=passenger-test \
  -email=passenger@test.com \
  -role=PASSENGER \
  2>/dev/null | grep '^eyJ' | head -n1 | xargs)

curl -s -X GET "$ADMIN_API_URL/admin/users" \
  -H "Authorization: Bearer $PASSENGER_TOKEN" | jq .
echo ""

# 10. Тест создания пользователя с коротким паролем (должен вернуть 400)
echo "10. Testing short password (should fail with 400)..."
curl -s -X POST "$ADMIN_API_URL/admin/users" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "123",
    "role": "PASSENGER"
  }' | jq .
echo ""

# 11. Тест дублирования email (должен вернуть 409 Conflict)
echo "11. Testing duplicate email (should fail with 409)..."
curl -s -X POST "$ADMIN_API_URL/admin/users" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "passenger1@example.com",
    "password": "AnotherPass123",
    "role": "PASSENGER"
  }' | jq .
echo ""

echo "=== All Admin API tests completed ==="