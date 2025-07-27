.PHONY: build up down logs clean test dev dev-up dev-down dev-test

# Production commands
build:
	docker-compose -f docker-compose.prod.yml build

up:
	docker-compose -f docker-compose.prod.yml up --build -d

down:
	docker-compose -f docker-compose.prod.yml down

# Legacy/Simple production (single service)
build-simple:
	docker-compose build

up-simple:
	docker-compose up --build -d

down-simple:
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
	docker-compose -f docker-compose.prod.yml logs -f

logs-simple:
	docker-compose logs -f

# View logs for specific service (production)
logs-api:
	docker-compose -f docker-compose.prod.yml logs -f api

logs-demo:
	docker-compose -f docker-compose.prod.yml logs -f demo-app

logs-dashboard:
	docker-compose -f docker-compose.prod.yml logs -f dashboard

logs-nginx:
	docker-compose -f docker-compose.prod.yml logs -f nginx

logs-db:
	docker-compose -f docker-compose.prod.yml logs -f postgres

logs-redis:
	docker-compose -f docker-compose.prod.yml logs -f redis

# Clean up everything
clean:
	docker-compose -f docker-compose.prod.yml down -v
	docker-compose down -v
	docker system prune -f

# Testing commands
# Check if Go is installed locally, otherwise use Docker
GO_CMD := $(shell command -v go 2> /dev/null)
ifndef GO_CMD
    # Use golang:1.21 (debian-based) instead of alpine for CGO support (needed for -race)
    GO_CMD = docker run --rm -v $(PWD):/app -w /app golang:1.21 go
    GO_INFO = "Using Docker (golang:1.21 with CGO support)"
    RACE_FLAG = -race
else
    GO_INFO = "Using local Go: $(GO_CMD)"
    RACE_FLAG = -race
endif

# Show which Go is being used
go-info:
	@echo $(GO_INFO)

# Fast unit tests (no external Docker dependencies, but may use Docker for Go)
test-unit:
	$(GO_CMD) test -v $(RACE_FLAG) ./internal/handlers/... ./internal/middleware/... ./internal/sse/... ./internal/models/...
	$(GO_CMD) test -v $(RACE_FLAG) ./internal/cache/... -run "Unit"

# Unit tests without race detection (faster, works on any system)
test-unit-fast:
	$(GO_CMD) test -v ./internal/handlers/... ./internal/middleware/... ./internal/sse/... ./internal/models/...
	$(GO_CMD) test -v ./internal/cache/... -run "Unit"

# Unit tests with coverage
test-unit-coverage:
	$(GO_CMD) test -v $(RACE_FLAG) -coverprofile=coverage.out ./internal/handlers/... ./internal/middleware/... ./internal/sse/... ./internal/models/...
	$(GO_CMD) test -v $(RACE_FLAG) -coverprofile=coverage_cache.out ./internal/cache/... -run "Unit"
	echo "mode: atomic" > combined_coverage.out
	tail -n +2 coverage.out >> combined_coverage.out 2>/dev/null || true
	tail -n +2 coverage_cache.out >> combined_coverage.out 2>/dev/null || true
	$(GO_CMD) tool cover -html=combined_coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Integration tests (requires Docker services)
test-integration: dev-up
	@echo "Waiting for services to be ready..."
	@sleep 10
	$(GO_CMD) test -v $(RACE_FLAG) ./internal/db/... -run "TestOrganizationRepository_"
	$(GO_CMD) test -v $(RACE_FLAG) ./internal/cache/... -run "TestRedisCache_"
	$(GO_CMD) test -v $(RACE_FLAG) ./internal/integration/...

# Run specific test packages (unit tests)
test-services:
	$(GO_CMD) test -v $(RACE_FLAG) ./internal/services/...

test-handlers:
	$(GO_CMD) test -v $(RACE_FLAG) ./internal/handlers/...



test-cache-unit:
	$(GO_CMD) test -v $(RACE_FLAG) ./internal/cache/... -run "Unit"

test-middleware:
	$(GO_CMD) test -v $(RACE_FLAG) ./internal/middleware/...

test-sse:
	$(GO_CMD) test -v $(RACE_FLAG) ./internal/sse/...

test-models:
	$(GO_CMD) test -v $(RACE_FLAG) ./internal/models/...

# Development utilities
dev-shell:
	docker-compose -f docker-compose.dev.yml exec api sh

dev-logs:
	docker-compose -f docker-compose.dev.yml logs -f

dev-clean:
	docker-compose -f docker-compose.dev.yml down -v

# Main test commands
test: test-unit
	@echo "✅ All unit tests passed!"

test-all: test-unit test-integration
	@echo "✅ All tests (unit + integration) passed!"

# Benchmark tests
test-bench:
	$(GO_CMD) test -bench=. ./...

# Clean test cache
test-clean:
	$(GO_CMD) clean -testcache

# Go module management
tidy:
	$(GO_CMD) mod tidy

mod-download:
	$(GO_CMD) mod download

# Linting and code quality (requires Go tools)
lint:
	$(GO_CMD) vet ./...
ifndef GO_CMD
	docker run --rm -v $(PWD):/app -w /app golang:1.21 sh -c "go install honnef.co/go/tools/cmd/staticcheck@latest && staticcheck ./..."
else
	$(GO_CMD) install honnef.co/go/tools/cmd/staticcheck@latest
	staticcheck ./...
endif





# Restart specific service
restart-api:
	docker-compose restart api

restart-demo:
	docker-compose restart demo-app

# Database operations
db-migrate:
	docker-compose exec postgres psql -U postgres -d remote_config -f /docker-entrypoint-initdb.d/001_initial.sql

# Demo commands
demo-setup:
	@echo "Setting up demo application..."
	./scripts/setup-demo.sh

demo-data:
	@echo "Creating demo data..."
	./scripts/create-demo-data.sh

demo: demo-setup up demo-data
	@echo "🎉 Demo is ready!"
	@echo ""
	@echo "🌐 Access points:"
	@echo "  Main Demo:     http://localhost/demo"
	@echo "  Dashboard:     http://localhost/dashboard"
	@echo "  SSE Demo:      http://localhost/demo/sse"
	@echo ""
	@echo "🔧 Direct service access:"
	@echo "  Demo App:      http://localhost:3000"
	@echo "  Dashboard:     http://localhost:4000"
	@echo "  API:           http://localhost:8080 (internal)"