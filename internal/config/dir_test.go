package config

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestDir_Default(t *testing.T) {
	// Clear overrides
	t.Setenv("TIMBERS_CONFIG_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")

	dir := Dir()
	if dir == "" {
		t.Fatal("Dir() returned empty string")
	}

	if runtime.GOOS != "windows" {
		if filepath.Base(dir) != "timbers" {
			t.Errorf("Dir() = %q, want path ending in 'timbers'", dir)
		}
	}
}

func TestDir_ExplicitOverride(t *testing.T) {
	t.Setenv("TIMBERS_CONFIG_HOME", "/custom/path")
	if got := Dir(); got != "/custom/path" {
		t.Errorf("Dir() = %q, want %q", got, "/custom/path")
	}
}

func TestDir_XDGOverride(t *testing.T) {
	t.Setenv("TIMBERS_CONFIG_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "/xdg/config")
	if got := Dir(); got != filepath.Join("/xdg/config", "timbers") {
		t.Errorf("Dir() = %q, want %q", got, filepath.Join("/xdg/config", "timbers"))
	}
}
