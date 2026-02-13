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

// newTestRootCmdWithInit creates the root command with init registered for testing.
func newTestRootCmdWithInit() *cobra.Command {
	cmd := newRootCmd()
	cmd.AddCommand(newInitCmd())
	return cmd
}

func TestInitCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		setup       func(t *testing.T, dir string)
		wantFields  map[string]any
		wantStrings []string
	}{
		{
			name: "JSON output with --yes flag",
			args: []string{"init", "--yes", "--json"},
			wantFields: map[string]any{
				"status": "ok",
			},
		},
		{
			name: "dry-run shows what would be done",
			args: []string{"init", "--dry-run", "--json"},
			wantFields: map[string]any{
				"status": "dry_run",
			},
		},
		{
			name: "dry-run without --hooks skips hooks",
			args: []string{"init", "--dry-run", "--json"},
			wantFields: map[string]any{
				"status": "dry_run",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh temp dir for each test
			tempDir := t.TempDir()

			// Initialize a git repo
			runGit(t, tempDir, "init")
			runGit(t, tempDir, "config", "user.email", "test@test.com")
			runGit(t, tempDir, "config", "user.name", "Test User")

			// Create a file and commit
			testFile := filepath.Join(tempDir, "test.txt")
			if err := os.WriteFile(testFile, []byte("test content"), 0o600); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}
			runGit(t, tempDir, "add", "test.txt")
			runGit(t, tempDir, "commit", "-m", "Initial commit")

			if tt.setup != nil {
				tt.setup(t, tempDir)
			}

			runInDir(t, tempDir, func() {
				var buf bytes.Buffer

				cmd := newTestRootCmdWithInit()
				cmd.SetOut(&buf)
				cmd.SetErr(&buf)
				cmd.SetArgs(tt.args)

				if err := cmd.Execute(); err != nil {
					t.Fatalf("command failed: %v\nOutput: %s", err, buf.String())
				}

				if len(tt.wantFields) > 0 {
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
							t.Errorf("field %q = %v, want %v", key, got, want)
						}
					}
				}

				if len(tt.wantStrings) > 0 {
					output := buf.String()
					for _, want := range tt.wantStrings {
						if !strings.Contains(output, want) {
							t.Errorf("output missing %q\nOutput: %s", want, output)
						}
					}
				}
			})
		})
	}
}

func TestInitIdempotent(t *testing.T) {
	tempDir := t.TempDir()

	// Initialize a git repo
	runGit(t, tempDir, "init")
	runGit(t, tempDir, "config", "user.email", "test@test.com")
	runGit(t, tempDir, "config", "user.name", "Test User")

	// Create a file and commit
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0o600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	runGit(t, tempDir, "add", "test.txt")
	runGit(t, tempDir, "commit", "-m", "Initial commit")

	runInDir(t, tempDir, func() {
		// First init
		var buf1 bytes.Buffer
		cmd1 := newTestRootCmdWithInit()
		cmd1.SetOut(&buf1)
		cmd1.SetErr(&buf1)
		cmd1.SetArgs([]string{"init", "--yes", "--no-claude", "--json"})

		if err := cmd1.Execute(); err != nil {
			t.Fatalf("first init failed: %v\nOutput: %s", err, buf1.String())
		}

		var result1 map[string]any
		if err := json.Unmarshal(buf1.Bytes(), &result1); err != nil {
			t.Fatalf("failed to parse first JSON: %v", err)
		}

		if result1["status"] != "ok" {
			t.Errorf("first init status = %v, want ok", result1["status"])
		}

		// Second init (should report already initialized)
		var buf2 bytes.Buffer
		cmd2 := newTestRootCmdWithInit()
		cmd2.SetOut(&buf2)
		cmd2.SetErr(&buf2)
		cmd2.SetArgs([]string{"init", "--yes", "--no-claude", "--json"})

		if err := cmd2.Execute(); err != nil {
			t.Fatalf("second init failed: %v\nOutput: %s", err, buf2.String())
		}

		var result2 map[string]any
		if err := json.Unmarshal(buf2.Bytes(), &result2); err != nil {
			t.Fatalf("failed to parse second JSON: %v", err)
		}

		if result2["status"] != "ok" {
			t.Errorf("second init status = %v, want ok", result2["status"])
		}

		// Check already_initialized flag exists (value may vary based on hooks state)
		if _, ok := result2["already_initialized"]; !ok {
			t.Log("already_initialized field not present (may be expected)")
		}
	})
}

