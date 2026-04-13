# Intelligent Process Reaper
**v1.2.9** – Linux process reaper with forensic capture, audit logging, and optional UniVerse integration.

![Go Version](https://img.shields.io/badge/go-1.26.1-blue)
![License](https://img.shields.io/badge/license-MIT-green)
![Platform](https://img.shields.io/badge/platform-linux-lightgrey)

Intelligent Process Reaper is a standalone Go daemon for Linux that identifies orphaned processes whose command lines match a configurable regular expression, captures forensic data, and terminates them with a SIGTERM-to-SIGKILL flow.

## Key Features
- Regex-based targeting over process command lines
- Forensic JSON capture before termination
- Audit log for scans, forensic collection, and signals
- Grace period between SIGTERM and SIGKILL
- Safety filters for self-exclusion, PID 1 exclusion, orphan-only matching, and minimum uptime
- Optional UniVerse forensic extensions
- Systemd-ready deployment with environment-based configuration

## Quick Start

### Install from packages

**Debian/Ubuntu**
```bash
wget https://github.com/clawbotari/process-reaper/releases/download/v1.2.9/process-reaper_1.2.9_amd64.deb
sudo dpkg -i process-reaper_1.2.9_amd64.deb
```

**RHEL/CentOS/Fedora**
```bash
wget https://github.com/clawbotari/process-reaper/releases/download/v1.2.9/process-reaper-1.2.9-1.x86_64.rpm
sudo rpm -i process-reaper-1.2.9-1.x86_64.rpm
```

After installation, edit the packaged service file and reload systemd:
```bash
sudo nano /lib/systemd/system/process-reaper.service
sudo systemctl daemon-reload
sudo systemctl restart process-reaper
```

### Manual install from source
```bash
git clone https://github.com/clawbotari/process-reaper.git
cd process-reaper
CGO_ENABLED=0 go build -o process-reaper ./cmd/process-reaper
sudo cp process-reaper /usr/local/bin/
sudo mkdir -p /var/log/process-reaper
sudo cp process-reaper.service /etc/systemd/system/
sudo systemctl daemon-reload
```

## Configuration

The daemon is configured with environment variables, usually in the systemd unit.

| Variable | Default | Description |
|----------|---------|-------------|
| `REAPER_PATTERN` | `.*` | Regular expression matched against process command lines. |
| `REAPER_INTERVAL` | `60` | Scan interval in seconds. |
| `REAPER_LOG_DIR` | `/var/log/process-reaper` | Base directory for the audit log, service log, and forensic files. |
| `REAPER_GRACE_PERIOD` | `10` | Seconds to wait between SIGTERM and SIGKILL. |
| `REAPER_MIN_UPTIME` | `5` | Minimum process age in minutes before a match is considered killable. |
| `REAPER_HEARTBEAT_QUIET` | `false` | Suppress heartbeat logs when no candidates are found. |
| `REAPER_KILL` | `true` | If `false`, collect forensic data and audit only. |
| `REAPER_UV_DIR` | disabled | UniVerse installation directory. When unset, UniVerse integration is off. |
| `REAPER_UV_DEBUG` | auto-detected | Optional UniVerse debug directory override. Used only when `REAPER_UV_DIR` is set. |
| `REAPER_RETENTION_DAYS` | `30` | Retention period for forensic `.json` and copied `.gz` debug artifacts. |
| `REAPER_DEBUG_FORENSIC` | `false` | Emit detailed UniVerse forensic command diagnostics without dumping the full process environment. |

Filtering behavior:
- Only orphaned processes are considered (`PPID == 1`).
- PID 1 and the reaper's own PID are always excluded.
- Processes younger than `REAPER_MIN_UPTIME` are skipped.

Example service configuration:
```ini
[Service]
Environment=REAPER_PATTERN=python3.*worker.*
Environment=REAPER_INTERVAL=30
Environment=REAPER_LOG_DIR=/var/log/process-reaper
Environment=REAPER_GRACE_PERIOD=5
Environment=REAPER_MIN_UPTIME=5
Environment=REAPER_HEARTBEAT_QUIET=false
Environment=REAPER_KILL=true
# Optional UniVerse integration:
# Environment=REAPER_UV_DIR=/usr/uv
# Environment=REAPER_UV_DEBUG=/usr/uv/uvdebug
```

## UniVerse Integration

When `REAPER_UV_DIR` is set, the reaper enriches forensic reports with UniVerse data:

- `port_status` from `bin/port.status`
- `list_readu` lock information
- `user_no` extracted from `listuser` or `list_readu`
- `uv_debug_file` when a matching debug file is found
- `uv_error` extracted from the debug file
- `uv_file` extracted from the debug file

If `REAPER_UV_DEBUG` is unset, the reaper attempts to derive the debug directory from `serverdebug`. Failures in UniVerse helpers never block process termination.

## Forensic Output

Forensic reports are written under `REAPER_LOG_DIR/forensics/` with names like:

```text
reaper_12345_20260306_183456.json
```

Example fields:

```json
{
  "pid": 12345,
  "cmdline": "/usr/bin/python3 /home/user/app.py",
  "executable": "/usr/bin/python3",
  "rss_bytes": 25468928,
  "vms_bytes": 10104832,
  "open_files": ["socket:[291202]", "/var/log/app.log"],
  "create_time": 1712345678901,
  "cpu_percent": 2.5,
  "username": "appuser",
  "status": ["running"],
  "timestamp": "2026-03-06T18:34:56Z"
}
```

When UniVerse debug files are copied, they are compressed into the same `forensics/` directory as `.gz` files.

## Audit Log

Audit entries are written to:

```text
REAPER_LOG_DIR/process-reaper-audit.log
```

Format:

```text
[2026-03-06T18:34:56Z] action=forensic pid=12345 collection=success
[2026-03-06T18:34:56Z] action=kill pid=12345 signal=SIGTERM status=success
[2026-03-06T18:34:58Z] action=terminated pid=12345 process exited after SIGTERM
[2026-03-06T18:34:59Z] action=scan pid= found 0 matching processes
```

## Development

### Build
```bash
CGO_ENABLED=0 go build -o process-reaper ./cmd/process-reaper
```

### Validate
```bash
go vet ./...
go test ./...
go build ./...
bash test/fire_test.sh
```

### Rocky Linux 9 Test Bed

The repository includes a Rocky Linux 9 Docker-based validation environment for Linux-targeted checks, integration testing, and packaging:

```bash
bash scripts/validate-rocky9.sh
```

The script builds `docker/rocky9/Dockerfile`, mounts the current checkout into the container, and runs:
- `go vet ./...`
- `go test ./...`
- `go build ./...`
- `bash test/fire_test.sh`
- RPM and DEB packaging with `nfpm`

### Package
Install `nfpm` first:
```bash
go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest
```

Then build the release binary and create the output directory:
```bash
mkdir -p build dist
CGO_ENABLED=0 go build -o build/process-reaper ./cmd/process-reaper
nfpm package --config nfpm.yaml --target dist/ --packager deb
nfpm package --config nfpm.yaml --target dist/ --packager rpm
```

### Integration Test
`test/fire_test.sh` is Linux-only. It starts a daemonized Python process, reads the actual orphan PID from `/tmp/hanging.pid`, runs the reaper with `REAPER_MIN_UPTIME=0` and UniVerse disabled, and verifies both termination and forensic output under `/tmp/reaper_fire_test/forensics`. On non-Linux hosts it exits with a skip message and points to `scripts/validate-rocky9.sh`.

## Validation Matrix

- `go vet`, `go test`, and `go build` are expected to pass anywhere the Go toolchain supports the code.
- The integration test is Unix-oriented.
- The daemon, packaging, and systemd unit are intended for Linux deployments.
- `scripts/validate-rocky9.sh` is the primary Linux validation path for this repository.

## License

MIT – see [LICENSE](LICENSE).
