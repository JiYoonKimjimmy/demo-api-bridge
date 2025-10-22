#!/bin/bash

# API Bridge Shutdown Script (Bash)
# This script uses the API endpoint to gracefully shutdown the service

set -e # Exit on any error

# Default values
PORT="10019"
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
    echo "Usage: ./scripts/shutdown.sh [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -p, --port PORT   Target port (default: 10019)"
    echo "  -h, --help        Show this help message"
    echo ""
    echo "Examples:"
    echo "  ./scripts/shutdown.sh                    # Graceful shutdown on default port"
    echo "  ./scripts/shutdown.sh -p 8080           # Graceful shutdown on port 8080"
    echo ""
}

# Parse arguments
while [[ "$#" -gt 0 ]]; do
    case $1 in
        -p|--port)
            PORT="$2"
            shift
            ;;
        -h|--help)
            HELP=true
            ;;
        *)
            print_error "Unknown parameter: $1"
            show_usage
            exit 1
            ;;
    esac
    shift
done

if [ "$HELP" = true ]; then
    show_usage
    exit 0
fi

# Function to check if service is running
test_service_running() {
    local port=$1
    if curl -s -f "http://localhost:$port/health" > /dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# Function to send graceful shutdown request
send_graceful_shutdown() {
    local port=$1
    print_info "Sending graceful shutdown request to http://localhost:$port/api/v1/shutdown..."
    
    local response
    local http_code
    
    response=$(curl -s -w "%{http_code}" -X POST "http://localhost:$port/api/v1/shutdown" -o /tmp/shutdown_response.json)
    http_code="${response: -3}"
    
    if [ "$http_code" -eq 200 ]; then
        print_success "Graceful shutdown initiated successfully"
        if [ -f /tmp/shutdown_response.json ]; then
            local message=$(cat /tmp/shutdown_response.json | grep -o '"message":"[^"]*"' | cut -d'"' -f4)
            local timestamp=$(cat /tmp/shutdown_response.json | grep -o '"timestamp":"[^"]*"' | cut -d'"' -f4)
            echo "Response: $message"
            echo "Timestamp: $timestamp"
            rm -f /tmp/shutdown_response.json
        fi
        return 0
    else
        print_warning "Unexpected response code: $http_code"
        return 1
    fi
}

# Function to wait for service to stop
wait_for_service_stop() {
    local port=$1
    local timeout=${2:-30}
    
    print_info "Waiting for service to stop (timeout: $timeout seconds)..."
    
    local elapsed=0
    while [ $elapsed -lt $timeout ]; do
        if ! test_service_running "$port"; then
            print_success "Service has stopped successfully"
            return 0
        fi
        
        sleep 1
        elapsed=$((elapsed + 1))
        
        if [ $((elapsed % 5)) -eq 0 ]; then
            print_info "Still waiting... ($elapsed/$timeout seconds)"
        fi
    done
    
    print_warning "Service did not stop within $timeout seconds"
    return 1
}

# Main function
main() {
    echo "=================================="
    echo "API Bridge Shutdown Script"
    echo "=================================="
    
    print_info "Checking if service is running on port $PORT..."
    
    if ! test_service_running "$PORT"; then
        print_warning "Service is not running on port $PORT"
        return
    fi
    
    print_success "Service is running on port $PORT"
    
    # Send graceful shutdown request
    if send_graceful_shutdown "$PORT"; then
        # Wait for service to stop
        if wait_for_service_stop "$PORT"; then
            print_success "Graceful shutdown completed successfully"
        else
            print_warning "Graceful shutdown may not have completed properly"
            print_info "You may need to use force shutdown methods"
        fi
    else
        print_error "Failed to initiate graceful shutdown"
        print_info "You may need to use force shutdown methods"
    fi
    
    echo "=================================="
}

# Run main function
main