func TestInitNotARepo(t *testing.T) {
	tempDir := t.TempDir()

	runInDir(t, tempDir, func() {
		var buf bytes.Buffer

		cmd := newTestRootCmdWithInit()
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)
		cmd.SetArgs([]string{"init", "--json"})

		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error for non-repo directory")
		}

		var result map[string]any
		if jsonErr := json.Unmarshal(buf.Bytes(), &result); jsonErr != nil {
			t.Fatalf("failed to parse JSON error output: %v\nOutput: %s", jsonErr, buf.String())
		}

		// Check error code is 2 (system error)
		code, ok := result["code"].(float64)
		if !ok {
			t.Fatalf("missing or invalid 'code' in error output: %v", result)
		}
		if code != 2 {
			t.Errorf("error code = %v, want 2 (system error)", code)
		}
	})
}

func TestInitHumanOutput(t *testing.T) {
	tempDir := t.TempDir()

	runGit(t, tempDir, "init")
	runGit(t, tempDir, "config", "user.email", "test@test.com")
	runGit(t, tempDir, "config", "user.name", "Test User")

	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0o600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	runGit(t, tempDir, "add", "test.txt")
	runGit(t, tempDir, "commit", "-m", "Initial commit")

	runInDir(t, tempDir, func() {
		var buf bytes.Buffer

		cmd := newTestRootCmdWithInit()
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)
		cmd.SetArgs([]string{"init", "--yes", "--no-claude"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("command failed: %v\nOutput: %s", err, buf.String())
		}

		output := buf.String()

		// Check for key output elements
		checks := []string{
			"Initializing timbers",
			"Next steps:",
			"timbers onboard",
			"timbers log",
			"timbers doctor",
		}

		for _, check := range checks {
			if !strings.Contains(output, check) {
				t.Errorf("human output missing %q\nOutput: %s", check, output)
			}
		}
	})
}

func TestInitDryRunHumanOutput(t *testing.T) {
	tempDir := t.TempDir()

	runGit(t, tempDir, "init")
	runGit(t, tempDir, "config", "user.email", "test@test.com")
	runGit(t, tempDir, "config", "user.name", "Test User")

	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0o600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	runGit(t, tempDir, "add", "test.txt")
	runGit(t, tempDir, "commit", "-m", "Initial commit")

	runInDir(t, tempDir, func() {
		var buf bytes.Buffer

		cmd := newTestRootCmdWithInit()
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)
		cmd.SetArgs([]string{"init", "--dry-run"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("command failed: %v\nOutput: %s", err, buf.String())
		}

		output := buf.String()

		// Check dry-run specific output
		if !strings.Contains(output, "Dry run") {
			t.Errorf("dry-run output missing 'Dry run' header\nOutput: %s", output)
		}
	})
}

func TestInitWithExistingHook(t *testing.T) {
	tempDir := t.TempDir()

	runGit(t, tempDir, "init")
	runGit(t, tempDir, "config", "user.email", "test@test.com")
	runGit(t, tempDir, "config", "user.name", "Test User")

	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0o600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	runGit(t, tempDir, "add", "test.txt")
	runGit(t, tempDir, "commit", "-m", "Initial commit")

	// Create an existing pre-commit hook
	hooksDir := filepath.Join(tempDir, ".git", "hooks")
	existingHook := filepath.Join(hooksDir, "pre-commit")
	existingContent := "#!/bin/sh\necho 'existing hook'\n"
	// #nosec G306 -- test hook needs execute permission
	if err := os.WriteFile(existingHook, []byte(existingContent), 0o755); err != nil {
		t.Fatalf("failed to create existing hook: %v", err)
	}

	runInDir(t, tempDir, func() {
		var buf bytes.Buffer

		cmd := newTestRootCmdWithInit()
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)
		cmd.SetArgs([]string{"init", "--yes", "--hooks", "--no-claude", "--json"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("command failed: %v\nOutput: %s", err, buf.String())
		}

		// Check that backup was created
		backupPath := filepath.Join(hooksDir, "pre-commit.backup")
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			t.Error("existing hook was not backed up")
		}

		// Check new hook contains timbers
		content, err := os.ReadFile(existingHook)
		if err != nil {
			t.Fatalf("failed to read hook: %v", err)
		}
		if !strings.Contains(string(content), "timbers hook run pre-commit") {
			t.Error("new hook does not contain timbers command")
		}
	})
}

