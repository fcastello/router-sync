#!/bin/bash

# Test script for router-sync installation
# This script tests the installation process in a controlled environment

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${GREEN}[TEST]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Test directory
TEST_DIR="/tmp/router-sync-test"
BINARY_NAME="router-sync"

# Cleanup function
cleanup() {
    print_status "Cleaning up test environment..."
    
    # Stop and disable service if it exists
    if systemctl is-active --quiet router-sync 2>/dev/null; then
        systemctl stop router-sync || true
    fi
    
    if systemctl is-enabled --quiet router-sync 2>/dev/null; then
        systemctl disable router-sync || true
    fi
    
    # Remove test files
    rm -rf "$TEST_DIR" || true
    rm -f /etc/systemd/system/router-sync.service || true
    rm -f /usr/local/bin/router-sync || true
    rm -rf /etc/router-sync || true
    rm -rf /var/log/router-sync || true
    rm -rf /var/lib/router-sync || true
    
    # Remove test user if it exists
    userdel router-sync 2>/dev/null || true
    groupdel router-sync 2>/dev/null || true
    
    systemctl daemon-reload || true
    
    print_status "Cleanup completed"
}

# Setup test environment
setup_test() {
    print_status "Setting up test environment..."
    
    # Create test directory
    mkdir -p "$TEST_DIR"
    cd "$TEST_DIR"
    
    # Extract the latest release
    if [[ -f "../../release/router-sync-v0.0.2-linux-amd64.tar.gz" ]]; then
        tar -xzf ../../release/router-sync-v0.0.2-linux-amd64.tar.gz
        cd router-sync-v0.0.2-linux-amd64
    else
        print_error "Release file not found. Please run 'make release' first."
        exit 1
    fi
    
    print_status "Test environment ready"
}

# Test installation
test_installation() {
    print_status "Testing installation process..."
    
    # Run installation script
    if [[ -f "install.sh" ]]; then
        chmod +x install.sh
        print_status "Running installation script..."
        ./install.sh
    else
        print_error "Installation script not found"
        exit 1
    fi
}

# Test service functionality
test_service() {
    print_status "Testing service functionality..."
    
    # Check if service is running
    if systemctl is-active --quiet router-sync; then
        print_status "Service is running"
    else
        print_error "Service is not running"
        systemctl status router-sync --no-pager -l
        exit 1
    fi
    
    # Check if service is enabled
    if systemctl is-enabled --quiet router-sync; then
        print_status "Service is enabled"
    else
        print_error "Service is not enabled"
        exit 1
    fi
    
    # Check service logs
    print_status "Checking service logs..."
    journalctl -u router-sync --no-pager -n 10 || true
}

# Test configuration
test_configuration() {
    print_status "Testing configuration..."
    
    if [[ -f "/etc/router-sync/config.yaml" ]]; then
        print_status "Configuration file exists"
    else
        print_error "Configuration file not found"
        exit 1
    fi
    
    # Test binary execution
    if /usr/local/bin/router-sync --help >/dev/null 2>&1; then
        print_status "Binary is executable"
    else
        print_error "Binary is not executable"
        exit 1
    fi
}

# Main test function
main() {
    print_status "Starting router-sync installation test..."
    
    # Check if running as root
    if [[ $EUID -ne 0 ]]; then
        print_error "This test script must be run as root (use sudo)"
        exit 1
    fi
    
    # Setup trap for cleanup
    trap cleanup EXIT
    
    # Run tests
    setup_test
    test_installation
    test_service
    test_configuration
    
    print_status "All tests passed! Installation is working correctly."
    print_warning "Remember to run cleanup when done testing"
}

# Run main function
main "$@" 