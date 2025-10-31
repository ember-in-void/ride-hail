#!/bin/bash

#═══════════════════════════════════════════════════════════════════════════════
# WebSocket клиент для принятия поездки водителем
#═══════════════════════════════════════════════════════════════════════════════

set -e

DRIVER_ID=$1
DRIVER_TOKEN=$2
RIDE_ID=$3

if [ -z "$DRIVER_ID" ] || [ -z "$DRIVER_TOKEN" ] || [ -z "$RIDE_ID" ]; then
    echo "Usage: $0 <driver_id> <driver_token> <ride_id>"
    exit 1
fi

echo "🚗 Connecting to Driver WebSocket..."
echo "Driver ID: $DRIVER_ID"
echo "Ride ID: $RIDE_ID"
echo ""

# Создаем временный файл с командами WebSocket
TEMP_FILE=$(mktemp)
cat > "$TEMP_FILE" << EOF
{
  "type": "ride_response",
  "data": {
    "ride_id": "$RIDE_ID",
    "accepted": true,
    "current_location": {
      "latitude": 43.238949,
      "longitude": 76.889709
    }
  }
}
EOF

echo "📤 Отправка ride_response..."
cat "$TEMP_FILE"
echo ""

# Подключаемся к WebSocket и отправляем сообщение
# Используем timeout чтобы автоматически закрыть соединение после отправки
timeout 3 wscat -c "ws://localhost:3001/ws?token=${DRIVER_TOKEN}" \
    --exec "cat $TEMP_FILE" 2>/dev/null || true

rm -f "$TEMP_FILE"

echo ""
echo "✅ Сообщение отправлено!"
