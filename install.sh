#!/bin/bash
set -e

# Installation script for Intelligent Process Reaper
BINARY_NAME="process-reaper"
INSTALL_DIR="/usr/local/bin"
SERVICE_FILE="process-reaper.service"
SYSTEMD_DIR="/etc/systemd/system"
LOG_DIR="/var/log/process-reaper"

# Ensure script is run as root
if [[ $EUID -ne 0 ]]; then
   echo "This script must be run as root" 
   exit 1
fi

echo "Installing Intelligent Process Reaper..."

# 1. Build the binary (if not already built)
if [[ ! -f "./$BINARY_NAME" ]]; then
    echo "Building binary..."
    go build -o "$BINARY_NAME" ./cmd/process-reaper
fi

# 2. Copy binary
echo "Installing binary to $INSTALL_DIR"
cp "$BINARY_NAME" "$INSTALL_DIR/"
chmod 755 "$INSTALL_DIR/$BINARY_NAME"

# 3. Create log directory
echo "Creating log directory $LOG_DIR"
mkdir -p "$LOG_DIR"
chown root:root "$LOG_DIR"
chmod 755 "$LOG_DIR"

# 4. Install systemd service
echo "Installing systemd service"
cp "$SERVICE_FILE" "$SYSTEMD_DIR/"
chmod 644 "$SYSTEMD_DIR/$SERVICE_FILE"

# 5. Reload systemd
echo "Reloading systemd daemon"
systemctl daemon-reload

echo "Installation complete."
echo ""
echo "To start the service:"
echo "  systemctl start process-reaper"
echo ""
echo "To enable at boot:"
echo "  systemctl enable process-reaper"
echo ""
echo "Edit configuration in $SYSTEMD_DIR/$SERVICE_FILE and restart."