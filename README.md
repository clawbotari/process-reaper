# Intelligent Process Reaper
**v1.2.6** – Fixes log overlap on service restart and improves UniVerse forensic robustness.

![Go Version](https://img.shields.io/badge/go-1.26.1-blue)
![License](https://img.shields.io/badge/license-MIT-green)
![Platform](https://img.shields.io/badge/platform-linux-lightgrey)

A standalone Go daemon for Linux that identifies, analyzes, and kills hanging processes based on configurable regex patterns. Designed for production servers, it collects forensic data before termination, respects grace periods, and logs every action for audit.

**Key Features**
- **Regex‑based targeting** – match processes by command‑line pattern
- **Forensic data collection** – RSS/VMS, open files, execution time saved as JSON
- **Graceful termination** – SIGTERM → configurable grace period → SIGKILL
- **Self‑exclusion** – never targets its own PID or PID 1 (systemd/init)
- **Audit logging** – structured log of every scan, forensic capture, and kill
- **Static binary** – zero dependencies, ready to deploy on any Linux amd64 system
- **Systemd‑native** – runs as a service with environment‑based configuration

## Quick Start

### Install from .deb or .rpm (recommended)

**Debian/Ubuntu**
```bash
wget https://github.com/clawbotari/process-reaper/releases/download/v1.2.6/process-reaper_1.2.6_amd64.deb
sudo dpkg -i process-reaper_1.2.6_amd64.deb
```


**Operational note:** After changing any variable in the systemd service file, run:
```bash
sudo systemctl daemon-reload && sudo systemctl restart process-reaper
```
**RHEL/CentOS/Fedora**
```bash
wget https://github.com/clawbotari/process-reaper/releases/download/v1.2.6/process-reaper-1.2.6-1.x86_64.rpm
sudo rpm -i process-reaper-1.2.6-1.x86_64.rpm
```

After installation, edit the systemd service file to set your pattern and other options:
```bash
sudo nano /lib/systemd/system/process-reaper.service
```

Then start and enable the service:
```bash
sudo systemctl daemon-reload
sudo systemctl start process-reaper
sudo systemctl enable process-reaper
```

### Manual installation (from source)

```bash
git clone https://github.com/clawbotari/process-reaper.git
cd process-reaper
CGO_ENABLED=0 go build -o process-reaper ./cmd/process-reaper
sudo cp process-reaper /usr/local/bin/
sudo mkdir -p /var/log/process-reaper
sudo cp process-reaper.service /lib/systemd/system/
sudo systemctl daemon-reload
```

## Configuration

The reaper is configured exclusively through environment variables, which are best set in the systemd service file.

| Variable | Default | Description |
|----------|---------|-------------|
| `REAPER_PATTERN` | `.*` | Regular expression to match against process command lines. |
| `REAPER_INTERVAL` | `60` | Scan interval in seconds. |
| `REAPER_LOG_DIR` | `/var/log/process-reaper` | Directory for forensic JSON files and audit log. |
| `REAPER_GRACE_PERIOD` | `10` | Seconds to wait between SIGTERM and SIGKILL. |
| `REAPER_MIN_UPTIME` | `5` | Minimum process age in minutes (only processes older than this are considered). |
| `REAPER_HEARTBEAT_QUIET` | `false` | Suppress heartbeat logs when no candidates are found. |
| `REAPER_KILL` | `true` | If `false`, the reaper only logs and collects forensic data (audit mode). |
| `REAPER_UV_DIR` | *(none)* | Path to UniVerse installation directory (enables deep forensic collection for `uvapi_slave` processes). |
| `REAPER_UV_DEBUG` | *(auto‑detected)* | Path to UniVerse debug directory (read from `serverdebug`). |
| `REAPER_RETENTION_DAYS` | `30` | Forensic file retention in days (automatic cleanup). |
**Filtering logic:** The reaper now only selects processes that are *orphaned* (parent PID = 1) and have been running longer than `REAPER_MIN_UPTIME` minutes. This prevents killing short‑lived or child processes that still have a living parent.

**Example service file snippet** (`/lib/systemd/system/process-reaper.service`):
```ini
[Service]
Environment=REAPER_PATTERN=python3.*myapp.*
Environment=REAPER_INTERVAL=30
Environment=REAPER_LOG_DIR=/var/log/process-reaper
Environment=REAPER_GRACE_PERIOD=5
Environment=REAPER_MIN_UPTIME=5
Environment=REAPER_HEARTBEAT_QUIET=false
Environment=REAPER_KILL=true
Environment=REAPER_UV_DIR=/opt/uv
Environment=REAPER_UV_DEBUG=/opt/uv/debug
```


## UniVerse Deep Forensic Integration

When `REAPER_UV_DIR` is set and the pattern matches `uvapi_slave`, the reaper performs extended forensic collection for UniVerse processes:

1. **Initialization** – Reads `serverdebug` inside `REAPER_UV_DIR` to locate the debug directory (`REAPER_UV_DEBUG`).
2. **Extended forensic fields** (added to the JSON report):
   - `port_status` – Output of `bin/port.status PID … LAYER.STACK FILEMAP`.
   - `list_readu` – Lock information for the process's USERNO.
   - `user_no` – USERNO extracted via `bin/listuser` or `bin/list_readu`.
   - `uv_debug_file` – Name of the UniVerse debug file containing the PID.
   - `uv_error` – Last `returncode=` found in the debug file.
   - `uv_file` – Last `arg[0]=` (file path) from the debug file.
   - `uv_ps` – Output of `ps -fp PID --no-headers`.
3. **Debug file preservation** – The relevant debug file is copied to `REAPER_LOG_DIR` and compressed (`.gz`).

All UniVerse commands are executed with the working directory set to `REAPER_UV_DIR`. If a command fails or a path is missing, the error is logged and the reaper continues (no blocking).

## Usage Examples

| Scenario | Pattern | Description |
|----------|---------|-------------|
| **UVAPI Slave processes** | `uvapi_slave` | Cleans up child processes of the API that became orphaned after a main process crash. |
| **Python generic workers** | `python3.*worker\.py` | Locates worker scripts that often hang in background tasks. |
| **Java leaking processes** | `java.*-jar.*myapp\.jar` | For Java applications that fail to close threads properly after a crash. |

## Pattern Examples

| Use case | Pattern | Notes |
|----------|---------|-------|
| **Python workers stuck in a loop** | `python3.*worker.*` | Matches any Python command containing “worker”. |
| **Java processes leaking memory** | `java.*-Xmx4G` | Targets Java processes with a specific heap flag. |
| **Zombie cron jobs** | `bash.*/home/.*/script.sh` | Catches user‑space bash scripts that have hung. |
| **Custom binary with a known path** | `/opt/myapp/bin/daemon` | Exact path matching. |
| **Multiple patterns** | `(python3.*flask|node.*server)` | Combined regex for several application types. |

## Forensic Data

Before sending SIGTERM, the reaper collects the following information and writes it as a timestamped JSON file in `REAPER_LOG_DIR` (e.g., `reaper_12345_20260306_183456.json`):

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

## Audit Log

All actions are recorded in `REAPER_LOG_DIR/process-reaper-audit.log` with the following format:
```
[2026-03-06T18:34:56Z] action=forensic pid=12345 collection=success
[2026-03-06T18:34:56Z] action=kill pid=12345 signal=SIGTERM status=success
[2026-03-06T18:34:58Z] action=terminated pid=12345 process exited after SIGTERM
[2026-03-06T18:34:59Z] action=scan pid= found 0 matching processes
```

## Safety Mechanisms

1. **Self‑exclusion** – The reaper never selects its own PID.
2. **PID 1 protection** – The init/systemd process is always excluded, even if the pattern matches.
3. **Grace period** – Processes that exit cleanly after SIGTERM are not forcefully killed.
4. **Error‑handling** – Failures in forensic collection or signalling are logged and do not crash the daemon.

## Development

### Building a static binary
```bash
CGO_ENABLED=0 go build -o process-reaper ./cmd/process-reaper
```

### Running the integration test
```bash
bash test/fire_test.sh
```

### Creating distribution packages
Install `nfpm` (`go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest`), then:
```bash
# Build binary first
CGO_ENABLED=0 go build -o build/process-reaper ./cmd/process-reaper
# Generate .deb and .rpm
nfpm package --config nfpm.yaml --target dist/ --packager deb
nfpm package --config nfpm.yaml --target dist/ --packager rpm
```

### Project structure
```
process-reaper/
├── cmd/process-reaper/main.go          # Daemon entry point
├── internal/config/config.go           # Environment variable parsing
├── internal/reaper/scanner.go          # PID scanning with regex & exclusions
├── internal/reaper/killer.go           # SIGTERM → grace → SIGKILL logic
├── internal/forensic/recorder.go       # Forensic JSON writer
├── internal/logging/audit.go           # Audit log manager
├── test/                              # Integration tests
├── scripts/                           # Packaging scripts
├── nfpm.yaml                          # Package configuration
├── process-reaper.service             # Systemd unit
└── go.mod (gopsutil/v3 dependency)
```

## License

MIT – see [LICENSE](LICENSE) file.

## Author

Ari Ben Canaan (🦞) – Senior Systems Programmer & DevOps Engineer – [clawbotari](https://github.com/clawbotari)