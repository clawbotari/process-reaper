package forensic

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindDebugFile(t *testing.T) {
	debugDir := t.TempDir()
	other := filepath.Join(debugDir, "other.log")
	match := filepath.Join(debugDir, "uvcs_001")

	if err := os.WriteFile(other, []byte("no matching pid here"), 0644); err != nil {
		t.Fatalf("write other file: %v", err)
	}
	if err := os.WriteFile(match, []byte("header\npid 12345 footer\n"), 0644); err != nil {
		t.Fatalf("write match file: %v", err)
	}

	found, err := findDebugFile(12345, debugDir)
	if err != nil {
		t.Fatalf("findDebugFile() returned unexpected error: %v", err)
	}
	if found != match {
		t.Fatalf("expected %q, got %q", match, found)
	}
}

func TestContainsPIDToken(t *testing.T) {
	if !containsPIDToken("header\npid 12345 footer\n", "12345") {
		t.Fatal("expected token match")
	}
	if containsPIDToken("pid 123456 footer", "12345") {
		t.Fatal("did not expect partial token match")
	}
}