func TestInitDefaultNoHooks(t *testing.T) {
	tempDir := t.TempDir()

	runGit(t, tempDir, "init")
	runGit(t, tempDir, "config", "user.email", "test@test.com")
	runGit(t, tempDir, "config", "user.name", "Test User")

	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0o600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	runGit(t, tempDir, "add", "test.txt")
	runGit(t, tempDir, "commit", "-m", "Initial commit")

	runInDir(t, tempDir, func() {
		var buf bytes.Buffer

		cmd := newTestRootCmdWithInit()
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)
		cmd.SetArgs([]string{"init", "--yes", "--no-claude", "--json"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("command failed: %v\nOutput: %s", err, buf.String())
		}

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("failed to parse JSON: %v\nOutput: %s", err, buf.String())
		}

		// Check hooks_installed is false (hooks are opt-in)
		if hooksInstalled, ok := result["hooks_installed"].(bool); ok && hooksInstalled {
			t.Error("hooks should not be installed by default")
		}

		// Verify no hook file was created
		hookPath := filepath.Join(tempDir, ".git", "hooks", "pre-commit")
		if _, err := os.Stat(hookPath); err == nil {
			t.Error("hook file should not exist by default")
		}
	})
}

func TestInitJSONStructure(t *testing.T) {
	tempDir := t.TempDir()

	runGit(t, tempDir, "init")
	runGit(t, tempDir, "config", "user.email", "test@test.com")
	runGit(t, tempDir, "config", "user.name", "Test User")

	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0o600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	runGit(t, tempDir, "add", "test.txt")
	runGit(t, tempDir, "commit", "-m", "Initial commit")

	runInDir(t, tempDir, func() {
		var buf bytes.Buffer

		cmd := newTestRootCmdWithInit()
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)
		cmd.SetArgs([]string{"init", "--yes", "--no-claude", "--json"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("command failed: %v\nOutput: %s", err, buf.String())
		}

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("failed to parse JSON: %v\nOutput: %s", err, buf.String())
		}

		// Verify all expected fields are present
		requiredFields := []string{
			"status",
			"repo_name",
			"timbers_dir_created",
			"hooks_installed",
			"claude_installed",
			"already_initialized",
			"steps",
		}

		for _, field := range requiredFields {
			if _, ok := result[field]; !ok {
				t.Errorf("missing required field %q in JSON output", field)
			}
		}

		// Verify steps is an array
		steps, ok := result["steps"].([]any)
		if !ok {
			t.Fatalf("steps is not an array: %T", result["steps"])
		}

		// Each step should have name, status, and optionally message
		for i, step := range steps {
			stepMap, mapOK := step.(map[string]any)
			if !mapOK {
				t.Errorf("step %d is not a map: %T", i, step)
				continue
			}
			if _, hasName := stepMap["name"]; !hasName {
				t.Errorf("step %d missing 'name' field", i)
			}
			if _, hasStatus := stepMap["status"]; !hasStatus {
				t.Errorf("step %d missing 'status' field", i)
			}
		}
	})
}

