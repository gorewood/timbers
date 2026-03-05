// Package main provides the entry point for the timbers CLI.
package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// newTestRootCmdWithHooks creates a root command that includes hooks commands for testing.
// This is needed because main.go hasn't been updated yet to include hooks.
func newTestRootCmdWithHooks() *cobra.Command {
	cmd := newRootCmd()
	cmd.AddCommand(newHooksCmd())
	cmd.AddCommand(newHookCmd())
	return cmd
}

func TestHooksListCommand(t *testing.T) {
	tempDir := t.TempDir()

	// Initialize a git repo
	runGit(t, tempDir, "init")
	runGit(t, tempDir, "config", "user.email", "test@test.com")
	runGit(t, tempDir, "config", "user.name", "Test User")

	tests := []struct {
		name       string
		args       []string
		setup      func(t *testing.T, dir string)
		wantFields map[string]any
	}{
		{
			name: "JSON output - no hooks installed",
			args: []string{"hooks", "list", "--json"},
			wantFields: map[string]any{
				"pre_commit": map[string]any{
					"installed": false,
					"chained":   false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(t, tempDir)
			}

			runInDir(t, tempDir, func() {
				var buf bytes.Buffer

				cmd := newTestRootCmdWithHooks()
				cmd.SetOut(&buf)
				cmd.SetErr(&buf)
				cmd.SetArgs(tt.args)

				if err := cmd.Execute(); err != nil {
					t.Fatalf("command failed: %v\nOutput: %s", err, buf.String())
				}

				var result map[string]any
				if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
					t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, buf.String())
				}

				for key, want := range tt.wantFields {
					got, ok := result[key]
					if !ok {
						t.Errorf("missing field %q in output", key)
						continue
					}
					// For nested maps, compare as JSON
					wantJSON, _ := json.Marshal(want)
					gotJSON, _ := json.Marshal(got)
					if string(wantJSON) != string(gotJSON) {
						t.Errorf("field %q = %s, want %s", key, gotJSON, wantJSON)
					}
				}
			})
		})
	}
}

func TestHooksListHumanOutput(t *testing.T) {
	tempDir := t.TempDir()

	runGit(t, tempDir, "init")
	runGit(t, tempDir, "config", "user.email", "test@test.com")
	runGit(t, tempDir, "config", "user.name", "Test User")

	runInDir(t, tempDir, func() {
		var buf bytes.Buffer

		cmd := newTestRootCmdWithHooks()
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)
		cmd.SetArgs([]string{"hooks", "list"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("command failed: %v", err)
		}

		output := buf.String()

		checks := []string{
			"pre-commit",
			"not installed",
		}

		for _, check := range checks {
			if !strings.Contains(output, check) {
				t.Errorf("human output missing %q\nOutput: %s", check, output)
			}
		}
	})
}

func TestHooksInstallCommand(t *testing.T) {
	tempDir := t.TempDir()

	// Initialize a git repo
	runGit(t, tempDir, "init")
	runGit(t, tempDir, "config", "user.email", "test@test.com")
	runGit(t, tempDir, "config", "user.name", "Test User")

	tests := []struct {
		name       string
		args       []string
		setup      func(t *testing.T, dir string)
		wantFields map[string]any
		wantErr    bool
		checkHook  func(t *testing.T, dir string)
	}{
		{
			name: "install hook JSON output",
			args: []string{"hooks", "install", "--json"},
			wantFields: map[string]any{
				"status": "ok",
			},
			checkHook: func(t *testing.T, dir string) {
				hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
				if _, err := os.Stat(hookPath); os.IsNotExist(err) {
					t.Error("hook file not created")
					return
				}

				content, err := os.ReadFile(hookPath)
				if err != nil {
					t.Fatalf("failed to read hook: %v", err)
				}

				if !strings.Contains(string(content), "timbers hook run pre-commit") {
					t.Error("hook does not contain expected timbers command")
				}
			},
		},
		{
			name: "dry-run does not create hook",
			args: []string{"hooks", "install", "--dry-run", "--json"},
			wantFields: map[string]any{
				"status": "dry_run",
			},
			checkHook: func(t *testing.T, dir string) {
				hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
				if _, err := os.Stat(hookPath); err == nil {
					t.Error("hook file should not be created in dry-run mode")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh temp dir for each test to avoid conflicts
			testDir := t.TempDir()
			runGit(t, testDir, "init")
			runGit(t, testDir, "config", "user.email", "test@test.com")
			runGit(t, testDir, "config", "user.name", "Test User")

			if tt.setup != nil {
				tt.setup(t, testDir)
			}

			runInDir(t, testDir, func() {
				var buf bytes.Buffer

				cmd := newTestRootCmdWithHooks()
				cmd.SetOut(&buf)
				cmd.SetErr(&buf)
				cmd.SetArgs(tt.args)

				err := cmd.Execute()
				if (err != nil) != tt.wantErr {
					t.Fatalf("Execute() error = %v, wantErr %v\nOutput: %s", err, tt.wantErr, buf.String())
				}

				if tt.wantErr {
					return
				}

				var result map[string]any
				if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
					t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, buf.String())
				}

				for key, want := range tt.wantFields {
					got, ok := result[key]
					if !ok {
						t.Errorf("missing field %q in output", key)
						continue
					}
					if got != want {
						t.Errorf("field %q = %v (%T), want %v (%T)", key, got, got, want, want)
					}
				}

				if tt.checkHook != nil {
					tt.checkHook(t, testDir)
				}
			})
		})
	}
}

