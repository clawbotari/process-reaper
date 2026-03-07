#!/bin/sh
set -e

echo "Stopping process-reaper service if running"
systemctl stop process-reaper >/dev/null 2>&1 || true
systemctl disable process-reaper >/dev/null 2>&1 || true