#!/usr/bin/env python3
"""
Test process that becomes an orphan (PPID=1) and hangs indefinitely.
Used to validate the Intelligent Process Reaper's orphan‑process filtering.
"""
import os
import sys
import time

def spawn_test_process(name):
    """
    Double‑fork daemonization that ensures the final process:
      - Has PPID = 1 (adopted by systemd/init)
      - Shows `name` as its argv[0] (visible in /proc/.../cmdline)
      - Runs an infinite sleep loop.
    Returns the PID of the spawned process to the original caller.
    """
    try:
        # First fork: detach from terminal/script parent
        pid = os.fork()
        if pid > 0:
            # Original parent exits; child continues
            sys.exit(0)

        # Create new session, become session leader
        os.setsid()

        # Second fork: guarantee adoption by PID 1
        pid = os.fork()
        if pid > 0:
            # Intermediate parent prints the final PID and exits
            print(f"Test process spawned with PID: {pid}", file=sys.stderr)
            # Write PID to file for external verification
            with open("/tmp/hanging.pid", "w") as f:
                f.write(str(pid))
            sys.exit(0)

        # Final child process – will be orphaned (PPID = 1)
        # Replace ourselves with a Python interpreter that runs an infinite loop,
        # using `name` as argv[0] so it appears in the command line.
        os.execlp(
            "python3",
            name,                     # argv[0] – appears as process name
            "-c",
            "import time\n"
            "while True:\n"
            "    time.sleep(10)\n"
        )
        # exec never returns unless it fails
    except Exception as e:
        print(f"Error spawning process: {e}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    # Default process name matches the example pattern in the README
    process_name = sys.argv[1] if len(sys.argv) > 1 else "uvapi_slave"
    spawn_test_process(process_name)
