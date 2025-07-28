#!/bin/sh

# Set default values for environment variables
export API_SERVICE_NAME=${API_SERVICE_NAME:-"remote-config-system-api"}
export API_SERVICE_PORT=${API_SERVICE_PORT:-"8080"}

echo "🔧 Configuring nginx with:"
echo "   API_SERVICE_NAME: $API_SERVICE_NAME"
echo "   API_SERVICE_PORT: $API_SERVICE_PORT"

# Create writable config directory
mkdir -p /tmp/nginx/conf.d

# Substitute environment variables in nginx template
envsubst '${API_SERVICE_NAME} ${API_SERVICE_PORT}' < /etc/nginx/conf.d/default.conf.template > /tmp/nginx/conf.d/default.conf

echo "✅ Nginx configuration generated"

# Start nginx with custom config path
exec nginx -g "daemon off;" -c /etc/nginx/nginx.conf
