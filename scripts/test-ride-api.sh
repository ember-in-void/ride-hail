#!/bin/bash

# Скрипт для тестирования Ride Service API

set -e

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

RIDE_SERVICE_URL=${RIDE_SERVICE_URL:-"http://localhost:3000"}
ADMIN_SERVICE_URL=${ADMIN_SERVICE_URL:-"http://localhost:3004"}
PASSENGER_EMAIL=${PASSENGER_EMAIL:-"passenger@ridehail.com"}

echo -e "${BLUE}╔════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║   🚗 Testing Ride Service API                     ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════╝${NC}"
echo ""

# Получаем passenger_id из базы данных
echo -e "${YELLOW}⚠️  Looking up passenger ID...${NC}"

# Генерация admin токена
ADMIN_TOKEN=$(go run cmd/generate-jwt/main.go \
  -user="11111111-1111-1111-1111-111111111111" \
  -email="admin@ridehail.com" \
  -role=ADMIN \
  2>/dev/null | grep '^eyJ' | head -n1 | xargs)

# Получаем список пассажиров
USERS_RESPONSE=$(curl -s "$ADMIN_SERVICE_URL/admin/users?role=PASSENGER&limit=1" \
  -H "Authorization: Bearer $ADMIN_TOKEN")

PASSENGER_ID=$(echo "$USERS_RESPONSE" | grep -o '"user_id":"[^"]*"' | head -n1 | cut -d'"' -f4)

if [ -z "$PASSENGER_ID" ]; then
    echo -e "${RED}❌ No passenger found. Run ./scripts/setup-test-passenger.sh first${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Found passenger: $PASSENGER_ID${NC}"
echo ""

# Генерация passenger токена
if [ -z "$PASSENGER_TOKEN" ]; then
    echo -e "${YELLOW}⚠️  Generating passenger token...${NC}"
    PASSENGER_TOKEN=$(go run cmd/generate-jwt/main.go \
      -user="$PASSENGER_ID" \
      -email="$PASSENGER_EMAIL" \
      -role=PASSENGER \
      2>/dev/null | grep '^eyJ' | head -n1 | xargs)
    
    if [ -z "$PASSENGER_TOKEN" ]; then
        echo -e "${RED}❌ Failed to generate token${NC}"
        exit 1
    fi
    echo -e "${GREEN}✅ Token generated${NC}"
    echo ""
fi

AUTH_HEADER="Authorization: Bearer $PASSENGER_TOKEN"

echo -e "${BLUE}Using:${NC}"
echo -e "  Passenger ID: ${YELLOW}$PASSENGER_ID${NC}"
echo -e "  Service URL: ${YELLOW}$RIDE_SERVICE_URL${NC}"
echo ""

pretty_json() {
    if command -v jq &> /dev/null; then
        echo "$1" | jq '.'
    else
        echo "$1"
    fi
}

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

