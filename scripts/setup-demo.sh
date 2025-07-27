#!/bin/bash

# Setup demo - React app will be built by Docker

set -e

echo "🚀 Setting up ShopFlow Lite Demo..."

# Check if Docker is available
if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed. Please install Docker first."
    exit 1
fi

echo "✅ Docker is available"
echo "📦 React app will be built automatically by Docker during startup"
echo "🎯 No additional setup required - the demo-app Dockerfile handles the build process"

echo ""
echo "✅ Demo setup complete!"
echo ""
echo "📋 Next steps:"
echo "1. Start the services: make up (production) or make dev (development)"
echo "2. Set up demo data: ./scripts/create-demo-data.sh"
echo ""
echo "🌐 Development access points:"
echo "  React Demo:    http://localhost:3000"
echo "  Dashboard:     http://localhost:8080/dashboard"
echo "  SSE Demo:      http://localhost:8080/demo/sse"
echo "  API:           http://localhost:8080"
echo ""
echo "🌐 Production access points:"
echo "  Main Demo:     http://localhost/demo"
echo "  Dashboard:     http://localhost/dashboard"
echo "  SSE Demo:      http://localhost/demo/sse"
echo ""
echo "🎯 Demo features:"
echo "- Real-time configuration updates via Server-Sent Events"
echo "- Theme switching (light/dark/colorful)"
echo "- Feature toggles (prices, ratings, layout)"
echo "- Countdown timers for promotions"
echo "- Live configuration panel"
