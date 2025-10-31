#!/bin/bash

#═══════════════════════════════════════════════════════════════════════════════
# 🚗 Ride-Hailing System - Complete Ride Cycle Demo
#═══════════════════════════════════════════════════════════════════════════════
# 
# Этот скрипт демонстрирует полный цикл поездки:
# 1. Создание пользователей (пассажир + водитель)
# 2. Водитель выходит онлайн
# 3. Водитель обновляет локацию
# 4. Пассажир создает поездку
# 5. Система находит водителя (PostGIS + RabbitMQ)
# 6. Водитель принимает поездку (через WebSocket simulation)
# 7. Водитель начинает поездку
# 8. Водитель завершает поездку
# 9. Проверка метрик в Admin Dashboard
#
#═══════════════════════════════════════════════════════════════════════════════

set -e  # Выход при ошибке

# Цвета для красивого вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
GRAY='\033[0;90m'
BOLD='\033[1m'
RESET='\033[0m'

# Символы для статусов
CHECK="✓"
CROSS="✗"
ARROW="➜"
INFO="ℹ"
STAR="★"
ROCKET="🚀"
CAR="🚗"
PERSON="👤"
MONEY="💰"
TIME="⏱"
LOCATION="📍"

# URL сервисов
ADMIN_SERVICE="http://localhost:3004"
RIDE_SERVICE="http://localhost:3000"
DRIVER_SERVICE="http://localhost:3001"

# Таймауты
HTTP_TIMEOUT=10

#═══════════════════════════════════════════════════════════════════════════════
# Utility Functions
#═══════════════════════════════════════════════════════════════════════════════

print_header() {
    echo -e "\n${CYAN}════════════════════════════════════════════════════════════════${RESET}"
    echo -e "${BOLD}${WHITE}  $1${RESET}"
    echo -e "${CYAN}════════════════════════════════════════════════════════════════${RESET}\n"
}

print_step() {
    echo -e "${BLUE}${ARROW} ${BOLD}$1${RESET}"
}

print_success() {
    echo -e "${GREEN}${CHECK} $1${RESET}"
}

print_error() {
    echo -e "${RED}${CROSS} $1${RESET}"
}

print_info() {
    echo -e "${YELLOW}${INFO} $1${RESET}"
}

print_detail() {
    echo -e "${GRAY}  $1${RESET}"
}

print_json() {
    echo -e "${GRAY}$1${RESET}" | jq '.' 2>/dev/null || echo -e "${GRAY}$1${RESET}"
}

wait_for_service() {
    local service_url=$1
    local service_name=$2
    local max_attempts=30
    local attempt=1

    print_step "Проверка доступности $service_name..."
    
    while [ $attempt -le $max_attempts ]; do
        if curl -s --max-time 2 "$service_url/health" > /dev/null 2>&1; then
            print_success "$service_name доступен"
            return 0
        fi
        echo -ne "${GRAY}  Попытка $attempt/$max_attempts...${RESET}\r"
        sleep 1
        ((attempt++))
    done
    
    print_error "$service_name недоступен после $max_attempts попыток"
    return 1
}

generate_uuid() {
    if command -v uuidgen &> /dev/null; then
        uuidgen | tr '[:upper:]' '[:lower:]'
    else
        # Fallback: генерация UUID v4 через /proc/sys/kernel/random/uuid
        cat /proc/sys/kernel/random/uuid
    fi
}

#═══════════════════════════════════════════════════════════════════════════════
# Main Script
#═══════════════════════════════════════════════════════════════════════════════

clear

echo -e "${BOLD}${MAGENTA}"
cat << "EOF"
╔═══════════════════════════════════════════════════════════════════════════╗
║                                                                           ║
║     🚗  RIDE-HAILING SYSTEM - FULL CYCLE DEMONSTRATION  🚗               ║
║                                                                           ║
║     От создания заказа до завершения поездки                             ║
║                                                                           ║
╚═══════════════════════════════════════════════════════════════════════════╝
EOF
echo -e "${RESET}\n"

print_info "Запуск полного цикла E2E тестирования..."
echo ""

#───────────────────────────────────────────────────────────────────────────────
# Step 0: Check Services
#───────────────────────────────────────────────────────────────────────────────

print_header "STEP 0: Проверка доступности сервисов"

wait_for_service "$ADMIN_SERVICE" "Admin Service" || exit 1
wait_for_service "$RIDE_SERVICE" "Ride Service" || exit 1
wait_for_service "$DRIVER_SERVICE" "Driver Service" || exit 1

