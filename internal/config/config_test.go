package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDisablesUVByDefault(t *testing.T) {
	t.Setenv("REAPER_UV_DIR", "")
	t.Setenv("REAPER_UV_DEBUG", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}
	if cfg.UVDir != "" {
		t.Fatalf("expected UVDir to be empty, got %q", cfg.UVDir)
	}
	if cfg.UVDebug != "" {
		t.Fatalf("expected UVDebug to be empty, got %q", cfg.UVDebug)
	}
}

func TestLoadEnablesUVWhenDirectoryIsConfigured(t *testing.T) {
	uvDir := t.TempDir()
	debugDir := filepath.Join(uvDir, "uvdebug")
	if err := os.Mkdir(debugDir, 0755); err != nil {
		t.Fatalf("mkdir debug dir: %v", err)
	}
	serverdebug := []byte("uvcs 10 " + filepath.Join(debugDir, "uvcs_") + "\n")
	if err := os.WriteFile(filepath.Join(uvDir, "serverdebug"), serverdebug, 0644); err != nil {
		t.Fatalf("write serverdebug: %v", err)
	}

	t.Setenv("REAPER_UV_DIR", uvDir)
	t.Setenv("REAPER_UV_DEBUG", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}
	if cfg.UVDir != uvDir {
		t.Fatalf("expected UVDir %q, got %q", uvDir, cfg.UVDir)
	}
	if cfg.UVDebug != debugDir {
		t.Fatalf("expected UVDebug %q, got %q", debugDir, cfg.UVDebug)
	}
}

func TestLoadFailsForInvalidExplicitUVDir(t *testing.T) {
	t.Setenv("REAPER_UV_DIR", filepath.Join(t.TempDir(), "missing"))

	if _, err := Load(); err == nil {
		t.Fatal("expected Load() to fail for invalid REAPER_UV_DIR")
	}
}

func TestLoadHonorsExplicitUVDebugOverride(t *testing.T) {
	uvDir := t.TempDir()
	overrideDebug := t.TempDir()
	serverdebug := []byte("uvcs 10 /tmp/ignored/uvcs_\n")
	if err := os.WriteFile(filepath.Join(uvDir, "serverdebug"), serverdebug, 0644); err != nil {
		t.Fatalf("write serverdebug: %v", err)
	}

	t.Setenv("REAPER_UV_DIR", uvDir)
	t.Setenv("REAPER_UV_DEBUG", overrideDebug)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}
	if cfg.UVDebug != overrideDebug {
		t.Fatalf("expected explicit UVDebug %q, got %q", overrideDebug, cfg.UVDebug)
	}
}

func TestExtractUVDebugPath(t *testing.T) {
	uvDir := t.TempDir()
	debugDir := filepath.Join(uvDir, "uvdebug")
	if err := os.Mkdir(debugDir, 0755); err != nil {
		t.Fatalf("mkdir debug dir: %v", err)
	}
	serverdebug := []byte("uvcs 10 ./uvdebug/uvcs_\n")
	if err := os.WriteFile(filepath.Join(uvDir, "serverdebug"), serverdebug, 0644); err != nil {
		t.Fatalf("write serverdebug: %v", err)
	}

	got, err := extractUVDebugPath(uvDir)
	if err != nil {
		t.Fatalf("extractUVDebugPath() returned unexpected error: %v", err)
	}
	if got != debugDir {
		t.Fatalf("expected %q, got %q", debugDir, got)
	}
}

func TestExtractUVDebugPathErrors(t *testing.T) {
	uvDir := t.TempDir()

	if _, err := extractUVDebugPath(uvDir); err == nil {
		t.Fatal("expected missing serverdebug to fail")
	}

	if err := os.WriteFile(filepath.Join(uvDir, "serverdebug"), []byte("bad line\n"), 0644); err != nil {
		t.Fatalf("write malformed serverdebug: %v", err)
	}
	if _, err := extractUVDebugPath(uvDir); err == nil {
		t.Fatal("expected malformed serverdebug to fail")
	}
}
