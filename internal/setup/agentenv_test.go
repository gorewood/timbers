package setup

import (
	"testing"
)

func TestRegistryHasClaude(t *testing.T) {
	env := GetAgentEnv("claude")
	if env == nil {
		t.Fatal("claude agent env should be registered")
	}
	if env.Name() != "claude" {
		t.Errorf("Name() = %q, want %q", env.Name(), "claude")
	}
	if env.DisplayName() != "Claude Code" {
		t.Errorf("DisplayName() = %q, want %q", env.DisplayName(), "Claude Code")
	}
}

func TestGetAgentEnvUnknown(t *testing.T) {
	if GetAgentEnv("nonexistent") != nil {
		t.Error("GetAgentEnv(\"nonexistent\") should return nil")
	}
}

func TestAllAgentEnvs(t *testing.T) {
	envs := AllAgentEnvs()
	if len(envs) == 0 {
		t.Fatal("AllAgentEnvs() should return at least one env")
	}
	// First should be claude (stable ordering).
	if envs[0].Name() != "claude" {
		t.Errorf("first env = %q, want %q", envs[0].Name(), "claude")
	}
}

func TestClaudeEnvDetectNotInstalled(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	env := &ClaudeEnv{}
	_, _, installed := env.Detect()
	if installed {
		t.Error("Detect() should return false when nothing is installed")
	}
}

func TestClaudeEnvInstallAndDetect(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	env := &ClaudeEnv{}

	// Install at global scope.
	path, err := env.Install(false)
	if err != nil {
		t.Fatalf("Install() error: %v", err)
	}
	if path == "" {
		t.Fatal("Install() returned empty path")
	}

	// Detect should find it.
	_, scope, installed := env.Detect()
	if !installed {
		t.Error("Detect() should return true after install")
	}
	if scope != "global" {
		t.Errorf("scope = %q, want %q", scope, "global")
	}
}

func TestClaudeEnvRemove(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	env := &ClaudeEnv{}

	// Install then remove.
	if _, err := env.Install(false); err != nil {
		t.Fatalf("Install() error: %v", err)
	}

	if err := env.Remove(false); err != nil {
		t.Fatalf("Remove() error: %v", err)
	}

	_, _, installed := env.Detect()
	if installed {
		t.Error("Detect() should return false after remove")
	}
}

func TestClaudeEnvCheck(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	env := &ClaudeEnv{}

	path, scope, installed, err := env.Check(false)
	if err != nil {
		t.Fatalf("Check() error: %v", err)
	}
	if path == "" {
		t.Error("Check() should return a path even when not installed")
	}
	if scope != "global" {
		t.Errorf("scope = %q, want %q", scope, "global")
	}
	if installed {
		t.Error("Check() should return false when not installed")
	}
}

func TestDetectedAgentEnvsEmpty(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	detected := DetectedAgentEnvs()
	if len(detected) != 0 {
		t.Errorf("DetectedAgentEnvs() = %d, want 0 when nothing installed", len(detected))
	}
}

func TestDetectedAgentEnvsWithClaude(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	env := &ClaudeEnv{}
	if _, err := env.Install(false); err != nil {
		t.Fatalf("Install() error: %v", err)
	}

	detected := DetectedAgentEnvs()
	if len(detected) != 1 {
		t.Fatalf("DetectedAgentEnvs() = %d, want 1", len(detected))
	}
	if detected[0].Name() != "claude" {
		t.Errorf("detected[0].Name() = %q, want %q", detected[0].Name(), "claude")
	}
}
