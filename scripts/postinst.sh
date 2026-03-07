#!/bin/sh
set -e

LOG_DIR="/var/log/process-reaper"

echo "Creating log directory $LOG_DIR"
mkdir -p "$LOG_DIR"
chown root:root "$LOG_DIR"
chmod 755 "$LOG_DIR"

echo "Reloading systemd daemon"
systemctl daemon-reload >/dev/null 2>&1 || true

# Optional: start and enable service automatically?
# Uncomment the following lines if you want the service to start on install
# echo "Starting process-reaper service"
# systemctl start process-reaper >/dev/null 2>&1 || true
# systemctl enable process-reaper >/dev/null 2>&1 || true

echo "process-reaper installed successfully."
echo "Edit /lib/systemd/system/process-reaper.service to adjust environment variables."
echo "Then start with: systemctl start process-reaper"