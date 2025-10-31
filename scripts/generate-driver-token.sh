#!/bin/bash

# Скрипт для генерации JWT токена для водителя

set -e

DRIVER_ID=${1:-"11111111-1111-1111-1111-111111111111"}
DRIVER_EMAIL=${2:-"driver@ridehail.com"}

echo "🔐 Generating JWT token for driver..."
echo "Driver ID: $DRIVER_ID"
echo "Email: $DRIVER_EMAIL"
echo ""

TOKEN=$(go run cmd/generate-jwt/main.go \
  -user="$DRIVER_ID" \
  -email="$DRIVER_EMAIL" \
  -role=DRIVER \
  2>/dev/null | grep '^eyJ' | head -n1 | xargs)

if [ -z "$TOKEN" ]; then
    echo "❌ Failed to generate token"
    exit 1
fi

echo "✅ Driver token:"
echo "$TOKEN"
echo ""
echo "📋 Copy this for API requests:"
echo "Authorization: Bearer $TOKEN"
echo ""
echo "💡 Export to use in tests:"
echo "export DRIVER_TOKEN=\"$TOKEN\""
echo "export DRIVER_ID=\"$DRIVER_ID\""
