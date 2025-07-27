#!/bin/bash

# Remote Config System - Test Runner Script
# This script provides an easy way to run tests with proper setup

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

# Function to check if development environment is running
check_dev_env() {
    if ! docker-compose -f docker-compose.dev.yml ps | grep -q "Up"; then
        return 1
    fi
    return 0
}

# Function to wait for services to be ready
wait_for_services() {
    print_status "Waiting for services to be ready..."
    local max_attempts=30
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if curl -s http://localhost:8080/health > /dev/null 2>&1; then
            print_success "Services are ready!"
            return 0
        fi
        
        print_status "Attempt $attempt/$max_attempts - waiting for services..."
        sleep 2
        attempt=$((attempt + 1))
    done
    
    print_error "Services failed to start within expected time"
    return 1
}

# Main function
main() {
    local command=${1:-"all"}
    
    case $command in
        "setup")
            print_status "Setting up development environment..."
            make dev-down || true
            make dev-build
            make dev-up
            wait_for_services
            print_success "Development environment is ready!"
            ;;
            
        "quick")
            print_status "Running quick tests (assumes dev environment is running)..."
            if ! check_dev_env; then
                print_error "Development environment is not running. Run: $0 setup"
                exit 1
            fi
            make dev-test
            ;;
            
        "all")
            print_status "Running full test suite..."
            make dev-down || true
            make dev-build
            make dev-up
            wait_for_services
            make dev-test
            print_success "All tests completed successfully!"
            ;;
            
        "unit")
            print_status "Running unit tests only..."
            make test-unit
            ;;
            
        "coverage")
            print_status "Running tests with coverage..."
            if ! check_dev_env; then
                print_error "Development environment is not running. Run: $0 setup"
                exit 1
            fi
            make test-coverage
            print_success "Coverage report generated!"
            ;;
            
        "clean")
            print_status "Cleaning up test environment..."
            make dev-down
            docker system prune -f
            print_success "Environment cleaned!"
            ;;
            
        "help"|*)
            echo "Remote Config System - Test Runner"
            echo ""
            echo "Usage: $0 [command]"
            echo ""
            echo "Commands:"
            echo "  setup     - Set up development environment"
            echo "  quick     - Run tests (assumes dev env is running)"
            echo "  all       - Full test suite with environment setup (default)"
            echo "  unit      - Run unit tests only (no external dependencies)"
            echo "  coverage  - Run tests with coverage report"
            echo "  clean     - Clean up test environment"
            echo "  help      - Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0 setup     # Set up environment"
            echo "  $0 quick     # Quick test run"
            echo "  $0 all       # Full test suite"
            echo "  $0 coverage  # Generate coverage report"
            ;;
    esac
}

# Run main function with all arguments
main "$@"
