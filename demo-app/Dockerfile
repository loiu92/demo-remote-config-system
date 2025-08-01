# React Demo App Dockerfile
# Build stage
FROM node:18-alpine AS builder

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install all dependencies (including devDependencies for build)
RUN npm install

# Copy source code
COPY . .

# Build the application
RUN npm run build

# Production stage
FROM nginx:alpine

# Install wget for health checks
RUN apk --no-cache add wget

# Copy built app from builder stage
COPY --from=builder /app/dist /usr/share/nginx/html

# Copy nginx configurations
COPY nginx.conf.template /etc/nginx/conf.d/default.conf.template
COPY nginx-main.conf /etc/nginx/nginx.conf
COPY docker-entrypoint.sh /docker-entrypoint.sh

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Change ownership of nginx directories and create temp directories
RUN chown -R appuser:appgroup /var/cache/nginx /var/run /var/log/nginx /usr/share/nginx/html && \
    mkdir -p /tmp/client_temp /tmp/proxy_temp_path /tmp/fastcgi_temp /tmp/uwsgi_temp /tmp/scgi_temp /tmp/nginx/conf.d && \
    chown -R appuser:appgroup /tmp/client_temp /tmp/proxy_temp_path /tmp/fastcgi_temp /tmp/uwsgi_temp /tmp/scgi_temp /tmp/nginx && \
    chmod +x /docker-entrypoint.sh

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 3000

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:3000/ || exit 1

# Start with custom entrypoint
CMD ["/docker-entrypoint.sh"]
