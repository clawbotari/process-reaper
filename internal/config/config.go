package config

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"
)

const (
	DefaultPattern      = `.*`
	DefaultInterval     = 60  // seconds
	DefaultLogDir       = "/var/log/process-reaper"
	DefaultGracePeriod  = 10  // seconds
	DefaultMinUptime    = 5   // minutes
)

// Config holds all reaper configuration parsed from environment variables.
type Config struct {
	Pattern     *regexp.Regexp // REAPER_PATTERN: regex to match process command lines
	Interval    time.Duration  // REAPER_INTERVAL: scan frequency in seconds
	LogDir      string         // REAPER_LOG_DIR: directory for logs and forensic JSON
	GracePeriod time.Duration  // REAPER_GRACE_PERIOD: seconds between SIGTERM and SIGKILL
	MinUptime   time.Duration  // REAPER_MIN_UPTIME: minimum process age in minutes
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

	return &Config{
		Pattern:     pattern,
		Interval:    interval,
		LogDir:      logDir,
		GracePeriod: gracePeriod,
		MinUptime:   minUptime,
	}, nil
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