func TestHooksInstallChaining(t *testing.T) {
	// --chain is now a hidden deprecated alias that maps to default append behavior.
	// With the new append-section approach, existing hooks are preserved in-place
	// and timbers appends a delimited section.
	testDir := t.TempDir()
	runGit(t, testDir, "init")
	runGit(t, testDir, "config", "user.email", "test@test.com")
	runGit(t, testDir, "config", "user.name", "Test User")

	// Create an existing hook
	hooksDir := filepath.Join(testDir, ".git", "hooks")
	existingHook := filepath.Join(hooksDir, "pre-commit")
	existingContent := "#!/bin/sh\necho 'existing hook'\n"
	// #nosec G306 -- test hook needs execute permission
	if err := os.WriteFile(existingHook, []byte(existingContent), 0o755); err != nil {
		t.Fatalf("failed to create existing hook: %v", err)
	}

	runInDir(t, testDir, func() {
		var buf bytes.Buffer

		cmd := newTestRootCmdWithHooks()
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)
		cmd.SetArgs([]string{"hooks", "install", "--chain", "--json"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("command failed: %v\nOutput: %s", err, buf.String())
		}

		// With append-section, the existing hook content is preserved
		// and timbers section is appended (no backup created).
		content, err := os.ReadFile(existingHook)
		if err != nil {
			t.Fatalf("failed to read hook: %v", err)
		}

		contentStr := string(content)
		if !strings.Contains(contentStr, "echo 'existing hook'") {
			t.Error("existing hook content was not preserved")
		}
		if !strings.Contains(contentStr, "timbers hook run pre-commit") {
			t.Error("timbers section was not appended")
		}
		if !strings.Contains(contentStr, "--- timbers section") {
			t.Error("timbers section delimiters not found")
		}
	})
}

func TestHooksInstallForceOverwrite(t *testing.T) {
	// --force now means "install even in Tier 4 (unknown override)".
	// For Tier 2 (existing hook, standard path), it behaves the same as
	// default: append timbers section alongside existing content.
	testDir := t.TempDir()
	runGit(t, testDir, "init")
	runGit(t, testDir, "config", "user.email", "test@test.com")
	runGit(t, testDir, "config", "user.name", "Test User")

	// Create an existing hook
	hooksDir := filepath.Join(testDir, ".git", "hooks")
	existingHook := filepath.Join(hooksDir, "pre-commit")
	existingContent := "#!/bin/sh\necho 'existing hook'\n"
	// #nosec G306 -- test hook needs execute permission
	if err := os.WriteFile(existingHook, []byte(existingContent), 0o755); err != nil {
		t.Fatalf("failed to create existing hook: %v", err)
	}

	runInDir(t, testDir, func() {
		var buf bytes.Buffer

		cmd := newTestRootCmdWithHooks()
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)
		cmd.SetArgs([]string{"hooks", "install", "--force", "--json"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("command failed: %v\nOutput: %s", err, buf.String())
		}

		// Existing content preserved, timbers section appended.
		content, err := os.ReadFile(existingHook)
		if err != nil {
			t.Fatalf("failed to read hook: %v", err)
		}

		contentStr := string(content)
		if !strings.Contains(contentStr, "echo 'existing hook'") {
			t.Error("existing hook content was not preserved")
		}
		if !strings.Contains(contentStr, "timbers hook run pre-commit") {
			t.Error("timbers section was not appended")
		}
	})
}