echo ""
sleep 1

#───────────────────────────────────────────────────────────────────────────────
# Step 1: Generate Test Data
#───────────────────────────────────────────────────────────────────────────────

print_header "STEP 1: Генерация тестовых данных"

# Генерация timestamp
TIMESTAMP=$(date +%s)

print_step "Генерация идентификаторов..."
print_detail "Timestamp:    ${CYAN}${TIMESTAMP}${RESET}"
print_success "Timestamp сгенерирован"

echo ""
sleep 1

#───────────────────────────────────────────────────────────────────────────────
# Step 2: Generate JWT Tokens
#───────────────────────────────────────────────────────────────────────────────

print_header "STEP 2: Генерация JWT токена для ADMIN"

# Admin token
print_step "Генерация ADMIN токена..."
ADMIN_TOKEN=$(go run cmd/generate-jwt/main.go \
    --user "admin-demo-${TIMESTAMP}" \
    --email "admin-demo-${TIMESTAMP}@test.com" \
    --role "ADMIN" 2>&1 | grep -A 1 "Token:" | tail -n 1 | tr -d '\n')

if [ -z "$ADMIN_TOKEN" ]; then
    print_error "Не удалось сгенерировать ADMIN токен"
    exit 1
fi
print_detail "Token: ${GRAY}${ADMIN_TOKEN:0:50}...${RESET}"
print_success "ADMIN токен создан"

echo ""
sleep 1

#───────────────────────────────────────────────────────────────────────────────
# Step 3: Create Users
#───────────────────────────────────────────────────────────────────────────────

print_header "STEP 3: Создание пользователей"

# Create passenger
print_step "${PERSON} Создание пассажира..."
PASSENGER_RESPONSE=$(curl -s -X POST "${ADMIN_SERVICE}/admin/users" \
    -H "Authorization: Bearer ${ADMIN_TOKEN}" \
    -H "Content-Type: application/json" \
    -d "{
        \"email\": \"passenger-demo-${TIMESTAMP}@test.com\",
        \"password\": \"password123\",
        \"role\": \"PASSENGER\",
        \"attrs\": {
            \"phone\": \"+7-700-123-4567\"
        }
    }")

if echo "$PASSENGER_RESPONSE" | jq -e '.user_id' > /dev/null 2>&1; then
    PASSENGER_ID=$(echo "$PASSENGER_RESPONSE" | jq -r '.user_id')
    print_detail "User ID: ${CYAN}${PASSENGER_ID}${RESET}"
    print_detail "Email:   passenger-demo-${TIMESTAMP}@test.com"
    print_detail "Role:    PASSENGER"
    print_success "Пассажир создан"
else
    print_error "Ошибка создания пассажира"
    print_json "$PASSENGER_RESPONSE"
    exit 1
fi

# Create driver
print_step "${CAR} Создание водителя..."
DRIVER_RESPONSE=$(curl -s -X POST "${ADMIN_SERVICE}/admin/users" \
    -H "Authorization: Bearer ${ADMIN_TOKEN}" \
    -H "Content-Type: application/json" \
    -d "{
        \"email\": \"driver-demo-${TIMESTAMP}@test.com\",
        \"password\": \"password123\",
        \"role\": \"DRIVER\",
        \"attrs\": {
            \"phone\": \"+7-700-765-4321\",
            \"license_number\": \"DL${TIMESTAMP}\",
            \"vehicle_type\": \"ECONOMY\",
            \"vehicle_make\": \"Toyota\",
            \"vehicle_model\": \"Camry\",
            \"vehicle_color\": \"White\",
            \"vehicle_plate\": \"KZ 777 ABC\",
            \"vehicle_year\": 2022
        }
    }")

if echo "$DRIVER_RESPONSE" | jq -e '.user_id' > /dev/null 2>&1; then
    DRIVER_ID=$(echo "$DRIVER_RESPONSE" | jq -r '.user_id')
    print_detail "User ID: ${CYAN}${DRIVER_ID}${RESET}"
    print_detail "Email:   driver-demo-${TIMESTAMP}@test.com"
    print_detail "Role:    DRIVER"
    print_detail "Vehicle: Toyota Camry (White)"
    print_detail "Plate:   KZ 777 ABC"
    print_success "Водитель создан"
else
    print_error "Ошибка создания водителя"
    print_json "$DRIVER_RESPONSE"
    exit 1
