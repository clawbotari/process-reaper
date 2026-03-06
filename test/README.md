# Test Suite for Intelligent Process Reaper

This directory contains integration tests for the reaper.

## `hanging.py`
A simple Python process that runs indefinitely, printing a marker
`HANGING_TEST_PROCESS`. Used as a target for the reaper.

## `fire_test.sh`
End‑to‑end test that:
1. Starts a hanging Python process.
2. Launches the reaper with a pattern matching `python3.*hanging`.
3. Waits for the reaper to scan, collect forensic data, and kill the process.
4. Verifies that the process is terminated, a forensic JSON file is created,
   and the audit log contains the expected entries.

Run with:
```bash
cd /path/to/process-reaper
bash test/fire_test.sh
```

## Notes
- The test cleans up `/tmp/reaper_fire_test` before each run.
- The reaper runs with `REAPER_INTERVAL=1` and `REAPER_GRACE_PERIOD=2`.
- The test expects the reaper binary to be present in the project root
  (builds it statically if missing).
