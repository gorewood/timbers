// Package main provides the entry point for the timbers CLI.
package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeSettingsJSON creates a Claude Code settings file with the given content.
func writeSettingsJSON(t *testing.T, path string, data map[string]any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("failed to create settings dir: %v", err)
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatalf("failed to write settings: %v", err)
	}
}

// readSettingsJSON reads a Claude Code settings file.
func readSettingsJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read settings: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("failed to parse settings: %v", err)
	}
	return m
}

// timbersSettings creates a minimal settings map with timbers prime installed.
func timbersSettings() map[string]any {
	return map[string]any{
		"hooks": map[string]any{
			"SessionStart": []any{
				map[string]any{
					"matcher": "",
					"hooks": []any{
						map[string]any{"type": "command", "command": "timbers prime"},
					},
				},
			},
		},
	}
}

// TestSetupClaudeCheck verifies the check flag for Claude integration.
func TestSetupClaudeCheck(t *testing.T) {
	tests := []struct {
		name       string
		setupHook  bool // If true, create the settings file first
		wantStatus bool // Expected installed status
		wantHuman  string
	}{
		{
			name:       "not installed",
			setupHook:  false,
			wantStatus: false,
			wantHuman:  "not installed",
		},
		{
			name:       "already installed",
			setupHook:  true,
			wantStatus: true,
			wantHuman:  "installed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpHome := t.TempDir()
			t.Setenv("HOME", tmpHome)

			if tc.setupHook {
				settingsPath := filepath.Join(tmpHome, ".claude", "settings.json")
				writeSettingsJSON(t, settingsPath, timbersSettings())
			}

			// Test JSON output (use --global to check global scope)
			t.Run("json", func(t *testing.T) {
				var buf bytes.Buffer
				cmd := newSetupCmd()
				cmd.PersistentFlags().Bool("json", false, "")
				_ = cmd.PersistentFlags().Set("json", "true")
				cmd.SetOut(&buf)
				cmd.SetArgs([]string{"claude", "--global", "--check"})

				err := cmd.Execute()
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				var result map[string]any
				if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
					t.Fatalf("failed to parse JSON: %v (output: %s)", err, buf.String())
				}

				if result["integration"] != "claude" {
					t.Errorf("expected integration=claude, got %v", result["integration"])
				}
				if result["installed"] != tc.wantStatus {
					t.Errorf("expected installed=%v, got %v", tc.wantStatus, result["installed"])
				}
				if result["scope"] != "global" {
					t.Errorf("expected scope=global, got %v", result["scope"])
				}
			})

			// Test human output (use --global to check global scope)
			t.Run("human", func(t *testing.T) {
				var buf bytes.Buffer
				cmd := newSetupCmd()
				cmd.SetOut(&buf)
				cmd.SetArgs([]string{"claude", "--global", "--check"})

				err := cmd.Execute()
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				output := buf.String()
				if !strings.Contains(strings.ToLower(output), tc.wantHuman) {
					t.Errorf("expected output to contain %q, got: %s", tc.wantHuman, output)
				}
			})
		})
	}
}

