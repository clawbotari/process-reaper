# Test Suite for Intelligent Process Reaper

This directory contains integration tests for the reaper.

## `hanging.py`
A simple Python process that runs indefinitely, printing a marker
`HANGING_TEST_PROCESS`. Used as a target for the reaper.

## `fire_test.sh`
End‑to‑end test that:
1. Starts a hanging Python process.
2. Reads the daemonized child PID from `/tmp/hanging.pid`.
3. Launches the reaper with a pattern matching `hanging-test-process`, `REAPER_MIN_UPTIME=0`, and UniVerse disabled.
4. Waits for the reaper to scan, collect forensic data, and kill the process.
5. Verifies that the process is terminated, a forensic JSON file is created under `/tmp/reaper_fire_test/forensics`, and the audit log contains the expected entries.

Run with:
```bash
cd /path/to/process-reaper
bash test/fire_test.sh
```

## Notes
- The test cleans up `/tmp/reaper_fire_test` before each run.
- The reaper runs with `REAPER_INTERVAL=1`, `REAPER_GRACE_PERIOD=2`, and `REAPER_MIN_UPTIME=0`.
- The test expects the reaper binary to be present in the project root
  (builds it statically if missing).
- On non-Linux hosts the script exits early with a skip message instead of producing a false failure.
- `bash scripts/validate-rocky9.sh` is the preferred Linux-targeted way to run this test in a Rocky 9 environment.
