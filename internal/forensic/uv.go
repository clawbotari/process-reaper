package forensic

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"log"
)

// UVData holds UniVerse‑specific forensic information.
type UVData struct {
	PortStatus  string `json:"port_status,omitempty"`
	ListReadU   string `json:"list_readu,omitempty"`
	UserNo      string `json:"user_no,omitempty"`
	UVDebugFile string `json:"uv_debug_file,omitempty"`
	UVError     string `json:"uv_error,omitempty"`
	UVFile      string `json:"uv_file,omitempty"`
}

// CollectUVData gathers UniVerse‑specific information for a process.
// If uvDir is empty or any command fails, the function returns an empty UVData
// (no error is propagated, to avoid blocking the reaper).
func CollectUVData(pid int32, uvDir, uvDebug string, debug bool) UVData {
	var data UVData
	if uvDir == "" {
		return data
	}

	// Ensure commands are executed inside the UniVerse installation directory
	// Debug: log original environment if debug is enabled
	if debug {
		log.Printf("[DEBUG forensic] original environment (%d variables):", len(os.Environ()))
		for _, e := range os.Environ() {
			log.Printf("[DEBUG forensic]   %s", e)
		}
	}

	runUV := func(uvBin string, args ...string) (stdout, stderr string, err error) {
		// Use /usr/bin/env -i to ensure completely clean environment
		envArgs := []string{"-i", "TERM=vt100", uvBin}
		envArgs = append(envArgs, args...)
		c := exec.Command("/usr/bin/env", envArgs...)
		c.Dir = uvDir
		c.Env = []string{} // reinforce empty environment
		var outBuf, errBuf bytes.Buffer
		c.Stdout = &outBuf
		c.Stderr = &errBuf
		err = c.Run()
		stdout = strings.TrimSpace(outBuf.String())
		stderr = strings.TrimSpace(errBuf.String())
		if debug && (err != nil || stdout == "") {
						// Build the exact command line as it would appear in a shell
						cmdLine := strings.Join(append([]string{"/usr/bin/env", "-i", "TERM=vt100", uvBin}, args...), " ")
						log.Printf("[DEBUG forensic] cmd=%s", cmdLine)
						log.Printf("[DEBUG forensic] working dir=%s", uvDir)
						log.Printf("[DEBUG forensic] stdout=%q stderr=%q error=%v", stdout, stderr, err)
		}
		return
	}
	// 1. port.status
	stdout, stderr, err := runUV(filepath.Join(uvDir, "bin", "port.status"), "PID", fmt.Sprintf("%d", pid), "LAYER.STACK", "FILEMAP")
	if err != nil || stdout == "" {
		data.PortStatus = "No port status info or command failed"
		if debug {
			log.Printf("[DEBUG forensic] port.status failed: err=%v stderr=%q", err, stderr)
		}
	} else {
		data.PortStatus = stdout
	}
	// 2. listuser / list_readu to find USERNO
	userNo := ""
	if stdout, _, err := runUV(filepath.Join(uvDir, "bin", "listuser")); err == nil {
		scanner := bufio.NewScanner(strings.NewReader(stdout))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, fmt.Sprintf("%d", pid)) {
				fields := strings.Fields(line)
				if len(fields) > 0 {
					userNo = fields[0]
					break
				}
			}
		}
	}
	if userNo == "" {
		if stdout, _, err := runUV(filepath.Join(uvDir, "bin", "list_readu")); err == nil {
			scanner := bufio.NewScanner(strings.NewReader(stdout))
			for scanner.Scan() {
				line := scanner.Text()
				if strings.Contains(line, fmt.Sprintf("%d", pid)) {
					fields := strings.Fields(line)
					if len(fields) > 0 {
						userNo = fields[0]
						break
					}
				}
			}
		}
	}
	data.UserNo = userNo

	// 3. list_readu every USER $USERNO
	if userNo != "" {
		if stdout, _, err := runUV(filepath.Join(uvDir, "bin", "list_readu"), "every", "USER", userNo); err == nil {
			data.ListReadU = stdout
		}
	} else {
		data.ListReadU = "No locks found"
	}

	// 4. Search for debug file containing the PID
	if uvDebug != "" {
		debugFile, err := findDebugFile(pid, uvDebug)
		if err == nil && debugFile != "" {
			data.UVDebugFile = filepath.Base(debugFile)
			// Copy debug file compressed to log directory (if logDir known, caller will handle)
			// Extract error and file info
			uvError, uvFile := extractDebugInfo(debugFile)
			data.UVError = uvError
			data.UVFile = uvFile
		}
	}

	return data
}

// findDebugFile searches recursively in uvDebug for a file that contains the PID.
func findDebugFile(pid int32, uvDebug string) (string, error) {
	// Use grep -l for efficiency
	cmd := exec.Command("grep", "-l", fmt.Sprintf("\\b%d\\b", pid), "-r", uvDebug)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("grep failed: %w", err)
	}
	firstLine := strings.Split(out.String(), "\n")[0]
	return strings.TrimSpace(firstLine), nil
}

// extractDebugInfo reads the debug file and extracts returncode= and arg[0]= from last 20 lines.
func extractDebugInfo(path string) (errorStr, fileStr string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", ""
	}
	lines := strings.Split(string(data), "\n")
	// Search from bottom up
	start := len(lines) - 20
	if start < 0 {
		start = 0
	}
	for i := len(lines) - 1; i >= start; i-- {
		line := lines[i]
		if strings.Contains(line, "returncode=") && errorStr == "" {
			if idx := strings.Index(line, "returncode="); idx != -1 {
				errorStr = strings.TrimSpace(line[idx+len("returncode="):])
			}
		}
		if strings.Contains(line, "arg[0]=") && fileStr == "" {
			if idx := strings.Index(line, "arg[0]="); idx != -1 {
				fileStr = strings.TrimSpace(line[idx+len("arg[0]="):])
			}
		}
	}
	return errorStr, fileStr
}

// CopyDebugFile copies the debug file to logDir, compressed with gzip.
// Returns the destination file name (relative to logDir) if successful.
func CopyDebugFile(srcDebugFile, logDir string) (string, error) {
	if srcDebugFile == "" {
		return "", fmt.Errorf("no debug file provided")
	}
	data, err := os.ReadFile(srcDebugFile)
	if err != nil {
		return "", fmt.Errorf("cannot read debug file: %w", err)
	}
	destName := filepath.Join(logDir, filepath.Base(srcDebugFile)+".gz")
	dest, err := os.Create(destName)
	if err != nil {
		return "", fmt.Errorf("cannot create gzip file: %w", err)
	}
	defer dest.Close()
	gz := gzip.NewWriter(dest)
	if _, err := gz.Write(data); err != nil {
		return "", fmt.Errorf("cannot write compressed data: %w", err)
	}
	if err := gz.Close(); err != nil {
		return "", fmt.Errorf("cannot close gzip writer: %w", err)
	}
	return filepath.Base(destName), nil
}
