package logging

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Audit logs reaper actions to a dedicated audit log file.
type Audit struct {
	logger *log.Logger
	file   *os.File
}

// NewAudit creates or opens the audit log file in the given directory.
func NewAudit(logDir string) (*Audit, error) {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("cannot create log directory %s: %w", logDir, err)
	}
	logPath := filepath.Join(logDir, "process-reaper-audit.log")
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("cannot open audit log %s: %w", logPath, err)
	}
	logger := log.New(file, "", 0)
	return &Audit{logger: logger, file: file}, nil
}

// Close closes the audit log file.
func (a *Audit) Close() error {
	return a.file.Close()
}

// Log writes an audit entry with the given action and details.
func (a *Audit) Log(action, pid, details string) {
	entry := fmt.Sprintf("[%s] action=%s pid=%s %s", time.Now().UTC().Format(time.RFC3339), action, pid, details)
	a.logger.Println(entry)
}

// LogScan logs a scan event.
func (a *Audit) LogScan(count int) {
	a.Log("scan", "", fmt.Sprintf("found %d matching processes", count))
}

// LogKill logs a kill event (SIGTERM/SIGKILL).
func (a *Audit) LogKill(pid int32, signal string, success bool, errMsg string) {
	status := "success"
	if !success {
		status = "failed"
	}
	details := fmt.Sprintf("signal=%s status=%s", signal, status)
	if errMsg != "" {
		details += " error=" + errMsg
	}
	a.Log("kill", fmt.Sprintf("%d", pid), details)
}

// LogForensic logs forensic data collection.
func (a *Audit) LogForensic(pid int32, success bool) {
	status := "success"
	if !success {
		status = "failed"
	}
	a.Log("forensic", fmt.Sprintf("%d", pid), "collection="+status)
}
