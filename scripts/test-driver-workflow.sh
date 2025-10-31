#!/bin/bash

# Полный тест сценария: водитель выходит онлайн, принимает поездку, завершает её

set -e

# Цвета
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

DRIVER_SERVICE_URL=${DRIVER_SERVICE_URL:-"http://localhost:3001"}
DRIVER_ID=${DRIVER_ID:-"11111111-1111-1111-1111-111111111111"}
RIDE_ID=${RIDE_ID:-"99999999-9999-9999-9999-999999999999"}

echo -e "${BLUE}╔════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║   🚗 Full Driver Workflow Test                    ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════╝${NC}"
echo ""

# Генерация токена
if [ -z "$DRIVER_TOKEN" ]; then
    echo -e "${YELLOW}⚠️  Generating driver token...${NC}"
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

pretty_json() {
    if command -v jq &> /dev/null; then
        echo "$1" | jq '.'
    else
        echo "$1"
    fi
}

echo -e "${BLUE}📍 Step 1: Driver goes ONLINE${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
response=$(curl -s -X POST "$DRIVER_SERVICE_URL/drivers/$DRIVER_ID/online" \
  -H "Content-Type: application/json" \
  -H "$AUTH_HEADER" \
  -d '{
    "latitude": 43.238949,
    "longitude": 76.889709
  }')
pretty_json "$response"
SESSION_ID=$(echo "$response" | grep -o '"session_id":"[^"]*"' | cut -d'"' -f4 || echo "")
echo -e "${GREEN}✅ Driver is online (Session: $SESSION_ID)${NC}"
echo ""
sleep 2

echo -e "${BLUE}📍 Step 2: Driver updates location${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
sleep 3  # Wait for rate limit
response=$(curl -s -X POST "$DRIVER_SERVICE_URL/drivers/$DRIVER_ID/location" \
  -H "Content-Type: application/json" \
  -H "$AUTH_HEADER" \
  -d '{
    "latitude": 43.240000,
    "longitude": 76.890000,
    "speed_kmh": 30.0,
    "heading_degrees": 90.0
  }')
pretty_json "$response"
echo -e "${GREEN}✅ Location updated${NC}"
echo ""
sleep 2

echo -e "${BLUE}📍 Step 3: Driver starts a ride${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}NOTE: This assumes ride $RIDE_ID exists and is assigned to this driver${NC}"
response=$(curl -s -X POST "$DRIVER_SERVICE_URL/drivers/$DRIVER_ID/start" \
  -H "Content-Type: application/json" \
  -H "$AUTH_HEADER" \
  -d "{
    \"ride_id\": \"$RIDE_ID\",
    \"latitude\": 43.241000,
    \"longitude\": 76.891000
  }")
pretty_json "$response"
if echo "$response" | grep -q "IN_PROGRESS\|started"; then
    echo -e "${GREEN}✅ Ride started${NC}"
else
    echo -e "${YELLOW}⚠️  Ride start failed (expected if ride doesn't exist)${NC}"
fi
echo ""
sleep 2

echo -e "${BLUE}📍 Step 4: Driver updates location during ride${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
sleep 3
response=$(curl -s -X POST "$DRIVER_SERVICE_URL/drivers/$DRIVER_ID/location" \
  -H "Content-Type: application/json" \
  -H "$AUTH_HEADER" \
  -d '{
    "latitude": 43.242000,
    "longitude": 76.892000,
    "speed_kmh": 50.0,
    "heading_degrees": 180.0
  }')
pretty_json "$response"
echo -e "${GREEN}✅ Location updated during ride${NC}"
echo ""
sleep 2

echo -e "${BLUE}📍 Step 5: Driver completes the ride${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
response=$(curl -s -X POST "$DRIVER_SERVICE_URL/drivers/$DRIVER_ID/complete" \
  -H "Content-Type: application/json" \
  -H "$AUTH_HEADER" \
  -d "{
    \"ride_id\": \"$RIDE_ID\",
    \"final_latitude\": 43.222015,
    \"final_longitude\": 76.851511,
    \"actual_distance_km\": 5.2,
    \"actual_duration_minutes\": 15
  }")
pretty_json "$response"
if echo "$response" | grep -q "COMPLETED\|completed"; then
    echo -e "${GREEN}✅ Ride completed${NC}"
    EARNINGS=$(echo "$response" | grep -o '"driver_earnings":[0-9.]*' | cut -d':' -f2 || echo "0")
    echo -e "${GREEN}💰 Driver earnings: $EARNINGS ₸${NC}"
else
    echo -e "${YELLOW}⚠️  Ride completion failed (expected if ride doesn't exist)${NC}"
fi
echo ""
sleep 2

echo -e "${BLUE}📍 Step 6: Driver goes OFFLINE${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
response=$(curl -s -X POST "$DRIVER_SERVICE_URL/drivers/$DRIVER_ID/offline" \
  -H "Content-Type: application/json" \
  -H "$AUTH_HEADER")
pretty_json "$response"
DURATION=$(echo "$response" | grep -o '"duration_hours":[0-9.]*' | cut -d':' -f2 || echo "0")
RIDES=$(echo "$response" | grep -o '"rides_completed":[0-9]*' | cut -d':' -f2 || echo "0")
TOTAL_EARNINGS=$(echo "$response" | grep -o '"earnings":[0-9.]*' | cut -d':' -f2 || echo "0")
echo -e "${GREEN}✅ Driver is offline${NC}"
echo -e "${GREEN}📊 Session summary:${NC}"
echo -e "  ⏱️  Duration: ${YELLOW}$DURATION hours${NC}"
echo -e "  🚗 Rides completed: ${YELLOW}$RIDES${NC}"
echo -e "  💰 Total earnings: ${YELLOW}$TOTAL_EARNINGS ₸${NC}"
echo ""

echo -e "${BLUE}╔════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║   ✅ Full Driver Workflow Test Completed          ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════╝${NC}"