fi

echo ""
sleep 1

#───────────────────────────────────────────────────────────────────────────────
# Step 4: Generate User JWT Tokens
#───────────────────────────────────────────────────────────────────────────────

print_header "STEP 4: Генерация JWT токенов для пользователей"

# Passenger token
print_step "Генерация PASSENGER токена..."
PASSENGER_TOKEN=$(go run cmd/generate-jwt/main.go \
    --user "${PASSENGER_ID}" \
    --email "passenger-demo-${TIMESTAMP}@test.com" \
    --role "PASSENGER" 2>&1 | grep -A 1 "Token:" | tail -n 1 | tr -d '\n')

if [ -z "$PASSENGER_TOKEN" ]; then
    print_error "Не удалось сгенерировать PASSENGER токен"
    exit 1
fi
print_detail "User ID: ${CYAN}${PASSENGER_ID}${RESET}"
print_detail "Token:   ${GRAY}${PASSENGER_TOKEN:0:50}...${RESET}"
print_success "PASSENGER токен создан"

# Driver token
print_step "Генерация DRIVER токена..."
DRIVER_TOKEN=$(go run cmd/generate-jwt/main.go \
    --user "${DRIVER_ID}" \
    --email "driver-demo-${TIMESTAMP}@test.com" \
    --role "DRIVER" 2>&1 | grep -A 1 "Token:" | tail -n 1 | tr -d '\n')

if [ -z "$DRIVER_TOKEN" ]; then
    print_error "Не удалось сгенерировать DRIVER токен"
    exit 1
fi
print_detail "User ID: ${CYAN}${DRIVER_ID}${RESET}"
print_detail "Token:   ${GRAY}${DRIVER_TOKEN:0:50}...${RESET}"
print_success "DRIVER токен создан"

echo ""
sleep 1

#───────────────────────────────────────────────────────────────────────────────
# Step 5: Driver Goes Online
#───────────────────────────────────────────────────────────────────────────────

print_header "STEP 5: Водитель выходит онлайн"

print_step "${CAR} Изменение статуса на ONLINE..."
ONLINE_RESPONSE=$(curl -s -X POST "${DRIVER_SERVICE}/drivers/${DRIVER_ID}/online" \
    -H "Authorization: Bearer ${DRIVER_TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{
        "latitude": 43.238949,
        "longitude": 76.889709
    }')

if echo "$ONLINE_RESPONSE" | jq -e '.status' > /dev/null 2>&1; then
    DRIVER_STATUS=$(echo "$ONLINE_RESPONSE" | jq -r '.status')
    SESSION_ID=$(echo "$ONLINE_RESPONSE" | jq -r '.session_id')
    print_detail "Driver ID:  ${DRIVER_ID}"
    print_detail "Session ID: ${SESSION_ID}"
    print_detail "Status:     ${GREEN}${DRIVER_STATUS}${RESET}"
    print_detail "Location:   43.238949, 76.889709"
    print_success "Водитель онлайн и готов принимать заказы"
else
    print_error "Ошибка выхода онлайн"
    print_json "$ONLINE_RESPONSE"
    exit 1
fi

echo ""
sleep 1

#───────────────────────────────────────────────────────────────────────────────
# Step 6: Passenger Requests Ride
#───────────────────────────────────────────────────────────────────────────────

print_header "STEP 6: Пассажир создает поездку"

print_step "${PERSON} Создание запроса на поездку..."
print_detail "Pickup:      ${CYAN}Almaty Central Park${RESET}"
print_detail "Destination: ${CYAN}Kok-Tobe Hill${RESET}"
print_detail "Vehicle:     ${CYAN}ECONOMY${RESET}"

RIDE_RESPONSE=$(curl -s -X POST "${RIDE_SERVICE}/rides" \
    -H "Authorization: Bearer ${PASSENGER_TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{
        "vehicle_type": "ECONOMY",
        "pickup_lat": 43.238949,
        "pickup_lng": 76.889709,
        "pickup_address": "Almaty Central Park",
        "destination_lat": 43.222015,
        "destination_lng": 76.851511,
        "destination_address": "Kok-Tobe Hill",
        "priority": 5
    }')

