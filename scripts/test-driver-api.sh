#!/bin/bash

# Скрипт для тестирования Driver API

set -e

# Цвета для вывода
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

DRIVER_SERVICE_URL=${DRIVER_SERVICE_URL:-"http://localhost:3001"}
DRIVER_ID=${DRIVER_ID:-"11111111-1111-1111-1111-111111111111"}

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}🚗 Testing Driver Service API${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Проверка наличия токена
if [ -z "$DRIVER_TOKEN" ]; then
    echo -e "${YELLOW}⚠️  DRIVER_TOKEN not set. Generating...${NC}"
    echo ""
    DRIVER_TOKEN=$(go run cmd/generate-jwt/main.go \
      -user="$DRIVER_ID" \
      -email=driver@ridehail.com \
      -role=DRIVER \
      2>/dev/null | grep '^eyJ' | head -n1 | xargs)
    
    if [ -z "$DRIVER_TOKEN" ]; then
        echo -e "${RED}❌ Failed to generate token${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}✅ Token generated${NC}"
    echo ""
fi

AUTH_HEADER="Authorization: Bearer $DRIVER_TOKEN"

echo -e "${BLUE}Using:${NC}"
echo -e "  Driver ID: ${YELLOW}$DRIVER_ID${NC}"
echo -e "  Service URL: ${YELLOW}$DRIVER_SERVICE_URL${NC}"
echo ""

# Функция для красивого вывода JSON
pretty_json() {
    if command -v jq &> /dev/null; then
        echo "$1" | jq '.'
    else
        echo "$1"
    fi
}

# Функция для проверки ответа
check_response() {
    local response="$1"
    local test_name="$2"
    
    if echo "$response" | grep -q '"error"' || echo "$response" | grep -q 'HTTP/.*[45][0-9][0-9]'; then
        echo -e "${RED}❌ FAILED: $test_name${NC}"
        pretty_json "$response"
        return 1
    else
        echo -e "${GREEN}✅ PASSED: $test_name${NC}"
        pretty_json "$response"
        return 0
    fi
}

# 1. Test: Health Check
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Test 1: Health Check${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
response=$(curl -s "$DRIVER_SERVICE_URL/health")
check_response "$response" "Health Check"
echo ""

# 2. Test: Go Online
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Test 2: Go Online${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
response=$(curl -s -X POST "$DRIVER_SERVICE_URL/drivers/$DRIVER_ID/online" \
  -H "Content-Type: application/json" \
  -H "$AUTH_HEADER" \
  -d '{
    "latitude": 43.238949,
    "longitude": 76.889709
  }')
check_response "$response" "Go Online"

# Извлекаем session_id для последующих тестов
SESSION_ID=$(echo "$response" | grep -o '"session_id":"[^"]*"' | cut -d'"' -f4 || echo "")
echo -e "${YELLOW}Session ID: $SESSION_ID${NC}"
echo ""

# 3. Test: Update Location
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Test 3: Update Location${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
sleep 4  # Ждем чтобы не нарваться на rate limit
response=$(curl -s -X POST "$DRIVER_SERVICE_URL/drivers/$DRIVER_ID/location" \
  -H "Content-Type: application/json" \
  -H "$AUTH_HEADER" \
  -d '{
    "latitude": 43.240000,
    "longitude": 76.890000,
    "speed_kmh": 45.5,
    "heading_degrees": 180.0
  }')
check_response "$response" "Update Location"
echo ""

# 4. Test: Update Location with Rate Limit (should fail)
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Test 4: Update Location (Rate Limited - should fail)${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
response=$(curl -s -X POST "$DRIVER_SERVICE_URL/drivers/$DRIVER_ID/location" \
  -H "Content-Type: application/json" \
  -H "$AUTH_HEADER" \
  -d '{
    "latitude": 43.241000,
    "longitude": 76.891000
  }')
if echo "$response" | grep -q "rate limit"; then
    echo -e "${GREEN}✅ PASSED: Rate limit works correctly${NC}"
else
    echo -e "${YELLOW}⚠️  Rate limit might not be working${NC}"
fi
pretty_json "$response"
echo ""

# 5. Test: Go Offline
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Test 5: Go Offline${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
response=$(curl -s -X POST "$DRIVER_SERVICE_URL/drivers/$DRIVER_ID/offline" \
  -H "Content-Type: application/json" \
  -H "$AUTH_HEADER")
check_response "$response" "Go Offline"
echo ""

# 6. Test: Invalid Token (should fail)
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Test 6: Invalid Token (should fail)${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
response=$(curl -s -X POST "$DRIVER_SERVICE_URL/drivers/$DRIVER_ID/online" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer invalid_token" \
  -d '{
    "latitude": 43.238949,
    "longitude": 76.889709
  }')
if echo "$response" | grep -qi "invalid\|unauthorized"; then
    echo -e "${GREEN}✅ PASSED: Invalid token rejected${NC}"
else
    echo -e "${RED}❌ FAILED: Invalid token should be rejected${NC}"
fi
pretty_json "$response"
echo ""

# 7. Test: Wrong Driver ID (should fail)
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Test 7: Wrong Driver ID (should fail)${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
WRONG_DRIVER_ID="22222222-2222-2222-2222-222222222222"
response=$(curl -s -X POST "$DRIVER_SERVICE_URL/drivers/$WRONG_DRIVER_ID/online" \
  -H "Content-Type: application/json" \
  -H "$AUTH_HEADER" \
  -d '{
    "latitude": 43.238949,
    "longitude": 76.889709
  }')
if echo "$response" | grep -qi "mismatch\|forbidden"; then
    echo -e "${GREEN}✅ PASSED: ID mismatch detected${NC}"
else
    echo -e "${RED}❌ FAILED: Should reject ID mismatch${NC}"
fi
pretty_json "$response"
echo ""

# 8. Test: Invalid Coordinates (should fail)
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Test 8: Invalid Coordinates (should fail)${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
response=$(curl -s -X POST "$DRIVER_SERVICE_URL/drivers/$DRIVER_ID/online" \
  -H "Content-Type: application/json" \
  -H "$AUTH_HEADER" \
  -d '{
    "latitude": 91.0,
    "longitude": 181.0
  }')
if echo "$response" | grep -qi "invalid.*coordinate\|latitude\|longitude"; then
    echo -e "${GREEN}✅ PASSED: Invalid coordinates rejected${NC}"
else
    echo -e "${RED}❌ FAILED: Should reject invalid coordinates${NC}"
fi
pretty_json "$response"
echo ""

echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}✅ Driver Service API Tests Completed${NC}"
echo -e "${BLUE}========================================${NC}"
