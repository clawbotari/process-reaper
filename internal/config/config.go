package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultPattern       = `.*`
	DefaultInterval      = 60 // seconds
	DefaultLogDir        = "/var/log/process-reaper"
	DefaultGracePeriod   = 10    // seconds
	DefaultMinUptime     = 5     // minutes
	DefaultRetentionDays = 30    // days
	DefaultDebugForensic = false // REAPER_DEBUG_FORENSIC: log detailed forensic command errors
)

// Config holds all reaper configuration parsed from environment variables.
type Config struct {
	Pattern        *regexp.Regexp // REAPER_PATTERN: regex to match process command lines
	Interval       time.Duration  // REAPER_INTERVAL: scan frequency in seconds
	LogDir         string         // REAPER_LOG_DIR: directory for logs and forensic JSON
	GracePeriod    time.Duration  // REAPER_GRACE_PERIOD: seconds between SIGTERM and SIGKILL
	MinUptime      time.Duration  // REAPER_MIN_UPTIME: minimum process age in minutes
	HeartbeatQuiet bool           // REAPER_HEARTBEAT_QUIET: suppress heartbeat logs
	Kill           bool           // REAPER_KILL: actually send signals (false = audit mode)
	DebugForensic  bool           // REAPER_DEBUG_FORENSIC: log detailed forensic command errors
	UVDir          string         // REAPER_UV_DIR: UniVerse installation directory (optional)
	UVDebug        string         // REAPER_UV_DEBUG: UniVerse debug directory (override or extracted from serverdebug)
	RetentionDays  int            // REAPER_RETENTION_DAYS: forensic file retention in days
}

// Load reads environment variables and returns a validated Config.
func Load() (*Config, error) {
	patternStr := getEnvOrDefault("REAPER_PATTERN", DefaultPattern)
	pattern, err := regexp.Compile(patternStr)
	if err != nil {
		return nil, fmt.Errorf("invalid REAPER_PATTERN regex %q: %w", patternStr, err)
	}

	intervalSec := parseIntEnv("REAPER_INTERVAL", DefaultInterval)
	if intervalSec < 1 {
		return nil, fmt.Errorf("REAPER_INTERVAL must be >= 1 second, got %d", intervalSec)
	}
	interval := time.Duration(intervalSec) * time.Second

	logDir := getEnvOrDefault("REAPER_LOG_DIR", DefaultLogDir)
	if logDir == "" {
		return nil, fmt.Errorf("REAPER_LOG_DIR cannot be empty")
	}

	graceSec := parseIntEnv("REAPER_GRACE_PERIOD", DefaultGracePeriod)
	if graceSec < 0 {
		return nil, fmt.Errorf("REAPER_GRACE_PERIOD must be >= 0, got %d", graceSec)
	}
	gracePeriod := time.Duration(graceSec) * time.Second

	minUptimeMin := parseIntEnv("REAPER_MIN_UPTIME", DefaultMinUptime)
	if minUptimeMin < 0 {
		return nil, fmt.Errorf("REAPER_MIN_UPTIME must be >= 0, got %d", minUptimeMin)
	}
	minUptime := time.Duration(minUptimeMin) * time.Minute

	heartbeatQuiet := parseBoolEnv("REAPER_HEARTBEAT_QUIET", false)
	kill := parseBoolEnv("REAPER_KILL", true)
	debugForensic := parseBoolEnv("REAPER_DEBUG_FORENSIC", DefaultDebugForensic)

	uvDir := strings.TrimRight(os.Getenv("REAPER_UV_DIR"), "/")
	uvDebug := strings.TrimRight(os.Getenv("REAPER_UV_DEBUG"), "/")
	if uvDir != "" {
		dirInfo, err := os.Stat(uvDir)
		if err != nil {
			return nil, fmt.Errorf("REAPER_UV_DIR %q does not exist or is inaccessible: %w", uvDir, err)
		}
		if !dirInfo.IsDir() {
			return nil, fmt.Errorf("REAPER_UV_DIR %q is not a directory", uvDir)
		}

		if uvDebug != "" {
			debugInfo, err := os.Stat(uvDebug)
			if err != nil {
				return nil, fmt.Errorf("REAPER_UV_DEBUG %q does not exist or is inaccessible: %w", uvDebug, err)
			}
			if !debugInfo.IsDir() {
				return nil, fmt.Errorf("REAPER_UV_DEBUG %q is not a directory", uvDebug)
			}
		} else {
			debugPath, err := extractUVDebugPath(uvDir)
			if err == nil {
				uvDebug = debugPath
			}
		}
	}

	retentionDays := parseIntEnv("REAPER_RETENTION_DAYS", DefaultRetentionDays)
	if retentionDays < 0 {
		return nil, fmt.Errorf("REAPER_RETENTION_DAYS must be >= 0, got %d", retentionDays)
	}

	return &Config{
		Pattern:        pattern,
		Interval:       interval,
		LogDir:         logDir,
		GracePeriod:    gracePeriod,
		MinUptime:      minUptime,
		HeartbeatQuiet: heartbeatQuiet,
		Kill:           kill,
		DebugForensic:  debugForensic,
		UVDir:          uvDir,
		UVDebug:        uvDebug,
		RetentionDays:  retentionDays,
	}, nil
}

// UVEnabled returns true if UniVerse integration is configured.
func (c *Config) UVEnabled() bool {
	return c.UVDir != ""
}

// UVPatternMatches returns true if the configured pattern is likely targeting UVAPI slaves.
func (c *Config) UVPatternMatches() bool {
	return strings.Contains(c.Pattern.String(), "uvapi_slave")
}

// extractUVDebugPath reads the serverdebug file inside uvDir and extracts the debug directory path.
// The file format is: "uvcs 10 /usr/uv/uvdebug/uvcs_" (third column is full debug file path).
func extractUVDebugPath(uvDir string) (string, error) {
	serverdebug := filepath.Join(uvDir, "serverdebug")
	data, err := os.ReadFile(serverdebug)
	if err != nil {
		return "", fmt.Errorf("cannot read %s: %w", serverdebug, err)
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		debugFile := fields[2]
		if !filepath.IsAbs(debugFile) {
			debugFile = filepath.Join(uvDir, debugFile)
		}
		return filepath.Dir(debugFile), nil
	}
	return "", fmt.Errorf("no valid three-column line found in %s", serverdebug)
}

func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

func parseIntEnv(key string, defaultValue int) int {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultValue
	}
	val, err := strconv.Atoi(valStr)
	if err != nil || val < 0 {
		return defaultValue
	}
	return val
}

func parseBoolEnv(key string, defaultValue bool) bool {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultValue
	}
	valStr = strings.ToLower(valStr)
	switch valStr {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return defaultValue
	}
}
