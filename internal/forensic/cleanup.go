package forensic

import (
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// CleanupForensics deletes forensic files older than retentionDays from the forensic directory.
// The forensic directory is assumed to be <logDir>/forensics.
func CleanupForensics(logDir string, retentionDays int) (int, error) {
	forensicDir := filepath.Join(logDir, "forensics")
	// Ensure forensic directory exists (if not, nothing to clean)
	if _, err := os.Stat(forensicDir); os.IsNotExist(err) {
		return 0, nil
	}

	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	var deleted int

	err := filepath.WalkDir(forensicDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip problematic entries
		}
		if d.IsDir() {
			return nil
		}
		// Only consider .json and .gz files (our forensic outputs)
		ext := filepath.Ext(path)
		if ext != ".json" && ext != ".gz" {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.ModTime().Before(cutoff) {
			if err := os.Remove(path); err == nil {
				deleted++
			}
			// Log removal is done by the caller
		}
		return nil
	})
	return deleted, err
}
