#!/bin/bash

# Development Environment Setup Script
# This script sets up the development environment with Go toolchain for testing

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to show usage
show_usage() {
    cat << EOF
Usage: $0 [COMMAND]

Development environment setup and management

COMMANDS:
    setup       Set up development environment
    start       Start development environment
    stop        Stop development environment
    test        Run tests in development environment
    shell       Open shell in development container
    logs        Show development logs
    clean       Clean up development environment
    help        Show this help message

EXAMPLES:
    $0 setup     # Initial setup
    $0 start     # Start dev environment
    $0 test      # Run all tests
    $0 shell     # Open development shell

EOF
}

# Function to check prerequisites
check_prerequisites() {
    print_status "Checking prerequisites..."
    
    # Check if Docker is installed
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed or not in PATH"
        print_error "Please install Docker to continue"
        exit 1
    fi
    
    # Check if Docker Compose is installed
    if ! command -v docker-compose &> /dev/null; then
        print_error "Docker Compose is not installed or not in PATH"
        print_error "Please install Docker Compose to continue"
        exit 1
    fi
    
    # Check if Docker is running
    if ! docker info &> /dev/null; then
        print_error "Docker is not running"
        print_error "Please start Docker and try again"
        exit 1
    fi
    
    print_success "Prerequisites check passed"
}

# Function to setup development environment
setup_dev_env() {
    print_status "Setting up development environment..."
    
    # Build development images
    print_status "Building development Docker images..."
    docker-compose -f docker-compose.dev.yml build
    
    # Create necessary directories
    print_status "Creating development directories..."
    mkdir -p tmp coverage
    
    # Set up environment files
    if [[ ! -f ".env.dev" ]]; then
        print_status "Creating development environment file..."
        cat > .env.dev << EOF
# Development Environment Configuration
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=remote_config
DB_SSL_MODE=disable

REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

PORT=8080
GIN_MODE=debug

# Cache Configuration
CACHE_TTL=300
CACHE_SHORT_TTL=60
CACHE_LONG_TTL=3600
CACHE_ENABLE_COMPRESSION=false

# Development settings
LOG_LEVEL=debug
ENABLE_PROFILING=true
EOF
        print_success "Created .env.dev file"
    fi
    
    print_success "Development environment setup complete!"
}

# Function to start development environment
start_dev_env() {
    print_status "Starting development environment..."
    
    # Start services
    docker-compose -f docker-compose.dev.yml up -d
    
    # Wait for services to be healthy
    print_status "Waiting for services to be ready..."
    sleep 10
    
    # Check service health
    if docker-compose -f docker-compose.dev.yml ps | grep -q "Up"; then
        print_success "Development environment started successfully!"
        print_status "Services available:"
        print_status "  - API: http://localhost:8080"
        print_status "  - PostgreSQL: localhost:5432"
        print_status "  - Redis: localhost:6379"
        print_status ""
        print_status "To view logs: make dev-logs"
        print_status "To run tests: make dev-test"
        print_status "To open shell: make dev-shell"
    else
        print_error "Failed to start development environment"
        docker-compose -f docker-compose.dev.yml logs
        exit 1
    fi
}

# Function to stop development environment
stop_dev_env() {
    print_status "Stopping development environment..."
    docker-compose -f docker-compose.dev.yml down
    print_success "Development environment stopped"
}

# Function to run tests
run_tests() {
    print_status "Running tests in development environment..."
    
    # Check if development environment is running
    if ! docker-compose -f docker-compose.dev.yml ps | grep -q "Up"; then
        print_warning "Development environment is not running. Starting it now..."
        start_dev_env
    fi
    
    # Run tests
    print_status "Executing test suite..."
    docker-compose -f docker-compose.dev.yml exec api go test -v ./...
}

# Function to open development shell
open_shell() {
    print_status "Opening development shell..."
    
    # Check if development environment is running
    if ! docker-compose -f docker-compose.dev.yml ps | grep -q "Up"; then
        print_warning "Development environment is not running. Starting it now..."
        start_dev_env
    fi
    
    # Open shell
    docker-compose -f docker-compose.dev.yml exec api sh
}

# Function to show logs
show_logs() {
    print_status "Showing development logs..."
    docker-compose -f docker-compose.dev.yml logs -f
}

# Function to clean up
clean_dev_env() {
    print_status "Cleaning up development environment..."
    docker-compose -f docker-compose.dev.yml down -v
    docker system prune -f
    print_success "Development environment cleaned up"
}

# Main execution
main() {
    case "${1:-help}" in
        setup)
            check_prerequisites
            setup_dev_env
            ;;
        start)
            check_prerequisites
            start_dev_env
            ;;
        stop)
            stop_dev_env
            ;;
        test)
            check_prerequisites
            run_tests
            ;;
        shell)
            check_prerequisites
            open_shell
            ;;
        logs)
            show_logs
            ;;
        clean)
            clean_dev_env
            ;;
        help|--help|-h)
            show_usage
            ;;
        *)
            print_error "Unknown command: $1"
            show_usage
            exit 1
            ;;
    esac
}

# Run main function
main "$@"
