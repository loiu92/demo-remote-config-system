#!/bin/bash

# Create demo data for ShopFlow Lite

set -e

API_BASE="http://localhost:8080/admin"

echo "🎯 Creating demo data for ShopFlow Lite..."

# Function to make API calls with error handling
api_call() {
    local method=$1
    local endpoint=$2
    local data=$3
    local ignore_errors=$4

    echo "📡 $method $endpoint"

    if [ -n "$data" ]; then
        response=$(curl -s -X "$method" \
             -H "Content-Type: application/json" \
             -d "$data" \
             "$API_BASE$endpoint")
    else
        response=$(curl -s -X "$method" \
             "$API_BASE$endpoint")
    fi

    # Check if response contains error and ignore_errors is not set
    if [[ "$response" == *"error"* ]] && [[ "$ignore_errors" != "true" ]]; then
        echo "$response" | jq '.' 2>/dev/null || echo "$response"
        if [[ "$response" == *"already exists"* ]]; then
            echo "ℹ️  Resource already exists, continuing..."
        else
            echo "❌ Error occurred, but continuing..."
        fi
    else
        echo "$response" | jq '.' 2>/dev/null || echo "$response"
    fi

    echo ""
}

# Wait for server to be ready
echo "⏳ Waiting for server to be ready..."
until curl -s http://localhost:8080/health > /dev/null; do
    echo "Waiting for server..."
    sleep 2
done
echo "✅ Server is ready!"

# Create demo organization
echo "🏢 Creating demo organization..."
api_call "POST" "/orgs" '{
    "name": "Demo Organization",
    "slug": "demo",
    "description": "Demo organization for ShopFlow Lite"
}' "true"

# Create ShopFlow application
echo "🛍️ Creating ShopFlow application..."
api_call "POST" "/orgs/demo/apps" '{
    "name": "ShopFlow Lite",
    "slug": "shopflow",
    "description": "E-commerce demo application"
}' "true"

# Create production environment
echo "🌍 Creating production environment..."
api_call "POST" "/orgs/demo/apps/shopflow/envs" '{
    "name": "Production",
    "slug": "production",
    "description": "Production environment for ShopFlow Lite"
}' "true"

# Create initial configuration
echo "⚙️ Creating initial configuration..."

# Calculate promotion end time (24 hours from now)
# Use different date commands for macOS vs Linux
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    PROMOTION_END=$(date -v+24H -u +"%Y-%m-%dT%H:%M:%SZ")
else
    # Linux
    PROMOTION_END=$(date -d "+24 hours" -Iseconds)
fi

api_call "PUT" "/orgs/demo/apps/shopflow/envs/production/config" '{
    "config": {
        "theme": "light",
        "showPrices": true,
        "showRatings": true,
        "enablePromotions": true,
        "promotionEndTime": "'$PROMOTION_END'",
        "promotionTitle": "Black Friday Sale - 24 Hours Only!",
        "bannerMessage": "🎉 Welcome to ShopFlow Lite! Experience real-time configuration updates!",
        "showBanner": true,
        "maxItemsPerRow": 2,
        "features": {
            "newUserDiscount": true,
            "freeShipping": true,
            "loyaltyProgram": false
        },
        "ui": {
            "primaryColor": "#3b82f6",
            "secondaryColor": "#64748b",
            "borderRadius": "8px"
        }
    }
}'

echo ""
echo "✅ Demo data created successfully!"
echo ""
echo "🎮 Try these demo scenarios:"
echo ""
echo "1. 🎨 Change theme to dark:"
echo "   curl -X PUT $API_BASE/orgs/demo/apps/shopflow/envs/production/config \\"
echo "        -H \"Content-Type: application/json\" \\"
echo "        -d '{\"config\": {\"theme\": \"dark\"}}'"
echo ""
echo "2. 🌈 Change theme to colorful:"
echo "   curl -X PUT $API_BASE/orgs/demo/apps/shopflow/envs/production/config \\"
echo "        -H \"Content-Type: application/json\" \\"
echo "        -d '{\"config\": {\"theme\": \"colorful\"}}'"
echo ""
echo "3. 💰 Hide prices:"
echo "   curl -X PUT $API_BASE/orgs/demo/apps/shopflow/envs/production/config \\"
echo "        -H \"Content-Type: application/json\" \\"
echo "        -d '{\"config\": {\"showPrices\": false}}'"
echo ""
echo "4. 📱 Change layout to single column:"
echo "   curl -X PUT $API_BASE/orgs/demo/apps/shopflow/envs/production/config \\"
echo "        -H \"Content-Type: application/json\" \\"
echo "        -d '{\"config\": {\"maxItemsPerRow\": 1}}'"
echo ""
echo "5. 🎯 Start flash sale (1 hour countdown):"
if [[ "$OSTYPE" == "darwin"* ]]; then
    FLASH_SALE_END=$(date -v+1H -u +"%Y-%m-%dT%H:%M:%SZ")
else
    FLASH_SALE_END=$(date -d "+1 hour" -Iseconds)
fi
echo "   curl -X PUT $API_BASE/orgs/demo/apps/shopflow/envs/production/config \\"
echo "        -H \"Content-Type: application/json\" \\"
echo "        -d '{\"config\": {\"promotionEndTime\": \"$FLASH_SALE_END\", \"promotionTitle\": \"⚡ Flash Sale - 1 Hour Only!\"}}'"
echo ""
echo "🌐 Development access:"
echo "  React Demo:    http://localhost:3000"
echo "  Dashboard:     http://localhost:8080/dashboard"
echo "  SSE Demo:      http://localhost:8080/demo/sse"
echo ""
echo "🌐 Production access (if using make up):"
echo "  Main Demo:     http://localhost/demo"
echo "  Dashboard:     http://localhost/dashboard"
echo "  SSE Demo:      http://localhost/demo/sse"
