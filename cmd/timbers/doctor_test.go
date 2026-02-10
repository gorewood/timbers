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

// newTestRootCmdWithDoctor creates the root command with doctor registered for testing.
// This is needed because doctor is not yet registered in main.go (orchestrator handles that).
func newTestRootCmdWithDoctor() *cobra.Command {
	cmd := newRootCmd()
	cmd.AddCommand(newDoctorCmd())
	return cmd
}

func TestDoctorCommand(t *testing.T) {
	// Create a temp directory for test repo
	tempDir := t.TempDir()

	// Initialize a git repo with notes configured
	runGit(t, tempDir, "init")
	runGit(t, tempDir, "config", "user.email", "test@test.com")
	runGit(t, tempDir, "config", "user.name", "Test User")

	// Create a file and commit
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	runGit(t, tempDir, "add", "test.txt")
	runGit(t, tempDir, "commit", "-m", "Initial commit")

	tests := []struct {
		name       string
		args       []string
		wantFields map[string]any
		wantInJSON []string
	}{
		{
			name: "JSON output contains check categories",
			args: []string{"doctor", "--json"},
			wantInJSON: []string{
				"core",
				"workflow",
				"integration",
				"summary",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runInDir(t, tempDir, func() {
				var buf bytes.Buffer

				cmd := newTestRootCmdWithDoctor()
				cmd.SetOut(&buf)
				cmd.SetErr(&buf)
				cmd.SetArgs(tt.args)

				// Doctor command should not return error even if checks fail
				if err := cmd.Execute(); err != nil {
					t.Fatalf("command failed: %v", err)
				}

				if len(tt.wantInJSON) > 0 {
					var result map[string]any
					if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
						t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, buf.String())
					}

					for _, key := range tt.wantInJSON {
						if _, ok := result[key]; !ok {
							t.Errorf("missing field %q in JSON output", key)
						}
					}
				}
			})
		})
	}
}

