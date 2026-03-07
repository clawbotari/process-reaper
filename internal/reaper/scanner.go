package reaper

import (
	"fmt"
	"regexp"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

// ProcessInfo holds forensic data about a process.
type ProcessInfo struct {
	PID         int32    `json:"pid"`
	Cmdline     string   `json:"cmdline"`
	Executable  string   `json:"executable,omitempty"`
	RSS         uint64   `json:"rss_bytes"`  // Resident Set Size
	VMS         uint64   `json:"vms_bytes"`  // Virtual Memory Size
	OpenFiles   []string `json:"open_files,omitempty"`
	CreateTime  int64    `json:"create_time"`
	CPUPercent  float64  `json:"cpu_percent,omitempty"`
}

// Scan returns all processes whose command line matches the given regex,
// excluding the specified PIDs (typically the reaper's own PID and PID 1).
// Additional filters: minimum uptime (minUptime) and require orphan (PPID == 1).
func Scan(pattern *regexp.Regexp, minUptime time.Duration, requireOrphan bool, excludePID ...int32) ([]ProcessInfo, error) {
	pids, err := process.Pids()
	if err != nil {
		return nil, fmt.Errorf("failed to list PIDs: %w", err)
	}

	// Build set of excluded PIDs
	excluded := make(map[int32]bool)
	for _, pid := range excludePID {
		if pid > 0 {
			excluded[pid] = true
		}
	}
	// Always exclude PID 1 (systemd/init)
	excluded[1] = true

	var matches []ProcessInfo
	for _, pid := range pids {
		if excluded[pid] {
			continue
		}
		p, err := process.NewProcess(pid)
		if err != nil {
			// Process may have exited between listing and opening
			continue
		}

		cmdline, err := p.Cmdline()
		if err != nil {
			// Some processes hide their cmdline (kernel threads)
			cmdline = ""
		}

		if !pattern.MatchString(cmdline) {
			continue
		}

		// Check PPID == 1 if required
		if requireOrphan {
			ppid, err := p.Ppid()
			if err != nil || ppid != 1 {
				continue
			}
		}

		// Check minimum uptime
		if minUptime > 0 {
			create, err := p.CreateTime()
			if err != nil {
				continue
			}
			uptime := time.Since(time.Unix(create/1000, 0))
			if uptime < minUptime {
				continue
			}
		}

		info, err := collectForensicData(p, cmdline)
		if err != nil {
			continue
		}
		matches = append(matches, info)
	}
	return matches, nil
}

func collectForensicData(p *process.Process, cmdline string) (ProcessInfo, error) {
	exe, _ := p.Exe()
	mem, _ := p.MemoryInfo()
	if mem == nil {
		mem = &process.MemoryInfoStat{}
	}
	create, _ := p.CreateTime()
	files, _ := p.OpenFiles()
	openFiles := make([]string, len(files))
	for i, f := range files {
		openFiles[i] = f.Path
	}
	cpu, _ := p.CPUPercent()

	return ProcessInfo{
		PID:        p.Pid,
		Cmdline:    cmdline,
		Executable: exe,
		RSS:        mem.RSS,
		VMS:        mem.VMS,
		OpenFiles:  openFiles,
		CreateTime: create,
		CPUPercent: cpu,
	}, nil
}
