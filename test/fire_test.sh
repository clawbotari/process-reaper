#!/bin/bash
set -e
echo "=== Intelligent Process Reaper Fire Test ==="

# Clean up previous test
rm -rf /tmp/reaper_fire_test
mkdir -p /tmp/reaper_fire_test

# Kill any remaining hanging processes
pkill -f "python3.*hanging" 2>/dev/null || true
sleep 0.5

# Start hanging process
python3 test/hanging.py > /tmp/hanging.out 2>&1 &
HANG_PID=$!
echo "Hanging process PID: $HANG_PID"
sleep 0.5

# Verify it's running
if ps -p $HANG_PID > /dev/null; then
    echo "Hanging process is alive (PID $HANG_PID)"
else
    echo "ERROR: Hanging process died prematurely"
    exit 1
fi

# Build static binary if not present
if [[ ! -f ./process-reaper ]]; then
    CGO_ENABLED=0 go build -o process-reaper ./cmd/process-reaper
fi

# Run reaper with pattern that matches the hanging process
echo "Starting reaper (pattern='python3.*hanging')..."
REAPER_PATTERN="python3.*hanging" \
REAPER_INTERVAL=1 \
REAPER_LOG_DIR=/tmp/reaper_fire_test \
REAPER_GRACE_PERIOD=2 \
./process-reaper > /tmp/reaper.out 2>&1 &
REAPER_PID=$!
echo "Reaper PID: $REAPER_PID"

# Wait for reaper to scan and kill (should take max 3 seconds)
sleep 5

# Stop reaper gracefully
kill -TERM $REAPER_PID 2>/dev/null || true
wait $REAPER_PID 2>/dev/null || true

echo "Reaper stopped."

# Check if hanging process is gone
if ps -p $HANG_PID > /dev/null; then
    echo "FAIL: Hanging process still alive!"
    exit 1
else
    echo "SUCCESS: Hanging process was terminated."
fi

# Verify forensic JSON was created
JSON_COUNT=$(find /tmp/reaper_fire_test -name "reaper_*.json" -type f | wc -l)
if [[ $JSON_COUNT -gt 0 ]]; then
    echo "SUCCESS: $JSON_COUNT forensic JSON file(s) created."
    ls -la /tmp/reaper_fire_test/*.json 2>/dev/null || true
else
    echo "FAIL: No forensic JSON found."
    exit 1
fi

# Verify audit log
AUDIT_LOG="/tmp/reaper_fire_test/process-reaper-audit.log"
if [[ -f $AUDIT_LOG ]]; then
    echo "SUCCESS: Audit log exists."
    tail -5 $AUDIT_LOG
else
    echo "FAIL: Audit log missing."
    exit 1
fi

echo "=== Fire test completed successfully ==="