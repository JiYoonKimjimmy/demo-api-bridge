#!/bin/bash

# API Bridge Service Start Script
# This script starts the API Bridge service with proper configuration

set -e  # Exit on any error

# Default values
PORT="10019"
TARGET_HOST="localhost"
VERBOSE=false
HELP=false

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

# Function to show usage
show_usage() {
    echo "Usage: ./scripts/start.sh [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -p, --port PORT     Target port (default: 10019)"
    echo "  -h, --host HOST     Target host (default: localhost)"
    echo "  -v, --verbose       Show detailed output"
    echo "  --help             Show this help message"
    echo ""
    echo "Examples:"
    echo "  ./scripts/start.sh                    # Start with default settings"
    echo "  ./scripts/start.sh -p 8080           # Start on port 8080"
    echo "  ./scripts/start.sh --verbose          # Show detailed output"
    echo ""
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -p|--port)
            PORT="$2"
            shift 2
            ;;
        -h|--host)
            TARGET_HOST="$2"
            shift 2
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        --help)
            HELP=true
            shift
            ;;
        *)
            print_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

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

# Create necessary directories
create_directories() {
    print_info "Creating necessary directories..."
    mkdir -p bin logs
    print_success "Directories created"
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
    
    # 기존 바이너리 삭제 (캐시 제거)
    if [ -f "bin/api-bridge" ]; then
        print_info "Removing existing binary to ensure clean build..."

        # 실행 중인 프로세스 종료 (pgrep이 있는 경우에만)
        if command -v pgrep &> /dev/null; then
            if pgrep -f "api-bridge" > /dev/null 2>&1; then
                print_info "Stopping existing api-bridge processes..."
                pkill -f "api-bridge" || true
                sleep 2
            fi
        else
            # pgrep이 없는 경우 (Windows Git Bash 등)
            # 프로세스 확인/종료를 건너뛰고 바이너리만 삭제
            print_info "Process management tools not available, skipping process check..."
        fi

        # 바이너리 파일 삭제
        if rm -f "bin/api-bridge"; then
            print_info "Existing binary removed successfully"
        else
            print_warning "Could not remove existing binary, continuing with build..."
        fi
    fi
    
    # 새로 빌드
    if go build -o bin/api-bridge cmd/api-bridge/main.go; then
        print_success "Application built successfully"
    else
        print_error "Failed to build application"
        exit 1
    fi
}

# Set environment variables
set_environment() {
    print_info "Setting environment variables..."
    
    export PORT="$PORT"
    export GIN_MODE="release"
    export TZ="Asia/Seoul"
    
    print_info "PORT: $PORT"
    print_info "GIN_MODE: $GIN_MODE"
    print_info "TZ: $TZ"
}

# Start the application
start_application() {
    print_info "Starting API Bridge Service..."
    print_info "Service will be available at: http://$TARGET_HOST:$PORT"
    echo ""
    print_info "Available endpoints:"
    print_info "  ANY  /api/*path                 - API Bridge (handles all /api/* requests)"
    print_info "  GET  /abs/health         - Health check"
    print_info "  GET  /abs/ready          - Readiness check"
    print_info "  GET  /abs/v1/status      - Service status"
    print_info "  GET  /abs/metrics        - Prometheus metrics"
    echo ""
    print_info "Press Ctrl+C to stop the service"
    echo "=================================="
    
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

# Main function
main() {
    if [ "$HELP" = true ]; then
        show_usage
        return
    fi
    
    echo "=================================="
    echo "API Bridge Service Startup Script"
    echo "=================================="
    
    check_go
    check_project_structure
    create_directories
    download_dependencies
    build_application
    set_environment
    
    print_success "All checks passed. Starting service..."
    echo "=================================="
    
    start_application
}

# Run main function
main