func TestDoctorJSONStructure(t *testing.T) {
	tempDir := t.TempDir()

	runGit(t, tempDir, "init")
	runGit(t, tempDir, "config", "user.email", "test@test.com")
	runGit(t, tempDir, "config", "user.name", "Test User")

	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	runGit(t, tempDir, "add", "test.txt")
	runGit(t, tempDir, "commit", "-m", "Initial commit")

	runInDir(t, tempDir, func() {
		var buf bytes.Buffer

		cmd := newTestRootCmdWithDoctor()
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)
		cmd.SetArgs([]string{"doctor", "--json"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("command failed: %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, buf.String())
		}

		// Check summary structure
		summary, ok := result["summary"].(map[string]any)
		if !ok {
			t.Fatalf("summary is not a map: %T", result["summary"])
		}

		// Verify summary fields
		for _, field := range []string{"passed", "warnings", "failed"} {
			if _, exists := summary[field]; !exists {
				t.Errorf("summary missing field %q", field)
			}
		}

		// Check core structure
		core, ok := result["core"].([]any)
		if !ok {
			t.Fatalf("core is not an array: %T", result["core"])
		}

		if len(core) == 0 {
			t.Error("core checks should not be empty")
		}

		// Verify check structure
		firstCheck, ok := core[0].(map[string]any)
		if !ok {
			t.Fatalf("check is not a map: %T", core[0])
		}

		for _, field := range []string{"name", "status", "message"} {
			if _, ok := firstCheck[field]; !ok {
				t.Errorf("check missing field %q", field)
			}
		}
	})
}

func TestDoctorNotARepo(t *testing.T) {
	tempDir := t.TempDir()

	runInDir(t, tempDir, func() {
		var buf bytes.Buffer

		cmd := newTestRootCmdWithDoctor()
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)
		cmd.SetArgs([]string{"doctor", "--json"})

		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error for non-repo directory")
		}

		// Verify JSON error output
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

func TestDoctorHumanOutput(t *testing.T) {
	tempDir := t.TempDir()

	runGit(t, tempDir, "init")
	runGit(t, tempDir, "config", "user.email", "test@test.com")
	runGit(t, tempDir, "config", "user.name", "Test User")

	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	runGit(t, tempDir, "add", "test.txt")
	runGit(t, tempDir, "commit", "-m", "Initial commit")

	runInDir(t, tempDir, func() {
		var buf bytes.Buffer

		cmd := newTestRootCmdWithDoctor()
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)
		cmd.SetArgs([]string{"doctor"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("command failed: %v", err)
		}

		output := buf.String()

		// Check for section headers
		sections := []string{"CORE", "WORKFLOW", "INTEGRATION"}
		for _, section := range sections {
			if !strings.Contains(output, section) {
				t.Errorf("human output missing section %q\nOutput: %s", section, output)
			}
		}

		// Check for summary line
		if !strings.Contains(output, "passed") {
			t.Errorf("human output missing summary\nOutput: %s", output)
		}
	})
}

func TestDoctorQuietMode(t *testing.T) {
	tempDir := t.TempDir()

	runGit(t, tempDir, "init")
	runGit(t, tempDir, "config", "user.email", "test@test.com")
	runGit(t, tempDir, "config", "user.name", "Test User")

	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	runGit(t, tempDir, "add", "test.txt")
	runGit(t, tempDir, "commit", "-m", "Initial commit")

	// Configure notes to ensure most checks pass
	runGit(t, tempDir, "config", "--add", "remote.origin.fetch", "+refs/notes/timbers:refs/notes/timbers")

	runInDir(t, tempDir, func() {
		var buf bytes.Buffer

		cmd := newTestRootCmdWithDoctor()
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)
		cmd.SetArgs([]string{"doctor", "--quiet"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("command failed: %v", err)
		}

		output := buf.String()

		// In quiet mode, we expect warnings to still show up (pending commits, no entries, etc.)
		// The test verifies that --quiet runs without error
		// Quiet mode should still output warnings/failures, so we verify the output is not empty
		if len(output) == 0 {
			t.Error("quiet mode should still produce output when there are warnings")
		}
	})
}

func TestDoctorFixClaudeIntegration(t *testing.T) {
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

	// Set HOME to temp so global check doesn't find existing installs
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	runInDir(t, tempDir, func() {
		var buf bytes.Buffer

		cmd := newTestRootCmdWithDoctor()
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)
		cmd.SetArgs([]string{"doctor", "--fix", "--json"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("command failed: %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("failed to parse JSON: %v\nOutput: %s", err, buf.String())
		}

		// Find Claude Integration check in integration results
		integration, ok := result["integration"].([]any)
		if !ok {
			t.Fatalf("integration is not an array: %T", result["integration"])
		}

		var claudeCheck map[string]any
		for _, item := range integration {
			check, checkOK := item.(map[string]any)
			if checkOK && check["name"] == "Claude Integration" {
				claudeCheck = check
				break
			}
		}

		if claudeCheck == nil {
			t.Fatal("did not find Claude Integration check")
		}

		if claudeCheck["status"] != "pass" {
			t.Errorf("claude check status = %v, want pass (auto-fixed)", claudeCheck["status"])
		}

		// Verify hook was installed at project level, not global
		projectHookPath := filepath.Join(tempDir, ".claude", "hooks", "user_prompt_submit.sh")
		if _, err := os.Stat(projectHookPath); os.IsNotExist(err) {
			t.Error("--fix should install Claude hook at project level")
		}

		globalHookPath := filepath.Join(tmpHome, ".claude", "hooks", "user_prompt_submit.sh")
		if _, err := os.Stat(globalHookPath); !os.IsNotExist(err) {
			t.Error("--fix should not install Claude hook at global level")
		}
	})
}

func TestDoctorWithConfiguredNotes(t *testing.T) {
	tempDir := t.TempDir()

	runGit(t, tempDir, "init")
	runGit(t, tempDir, "config", "user.email", "test@test.com")
	runGit(t, tempDir, "config", "user.name", "Test User")

	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	runGit(t, tempDir, "add", "test.txt")
	runGit(t, tempDir, "commit", "-m", "Initial commit")

	// Configure notes fetch
	runGit(t, tempDir, "config", "--add", "remote.origin.fetch", "+refs/notes/timbers:refs/notes/timbers")

	runInDir(t, tempDir, func() {
		var buf bytes.Buffer

		cmd := newTestRootCmdWithDoctor()
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)
		cmd.SetArgs([]string{"doctor", "--json"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("command failed: %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, buf.String())
		}

		// Find the notes configured check in core
		core, ok := result["core"].([]any)
		if !ok {
			t.Fatalf("core is not an array: %T", result["core"])
		}

		var foundNotesCheck bool
		for _, item := range core {
			check, checkOK := item.(map[string]any)
			if !checkOK {
				continue
			}
			if check["name"] == "Remote Configured" {
				foundNotesCheck = true
				if check["status"] != "pass" {
					t.Errorf("notes configured check status = %v, want pass", check["status"])
				}
			}
		}

		if !foundNotesCheck {
			t.Error("did not find 'Remote Configured' check in core")
		}
	})
}
