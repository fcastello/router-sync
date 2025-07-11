#!/bin/bash

# Router Sync Installation Script
# This script installs router-sync binary and systemd service

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
BINARY_NAME="router-sync"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/router-sync"
SERVICE_USER="router-sync"
SERVICE_GROUP="router-sync"
SERVICE_NAME="router-sync"

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        print_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

# Function to detect architecture
detect_architecture() {
    local arch=$(uname -m)
    case $arch in
        x86_64)
            echo "linux-amd64"
            ;;
        aarch64|arm64)
            echo "linux-arm64"
            ;;
        *)
            print_error "Unsupported architecture: $arch"
            exit 1
            ;;
    esac
}

# Function to find the binary in the current directory
find_binary() {
    local arch=$1
    local binary_name="${BINARY_NAME}-${arch}"
    
    if [[ -f "$binary_name" ]]; then
        echo "$binary_name"
    elif [[ -f "${BINARY_NAME}" ]]; then
        echo "${BINARY_NAME}"
    else
        print_error "Binary not found. Please run this script from the directory containing the router-sync binary."
        exit 1
    fi
}

# Function to create service user
create_service_user() {
    if ! id "$SERVICE_USER" &>/dev/null; then
        print_status "Creating service user: $SERVICE_USER"
        useradd --system --no-create-home --shell /bin/false "$SERVICE_USER"
    else
        print_status "Service user $SERVICE_USER already exists"
    fi
}

# Function to create directories
create_directories() {
    print_status "Creating directories..."
    
    mkdir -p "$INSTALL_DIR"
    mkdir -p "$CONFIG_DIR"
    mkdir -p "/var/log/$SERVICE_NAME"
    mkdir -p "/var/lib/$SERVICE_NAME"
    
    # Set ownership
    chown "$SERVICE_USER:$SERVICE_GROUP" "/var/log/$SERVICE_NAME"
    chown "$SERVICE_USER:$SERVICE_GROUP" "/var/lib/$SERVICE_NAME"
    chmod 755 "/var/log/$SERVICE_NAME"
    chmod 755 "/var/lib/$SERVICE_NAME"
}

# Function to install binary
install_binary() {
    local binary_path=$1
    local target_path="$INSTALL_DIR/$BINARY_NAME"
    
    print_status "Installing binary to $target_path"
    cp "$binary_path" "$target_path"
    chmod 755 "$target_path"
    chown root:root "$target_path"
}

# Function to install systemd service
install_systemd_service() {
    local service_file="/etc/systemd/system/${SERVICE_NAME}.service"
    
    print_status "Installing systemd service..."
    
    cat > "$service_file" << EOF
[Unit]
Description=Router Sync Service
Documentation=https://github.com/your-org/router-sync
After=network.target

[Service]
Type=simple
User=$SERVICE_USER
Group=$SERVICE_GROUP
ExecStart=$INSTALL_DIR/$BINARY_NAME -config $CONFIG_DIR/config.yaml
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=$SERVICE_NAME

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/log/$SERVICE_NAME /var/lib/$SERVICE_NAME $CONFIG_DIR

[Install]
WantedBy=multi-user.target
EOF

    chmod 644 "$service_file"
}

# Function to create default config
create_default_config() {
    local config_file="$CONFIG_DIR/config.yaml"
    
    if [[ ! -f "$config_file" ]]; then
        print_status "Creating default configuration..."
        
        cat > "$config_file" << EOF
# Router Sync Configuration

# Log level (debug, info, warn, error)
log_level: info

# NATS configuration
nats:
  urls:
    - "nats://localhost:4222"
  username: ""
  password: ""
  token: ""
  cluster_id: "router-sync-cluster"
  client_id: "router-sync-client"

# API server configuration
api:
  address: ":8081"

# Synchronization configuration
sync:
  interval: 30s
EOF

        chown "$SERVICE_USER:$SERVICE_GROUP" "$config_file"
        chmod 644 "$config_file"
    else
        print_status "Configuration file already exists: $config_file"
    fi
}

# Function to enable and start service
enable_service() {
    print_status "Enabling and starting service..."
    systemctl daemon-reload
    systemctl enable "$SERVICE_NAME"
    systemctl start "$SERVICE_NAME"
}

# Function to show installation status
show_status() {
    print_status "Installation completed successfully!"
    echo
    echo "Service Status:"
    systemctl status "$SERVICE_NAME" --no-pager -l || true
    echo
    echo "Configuration file: $CONFIG_DIR/config.yaml"
    echo "Logs: journalctl -u $SERVICE_NAME"
    echo "Service control: systemctl {start|stop|restart|status} $SERVICE_NAME"
    echo
    print_warning "Please review and update the configuration file at $CONFIG_DIR/config.yaml"
}

# Function to show uninstall instructions
show_uninstall_info() {
    echo
    echo "To uninstall router-sync:"
    echo "  sudo systemctl stop $SERVICE_NAME"
    echo "  sudo systemctl disable $SERVICE_NAME"
    echo "  sudo rm -f /etc/systemd/system/${SERVICE_NAME}.service"
    echo "  sudo rm -f $INSTALL_DIR/$BINARY_NAME"
    echo "  sudo rm -rf $CONFIG_DIR"
    echo "  sudo rm -rf /var/log/$SERVICE_NAME"
    echo "  sudo rm -rf /var/lib/$SERVICE_NAME"
    echo "  sudo userdel $SERVICE_USER 2>/dev/null || true"
    echo "  sudo groupdel $SERVICE_GROUP 2>/dev/null || true"
    echo "  sudo systemctl daemon-reload"
}

# Main installation function
main() {
    print_status "Starting Router Sync installation..."
    
    # Check if running as root
    check_root
    
    # Detect architecture
    local arch=$(detect_architecture)
    print_status "Detected architecture: $arch"
    
    # Find binary
    local binary_path=$(find_binary "$arch")
    print_status "Found binary: $binary_path"
    
    # Create service user
    create_service_user
    
    # Create directories
    create_directories
    
    # Install binary
    install_binary "$binary_path"
    
    # Install systemd service
    install_systemd_service
    
    # Create default config
    create_default_config
    
    # Enable and start service
    enable_service
    
    # Show status
    show_status
    
    # Show uninstall info
    show_uninstall_info
}

# Run main function
main "$@" 