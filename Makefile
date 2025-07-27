.PHONY: build up down logs clean test dev dev-up dev-down dev-test

# Production commands
build:
	docker-compose build

up:
	docker-compose up --build -d

down:
	docker-compose down

# Development commands
dev-build:
	docker-compose -f docker-compose.dev.yml build

dev-up:
	docker-compose -f docker-compose.dev.yml up -d

dev-down:
	docker-compose -f docker-compose.dev.yml down

# Start development environment with hot reloading (foreground)
dev:
	docker-compose -f docker-compose.dev.yml up

# Stop development environment
dev-stop:
	docker-compose -f docker-compose.dev.yml down

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

# Development testing commands (with Go toolchain)
dev-test:
	docker-compose -f docker-compose.dev.yml exec api go test ./...

dev-test-verbose:
	docker-compose -f docker-compose.dev.yml exec api go test -v ./...

dev-test-coverage:
	docker-compose -f docker-compose.dev.yml exec api sh -c "go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out -o coverage.html"

dev-test-race:
	docker-compose -f docker-compose.dev.yml exec api go test -race ./...

dev-test-unit:
	docker-compose -f docker-compose.dev.yml exec api go test -short ./...

dev-test-integration:
	docker-compose -f docker-compose.dev.yml exec api go test -run Integration ./...

# Run specific test packages in development
dev-test-services:
	docker-compose -f docker-compose.dev.yml exec api go test ./internal/services/...

dev-test-handlers:
	docker-compose -f docker-compose.dev.yml exec api go test ./internal/handlers/...

dev-test-db:
	docker-compose -f docker-compose.dev.yml exec api go test ./internal/db/...

dev-test-cache:
	docker-compose -f docker-compose.dev.yml exec api go test ./internal/cache/...

dev-test-middleware:
	docker-compose -f docker-compose.dev.yml exec api go test ./internal/middleware/...

dev-test-sse:
	docker-compose -f docker-compose.dev.yml exec api go test ./internal/sse/...

# Run tests using dedicated test service
test-service:
	docker-compose -f docker-compose.dev.yml run --rm test

# Development utilities
dev-shell:
	docker-compose -f docker-compose.dev.yml exec api sh

dev-logs:
	docker-compose -f docker-compose.dev.yml logs -f

dev-clean:
	docker-compose -f docker-compose.dev.yml down -v

# Run all tests (requires dev environment to be running)
test: dev-test

# Run tests with verbose output
test-verbose:
	docker-compose -f docker-compose.dev.yml exec api go test -v ./...

# Run tests for specific package
test-package:
	@echo "Usage: make test-package PACKAGE=./internal/handlers"
	@if [ -z "$(PACKAGE)" ]; then echo "Please specify PACKAGE=<package_path>"; exit 1; fi
	docker-compose -f docker-compose.dev.yml exec api go test -v $(PACKAGE)

# Run unit tests only (no integration tests, no external dependencies)
test-unit:
	docker run --rm -v $(PWD):/app -w /app golang:1.22-alpine go test -short ./internal/models ./internal/middleware

# Setup and run all tests (one command)
test-all: dev-down dev-build dev-up
	@echo "Waiting for services to be ready..."
	@sleep 5
	$(MAKE) dev-test
	@echo "All tests completed!"

# Test with coverage
test-coverage:
	docker-compose -f docker-compose.dev.yml exec api go test -coverprofile=coverage.out ./...
	docker-compose -f docker-compose.dev.yml exec api go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run tests with race detection
test-race:
	docker-compose -f docker-compose.dev.yml exec api go test -race ./...

# Run integration tests only
test-integration:
	docker run --rm -v $(PWD):/app -w /app golang:1.21-alpine go test -run Integration ./...

# Run specific test package
test-services:
	docker run --rm -v $(PWD):/app -w /app golang:1.21-alpine go test ./internal/services/...

test-handlers:
	docker run --rm -v $(PWD):/app -w /app golang:1.21-alpine go test ./internal/handlers/...

test-db:
	docker run --rm -v $(PWD):/app -w /app golang:1.21-alpine go test ./internal/db/...

test-cache:
	docker run --rm -v $(PWD):/app -w /app golang:1.21-alpine go test ./internal/cache/...

test-middleware:
	docker run --rm -v $(PWD):/app -w /app golang:1.21-alpine go test ./internal/middleware/...

test-sse:
	docker run --rm -v $(PWD):/app -w /app golang:1.21-alpine go test ./internal/sse/...

# Run tests in running Docker Compose environment
test-docker-compose:
	docker-compose exec api go test ./...

# Run tests with coverage in running Docker Compose environment
test-docker-compose-coverage:
	docker-compose exec api go test -coverprofile=coverage.out ./...
	docker-compose exec api go tool cover -html=coverage.out -o coverage.html

# Benchmark tests
test-bench:
	docker run --rm -v $(PWD):/app -w /app golang:1.21-alpine go test -bench=. ./...

# Clean test cache
test-clean:
	docker run --rm -v $(PWD):/app -w /app golang:1.21-alpine go clean -testcache

# Run tests with local Go (if available)
test-local:
	go test ./...

test-local-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

test-local-verbose:
	go test -v ./...

# Tidy go modules
tidy:
	docker run --rm -v $(PWD):/app -w /app golang:1.21-alpine go mod tidy



# Restart specific service
restart-api:
	docker-compose restart api

restart-demo:
	docker-compose restart demo-app

# Database operations
db-migrate:
	docker-compose exec postgres psql -U postgres -d remote_config -f /docker-entrypoint-initdb.d/001_initial.sql


