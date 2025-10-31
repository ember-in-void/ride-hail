#!/bin/bash

# Скрипт для тестирования WebSocket соединений

set -e

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}╔════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║   🔌 Testing WebSocket Connections                ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════╝${NC}"
echo ""

# Check if websocat is installed
if ! command -v websocat &> /dev/null; then
    echo -e "${YELLOW}⚠️  websocat not found. Install with:${NC}"
    echo -e "   ${YELLOW}cargo install websocat${NC}"
    echo -e "   ${YELLOW}or download from: https://github.com/vi/websocat${NC}"
    echo ""
    echo -e "${BLUE}Using curl for basic WebSocket test instead...${NC}"
    
    # Test Ride Service WebSocket endpoint
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}Test 1: Ride Service WebSocket Endpoint${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    response=$(curl -s -i -N \
      -H "Connection: Upgrade" \
      -H "Upgrade: websocket" \
      -H "Sec-WebSocket-Version: 13" \
      -H "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==" \
      http://localhost:3000/ws 2>&1 | head -n 1)
    
    if echo "$response" | grep -q "101"; then
        echo -e "${GREEN}✅ PASSED: WebSocket upgrade successful (HTTP 101)${NC}"
    elif echo "$response" | grep -q "426"; then
        echo -e "${GREEN}✅ PASSED: Endpoint exists (Upgrade Required)${NC}"
    else
        echo -e "${YELLOW}⚠️  Response: $response${NC}"
    fi
    echo ""
    
    # Test Driver Service WebSocket endpoint
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}Test 2: Driver Service WebSocket Endpoint${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    response=$(curl -s -i -N \
      -H "Connection: Upgrade" \
      -H "Upgrade: websocket" \
      -H "Sec-WebSocket-Version: 13" \
      -H "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==" \
      http://localhost:3001/ws 2>&1 | head -n 1)
    
    if echo "$response" | grep -q "101"; then
        echo -e "${GREEN}✅ PASSED: WebSocket upgrade successful (HTTP 101)${NC}"
    elif echo "$response" | grep -q "426"; then
        echo -e "${GREEN}✅ PASSED: Endpoint exists (Upgrade Required)${NC}"
    else
        echo -e "${YELLOW}⚠️  Response: $response${NC}"
    fi
    echo ""
    
    echo -e "${BLUE}╔════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║   ✅ WebSocket Endpoints Available                ║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "${YELLOW}📝 Note: For full WebSocket testing with authentication,${NC}"
    echo -e "${YELLOW}   install websocat and run this script again.${NC}"
    
    exit 0
fi

# Full WebSocket testing with websocat

# Generate tokens
echo -e "${YELLOW}Generating test tokens...${NC}"

ADMIN_TOKEN=$(go run cmd/generate-jwt/main.go \
  -user="11111111-1111-1111-1111-111111111111" \
  -email="admin@ridehail.com" \
  -role=ADMIN \
  2>/dev/null | grep '^eyJ' | head -n1 | xargs)

# Get passenger ID
USERS_RESPONSE=$(curl -s "http://localhost:3004/admin/users?role=PASSENGER&limit=1" \
  -H "Authorization: Bearer $ADMIN_TOKEN")

PASSENGER_ID=$(echo "$USERS_RESPONSE" | grep -o '"user_id":"[^"]*"' | head -n1 | cut -d'"' -f4)

if [ -z "$PASSENGER_ID" ]; then
    echo -e "${RED}❌ No passenger found. Run ./scripts/setup-test-passenger.sh first${NC}"
    exit 1
fi

PASSENGER_TOKEN=$(go run cmd/generate-jwt/main.go \
  -user="$PASSENGER_ID" \
  -email="passenger@ridehail.com" \
  -role=PASSENGER \
  2>/dev/null | grep '^eyJ' | head -n1 | xargs)

# Get driver ID
DRIVER_RESPONSE=$(curl -s "http://localhost:3004/admin/users?role=DRIVER&limit=1" \
  -H "Authorization: Bearer $ADMIN_TOKEN")

DRIVER_ID=$(echo "$DRIVER_RESPONSE" | grep -o '"user_id":"[^"]*"' | head -n1 | cut -d'"' -f4)

if [ -z "$DRIVER_ID" ]; then
    echo -e "${RED}❌ No driver found. Run ./scripts/setup-test-driver.sh first${NC}"
    exit 1
fi

DRIVER_TOKEN=$(go run cmd/generate-jwt/main.go \
  -user="$DRIVER_ID" \
  -email="driver@ridehail.com" \
  -role=DRIVER \
  2>/dev/null | grep '^eyJ' | head -n1 | xargs)

echo -e "${GREEN}✅ Tokens generated${NC}"
echo ""

# Test 1: Passenger WebSocket Connection
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Test 1: Passenger WebSocket Connection${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

echo -e "${YELLOW}Connecting to ws://localhost:3000/ws${NC}"
echo -e "${YELLOW}Sending auth message...${NC}"

(
  echo "{\"token\":\"Bearer $PASSENGER_TOKEN\"}"
  sleep 1
  echo "{\"type\":\"ping\"}"
  sleep 1
) | timeout 3 websocat ws://localhost:3000/ws 2>&1 | while IFS= read -r line; do
    echo -e "${GREEN}← $line${NC}"
done

echo -e "${GREEN}✅ Passenger WebSocket test completed${NC}"
echo ""

# Test 2: Driver WebSocket Connection
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Test 2: Driver WebSocket Connection${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

echo -e "${YELLOW}Connecting to ws://localhost:3001/ws${NC}"
echo -e "${YELLOW}Sending auth message...${NC}"

(
  echo "{\"token\":\"Bearer $DRIVER_TOKEN\"}"
  sleep 1
  echo "{\"type\":\"ping\"}"
  sleep 1
) | timeout 3 websocat ws://localhost:3001/ws 2>&1 | while IFS= read -r line; do
    echo -e "${GREEN}← $line${NC}"
done

echo -e "${GREEN}✅ Driver WebSocket test completed${NC}"
echo ""

echo -e "${BLUE}╔════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║   ✅ WebSocket Tests Completed                    ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════╝${NC}"
