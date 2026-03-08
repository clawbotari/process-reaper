package reaper

import (
	"fmt"
	"os"
	"syscall"
	"time"

	"process-reaper/internal/forensic"
	"process-reaper/internal/logging"
)

// Killer terminates processes with a grace period between SIGTERM and SIGKILL.
// If KillEnabled is false, it only records forensic data (audit mode).
type Killer struct {
	GracePeriod time.Duration
	LogDir      string
	Audit       *logging.Audit
	KillEnabled bool
	UVDir       string
	UVDebug     string
}

// NewKiller creates a Killer with the given configuration.
func NewKiller(gracePeriod time.Duration, logDir string, audit *logging.Audit, killEnabled bool, uvDir, uvDebug string) *Killer {
	return &Killer{
		GracePeriod: gracePeriod,
		LogDir:      logDir,
		Audit:       audit,
		KillEnabled: killEnabled,
		UVDir:       uvDir,
		UVDebug:     uvDebug,
	}
}

// Kill terminates a process after recording forensic data.
// If KillEnabled is false, it only records forensic data and logs an audit message.
func (k *Killer) Kill(pid int32) error {
	// 1. Forensic recording (including UniVerse data if configured)
	if err := forensic.Record(k.LogDir, k.UVDir, k.UVDebug, pid); err != nil {
		k.Audit.LogForensic(pid, false)
		return fmt.Errorf("forensic recording failed for PID %d: %w", pid, err)
	}
	k.Audit.LogForensic(pid, true)

	// Audit mode: do not send signals
	if !k.KillEnabled {
		k.Audit.Log("audit", fmt.Sprintf("%d", pid), "process identified but not killed")
		return nil
	}

	// 2. Send SIGTERM
	if err := k.sendSignal(pid, syscall.SIGTERM); err != nil {
		k.Audit.LogKill(pid, "SIGTERM", false, err.Error())
		return fmt.Errorf("SIGTERM failed for PID %d: %w", pid, err)
	}
	k.Audit.LogKill(pid, "SIGTERM", true, "")

	// 3. Wait grace period, check if process still exists
	if k.GracePeriod > 0 {
		time.Sleep(k.GracePeriod)
	}

	// 4. If still alive, send SIGKILL
	if k.isAlive(pid) {
		if err := k.sendSignal(pid, syscall.SIGKILL); err != nil {
			k.Audit.LogKill(pid, "SIGKILL", false, err.Error())
			return fmt.Errorf("SIGKILL failed for PID %d: %w", pid, err)
		}
		k.Audit.LogKill(pid, "SIGKILL", true, "")
	} else {
		k.Audit.Log("terminated", fmt.Sprintf("%d", pid), "process exited after SIGTERM")
	}

	return nil
}

func (k *Killer) sendSignal(pid int32, sig syscall.Signal) error {
	proc, err := os.FindProcess(int(pid))
	if err != nil {
		return fmt.Errorf("cannot find process %d: %w", pid, err)
	}
	return proc.Signal(sig)
}

func (k *Killer) isAlive(pid int32) bool {
	proc, err := os.FindProcess(int(pid))
	if err != nil {
		return false
	}
	// Sending signal 0 checks if process exists
	return proc.Signal(syscall.Signal(0)) == nil
}

// Log is a helper for audit logging.
func (k *Killer) Log(pid int32, action, details string) {
	k.Audit.Log(action, fmt.Sprintf("%d", pid), details)
}