// TestSetupClaudeInstall verifies hook installation.
func TestSetupClaudeInstall(t *testing.T) {
	t.Run("install creates settings globally", func(t *testing.T) {
		tmpHome := t.TempDir()
		t.Setenv("HOME", tmpHome)

		settingsPath := filepath.Join(tmpHome, ".claude", "settings.json")

		var buf bytes.Buffer
		cmd := newSetupCmd()
		cmd.SetOut(&buf)
		cmd.SetArgs([]string{"claude", "--global"})

		err := cmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify settings were created
		settings := readSettingsJSON(t, settingsPath)

		// Check settings content
		hooks, ok := settings["hooks"].(map[string]any)
		if !ok {
			t.Fatal("expected hooks section in settings")
		}
		sessionStart, ok := hooks["SessionStart"].([]any)
		if !ok || len(sessionStart) == 0 {
			t.Fatal("expected SessionStart hook group")
		}

		// Verify timbers prime is present
		found := false
		for _, rawGroup := range sessionStart {
			group, _ := rawGroup.(map[string]any)
			rawHooks, _ := group["hooks"].([]any)
			for _, rawHook := range rawHooks {
				hook, _ := rawHook.(map[string]any)
				if cmd, _ := hook["command"].(string); cmd == "timbers prime" {
					found = true
				}
			}
		}
		if !found {
			t.Error("settings should contain 'timbers prime' hook")
		}
	})

	t.Run("install is idempotent", func(t *testing.T) {
		tmpHome := t.TempDir()
		t.Setenv("HOME", tmpHome)

		settingsPath := filepath.Join(tmpHome, ".claude", "settings.json")

		// Run install twice (global scope)
		for i := range 2 {
			var buf bytes.Buffer
			cmd := newSetupCmd()
			cmd.SetOut(&buf)
			cmd.SetArgs([]string{"claude", "--global"})

			err := cmd.Execute()
			if err != nil {
				t.Fatalf("install %d: unexpected error: %v", i+1, err)
			}
		}

		// Verify only one timbers hook entry exists
		settings := readSettingsJSON(t, settingsPath)
		count := 0
		hooks, _ := settings["hooks"].(map[string]any)
		sessionStart, _ := hooks["SessionStart"].([]any)
		for _, rawGroup := range sessionStart {
			group, _ := rawGroup.(map[string]any)
			rawHooks, _ := group["hooks"].([]any)
			for _, rawHook := range rawHooks {
				hook, _ := rawHook.(map[string]any)
				if cmd, _ := hook["command"].(string); cmd == "timbers prime" {
					count++
				}
			}
		}
		if count != 1 {
			t.Errorf("expected exactly 1 timbers prime hook, found %d", count)
		}
	})

	t.Run("preserves existing settings", func(t *testing.T) {
		tmpHome := t.TempDir()
		t.Setenv("HOME", tmpHome)

		settingsPath := filepath.Join(tmpHome, ".claude", "settings.json")

		// Create existing settings with other hooks
		writeSettingsJSON(t, settingsPath, map[string]any{
			"permissions": map[string]any{"allow": []any{"Bash(ls:*)"}},
			"hooks": map[string]any{
				"SessionStart": []any{
					map[string]any{
						"matcher": "",
						"hooks": []any{
							map[string]any{"type": "command", "command": "bd prime"},
						},
					},
				},
			},
		})

		var buf bytes.Buffer
		cmd := newSetupCmd()
		cmd.SetOut(&buf)
		cmd.SetArgs([]string{"claude", "--global"})

		err := cmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		settings := readSettingsJSON(t, settingsPath)

		// Verify existing settings preserved
		if _, ok := settings["permissions"]; !ok {
			t.Error("existing permissions should be preserved")
		}

		// Verify both hooks present
		hooks, _ := settings["hooks"].(map[string]any)
		sessionStart, _ := hooks["SessionStart"].([]any)
		foundBd, foundTimbers := false, false
		for _, rawGroup := range sessionStart {
			group, _ := rawGroup.(map[string]any)
			rawHooks, _ := group["hooks"].([]any)
			for _, rawHook := range rawHooks {
				hook, _ := rawHook.(map[string]any)
				cmd, _ := hook["command"].(string)
				if cmd == "bd prime" {
					foundBd = true
				}
				if cmd == "timbers prime" {
					foundTimbers = true
				}
			}
		}
		if !foundBd {
			t.Error("existing bd prime hook should be preserved")
		}
		if !foundTimbers {
			t.Error("timbers prime hook should be added")
		}
	})
}