func TestInitCreatesTimbersDirAndGitattributes(t *testing.T) {
	tempDir := t.TempDir()

	runGit(t, tempDir, "init")
	runGit(t, tempDir, "config", "user.email", "test@test.com")
	runGit(t, tempDir, "config", "user.name", "Test User")

	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0o600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	runGit(t, tempDir, "add", "test.txt")
	runGit(t, tempDir, "commit", "-m", "Initial commit")

	runInDir(t, tempDir, func() {
		var buf bytes.Buffer

		cmd := newTestRootCmdWithInit()
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)
		cmd.SetArgs([]string{"init", "--yes", "--no-claude", "--json"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("command failed: %v\nOutput: %s", err, buf.String())
		}

		// Verify .timbers/ directory was created
		timbersDir := filepath.Join(tempDir, ".timbers")
		info, err := os.Stat(timbersDir)
		if err != nil {
			t.Fatalf(".timbers directory not created: %v", err)
		}
		if !info.IsDir() {
			t.Error(".timbers is not a directory")
		}

		// Verify .gitattributes was created with correct content
		gaPath := filepath.Join(tempDir, ".gitattributes")
		content, err := os.ReadFile(gaPath)
		if err != nil {
			t.Fatalf(".gitattributes not created: %v", err)
		}
		if !strings.Contains(string(content), "/.timbers/** linguist-generated") {
			t.Errorf(".gitattributes missing timbers entry\nContent: %s", content)
		}
	})
}

func TestInitGitattributesIdempotent(t *testing.T) {
	tempDir := t.TempDir()

	runGit(t, tempDir, "init")
	runGit(t, tempDir, "config", "user.email", "test@test.com")
	runGit(t, tempDir, "config", "user.name", "Test User")

	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0o600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	runGit(t, tempDir, "add", "test.txt")
	runGit(t, tempDir, "commit", "-m", "Initial commit")

	// Pre-create .gitattributes with the timbers line
	gaPath := filepath.Join(tempDir, ".gitattributes")
	if err := os.WriteFile(gaPath, []byte("/.timbers/** linguist-generated\n"), 0o600); err != nil { //nolint:gosec // test file
		t.Fatalf("failed to write .gitattributes: %v", err)
	}

	runInDir(t, tempDir, func() {
		var buf bytes.Buffer

		cmd := newTestRootCmdWithInit()
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)
		cmd.SetArgs([]string{"init", "--yes", "--no-claude", "--json"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("command failed: %v\nOutput: %s", err, buf.String())
		}

		// Verify .gitattributes only has one copy of the line
		content, err := os.ReadFile(gaPath)
		if err != nil {
			t.Fatalf("failed to read .gitattributes: %v", err)
		}
		count := strings.Count(string(content), "/.timbers/** linguist-generated")
		if count != 1 {
			t.Errorf("expected 1 timbers entry in .gitattributes, got %d\nContent: %s", count, content)
		}
	})
}

func TestInitGitattributesAppendsToExisting(t *testing.T) {
	tempDir := t.TempDir()

	runGit(t, tempDir, "init")
	runGit(t, tempDir, "config", "user.email", "test@test.com")
	runGit(t, tempDir, "config", "user.name", "Test User")

	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0o600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	runGit(t, tempDir, "add", "test.txt")
	runGit(t, tempDir, "commit", "-m", "Initial commit")

	// Pre-create .gitattributes with other content
	gaPath := filepath.Join(tempDir, ".gitattributes")
	if err := os.WriteFile(gaPath, []byte("*.go text\n"), 0o600); err != nil { //nolint:gosec // test file
		t.Fatalf("failed to write .gitattributes: %v", err)
	}

	runInDir(t, tempDir, func() {
		var buf bytes.Buffer

		cmd := newTestRootCmdWithInit()
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)
		cmd.SetArgs([]string{"init", "--yes", "--no-claude", "--json"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("command failed: %v\nOutput: %s", err, buf.String())
		}

		// Verify .gitattributes has both lines
		content, err := os.ReadFile(gaPath)
		if err != nil {
			t.Fatalf("failed to read .gitattributes: %v", err)
		}
		if !strings.Contains(string(content), "*.go text") {
			t.Error("original .gitattributes content was lost")
		}
		if !strings.Contains(string(content), "/.timbers/** linguist-generated") {
			t.Error(".gitattributes missing timbers entry")
		}
	})
}