func TestHooksUninstallCommand(t *testing.T) {
	testDir := t.TempDir()
	runGit(t, testDir, "init")
	runGit(t, testDir, "config", "user.email", "test@test.com")
	runGit(t, testDir, "config", "user.name", "Test User")

	hooksDir := filepath.Join(testDir, ".git", "hooks")

	// First install the hook
	runInDir(t, testDir, func() {
		var buf bytes.Buffer
		cmd := newTestRootCmdWithHooks()
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)
		cmd.SetArgs([]string{"hooks", "install"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("install failed: %v", err)
		}
	})

	// Verify hook exists
	hookPath := filepath.Join(hooksDir, "pre-commit")
	if _, err := os.Stat(hookPath); os.IsNotExist(err) {
		t.Fatal("hook was not installed")
	}

	// Now uninstall
	runInDir(t, testDir, func() {
		var buf bytes.Buffer
		cmd := newTestRootCmdWithHooks()
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)
		cmd.SetArgs([]string{"hooks", "uninstall", "--json"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("uninstall failed: %v\nOutput: %s", err, buf.String())
		}

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("failed to parse JSON: %v\nOutput: %s", err, buf.String())
		}

		if result["status"] != "ok" {
			t.Errorf("status = %v, want ok", result["status"])
		}
	})

	// Verify hook is removed
	if _, err := os.Stat(hookPath); err == nil {
		t.Error("hook was not removed")
	}
}

func TestHooksUninstallRestoresLegacyBackup(t *testing.T) {
	// Tests that legacy .backup files from old chain installs are restored
	// during uninstall, even though the new install path doesn't create them.
	testDir := t.TempDir()
	runGit(t, testDir, "init")
	runGit(t, testDir, "config", "user.email", "test@test.com")
	runGit(t, testDir, "config", "user.name", "Test User")

	hooksDir := filepath.Join(testDir, ".git", "hooks")

	// Simulate a legacy chain install: timbers hook with section + .backup file.
	existingContent := "#!/bin/sh\necho 'original hook'\n"
	timbersHookContent := "#!/bin/sh\n" +
		"# --- timbers section (do not edit) ---\n" +
		"if command -v timbers >/dev/null 2>&1; then\n" +
		"  timbers hook run pre-commit \"$@\"\n" +
		"  rc=$?\n" +
		"  if [ $rc -ne 0 ]; then exit $rc; fi\n" +
		"fi\n" +
		"# --- end timbers section ---\n"

	hookPath := filepath.Join(hooksDir, "pre-commit")
	backupPath := filepath.Join(hooksDir, "pre-commit.backup")

	// #nosec G306 -- test hook needs execute permission
	if err := os.WriteFile(hookPath, []byte(timbersHookContent), 0o755); err != nil {
		t.Fatalf("failed to create hook: %v", err)
	}
	// #nosec G306 -- test hook needs execute permission
	if err := os.WriteFile(backupPath, []byte(existingContent), 0o755); err != nil {
		t.Fatalf("failed to create backup: %v", err)
	}

	// Uninstall
	runInDir(t, testDir, func() {
		var buf bytes.Buffer
		cmd := newTestRootCmdWithHooks()
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)
		cmd.SetArgs([]string{"hooks", "uninstall"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("uninstall failed: %v", err)
		}
	})

	// Verify backup was restored (section removal deleted the hook file since
	// it was only a shebang + section, then the backup was renamed in).
	content, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("failed to read hook: %v", err)
	}

	if string(content) != existingContent {
		t.Errorf("backup was not restored\ngot: %s\nwant: %s", content, existingContent)
	}

	// Verify backup file is removed
	if _, err := os.Stat(backupPath); err == nil {
		t.Error("backup file should be removed after restore")
	}
}

func TestHooksNotARepo(t *testing.T) {
	tempDir := t.TempDir()

	subcommands := []string{"list", "install", "uninstall"}

	for _, subcmd := range subcommands {
		t.Run(subcmd, func(t *testing.T) {
			runInDir(t, tempDir, func() {
				var buf bytes.Buffer

				cmd := newTestRootCmdWithHooks()
				cmd.SetOut(&buf)
				cmd.SetErr(&buf)
				cmd.SetArgs([]string{"hooks", subcmd, "--json"})

				err := cmd.Execute()
				if err == nil {
					t.Fatal("expected error for non-repo directory")
				}

				var result map[string]any
				if jsonErr := json.Unmarshal(buf.Bytes(), &result); jsonErr != nil {
					t.Fatalf("failed to parse JSON error output: %v\nOutput: %s", jsonErr, buf.String())
				}

				code, ok := result["code"].(float64)
				if !ok {
					t.Fatalf("missing or invalid 'code' in error output: %v", result)
				}
				if code != 2 {
					t.Errorf("error code = %v, want 2 (system error)", code)
				}
			})
		})
	}
}
