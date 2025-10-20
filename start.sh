#!/bin/bash

# API Bridge Service Start Script
# This script starts the API Bridge service with proper configuration

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() {
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

# Check if Go is installed
check_go() {
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed or not in PATH"
        print_info "Please install Go 1.25.1 or later"
        exit 1
    fi
    
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    print_info "Found Go version: $GO_VERSION"
}

# Check if required directories exist
check_project_structure() {
    print_info "Checking project structure..."
    
    if [ ! -f "go.mod" ]; then
        print_error "go.mod not found. Please run this script from the project root directory."
        exit 1
    fi
    
    if [ ! -f "cmd/api-bridge/main.go" ]; then
        print_error "Main application file not found: cmd/api-bridge/main.go"
        exit 1
    fi
    
    if [ ! -f "config/config.yaml" ]; then
        print_warning "config/config.yaml not found. Using default configuration."
    fi
    
    print_success "Project structure validated"
}

# Download dependencies
download_dependencies() {
    print_info "Downloading Go dependencies..."
    if go mod download; then
        print_success "Dependencies downloaded successfully"
    else
        print_error "Failed to download dependencies"
        exit 1
    fi
}

# Build the application
build_application() {
    print_info "Building the application..."
    if go build -o bin/api-bridge cmd/api-bridge/main.go; then
        print_success "Application built successfully"
    else
        print_error "Failed to build application"
        exit 1
    fi
}

# Create necessary directories
create_directories() {
    print_info "Creating necessary directories..."
    mkdir -p bin
    mkdir -p logs
    print_success "Directories created"
}

# Set environment variables
set_environment() {
    print_info "Setting environment variables..."
    
    # Set default port if not already set
    if [ -z "$PORT" ]; then
        export PORT=10019
    fi
    
    # Set Gin mode to release for production
    export GIN_MODE=release
    
    # Set timezone
    export TZ=Asia/Seoul
    
    print_info "PORT: $PORT"
    print_info "GIN_MODE: $GIN_MODE"
    print_info "TZ: $TZ"
}

# Start the application
start_application() {
    print_info "Starting API Bridge Service..."
    print_info "Service will be available at: http://localhost:$PORT"
    print_info ""
    print_info "Available endpoints:"
    print_info "  GET  /health                    - Health check"
    print_info "  GET  /ready                     - Readiness check"
    print_info "  GET  /api/v1/status             - Service status"
    print_info "  ANY  /api/v1/bridge/*           - API Bridge"
    print_info "  GET  /metrics                   - Prometheus metrics"
    print_info ""
    print_info "Press Ctrl+C to stop the service"
    print_info "=================================="
    
    # Start the application
    if [ -f "bin/api-bridge" ]; then
        # Use built binary
        ./bin/api-bridge
    else
        # Run directly with go run
        go run cmd/api-bridge/main.go
    fi
}

# Cleanup function
cleanup() {
    print_info "Shutting down API Bridge Service..."
    print_success "Service stopped"
}

# Set trap for cleanup
trap cleanup EXIT INT TERM

# Main execution
main() {
    print_info "=================================="
    print_info "API Bridge Service Startup Script"
    print_info "=================================="
    
    check_go
    check_project_structure
    create_directories
    download_dependencies
    build_application
    set_environment
    
    print_success "All checks passed. Starting service..."
    print_info "=================================="
    
    start_application
}

# Run main function
main "$@"
