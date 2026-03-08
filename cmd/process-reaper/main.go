package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"process-reaper/internal/config"
	"process-reaper/internal/logging"
	"process-reaper/internal/reaper"
)

const version = "1.2.0"

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("Intelligent Process Reaper v%s starting", version)

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}
	log.Printf("Configuration loaded: pattern=%s interval=%v log_dir=%s grace=%v min_uptime=%v",
		cfg.Pattern, cfg.Interval, cfg.LogDir, cfg.GracePeriod, cfg.MinUptime)
	if cfg.HeartbeatQuiet {
		log.Print("Heartbeat logs are suppressed (REAPER_HEARTBEAT_QUIET=true)")
	}
	if !cfg.Kill {
		log.Print("AUDIT MODE enabled – processes will be identified but NOT killed (REAPER_KILL=false)")
	}
	if cfg.UVEnabled() {
		log.Printf("UniVerse integration enabled: base=%s debug=%s", cfg.UVDir, cfg.UVDebug)
		if cfg.UVPatternMatches() {
			log.Print("Pattern matches 'uvapi_slave', UniVerse forensic extensions will be applied")
		}
	}

	audit, err := logging.NewAudit(cfg.LogDir)
	if err != nil {
		log.Fatalf("Failed to initialize audit log: %v", err)
	}
	defer audit.Close()

	killer := reaper.NewKiller(cfg.GracePeriod, cfg.LogDir, audit, cfg.Kill, cfg.UVDir, cfg.UVDebug)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle termination signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		log.Printf("Received signal %v, shutting down gracefully", sig)
		cancel()
	}()

	log.Printf("Starting scan loop with interval %v", cfg.Interval)
	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Shutdown completed")
			return
		case <-ticker.C:
			if err := scanAndKill(cfg, killer, audit); err != nil {
				log.Printf("Scan/kill cycle failed: %v", err)
			}
		}
	}
}

func scanAndKill(cfg *config.Config, killer *reaper.Killer, audit *logging.Audit) error {
	selfPID := int32(os.Getpid())
	// Require orphan processes (PPID == 1) and minimum uptime
	matches, err := reaper.Scan(cfg.Pattern, cfg.MinUptime, true, selfPID)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}
	audit.LogScan(len(matches))

	if len(matches) == 0 {
		if !cfg.HeartbeatQuiet {
			log.Printf("[Heartbeat] Scan complete. No candidates found.")
		}
		return nil
	}

	for _, proc := range matches {
		log.Printf("Terminating PID %d (%s)", proc.PID, proc.Cmdline)
		if err := killer.Kill(proc.PID); err != nil {
			log.Printf("Failed to kill PID %d: %v", proc.PID, err)
		}
	}
	return nil
}