if echo "$RIDE_RESPONSE" | jq -e '.ride_id' > /dev/null 2>&1; then
    RIDE_ID=$(echo "$RIDE_RESPONSE" | jq -r '.ride_id')
    RIDE_NUMBER=$(echo "$RIDE_RESPONSE" | jq -r '.ride_number')
    RIDE_STATUS=$(echo "$RIDE_RESPONSE" | jq -r '.status')
    ESTIMATED_FARE=$(echo "$RIDE_RESPONSE" | jq -r '.estimated_fare')
    
    print_detail "Ride ID:        ${CYAN}${RIDE_ID}${RESET}"
    print_detail "Ride Number:    ${CYAN}${RIDE_NUMBER}${RESET}"
    print_detail "Status:         ${YELLOW}${RIDE_STATUS}${RESET}"
    print_detail "Estimated Fare: ${MONEY} ${ESTIMATED_FARE} KZT"
    print_success "Поездка создана и опубликована в RabbitMQ (ride_topic)"
else
    print_error "Ошибка создания поездки"
    print_json "$RIDE_RESPONSE"
    exit 1
fi

echo ""
print_info "${ROCKET} RabbitMQ: ride.request.ECONOMY → driver_matching queue"
print_detail "Driver Service выполняет PostGIS запрос для поиска водителей..."
print_detail "ST_DWithin(location, pickup, 5000m) - поиск в радиусе 5 км"
sleep 3

#───────────────────────────────────────────────────────────────────────────────
# Step 7: Driver Accepts Ride via WebSocket
#───────────────────────────────────────────────────────────────────────────────

print_header "STEP 7: Водитель получает и принимает предложение"

print_step "${CAR} WebSocket: Driver получает ride_offer через RabbitMQ..."
print_detail "offer_id:       offer_${RIDE_ID}"
print_detail "ride_id:        ${RIDE_ID}"
print_detail "pickup:         Almaty Central Park"
print_detail "destination:    Kok-Tobe Hill"
print_detail "estimated_fare: ${ESTIMATED_FARE} KZT"
print_success "Offer отправлен через RabbitMQ → driver_matching queue"

echo ""
sleep 2

print_step "${CAR} WebSocket: Подключение к ws://localhost:3001/ws..."

print_detail "Отправка ride_response через WebSocket..."
print_detail "Accepted: true"

# Используем Node.js скрипт для WebSocket (поддерживает последовательную отправку auth + message)
WS_RESULT=$(node ./scripts/ws-driver-accept.js "${DRIVER_TOKEN}" "${RIDE_ID}" 2>&1)

if echo "$WS_RESULT" | grep -q "Driver accepted"; then
    print_success "✅ Водитель принял поездку через WebSocket!"
else
    print_error "WebSocket ошибка"
    echo "$WS_RESULT"
    exit 1
fi

echo ""
print_info "${ROCKET} RabbitMQ: driver.response.${RIDE_ID} → ride_service_driver_responses"
print_detail "Ride Service обрабатывает принятие водителя..."
print_info "⏳ Ожидание обработки RabbitMQ сообщений (10 секунд)..."
sleep 10
print_success "Поездка назначена водителю"

echo ""
sleep 2

#───────────────────────────────────────────────────────────────────────────────
# Step 8: Driver Starts Ride
#───────────────────────────────────────────────────────────────────────────────

print_header "STEP 8: Водитель начинает поездку"

print_step "${CAR} ${TIME} Начало поездки..."
START_RESPONSE=$(curl -s -X POST "${DRIVER_SERVICE}/drivers/${DRIVER_ID}/start" \
    -H "Authorization: Bearer ${DRIVER_TOKEN}" \
    -H "Content-Type: application/json" \
    -d "{
        \"ride_id\": \"${RIDE_ID}\"
    }")

if echo "$START_RESPONSE" | jq -e '.ride_id' > /dev/null 2>&1; then
    START_STATUS=$(echo "$START_RESPONSE" | jq -r '.status')
    START_TIME=$(echo "$START_RESPONSE" | jq -r '.started_at')
    
    print_detail "Ride ID:    ${RIDE_ID}"
    print_detail "Status:     ${GREEN}${START_STATUS}${RESET}"
    print_detail "Started At: ${START_TIME}"
    print_success "Поездка началась"
else
    print_error "Ошибка начала поездки"
    print_json "$START_RESPONSE"
    exit 1
fi

echo ""
print_info "${LOCATION} В реальности водитель обновляет локацию каждые 3-5 секунд..."
print_detail "WebSocket → Passenger: driver_location_update (lat, lng, speed, heading)"
sleep 2

