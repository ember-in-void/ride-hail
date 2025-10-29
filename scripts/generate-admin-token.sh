#!/bin/bash

# Быстрая генерация токена для дефолтного админа

set -e

echo "🔐 Generating JWT token for default admin..."
echo ""

TOKEN=$(go run cmd/generate-jwt/main.go \
  -user=admin-001 \
  -email=admin@ridehail.com \
  -role=ADMIN \
  2>/dev/null | grep '^eyJ' | head -n1 | xargs)

if [ -z "$TOKEN" ]; then
    echo "❌ Failed to generate token"
    exit 1
fi

echo "✅ Admin token:"
echo "$TOKEN"
echo ""
echo "📋 Copy this for API requests:"
echo "Authorization: Bearer $TOKEN"
echo ""
echo "💡 Example usage:"
echo "export ADMIN_TOKEN=\"$TOKEN\""
echo ""
echo "curl -X POST http://localhost:3004/admin/users \\"
echo "  -H \"Authorization: Bearer \$ADMIN_TOKEN\" \\"
echo "  -H \"Content-Type: application/json\" \\"
echo "  -d '{\"email\":\"user@example.com\",\"password\":\"Pass123\",\"role\":\"PASSENGER\"}'"