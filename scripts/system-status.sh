#!/bin/bash

# Скрипт для проверки состояния всех сервисов

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}╔═══════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║   🚗 Ride Hail System Health Check              ║${NC}"
echo -e "${BLUE}╚═══════════════════════════════════════════════════╝${NC}"
echo ""

check_service() {
    local name=$1
    local url=$2
    
    echo -n "Checking $name... "
    response=$(curl -s -o /dev/null -w "%{http_code}" "$url" 2>/dev/null)
    
    if [ "$response" = "200" ]; then
        echo -e "${GREEN}✓ OK${NC}"
        return 0
    else
        echo -e "${RED}✗ FAIL (HTTP $response)${NC}"
        return 1
    fi
}

echo -e "${BLUE}📡 Services Status:${NC}"
echo ""

check_service "Ride Service    " "http://localhost:3000/health"
check_service "Driver Service  " "http://localhost:3001/health"
check_service "Admin Service   " "http://localhost:3004/health"
check_service "RabbitMQ        " "http://localhost:15672"

echo ""
echo -e "${BLUE}🐳 Docker Containers:${NC}"
echo ""
docker-compose -f deployments/docker-compose.yml ps

echo ""
echo -e "${BLUE}📊 Database Status:${NC}"
echo ""
if docker exec ridehail-postgres pg_isready -U ridehail_user -d ridehail_db &>/dev/null; then
    echo -e "${GREEN}✓ PostgreSQL is ready${NC}"
else
    echo -e "${RED}✗ PostgreSQL is not ready${NC}"
fi

echo ""
echo -e "${BLUE}🐰 RabbitMQ Status:${NC}"
echo ""
if docker exec ridehail-rabbitmq rabbitmq-diagnostics check_port_connectivity &>/dev/null; then
    echo -e "${GREEN}✓ RabbitMQ is ready${NC}"
    echo -e "  Management UI: ${YELLOW}http://localhost:15672${NC} (guest/guest)"
else
    echo -e "${RED}✗ RabbitMQ is not ready${NC}"
fi

echo ""
echo -e "${BLUE}🔗 Quick Links:${NC}"
echo -e "  Ride Service:   ${YELLOW}http://localhost:3000${NC}"
echo -e "  Driver Service: ${YELLOW}http://localhost:3001${NC}"
echo -e "  Admin Service:  ${YELLOW}http://localhost:3004${NC}"
echo -e "  RabbitMQ UI:    ${YELLOW}http://localhost:15672${NC}"

echo ""
echo -e "${BLUE}🧪 Testing Commands:${NC}"
echo -e "  1. Setup driver:  ${YELLOW}./scripts/setup-test-driver.sh${NC}"
echo -e "  2. Run tests:     ${YELLOW}./scripts/test-driver-api.sh${NC}"
echo -e "  3. Full workflow: ${YELLOW}./scripts/test-driver-workflow.sh${NC}"

echo ""
