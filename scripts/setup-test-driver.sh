#!/bin/bash

# Скрипт для создания тестового водителя через Admin API

set -e

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

ADMIN_SERVICE_URL=${ADMIN_SERVICE_URL:-"http://localhost:3004"}
DRIVER_ID=${DRIVER_ID:-"11111111-1111-1111-1111-111111111111"}
DRIVER_EMAIL=${DRIVER_EMAIL:-"driver@ridehail.com"}
LICENSE_NUMBER=${LICENSE_NUMBER:-"DRV-$(date +%s)"}  # Уникальный номер на основе timestamp

echo -e "${BLUE}╔════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║   🚗 Setting up Test Driver                       ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════╝${NC}"
echo ""

# Генерация admin токена
echo -e "${YELLOW}⚠️  Generating admin token...${NC}"
ADMIN_TOKEN=$(go run cmd/generate-jwt/main.go \
  -user="00000000-0000-0000-0000-000000000001" \
  -email=admin@ridehail.com \
  -role=ADMIN \
  2>/dev/null | grep '^eyJ' | head -n1 | xargs)

if [ -z "$ADMIN_TOKEN" ]; then
    echo -e "${RED}❌ Failed to generate admin token${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Admin token generated${NC}"
echo ""

# Создание водителя
echo -e "${BLUE}📍 Step 1: Create Driver User${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
response=$(curl -s -X POST "$ADMIN_SERVICE_URL/admin/users" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d "{
    \"email\": \"$DRIVER_EMAIL\",
    \"password\": \"driver123\",
    \"role\": \"DRIVER\",
    \"attrs\": {
      \"license_number\": \"$LICENSE_NUMBER\",
      \"vehicle_type\": \"ECONOMY\",
      \"vehicle_make\": \"Toyota\",
      \"vehicle_model\": \"Camry\",
      \"vehicle_year\": 2020,
      \"vehicle_color\": \"White\",
      \"vehicle_plate\": \"ABC123\"
    }
  }")

if command -v jq &> /dev/null; then
    echo "$response" | jq '.'
else
    echo "$response"
fi

# Проверяем успешность создания
if echo "$response" | grep -qi '"user\?ID"'; then
    CREATED_ID=$(echo "$response" | grep -oiE '"user_?id":"[^"]*"' | cut -d'"' -f4)
    echo ""
    echo -e "${GREEN}✅ Driver created successfully!${NC}"
    echo -e "${YELLOW}Driver ID: $CREATED_ID${NC}"
    echo ""
    
    if [ "$CREATED_ID" != "$DRIVER_ID" ]; then
        echo -e "${YELLOW}⚠️  Note: Created driver has different ID than expected${NC}"
        echo -e "${YELLOW}   Expected: $DRIVER_ID${NC}"
        echo -e "${YELLOW}   Created:  $CREATED_ID${NC}"
        echo ""
        echo -e "${BLUE}💡 Update your tests to use:${NC}"
        echo -e "   export DRIVER_ID=\"$CREATED_ID\""
    fi
elif echo "$response" | grep -q "already exists\|duplicate"; then
    echo ""
    echo -e "${YELLOW}⚠️  Driver with email $DRIVER_EMAIL already exists${NC}"
    echo -e "${GREEN}✅ You can proceed with testing${NC}"
else
    echo ""
    echo -e "${RED}❌ Failed to create driver${NC}"
    exit 1
fi

echo ""
echo -e "${BLUE}╔════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║   ✅ Test Driver Setup Complete                   ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${BLUE}Next steps:${NC}"
echo -e "  1. Generate driver token: ${YELLOW}./scripts/generate-driver-token.sh${NC}"
echo -e "  2. Run API tests: ${YELLOW}./scripts/test-driver-api.sh${NC}"
echo -e "  3. Test workflow: ${YELLOW}./scripts/test-driver-workflow.sh${NC}"