// TestSetupClaudeRemove verifies hook removal.
func TestSetupClaudeRemove(t *testing.T) {
	t.Run("remove deletes timbers section only", func(t *testing.T) {
		tmpHome := t.TempDir()
		t.Setenv("HOME", tmpHome)

		settingsPath := filepath.Join(tmpHome, ".claude", "settings.json")

		// Create settings with timbers and other hooks
		writeSettingsJSON(t, settingsPath, map[string]any{
			"hooks": map[string]any{
				"SessionStart": []any{
					map[string]any{
						"matcher": "",
						"hooks": []any{
							map[string]any{"type": "command", "command": "bd prime"},
						},
					},
					map[string]any{
						"matcher": "",
						"hooks": []any{
							map[string]any{"type": "command", "command": "timbers prime"},
						},
					},
				},
			},
		})

		var buf bytes.Buffer
		cmd := newSetupCmd()
		cmd.SetOut(&buf)
		cmd.SetArgs([]string{"claude", "--global", "--remove"})

		err := cmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		settings := readSettingsJSON(t, settingsPath)

		// timbers should be gone
		hooks, _ := settings["hooks"].(map[string]any)
		sessionStart, _ := hooks["SessionStart"].([]any)
		for _, rawGroup := range sessionStart {
			group, _ := rawGroup.(map[string]any)
			rawHooks, _ := group["hooks"].([]any)
			for _, rawHook := range rawHooks {
				hook, _ := rawHook.(map[string]any)
				if cmd, _ := hook["command"].(string); cmd == "timbers prime" {
					t.Error("timbers prime should be removed")
				}
			}
		}

		// bd prime should still be there
		foundBd := false
		for _, rawGroup := range sessionStart {
			group, _ := rawGroup.(map[string]any)
			rawHooks, _ := group["hooks"].([]any)
			for _, rawHook := range rawHooks {
				hook, _ := rawHook.(map[string]any)
				if cmd, _ := hook["command"].(string); cmd == "bd prime" {
					foundBd = true
				}
			}
		}
		if !foundBd {
			t.Error("bd prime should be preserved")
		}
	})

	t.Run("remove when not installed", func(t *testing.T) {
		tmpHome := t.TempDir()
		t.Setenv("HOME", tmpHome)

		var buf bytes.Buffer
		cmd := newSetupCmd()
		cmd.SetOut(&buf)
		cmd.SetArgs([]string{"claude", "--global", "--remove"})

		err := cmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should succeed without error
		output := buf.String()
		if !strings.Contains(strings.ToLower(output), "not installed") {
			t.Errorf("expected 'not installed' message, got: %s", output)
		}
	})
}

// TestSetupClaudeDryRun verifies dry-run mode.
func TestSetupClaudeDryRun(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	settingsPath := filepath.Join(tmpHome, ".claude", "settings.json")

	var buf bytes.Buffer
	cmd := newSetupCmd()
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"claude", "--global", "--dry-run"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Settings file should NOT be created
	if _, err := os.Stat(settingsPath); !os.IsNotExist(err) {
		t.Error("dry-run should not create settings file")
	}

	// Output should describe what would happen
	output := buf.String()
	if !strings.Contains(strings.ToLower(output), "would") {
		t.Errorf("dry-run output should describe intended action, got: %s", output)
	}
}

// TestSetupClaudeDefaultIsProject verifies default installs to project level.
// Note: Cannot use t.Parallel() due to os.Chdir() usage.
func TestSetupClaudeDefaultIsProject(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	tmpProject := t.TempDir()

	// Create a git repo in the project
	gitDir := filepath.Join(tmpProject, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatalf("failed to create git dir: %v", err)
	}

	// Change to project directory
	oldDir, _ := os.Getwd()
	if err := os.Chdir(tmpProject); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldDir)
	}()

	var buf bytes.Buffer
	cmd := newSetupCmd()
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"claude"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should create settings in project directory (default)
	projectSettingsPath := filepath.Join(tmpProject, ".claude", "settings.local.json")
	if _, err := os.Stat(projectSettingsPath); os.IsNotExist(err) {
		t.Error("project settings file was not created")
	}

	// Should NOT create settings in global directory
	globalSettingsPath := filepath.Join(tmpHome, ".claude", "settings.json")
	if _, err := os.Stat(globalSettingsPath); !os.IsNotExist(err) {
		t.Error("global settings should not be created by default")
	}
}

// TestSetupList verifies the --list flag.
func TestSetupList(t *testing.T) {
	t.Run("json output", func(t *testing.T) {
		tmpHome := t.TempDir()
		t.Setenv("HOME", tmpHome)

		var buf bytes.Buffer
		cmd := newSetupCmd()
		cmd.PersistentFlags().Bool("json", false, "")
		_ = cmd.PersistentFlags().Set("json", "true")
		cmd.SetOut(&buf)
		cmd.SetArgs([]string{"--list"})

		err := cmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		integrations, ok := result["integrations"].([]any)
		if !ok {
			t.Fatalf("expected integrations array, got %T", result["integrations"])
		}

		found := false
		for _, i := range integrations {
			m, ok := i.(map[string]any)
			if ok && m["name"] == "claude" {
				found = true
				break
			}
		}
		if !found {
			t.Error("claude integration not found in list")
		}
	})

	t.Run("human output", func(t *testing.T) {
		tmpHome := t.TempDir()
		t.Setenv("HOME", tmpHome)

		var buf bytes.Buffer
		cmd := newSetupCmd()
		cmd.SetOut(&buf)
		cmd.SetArgs([]string{"--list"})

		err := cmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "claude") {
			t.Errorf("expected output to contain 'claude', got: %s", output)
		}
	})
}
