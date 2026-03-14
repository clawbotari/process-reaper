package forensic

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

// Record writes a JSON file with forensic data about a process before termination.
// If uvDir is non‑empty, UniVerse‑specific data is also collected.
// Record writes a JSON file with forensic data about a process before termination.
// If uvDir is non‑empty, UniVerse‑specific data is also collected.
func Record(logDir, uvDir, uvDebug string, pid int32, debug bool) error {
	p, err := process.NewProcess(pid)
	if err != nil {
		return fmt.Errorf("cannot open process %d: %w", pid, err)
	}

	info := collectFullInfo(p)
	info.Timestamp = time.Now() // local time

	// Forensic files go into a dedicated subdirectory
	forensicDir := filepath.Join(logDir, "forensics")
	if err := os.MkdirAll(forensicDir, 0755); err != nil {
		return fmt.Errorf("cannot create forensic directory %s: %w", forensicDir, err)
	}

	// Collect UniVerse data if directory provided
	if uvDir != "" {
		uv := CollectUVData(pid, uvDir, uvDebug, debug)
		info.UVData = &uv
		// If a debug file was found, copy it compressed to forensicDir
		if uv.UVDebugFile != "" {
			debugPath := filepath.Join(uvDebug, uv.UVDebugFile)
			if _, err := os.Stat(debugPath); err == nil {
				copied, err := CopyDebugFile(debugPath, forensicDir)
				if err == nil {
					info.UVData.UVDebugFile = copied // update to compressed name
				}
			}
		}
	}

	// Write JSON file with local timestamp in filename
	filename := filepath.Join(forensicDir, fmt.Sprintf("reaper_%d_%s.json",
		pid, info.Timestamp.Format("20060102_150405")))
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot marshal JSON: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("cannot write forensic file %s: %w", filename, err)
	}
	return nil
}


// ForensicInfo contains all collected data.
type ForensicInfo struct {
	PID         int32     `json:"pid"`
	Cmdline     string    `json:"cmdline"`
	Executable  string    `json:"executable"`
	RSS         uint64    `json:"rss_bytes"`
	VMS         uint64    `json:"vms_bytes"`
	OpenFiles   []string  `json:"open_files"`
	CreateTime  int64     `json:"create_time"`
	CPUPercent  float64   `json:"cpu_percent"`
	Username    string    `json:"username"`
	Status      []string  `json:"status"`
	Timestamp   time.Time `json:"timestamp"`
	UVData      *UVData   `json:"uv_data,omitempty"`
}

func collectFullInfo(p *process.Process) ForensicInfo {
	cmdline, _ := p.Cmdline()
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
	user, _ := p.Username()
	status, _ := p.Status()

	return ForensicInfo{
		PID:        p.Pid,
		Cmdline:    cmdline,
		Executable: exe,
		RSS:        mem.RSS,
		VMS:        mem.VMS,
		OpenFiles:  openFiles,
		CreateTime: create,
		CPUPercent: cpu,
		Username:   user,
		Status:     status,
	}
}