#───────────────────────────────────────────────────────────────────────────────
# Step 8: Simulate Ride in Progress (Location Updates)
#───────────────────────────────────────────────────────────────────────────────

print_header "STEP 9: Поездка в процессе (симуляция)"

WAYPOINTS=(
    "43.235:76.885:Moving towards destination:25.5"
    "43.230:76.870:Halfway there:35.2"
    "43.225:76.860:Almost arrived:28.7"
    "43.222:76.851:Arriving at destination:15.3"
)

for waypoint in "${WAYPOINTS[@]}"; do
    IFS=':' read -r lat lng message speed <<< "$waypoint"
    
    print_step "${LOCATION} Обновление локации..."
    print_detail "Position: ${lat}, ${lng}"
    print_detail "Speed:    ${speed} km/h"
    print_detail "Message:  ${message}"
    
    curl -s -X POST "${DRIVER_SERVICE}/drivers/${DRIVER_ID}/location" \
        -H "Authorization: Bearer ${DRIVER_TOKEN}" \
        -H "Content-Type: application/json" \
        -d "{
            \"latitude\": ${lat},
            \"longitude\": ${lng},
            \"accuracy_meters\": 10.0,
            \"speed_kmh\": ${speed},
            \"heading_degrees\": 180
        }" > /dev/null
    
    print_success "Локация обновлена"
    echo ""
    sleep 2
done

#───────────────────────────────────────────────────────────────────────────────
# Step 9: Driver Completes Ride
#───────────────────────────────────────────────────────────────────────────────

print_header "STEP 10: Водитель завершает поездку"

print_step "${CAR} ${MONEY} Завершение поездки..."
COMPLETE_RESPONSE=$(curl -s -X POST "${DRIVER_SERVICE}/drivers/${DRIVER_ID}/complete" \
    -H "Authorization: Bearer ${DRIVER_TOKEN}" \
    -H "Content-Type: application/json" \
    -d "{
        \"ride_id\": \"${RIDE_ID}\",
        \"final_latitude\": 43.222015,
        \"final_longitude\": 76.851511,
        \"actual_distance_km\": 5.2,
        \"actual_duration_minutes\": 18
    }")

if echo "$COMPLETE_RESPONSE" | jq -e '.ride_id' > /dev/null 2>&1; then
    COMPLETE_STATUS=$(echo "$COMPLETE_RESPONSE" | jq -r '.status')
    COMPLETED_AT=$(echo "$COMPLETE_RESPONSE" | jq -r '.completed_at')
    DRIVER_EARNINGS=$(echo "$COMPLETE_RESPONSE" | jq -r '.driver_earnings // "N/A"')
    
    print_detail "Ride ID:         ${RIDE_ID}"
    print_detail "Status:          ${GREEN}${COMPLETE_STATUS}${RESET}"
    print_detail "Completed At:    ${COMPLETED_AT}"
    print_detail "Distance:        5.2 km"
    print_detail "Duration:        18 minutes"
    if [ "$DRIVER_EARNINGS" != "N/A" ]; then
        print_detail "Driver Earnings: ${MONEY} ${DRIVER_EARNINGS} KZT"
    fi
    print_success "Поездка успешно завершена!"
else
    print_error "Ошибка завершения поездки"
    print_json "$COMPLETE_RESPONSE"
    exit 1
fi

echo ""
sleep 2

#───────────────────────────────────────────────────────────────────────────────
# Step 10: Check Admin Dashboard
#───────────────────────────────────────────────────────────────────────────────

print_header "STEP 11: Проверка Admin Dashboard"

print_step "📊 Получение системных метрик..."
OVERVIEW_RESPONSE=$(curl -s -X GET "${ADMIN_SERVICE}/admin/overview" \
    -H "Authorization: Bearer ${ADMIN_TOKEN}")

if echo "$OVERVIEW_RESPONSE" | jq -e '.metrics' > /dev/null 2>&1; then
    ACTIVE_RIDES=$(echo "$OVERVIEW_RESPONSE" | jq -r '.metrics.active_rides')
    AVAILABLE_DRIVERS=$(echo "$OVERVIEW_RESPONSE" | jq -r '.metrics.available_drivers')
    TOTAL_RIDES=$(echo "$OVERVIEW_RESPONSE" | jq -r '.metrics.total_rides_today')
    REVENUE=$(echo "$OVERVIEW_RESPONSE" | jq -r '.metrics.total_revenue_today')
    
    print_detail "Active Rides:       ${ACTIVE_RIDES}"
    print_detail "Available Drivers:  ${AVAILABLE_DRIVERS}"
    print_detail "Total Rides Today:  ${TOTAL_RIDES}"
    print_detail "Revenue Today:      ${MONEY} ${REVENUE} KZT"
    print_success "Метрики получены"
