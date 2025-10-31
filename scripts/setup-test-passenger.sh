#!/bin/bash

# Создание тестового пассажира для Ride Service

set -e

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

ADMIN_SERVICE_URL=${ADMIN_SERVICE_URL:-"http://localhost:3004"}
PASSENGER_ID=${PASSENGER_ID:-"22222222-2222-2222-2222-222222222222"}
PASSENGER_EMAIL=${PASSENGER_EMAIL:-"passenger@ridehail.com"}
PASSENGER_NAME=${PASSENGER_NAME:-"Test Passenger"}
PASSENGER_PHONE=${PASSENGER_PHONE:-"+77771234567"}

echo -e "${BLUE}╔════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║   👤 Creating Test Passenger                      ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════╝${NC}"
echo ""

# Генерация admin токена
echo -e "${YELLOW}Generating admin token...${NC}"
ADMIN_TOKEN=$(go run cmd/generate-jwt/main.go \
  -user="11111111-1111-1111-1111-111111111111" \
  -email="admin@ridehail.com" \
  -role=ADMIN \
  2>/dev/null | grep '^eyJ' | head -n1 | xargs)

if [ -z "$ADMIN_TOKEN" ]; then
    echo -e "${RED}❌ Failed to generate admin token${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Admin token generated${NC}"
echo ""

AUTH_HEADER="Authorization: Bearer $ADMIN_TOKEN"

# Создание пассажира
echo -e "${BLUE}Creating passenger...${NC}"
echo -e "  Email: ${YELLOW}$PASSENGER_EMAIL${NC}"
echo -e "  Name: ${YELLOW}$PASSENGER_NAME${NC}"
echo -e "  Phone: ${YELLOW}$PASSENGER_PHONE${NC}"
echo ""

response=$(curl -s -X POST "$ADMIN_SERVICE_URL/admin/users" \
  -H "Content-Type: application/json" \
  -H "$AUTH_HEADER" \
  -d "{
    \"email\": \"$PASSENGER_EMAIL\",
    \"password\": \"password123\",
    \"role\": \"PASSENGER\",
    \"attrs\": {
      \"name\": \"$PASSENGER_NAME\",
      \"phone_number\": \"$PASSENGER_PHONE\"
    }
  }")

if echo "$response" | grep -q '"UserID"\|"user_id"'; then
    echo -e "${GREEN}✅ Passenger created successfully${NC}"
    if command -v jq &> /dev/null; then
        echo "$response" | jq '.'
    else
        echo "$response"
    fi
else
    if echo "$response" | grep -qi "duplicate\|exists"; then
        echo -e "${YELLOW}⚠️  Passenger already exists${NC}"
    else
        echo -e "${RED}❌ Failed to create passenger${NC}"
        echo "$response"
        exit 1
    fi
fi
echo ""

echo -e "${BLUE}╔════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║   ✅ Test Passenger Setup Completed               ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════╝${NC}"
