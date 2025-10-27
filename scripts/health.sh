#!/bin/bash

# API Bridge Health Check Script
# This script tests the health endpoints of the API Bridge service

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default configuration
DEFAULT_HOST="localhost"
DEFAULT_PORT="10019"
DEFAULT_TIMEOUT=10

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
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -h, --host HOST      Target host (default: localhost)"
    echo "  -p, --port PORT      Target port (default: 10019)"
    echo "  -t, --timeout SEC    Request timeout in seconds (default: 10)"
    echo "  -v, --verbose        Show detailed response"
    echo "  --help               Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                           # Test localhost:10019"
    echo "  $0 -h 192.168.1.100 -p 8080  # Test specific host and port"
    echo "  $0 -v                        # Show detailed response"
    echo ""
}

# Parse command line arguments
HOST=$DEFAULT_HOST
PORT=$DEFAULT_PORT
TIMEOUT=$DEFAULT_TIMEOUT
VERBOSE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--host)
            HOST="$2"
            shift 2
            ;;
        -p|--port)
            PORT="$2"
            shift 2
            ;;
        -t|--timeout)
            TIMEOUT="$2"
            shift 2
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        --help)
            show_usage
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Base URL
BASE_URL="http://$HOST:$PORT"

# Function to check if curl is available
check_curl() {
    if ! command -v curl &> /dev/null; then
        print_error "curl is not installed or not in PATH"
        print_info "Please install curl to use this script"
        exit 1
    fi
    
    print_info "Found curl: $(curl --version | head -n1)"
}

# Function to test an endpoint
test_endpoint() {
    local endpoint=$1
    local expected_status=$2
    local description=$3
    
    local url="$BASE_URL$endpoint"
    
    print_info "Testing: $description"
    print_info "URL: $url"
    
    # Make the request
    local response
    local http_status
    local curl_exit_code
    
    if [ "$VERBOSE" = true ]; then
        response=$(curl -s -w "\n%{http_code}" --connect-timeout $TIMEOUT "$url" || echo -e "\n000")
    else
        response=$(curl -s -w "\n%{http_code}" --connect-timeout $TIMEOUT "$url" 2>/dev/null || echo -e "\n000")
    fi
    
    # Extract status code (last line)
    http_status=$(echo "$response" | tail -n1)
    
    # Extract response body (all lines except last)
    response_body=$(echo "$response" | head -n -1)
    
    # Check if curl command succeeded
    curl_exit_code=$?
    
    if [ $curl_exit_code -ne 0 ]; then
        print_error "Failed to connect to $url (curl exit code: $curl_exit_code)"
        return 1
    fi
    
    # Check HTTP status code
    if [ "$http_status" = "$expected_status" ]; then
        print_success "✓ Status: $http_status (Expected: $expected_status)"
    else
        print_error "✗ Status: $http_status (Expected: $expected_status)"
        return 1
    fi
    
    # Show response body if verbose or if there's an error
    if [ "$VERBOSE" = true ] || [ "$http_status" != "$expected_status" ]; then
        echo "Response:"
        echo "$response_body" | jq . 2>/dev/null || echo "$response_body"
    fi
    
    echo ""
    return 0
}

# Function to test service availability
test_service_availability() {
    print_info "Testing service availability..."

    # Try to connect to the service
    if curl -s --connect-timeout $TIMEOUT "$BASE_URL/management/health" > /dev/null 2>&1; then
        print_success "Service is available at $BASE_URL"
        return 0
    else
        print_error "Service is not available at $BASE_URL"
        print_info "Please make sure the API Bridge service is running"
        print_info "You can start it with: ./start.sh"
        return 1
    fi
}

# Function to run comprehensive health checks
run_health_checks() {
    print_info "=================================="
    print_info "API Bridge Health Check"
    print_info "=================================="
    print_info "Host: $HOST"
    print_info "Port: $PORT"
    print_info "Timeout: ${TIMEOUT}s"
    print_info "Verbose: $VERBOSE"
    print_info "=================================="
    echo ""
    
    local failed_tests=0
    
    # Test service availability first
    if ! test_service_availability; then
        exit 1
    fi
    
    # Test health endpoint
    if ! test_endpoint "/management/health" "200" "Health Check Endpoint"; then
        ((failed_tests++))
    fi

    # Test readiness endpoint
    if ! test_endpoint "/management/ready" "200" "Readiness Check Endpoint"; then
        ((failed_tests++))
    fi

    # Test status endpoint
    if ! test_endpoint "/management/v1/status" "200" "Service Status Endpoint"; then
        ((failed_tests++))
    fi

    # Test metrics endpoint
    if ! test_endpoint "/management/metrics" "200" "Prometheus Metrics Endpoint"; then
        ((failed_tests++))
    fi
    
    echo "=================================="
    if [ $failed_tests -eq 0 ]; then
        print_success "All health checks passed! ✓"
        print_info "API Bridge service is healthy and ready"
        exit 0
    else
        print_error "$failed_tests test(s) failed! ✗"
        print_info "Please check the service logs for more details"
        exit 1
    fi
}

# Function to run quick health check
run_quick_check() {
    print_info "Quick health check..."

    local response
    local http_status

    response=$(curl -s -w "\n%{http_code}" --connect-timeout $TIMEOUT "$BASE_URL/management/health" 2>/dev/null || echo -e "\n000")
    http_status=$(echo "$response" | tail -n1)

    if [ "$http_status" = "200" ]; then
        print_success "Service is healthy ✓"
        echo "$BASE_URL/management/health returned status $http_status"
    else
        print_error "Service is not healthy ✗"
        echo "$BASE_URL/management/health returned status $http_status"
        exit 1
    fi
}

# Main execution
main() {
    # Check prerequisites
    check_curl
    
    # Determine which check to run
    if [ "$VERBOSE" = true ]; then
        run_health_checks
    else
        # Check if user wants full health check or quick check
        if [ "$1" = "full" ] || [ "$1" = "--full" ]; then
            run_health_checks
        else
            run_quick_check
        fi
    fi
}

# Run main function
main "$@"
