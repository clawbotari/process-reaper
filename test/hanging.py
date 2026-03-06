#!/usr/bin/env python3
"""
Test process that hangs indefinitely, to be reaped by the Intelligent Process Reaper.
"""
import os
import time
import sys

MARKER = "HANGING_TEST_PROCESS"
pid = os.getpid()
print(f"{MARKER} started with PID {pid}", file=sys.stderr)
# Write PID to file for verification
with open("/tmp/hanging.pid", "w") as f:
    f.write(str(pid))

# Keep process alive forever
try:
    while True:
        time.sleep(1)
except KeyboardInterrupt:
    print(f"{MARKER} exiting", file=sys.stderr)
    sys.exit(0)
