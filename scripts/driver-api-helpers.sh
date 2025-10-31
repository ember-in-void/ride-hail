#!/bin/bash

# Простой скрипт для быстрых curl запросов к Driver API

DRIVER_SERVICE_URL=${DRIVER_SERVICE_URL:-"http://localhost:3001"}
DRIVER_ID=${DRIVER_ID:-"11111111-1111-1111-1111-111111111111"}

# Генерация токена если нет
if [ -z "$DRIVER_TOKEN" ]; then
    echo "Generating driver token..."
    export DRIVER_TOKEN=$(go run cmd/generate-jwt/main.go \
      -user="$DRIVER_ID" \
      -email=driver@ridehail.com \
      -role=DRIVER \
      2>/dev/null | grep '^eyJ' | head -n1 | xargs)
fi

# Функции для быстрого тестирования

# Health check
health() {
    curl -s "$DRIVER_SERVICE_URL/health" | jq '.'
}

# Go online
online() {
    local lat=${1:-43.238949}
    local lng=${2:-76.889709}
    
    curl -s -X POST "$DRIVER_SERVICE_URL/drivers/$DRIVER_ID/online" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $DRIVER_TOKEN" \
      -d "{
        \"latitude\": $lat,
        \"longitude\": $lng
      }" | jq '.'
}

# Go offline
offline() {
    curl -s -X POST "$DRIVER_SERVICE_URL/drivers/$DRIVER_ID/offline" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $DRIVER_TOKEN" | jq '.'
}

# Update location
location() {
    local lat=${1:-43.240000}
    local lng=${2:-76.890000}
    local speed=${3:-45.0}
    local heading=${4:-180.0}
    
    curl -s -X POST "$DRIVER_SERVICE_URL/drivers/$DRIVER_ID/location" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $DRIVER_TOKEN" \
      -d "{
        \"latitude\": $lat,
        \"longitude\": $lng,
        \"speed_kmh\": $speed,
        \"heading_degrees\": $heading
      }" | jq '.'
}

# Start ride
start_ride() {
    local ride_id=$1
    local lat=${2:-43.241000}
    local lng=${3:-76.891000}
    
    if [ -z "$ride_id" ]; then
        echo "Usage: start_ride <ride_id> [lat] [lng]"
        return 1
    fi
    
    curl -s -X POST "$DRIVER_SERVICE_URL/drivers/$DRIVER_ID/start" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $DRIVER_TOKEN" \
      -d "{
        \"ride_id\": \"$ride_id\",
        \"latitude\": $lat,
        \"longitude\": $lng
      }" | jq '.'
}

# Complete ride
complete_ride() {
    local ride_id=$1
    local lat=${2:-43.222015}
    local lng=${3:-76.851511}
    local distance=${4:-5.2}
    local duration=${5:-15}
    
    if [ -z "$ride_id" ]; then
        echo "Usage: complete_ride <ride_id> [lat] [lng] [distance_km] [duration_min]"
        return 1
    fi
    
    curl -s -X POST "$DRIVER_SERVICE_URL/drivers/$DRIVER_ID/complete" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $DRIVER_TOKEN" \
      -d "{
        \"ride_id\": \"$ride_id\",
        \"final_latitude\": $lat,
        \"final_longitude\": $lng,
        \"actual_distance_km\": $distance,
        \"actual_duration_minutes\": $duration
      }" | jq '.'
}

# Показываем доступные команды
cat << EOF
Driver API Quick Test Functions
================================

Available commands:
  health                           - Check service health
  online [lat] [lng]              - Go online (default: Almaty)
  offline                         - Go offline
  location [lat] [lng] [spd] [hd] - Update location
  start_ride <ride_id> [lat] [lng] - Start a ride
  complete_ride <ride_id> [lat] [lng] [dist] [dur] - Complete ride

Environment:
  DRIVER_SERVICE_URL: $DRIVER_SERVICE_URL
  DRIVER_ID: $DRIVER_ID
  DRIVER_TOKEN: ${DRIVER_TOKEN:0:20}...

Examples:
  health
  online
  location 43.25 76.90 50 270
  offline

EOF