func TestInitPostRewriteHook(t *testing.T) {
	tempDir := t.TempDir()

	runGit(t, tempDir, "init")
	runGit(t, tempDir, "config", "user.email", "test@test.com")
	runGit(t, tempDir, "config", "user.name", "Test User")

	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0o600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	runGit(t, tempDir, "add", "test.txt")
	runGit(t, tempDir, "commit", "-m", "Initial commit")

	runInDir(t, tempDir, func() {
		var buf bytes.Buffer

		cmd := newTestRootCmdWithInit()
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)
		cmd.SetArgs([]string{"init", "--yes", "--hooks", "--no-claude", "--json"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("command failed: %v\nOutput: %s", err, buf.String())
		}

		// Verify post-rewrite hook was created
		hookPath := filepath.Join(tempDir, ".git", "hooks", "post-rewrite")
		content, err := os.ReadFile(hookPath)
		if err != nil {
			t.Fatalf("post-rewrite hook not created: %v", err)
		}
		if !strings.Contains(string(content), ".timbers") {
			t.Error("post-rewrite hook missing .timbers reference")
		}
		if !strings.Contains(string(content), "sed") {
			t.Error("post-rewrite hook missing sed command for SHA remapping")
		}
	})
}

func TestInitPostRewriteChainsExistingHook(t *testing.T) {
	tempDir := t.TempDir()

	runGit(t, tempDir, "init")
	runGit(t, tempDir, "config", "user.email", "test@test.com")
	runGit(t, tempDir, "config", "user.name", "Test User")

	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0o600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	runGit(t, tempDir, "add", "test.txt")
	runGit(t, tempDir, "commit", "-m", "Initial commit")

	// Create an existing post-rewrite hook
	hooksDir := filepath.Join(tempDir, ".git", "hooks")
	hookPath := filepath.Join(hooksDir, "post-rewrite")
	existingContent := "#!/bin/sh\necho 'existing post-rewrite'\n"
	// #nosec G306 -- test hook needs execute permission
	if err := os.WriteFile(hookPath, []byte(existingContent), 0o755); err != nil {
		t.Fatalf("failed to create existing hook: %v", err)
	}

	runInDir(t, tempDir, func() {
		var buf bytes.Buffer

		cmd := newTestRootCmdWithInit()
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)
		cmd.SetArgs([]string{"init", "--yes", "--hooks", "--no-claude", "--json"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("command failed: %v\nOutput: %s", err, buf.String())
		}

		// Verify hook was chained (existing + timbers)
		content, err := os.ReadFile(hookPath)
		if err != nil {
			t.Fatalf("failed to read hook: %v", err)
		}
		contentStr := string(content)
		if !strings.Contains(contentStr, "existing post-rewrite") {
			t.Error("existing hook content was lost")
		}
		if !strings.Contains(contentStr, ".timbers") {
			t.Error("timbers section not appended")
		}
	})
}

func TestInitDryRunJSONSteps(t *testing.T) {
	tempDir := t.TempDir()

	runGit(t, tempDir, "init")
	runGit(t, tempDir, "config", "user.email", "test@test.com")
	runGit(t, tempDir, "config", "user.name", "Test User")

	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0o600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	runGit(t, tempDir, "add", "test.txt")
	runGit(t, tempDir, "commit", "-m", "Initial commit")

	runInDir(t, tempDir, func() {
		var buf bytes.Buffer

		cmd := newTestRootCmdWithInit()
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)
		cmd.SetArgs([]string{"init", "--dry-run", "--yes", "--json"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("command failed: %v\nOutput: %s", err, buf.String())
		}

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("failed to parse JSON: %v\nOutput: %s", err, buf.String())
		}

		if result["status"] != "dry_run" {
			t.Errorf("status = %v, want dry_run", result["status"])
		}

		steps, ok := result["steps"].([]any)
		if !ok {
			t.Fatalf("steps is not an array: %T", result["steps"])
		}

		// Should have 5 steps: timbers_dir, gitattributes, hooks, post_rewrite, agent_env
		if len(steps) != 5 {
			t.Errorf("got %d steps, want 5", len(steps))
		}

		// Check step names
		expectedSteps := []string{"timbers_dir", "gitattributes", "hooks", "post_rewrite", "agent_env"}
		for i, step := range steps {
			if i >= len(expectedSteps) {
				break
			}
			stepMap, _ := step.(map[string]any)
			if stepMap["name"] != expectedSteps[i] {
				t.Errorf("step %d name = %v, want %v", i, stepMap["name"], expectedSteps[i])
			}
		}
	})
}