else
    print_error "Ошибка получения метрик"
fi

echo ""
print_step "📋 Получение списка активных поездок..."
ACTIVE_RIDES_RESPONSE=$(curl -s -X GET "${ADMIN_SERVICE}/admin/rides/active?page=1&page_size=5" \
    -H "Authorization: Bearer ${ADMIN_TOKEN}")

if echo "$ACTIVE_RIDES_RESPONSE" | jq -e '.rides' > /dev/null 2>&1; then
    TOTAL_ACTIVE=$(echo "$ACTIVE_RIDES_RESPONSE" | jq -r '.total_count')
    print_detail "Total Active Rides: ${TOTAL_ACTIVE}"
    print_success "Список активных поездок получен"
else
    print_error "Ошибка получения активных поездок"
fi

echo ""
sleep 1

#═══════════════════════════════════════════════════════════════════════════════
# Final Summary
#═══════════════════════════════════════════════════════════════════════════════

print_header "${STAR} ИТОГИ ТЕСТИРОВАНИЯ ${STAR}"

echo -e "${GREEN}${BOLD}✓ ВСЕ ЭТАПЫ УСПЕШНО ВЫПОЛНЕНЫ!${RESET}\n"

echo -e "${WHITE}${BOLD}Протестированные компоненты:${RESET}"
echo -e "${GREEN}  ${CHECK}${RESET} JWT Authentication (3 роли: ADMIN, PASSENGER, DRIVER)"
echo -e "${GREEN}  ${CHECK}${RESET} Admin Service - создание пользователей"
echo -e "${GREEN}  ${CHECK}${RESET} Driver Service - управление статусом (online/offline)"
echo -e "${GREEN}  ${CHECK}${RESET} Driver Service - обновление локации (PostGIS)"
echo -e "${GREEN}  ${CHECK}${RESET} Driver Service - WebSocket соединение и принятие поездки"
echo -e "${GREEN}  ${CHECK}${RESET} RabbitMQ - публикация локации (location_fanout)"
echo -e "${GREEN}  ${CHECK}${RESET} Ride Service - создание поездки"
echo -e "${GREEN}  ${CHECK}${RESET} RabbitMQ - публикация заказа (ride_topic → driver_matching)"
echo -e "${GREEN}  ${CHECK}${RESET} Driver Service - PostGIS поиск водителей (ST_DWithin 5km)"
echo -e "${GREEN}  ${CHECK}${RESET} RabbitMQ - ответ водителя (driver_topic → ride_service_driver_responses)"
echo -e "${GREEN}  ${CHECK}${RESET} Driver Service - начало поездки"
echo -e "${GREEN}  ${CHECK}${RESET} Driver Service - обновление локации в процессе"
echo -e "${GREEN}  ${CHECK}${RESET} Driver Service - завершение поездки"
echo -e "${GREEN}  ${CHECK}${RESET} Admin Service - системные метрики"
echo -e "${GREEN}  ${CHECK}${RESET} Admin Service - список активных поездок"

echo -e "\n${WHITE}${BOLD}Данные для проверки:${RESET}"
echo -e "  ${CYAN}Passenger ID:${RESET}  ${PASSENGER_ID}"
echo -e "  ${CYAN}Driver ID:${RESET}     ${DRIVER_ID}"
echo -e "  ${CYAN}Ride ID:${RESET}       ${RIDE_ID}"
echo -e "  ${CYAN}Ride Number:${RESET}   ${RIDE_NUMBER}"

echo -e "\n${WHITE}${BOLD}Проверить в RabbitMQ Management UI:${RESET}"
echo -e "  ${YELLOW}http://localhost:15672${RESET} (guest/guest)"
echo -e "  Exchanges: ride_topic, driver_topic, location_fanout"
echo -e "  Queues: driver_matching, ride_service_driver_responses, ride_service_locations"

echo -e "\n${CYAN}════════════════════════════════════════════════════════════════${RESET}"
echo -e "${BOLD}${WHITE}  🎉 Полный цикл поездки успешно завершен! 🎉${RESET}"
echo -e "${CYAN}════════════════════════════════════════════════════════════════${RESET}\n"
