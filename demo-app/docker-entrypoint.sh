#!/bin/sh

# Set default values for environment variables
export API_SERVICE_NAME=${API_SERVICE_NAME:-"remote-config-system-api"}
export API_SERVICE_PORT=${API_SERVICE_PORT:-"8080"}

echo "🔧 Configuring nginx with:"
echo "   API_SERVICE_NAME: $API_SERVICE_NAME"
echo "   API_SERVICE_PORT: $API_SERVICE_PORT"

# Substitute environment variables in nginx template
envsubst '${API_SERVICE_NAME} ${API_SERVICE_PORT}' < /etc/nginx/conf.d/default.conf.template > /etc/nginx/conf.d/default.conf

echo "✅ Nginx configuration generated"

# Start nginx
exec nginx -g "daemon off;"
