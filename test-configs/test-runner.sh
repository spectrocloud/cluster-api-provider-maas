#!/bin/bash

# Test runner script for MAAS LXD VM provisioning
# Usage: ./test-runner.sh [test-name]

set -e

KUBECONFIG_PATH="/Users/junzhou/dl/admin.onyx.kubeconfig"
TEST_DIR="$(dirname "$0")"

export KUBECONFIG="$KUBECONFIG_PATH"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if MAAS credentials are configured
check_credentials() {
    log "Checking MAAS credentials..."
    if kubectl get secret maas-credentials -n cluster-api-provider-maas-system &> /dev/null; then
        success "MAAS credentials secret found"
        return 0
    else
        error "MAAS credentials not found. Please create the secret first:"
        echo "  kubectl apply -f $TEST_DIR/maas-credentials-template.yaml"
        return 1
    fi
}

# Deploy controller (assumes Docker image is available)
deploy_controller() {
    log "Deploying MAAS provider controller..."
    # Note: This requires Docker image to be built and available
    if kubectl get deployment -n cluster-api-provider-maas-system cluster-api-provider-maas-controller &> /dev/null; then
        warning "Controller deployment already exists, skipping deployment"
    else
        log "Controller deployment not found. Build and deploy the controller first:"
        echo "  make docker-build IMG=controller:latest"
        echo "  make deploy IMG=controller:latest"
        return 1
    fi
}

# Apply test configuration
apply_test() {
    local test_file="$1"
    log "Applying test configuration: $test_file"
    kubectl apply -f "$TEST_DIR/$test_file"
    success "Applied $test_file"
}

# Monitor test progress
monitor_test() {
    local machine_name="$1"
    local timeout=${2:-300}  # 5 minutes default
    
    log "Monitoring MaasMachine: $machine_name (timeout: ${timeout}s)"
    
    local start_time=$(date +%s)
    local end_time=$((start_time + timeout))
    
    while [ $(date +%s) -lt $end_time ]; do
        local status=$(kubectl get maasmachine "$machine_name" -o jsonpath='{.status.phase}' 2>/dev/null || echo "Not Found")
        local conditions=$(kubectl get maasmachine "$machine_name" -o jsonpath='{.status.conditions}' 2>/dev/null || echo "[]")
        
        log "Status: $status"
        
        case "$status" in
            "Running")
                success "VM $machine_name is running!"
                kubectl get maasmachine "$machine_name" -o yaml
                return 0
                ;;
            "Failed")
                error "VM $machine_name failed to provision!"
                kubectl describe maasmachine "$machine_name"
                return 1
                ;;
            "Not Found")
                error "MaasMachine $machine_name not found"
                return 1
                ;;
        esac
        
        sleep 10
    done
    
    warning "Timeout reached for $machine_name"
    kubectl describe maasmachine "$machine_name"
    return 1
}

# Clean up test resources
cleanup_test() {
    local test_file="$1"
    log "Cleaning up test: $test_file"
    kubectl delete -f "$TEST_DIR/$test_file" --ignore-not-found
    success "Cleaned up $test_file"
}

# Main test execution
run_test() {
    local test_name="$1"
    local test_file="${test_name}.yaml"
    
    if [ ! -f "$TEST_DIR/$test_file" ]; then
        error "Test file not found: $test_file"
        return 1
    fi
    
    log "Starting test: $test_name"
    
    # Apply test configuration
    apply_test "$test_file"
    
    # Extract machine names from the test file
    local machine_names=$(grep -E "^\s*name:\s+" "$TEST_DIR/$test_file" | awk '{print $2}' | head -5)
    
    # Monitor each machine
    for machine_name in $machine_names; do
        monitor_test "$machine_name" 600  # 10 minutes per VM
    done
    
    success "Test $test_name completed successfully!"
}

# Show available tests
show_tests() {
    log "Available tests:"
    for test_file in "$TEST_DIR"/*-test.yaml; do
        if [ -f "$test_file" ]; then
            local test_name=$(basename "$test_file" .yaml)
            echo "  - $test_name"
        fi
    done
}

# Main script
main() {
    local test_name="$1"
    
    log "MAAS LXD VM Provisioning Test Runner"
    echo "======================================"
    
    # Check prerequisites
    if ! check_credentials; then
        exit 1
    fi
    
    if [ -z "$test_name" ]; then
        show_tests
        echo ""
        echo "Usage: $0 <test-name>"
        echo "Example: $0 lxd-static-ip-test"
        exit 1
    fi
    
    # Run the test
    run_test "$test_name"
}

# Handle script arguments
case "${1:-}" in
    "cleanup")
        if [ -n "$2" ]; then
            cleanup_test "$2.yaml"
        else
            error "Please specify test name to clean up"
            exit 1
        fi
        ;;
    "monitor")
        if [ -n "$2" ]; then
            monitor_test "$2" "${3:-300}"
        else
            error "Please specify machine name to monitor"
            exit 1
        fi
        ;;
    *)
        main "$1"
        ;;
esac