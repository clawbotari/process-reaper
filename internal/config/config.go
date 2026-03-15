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
	DefaultPattern      = `.*`
	DefaultInterval     = 60  // seconds
	DefaultLogDir       = "/var/log/process-reaper"
	DefaultGracePeriod  = 10  // seconds
	DefaultMinUptime    = 5   // minutes
	DefaultRetentionDays = 30  // days
	DefaultDebugForensic = false // REAPER_DEBUG_FORENSIC: log detailed forensic command errors
	DefaultUVDir        = "/usr/uv" // REAPER_UV_DIR default installation path
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
	DebugForensic bool           // REAPER_DEBUG_FORENSIC: log detailed forensic command errors
	UVDir          string         // REAPER_UV_DIR: UniVerse installation directory (optional)
	UVDebug        string         // REAPER_UV_DEBUG: UniVerse debug directory (extracted from serverdebug)
	RetentionDays   int           // REAPER_RETENTION_DAYS: forensic file retention in days
}

// Load reads environment variables and returns a validated Config.
func Load() (*Config, error) {
	// REAPER_PATTERN
	patternStr := getEnvOrDefault("REAPER_PATTERN", DefaultPattern)
	pattern, err := regexp.Compile(patternStr)
	if err != nil {
		return nil, fmt.Errorf("invalid REAPER_PATTERN regex %q: %w", patternStr, err)
	}

	// REAPER_INTERVAL
	intervalSec := parseIntEnv("REAPER_INTERVAL", DefaultInterval)
	if intervalSec < 1 {
		return nil, fmt.Errorf("REAPER_INTERVAL must be >= 1 second, got %d", intervalSec)
	}
	interval := time.Duration(intervalSec) * time.Second

	// REAPER_LOG_DIR
	logDir := getEnvOrDefault("REAPER_LOG_DIR", DefaultLogDir)
	if logDir == "" {
		return nil, fmt.Errorf("REAPER_LOG_DIR cannot be empty")
	}

	// REAPER_GRACE_PERIOD
	graceSec := parseIntEnv("REAPER_GRACE_PERIOD", DefaultGracePeriod)
	if graceSec < 0 {
		return nil, fmt.Errorf("REAPER_GRACE_PERIOD must be >= 0, got %d", graceSec)
	}
	gracePeriod := time.Duration(graceSec) * time.Second

	// REAPER_MIN_UPTIME
	minUptimeMin := parseIntEnv("REAPER_MIN_UPTIME", DefaultMinUptime)
	if minUptimeMin < 0 {
		return nil, fmt.Errorf("REAPER_MIN_UPTIME must be >= 0, got %d", minUptimeMin)
	}
	minUptime := time.Duration(minUptimeMin) * time.Minute

	// REAPER_HEARTBEAT_QUIET
	heartbeatQuiet := parseBoolEnv("REAPER_HEARTBEAT_QUIET", false)

	// REAPER_KILL (default true for backward compatibility)
	kill := parseBoolEnv("REAPER_KILL", true)

	// REAPER_UV_DIR (optional)
	// REAPER_DEBUG_FORENSIC
	debugForensic := parseBoolEnv("REAPER_DEBUG_FORENSIC", DefaultDebugForensic)
	uvDirEnv := strings.TrimRight(os.Getenv("REAPER_UV_DIR"), "/")
	uvDir := uvDirEnv
	if uvDir == "" {
		uvDir = DefaultUVDir
	}
	var uvDebug string
	if uvDir != "" {
		// Verify directory exists
		if _, err := os.Stat(uvDir); err != nil {
			return nil, fmt.Errorf("REAPER_UV_DIR %q does not exist or is inaccessible: %w", uvDir, err)
		}
		// Try to read serverdebug file
		debugPath, err := extractUVDebugPath(uvDir)
		if err != nil {
			// Log but don't fail; UVDebug will remain empty
			uvDebug = ""
		} else {
			uvDebug = debugPath
		}
	}

	
	// REAPER_RETENTION_DAYS
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
		UVDir:          uvDir,
		UVDebug:        uvDebug,
		DebugForensic: debugForensic,
		RetentionDays:   retentionDays,
	}, nil
}

// UVEnabled returns true if UniVerse integration is configured.
func (c *Config) UVEnabled() bool {
	return c.UVDir != ""
}

// UVPatternMatches returns true if the configured pattern is likely targeting UVAPI slaves.
func (c *Config) UVPatternMatches() bool {
	// Simple heuristic: pattern contains "uvapi_slave"
	return strings.Contains(c.Pattern.String(), "uvapi_slave")
}

// extractUVDebugPath reads the serverdebug file inside uvDir and extracts the debug directory path.
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
		if len(fields) >= 3 {
			debugFile := fields[2]
			// If path is relative, make it absolute relative to uvDir
			if !filepath.IsAbs(debugFile) {
				debugFile = filepath.Join(uvDir, debugFile)
			}
			// We need the directory containing the debug file(s)
			debugDir := filepath.Dir(debugFile)
			return debugDir, nil
		}
	}
	return "", fmt.Errorf("no valid three‑column line found in %s", serverdebug)
}

// getEnvOrDefault returns the environment variable value or a default.
func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// parseIntEnv parses an integer environment variable, returns default on missing/invalid.
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

// parseBoolEnv parses a boolean environment variable.
// "true", "1", "yes", "on" (case-insensitive) => true
// "false", "0", "no", "off" => false
// missing or invalid => defaultValue
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
