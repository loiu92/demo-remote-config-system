.PHONY: build up down logs clean test

# Build and start all services
up:
	docker-compose up --build -d

# Stop all services
down:
	docker-compose down

# View logs
logs:
	docker-compose logs -f

# View logs for specific service
logs-api:
	docker-compose logs -f api

logs-db:
	docker-compose logs -f postgres

logs-redis:
	docker-compose logs -f redis

# Clean up everything
clean:
	docker-compose down -v
	docker system prune -f

# Run tests
test:
	docker-compose exec api go test ./...

# Tidy go modules
tidy:
	docker run --rm -v $(PWD):/app -w /app golang:1.21-alpine go mod tidy

# Build the application
build:
	docker-compose build

# Restart specific service
restart-api:
	docker-compose restart api

restart-demo:
	docker-compose restart demo-app

# Database operations
db-migrate:
	docker-compose exec postgres psql -U postgres -d remote_config -f /docker-entrypoint-initdb.d/001_initial.sql

# Development helpers
dev-up:
	docker-compose up postgres redis -d

dev-down:
	docker-compose down
