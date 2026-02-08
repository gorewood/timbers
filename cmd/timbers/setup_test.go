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

// TestSetupClaudeCheck verifies the check flag for Claude integration.
func TestSetupClaudeCheck(t *testing.T) {
	tests := []struct {
		name       string
		setupHook  bool // If true, create the hook file first
		wantJSON   map[string]any
		wantHuman  string
		wantStatus bool // Expected installed status
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
			// Create a temp home directory
			tmpHome := t.TempDir()
			t.Setenv("HOME", tmpHome)

			hookDir := filepath.Join(tmpHome, ".claude", "hooks")
			hookPath := filepath.Join(hookDir, "user_prompt_submit.sh")

			if tc.setupHook {
				if err := os.MkdirAll(hookDir, 0o755); err != nil {
					t.Fatalf("failed to create hook dir: %v", err)
				}
				// #nosec G306 -- hook needs execute permission for testing
				if err := os.WriteFile(hookPath, []byte("#!/bin/bash\n# BEGIN timbers\n# END timbers"), 0o755); err != nil {
					t.Fatalf("failed to write hook: %v", err)
				}
			}

			// Test JSON output
			t.Run("json", func(t *testing.T) {
				var buf bytes.Buffer
				cmd := newSetupCmd()
				cmd.PersistentFlags().Bool("json", false, "")
				_ = cmd.PersistentFlags().Set("json", "true")
				cmd.SetOut(&buf)
				cmd.SetArgs([]string{"claude", "--check"})

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

			// Test human output
			t.Run("human", func(t *testing.T) {
				var buf bytes.Buffer
				cmd := newSetupCmd()
				cmd.SetOut(&buf)
				cmd.SetArgs([]string{"claude", "--check"})

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
	t.Run("install creates hook", func(t *testing.T) {
		tmpHome := t.TempDir()
		t.Setenv("HOME", tmpHome)

		hookPath := filepath.Join(tmpHome, ".claude", "hooks", "user_prompt_submit.sh")

		var buf bytes.Buffer
		cmd := newSetupCmd()
		cmd.SetOut(&buf)
		cmd.SetArgs([]string{"claude"})

		err := cmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify hook was created
		content, err := os.ReadFile(hookPath)
		if err != nil {
			t.Fatalf("hook file not created: %v", err)
		}

		// Check hook content
		if !strings.Contains(string(content), "timbers prime") {
			t.Errorf("hook should contain 'timbers prime', got: %s", content)
		}
		if !strings.Contains(string(content), "#!/bin/bash") {
			t.Errorf("hook should have bash shebang, got: %s", content)
		}

		// Check permissions
		info, err := os.Stat(hookPath)
		if err != nil {
			t.Fatalf("failed to stat hook: %v", err)
		}
		if info.Mode().Perm()&0o111 == 0 {
			t.Error("hook should be executable")
		}
	})

	t.Run("install is idempotent", func(t *testing.T) {
		tmpHome := t.TempDir()
		t.Setenv("HOME", tmpHome)

		hookPath := filepath.Join(tmpHome, ".claude", "hooks", "user_prompt_submit.sh")

		// Run install twice
		for i := range 2 {
			var buf bytes.Buffer
			cmd := newSetupCmd()
			cmd.SetOut(&buf)
			cmd.SetArgs([]string{"claude"})

			err := cmd.Execute()
			if err != nil {
				t.Fatalf("install %d: unexpected error: %v", i+1, err)
			}
		}

		// Verify only one timbers section exists
		content, err := os.ReadFile(hookPath)
		if err != nil {
			t.Fatalf("hook file not found: %v", err)
		}

		count := strings.Count(string(content), "# BEGIN timbers")
		if count != 1 {
			t.Errorf("expected exactly 1 timbers section, found %d in:\n%s", count, content)
		}
	})

	t.Run("preserves existing hook content", func(t *testing.T) {
		tmpHome := t.TempDir()
		t.Setenv("HOME", tmpHome)

		hookDir := filepath.Join(tmpHome, ".claude", "hooks")
		hookPath := filepath.Join(hookDir, "user_prompt_submit.sh")

		// Create existing hook with other content
		if err := os.MkdirAll(hookDir, 0o755); err != nil {
			t.Fatalf("failed to create hook dir: %v", err)
		}
		existingContent := "#!/bin/bash\necho 'existing hook'\n"
		// #nosec G306 -- hook needs execute permission for testing
		if err := os.WriteFile(hookPath, []byte(existingContent), 0o755); err != nil {
			t.Fatalf("failed to write existing hook: %v", err)
		}

		var buf bytes.Buffer
		cmd := newSetupCmd()
		cmd.SetOut(&buf)
		cmd.SetArgs([]string{"claude"})

		err := cmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := os.ReadFile(hookPath)
		if err != nil {
			t.Fatalf("hook file not found: %v", err)
		}

		// Should contain both existing content and timbers
		if !strings.Contains(string(content), "existing hook") {
			t.Error("existing hook content was lost")
		}
		if !strings.Contains(string(content), "timbers prime") {
			t.Error("timbers section not added")
		}
	})
}

// TestSetupClaudeRemove verifies hook removal.
func TestSetupClaudeRemove(t *testing.T) {
	t.Run("remove deletes timbers section only", func(t *testing.T) {
		tmpHome := t.TempDir()
		t.Setenv("HOME", tmpHome)

		hookDir := filepath.Join(tmpHome, ".claude", "hooks")
		hookPath := filepath.Join(hookDir, "user_prompt_submit.sh")

		// Create hook with timbers and other content
		if err := os.MkdirAll(hookDir, 0o755); err != nil {
			t.Fatalf("failed to create hook dir: %v", err)
		}
		hookContent := `#!/bin/bash
echo 'before'
# BEGIN timbers
if command -v timbers >/dev/null 2>&1 && [ -d ".git" ]; then
  timbers prime 2>/dev/null
fi
# END timbers
echo 'after'
`
		// #nosec G306 -- hook needs execute permission for testing
		if err := os.WriteFile(hookPath, []byte(hookContent), 0o755); err != nil {
			t.Fatalf("failed to write hook: %v", err)
		}

		var buf bytes.Buffer
		cmd := newSetupCmd()
		cmd.SetOut(&buf)
		cmd.SetArgs([]string{"claude", "--remove"})

		err := cmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := os.ReadFile(hookPath)
		if err != nil {
			t.Fatalf("hook file not found: %v", err)
		}

		// Should have existing content but not timbers
		if !strings.Contains(string(content), "before") {
			t.Error("existing content before timbers was lost")
		}
		if !strings.Contains(string(content), "after") {
			t.Error("existing content after timbers was lost")
		}
		if strings.Contains(string(content), "timbers prime") {
			t.Error("timbers section should be removed")
		}
	})

	t.Run("remove when not installed", func(t *testing.T) {
		tmpHome := t.TempDir()
		t.Setenv("HOME", tmpHome)

		var buf bytes.Buffer
		cmd := newSetupCmd()
		cmd.SetOut(&buf)
		cmd.SetArgs([]string{"claude", "--remove"})

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

	hookPath := filepath.Join(tmpHome, ".claude", "hooks", "user_prompt_submit.sh")

	var buf bytes.Buffer
	cmd := newSetupCmd()
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"claude", "--dry-run"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Hook should NOT be created
	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Error("dry-run should not create hook file")
	}

	// Output should describe what would happen
	output := buf.String()
	if !strings.Contains(strings.ToLower(output), "would") {
		t.Errorf("dry-run output should describe intended action, got: %s", output)
	}
}

// TestSetupClaudeProject verifies project-level installation.
// Note: Cannot use t.Parallel() due to os.Chdir() usage.
func TestSetupClaudeProject(t *testing.T) {
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
	cmd.SetArgs([]string{"claude", "--project"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should create hook in project directory
	projectHookPath := filepath.Join(tmpProject, ".claude", "hooks", "user_prompt_submit.sh")
	if _, err := os.Stat(projectHookPath); os.IsNotExist(err) {
		t.Error("project hook was not created")
	}

	// Should NOT create hook in global directory
	globalHookPath := filepath.Join(tmpHome, ".claude", "hooks", "user_prompt_submit.sh")
	if _, err := os.Stat(globalHookPath); !os.IsNotExist(err) {
		t.Error("global hook should not be created with --project")
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
