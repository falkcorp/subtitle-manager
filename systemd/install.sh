#!/bin/bash
# file: systemd/install.sh
# version: 1.1.0
# last-edited: 2026-08-12
# guid: 890e2345-e89b-12d3-a456-426614174003

# Subtitle Manager systemd service installation script
# This script automates the installation of Subtitle Manager as a systemd service

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
USER_NAME="subtitle-manager"
SERVICE_NAME="subtitle-manager"
BINARY_PATH="/usr/local/bin/subtitle-manager"
CONFIG_DIR="/etc/subtitle-manager"
DATA_DIR="/var/lib/subtitle-manager"
LOG_DIR="/var/log/subtitle-manager"
WORK_DIR="/opt/subtitle-manager"

print_header() {
    echo -e "${BLUE}================================================${NC}"
    echo -e "${BLUE}   Subtitle Manager systemd Service Installer${NC}"
    echo -e "${BLUE}================================================${NC}"
    echo
}

print_step() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_root() {
    if [ "$EUID" -ne 0 ]; then
        print_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

check_systemd() {
    if ! command -v systemctl &> /dev/null; then
        print_error "systemd not found. This script requires a systemd-based Linux distribution."
        exit 1
    fi
}

check_binary() {
    local binary_source=""
    
    # Check if binary exists in current directory
    if [ -f "./bin/subtitle-manager" ]; then
        binary_source="./bin/subtitle-manager"
    elif [ -f "./subtitle-manager" ]; then
        binary_source="./subtitle-manager"
    else
        print_error "Subtitle Manager binary not found."
        print_error "Please build the binary first:"
        print_error "  CGO_ENABLED=0 go build -o subtitle-manager ."
        print_error "Alternatively, download from: https://github.com/jdfalk/subtitle-manager/releases"
        exit 1
    fi
    
    print_step "Found binary: $binary_source"
    echo "$binary_source"
}

create_user() {
    if id "$USER_NAME" &>/dev/null; then
        print_step "User '$USER_NAME' already exists"
    else
        print_step "Creating system user '$USER_NAME'..."
        useradd --system --home-dir "$DATA_DIR" --create-home --shell /bin/false "$USER_NAME"
    fi
}

create_directories() {
    print_step "Creating directories..."
    
    mkdir -p "$CONFIG_DIR"
    mkdir -p "$DATA_DIR/db"
    mkdir -p "$LOG_DIR"
    mkdir -p "$WORK_DIR"
    
    # Set ownership
    chown -R "$USER_NAME:$USER_NAME" "$DATA_DIR"
    chown -R "$USER_NAME:$USER_NAME" "$LOG_DIR"
    chown "$USER_NAME:$USER_NAME" "$WORK_DIR"
    
    print_step "Directories created and ownership set"
}

install_binary() {
    local binary_source="$1"
    
    print_step "Installing binary to $BINARY_PATH..."
    cp "$binary_source" "$BINARY_PATH"
    chmod +x "$BINARY_PATH"
    
    print_step "Binary installed successfully"
}

install_service_files() {
    print_step "Installing systemd service files..."
    
    if [ ! -f "systemd/subtitle-manager.service" ]; then
        print_error "Service file not found: systemd/subtitle-manager.service"
        print_error "Please run this script from the subtitle-manager repository root"
        exit 1
    fi
    
    # Copy service file
    cp "systemd/subtitle-manager.service" "/etc/systemd/system/"
    chmod 644 "/etc/systemd/system/subtitle-manager.service"
    
    # Copy environment file template
    if [ ! -f "$CONFIG_DIR/subtitle-manager.env" ]; then
        cp "systemd/subtitle-manager.env" "$CONFIG_DIR/"
        chmod 640 "$CONFIG_DIR/subtitle-manager.env"
        chown "root:$USER_NAME" "$CONFIG_DIR/subtitle-manager.env"
        print_step "Environment file template installed"
    else
        print_warning "Environment file already exists, not overwriting"
    fi
    
    print_step "Service files installed"
}

configure_systemd() {
    print_step "Configuring systemd..."
    
    # Reload systemd to recognize new service
    systemctl daemon-reload
    
    # Enable service to start on boot
    systemctl enable "$SERVICE_NAME"
    
    print_step "Service enabled for automatic startup"
}

show_next_steps() {
    echo
    echo -e "${GREEN}================================================${NC}"
    echo -e "${GREEN}   Installation completed successfully!${NC}"
    echo -e "${GREEN}================================================${NC}"
    echo
    echo -e "${BLUE}Next steps:${NC}"
    echo
    echo -e "1. ${YELLOW}Configure the service:${NC}"
    echo "   sudo nano $CONFIG_DIR/subtitle-manager.env"
    echo
    echo -e "2. ${YELLOW}Start the service:${NC}"
    echo "   sudo systemctl start $SERVICE_NAME"
    echo
    echo -e "3. ${YELLOW}Check service status:${NC}"
    echo "   sudo systemctl status $SERVICE_NAME"
    echo
    echo -e "4. ${YELLOW}View logs:${NC}"
    echo "   sudo journalctl -u $SERVICE_NAME -f"
    echo
    echo -e "5. ${YELLOW}Access web interface:${NC}"
    echo "   http://localhost:8080"
    echo
    echo -e "${BLUE}Management commands:${NC}"
    echo "   sudo systemctl start $SERVICE_NAME      # Start service"
    echo "   sudo systemctl stop $SERVICE_NAME       # Stop service"
    echo "   sudo systemctl restart $SERVICE_NAME    # Restart service"
    echo "   sudo systemctl status $SERVICE_NAME     # Check status"
    echo
    echo -e "${BLUE}Configuration files:${NC}"
    echo "   Service file: /etc/systemd/system/$SERVICE_NAME.service"
    echo "   Environment:  $CONFIG_DIR/subtitle-manager.env"
    echo "   Data:         $DATA_DIR/"
    echo "   Logs:         $LOG_DIR/"
    echo
}

main() {
    print_header
    
    check_root
    check_systemd
    
    local binary_source
    binary_source=$(check_binary)
    
    create_user
    create_directories
    install_binary "$binary_source"
    install_service_files
    configure_systemd
    
    show_next_steps
}

# Handle command line arguments
case "${1:-}" in
    -h|--help)
        echo "Usage: $0 [OPTIONS]"
        echo
        echo "Install Subtitle Manager as a systemd service"
        echo
        echo "OPTIONS:"
        echo "  -h, --help    Show this help message"
        echo
        echo "This script must be run as root (use sudo)"
        echo "Run from the subtitle-manager repository root directory"
        exit 0
        ;;
    *)
        main "$@"
        ;;
esac