# Test 1: Health Check
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Test 1: Health Check${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
response=$(curl -s "$RIDE_SERVICE_URL/health")
check_response "$response" "Health Check"
echo ""

# Test 2: Request ECONOMY Ride (Almaty)
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Test 2: Request ECONOMY Ride${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
response=$(curl -s -X POST "$RIDE_SERVICE_URL/rides" \
  -H "Content-Type: application/json" \
  -H "$AUTH_HEADER" \
  -d '{
    "vehicle_type": "ECONOMY",
    "pickup_lat": 43.238949,
    "pickup_lng": 76.889709,
    "pickup_address": "Almaty Central Park",
    "destination_lat": 43.222015,
    "destination_lng": 76.851511,
    "destination_address": "Kok-Tobe Hill"
  }')
check_response "$response" "Request ECONOMY Ride"

# Сохраняем ride_id для дальнейших тестов
RIDE_ID=$(echo "$response" | grep -o '"ride_id":"[^"]*"' | cut -d'"' -f4 || echo "")
if [ -n "$RIDE_ID" ]; then
    echo -e "${YELLOW}Ride ID: $RIDE_ID${NC}"
fi
echo ""

# Test 3: Request PREMIUM Ride
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Test 3: Request PREMIUM Ride${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
response=$(curl -s -X POST "$RIDE_SERVICE_URL/rides" \
  -H "Content-Type: application/json" \
  -H "$AUTH_HEADER" \
  -d '{
    "vehicle_type": "PREMIUM",
    "pickup_lat": 43.256910,
    "pickup_lng": 76.928640,
    "pickup_address": "Mega Alma-Ata",
    "destination_lat": 43.238949,
    "destination_lng": 76.889709,
    "destination_address": "Almaty Central Park",
    "priority": 5
  }')
check_response "$response" "Request PREMIUM Ride"
echo ""

# Test 4: Request XL Ride
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Test 4: Request XL Ride${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
response=$(curl -s -X POST "$RIDE_SERVICE_URL/rides" \
  -H "Content-Type: application/json" \
  -H "$AUTH_HEADER" \
  -d '{
    "vehicle_type": "XL",
    "pickup_lat": 43.222015,
    "pickup_lng": 76.851511,
    "pickup_address": "Kok-Tobe Hill",
    "destination_lat": 43.238949,
    "destination_lng": 76.889709,
    "destination_address": "Almaty Central Park"
  }')
check_response "$response" "Request XL Ride"
echo ""

# Test 5: Invalid Coordinates (should fail)
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Test 5: Invalid Coordinates (should fail)${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
response=$(curl -s -X POST "$RIDE_SERVICE_URL/rides" \
  -H "Content-Type: application/json" \
  -H "$AUTH_HEADER" \
  -d '{
    "vehicle_type": "ECONOMY",
    "pickup_lat": 91.0,
    "pickup_lng": 181.0,
    "pickup_address": "Invalid Location",
    "destination_lat": 43.222015,
    "destination_lng": 76.851511,
    "destination_address": "Kok-Tobe Hill"
  }')
if echo "$response" | grep -qi "invalid.*coordinate\|latitude"; then
    echo -e "${GREEN}✅ PASSED: Invalid coordinates rejected${NC}"
else
    echo -e "${RED}❌ FAILED: Should reject invalid coordinates${NC}"
fi
pretty_json "$response"
echo ""

# Test 6: Invalid Vehicle Type (should fail)
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Test 6: Invalid Vehicle Type (should fail)${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
response=$(curl -s -X POST "$RIDE_SERVICE_URL/rides" \
  -H "Content-Type: application/json" \
  -H "$AUTH_HEADER" \
  -d '{
    "vehicle_type": "INVALID_TYPE",
    "pickup_lat": 43.238949,
    "pickup_lng": 76.889709,
    "pickup_address": "Almaty Central Park",
    "destination_lat": 43.222015,
    "destination_lng": 76.851511,
    "destination_address": "Kok-Tobe Hill"
  }')
if echo "$response" | grep -qi "invalid.*vehicle"; then
    echo -e "${GREEN}✅ PASSED: Invalid vehicle type rejected${NC}"
else
    echo -e "${RED}❌ FAILED: Should reject invalid vehicle type${NC}"
fi
pretty_json "$response"
echo ""

# Test 7: Missing Required Fields (should fail)
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Test 7: Missing Required Fields (should fail)${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
response=$(curl -s -X POST "$RIDE_SERVICE_URL/rides" \
  -H "Content-Type: application/json" \
  -H "$AUTH_HEADER" \
  -d '{
    "vehicle_type": "ECONOMY",
    "pickup_lat": 43.238949,
    "pickup_lng": 76.889709
  }')
if echo "$response" | grep -qi "required\|address"; then
    echo -e "${GREEN}✅ PASSED: Missing fields rejected${NC}"
else
    echo -e "${RED}❌ FAILED: Should reject missing fields${NC}"
fi
pretty_json "$response"
echo ""

# Test 8: Invalid Token (should fail)
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Test 8: Invalid Token (should fail)${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
response=$(curl -s -X POST "$RIDE_SERVICE_URL/rides" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer invalid_token" \
  -d '{
    "vehicle_type": "ECONOMY",
    "pickup_lat": 43.238949,
    "pickup_lng": 76.889709,
    "pickup_address": "Almaty Central Park",
    "destination_lat": 43.222015,
    "destination_lng": 76.851511,
    "destination_address": "Kok-Tobe Hill"
  }')
if echo "$response" | grep -qi "invalid\|unauthorized"; then
    echo -e "${GREEN}✅ PASSED: Invalid token rejected${NC}"
else
    echo -e "${RED}❌ FAILED: Should reject invalid token${NC}"
fi
pretty_json "$response"
echo ""

echo -e "${BLUE}╔════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║   ✅ Ride Service API Tests Completed             ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════╝${NC}"
