package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestStatusCommand(t *testing.T) {
	// Create a temp directory for test repo
	tempDir := t.TempDir()

	// Initialize a git repo
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

	// Get expected values
	head := strings.TrimSpace(runGitOutput(t, tempDir, "rev-parse", "HEAD"))
	branch := strings.TrimSpace(runGitOutput(t, tempDir, "rev-parse", "--abbrev-ref", "HEAD"))
	repoName := filepath.Base(tempDir)

	tests := []struct {
		name       string
		args       []string
		wantFields map[string]any
	}{
		{
			name: "JSON output contains all fields",
			args: []string{"status", "--json"},
			wantFields: map[string]any{
				"repo":             repoName,
				"branch":           branch,
				"head":             head,
				"notes_ref":        "refs/notes/timbers",
				"notes_configured": false,
				"entry_count":      float64(0), // JSON numbers are float64
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runInDir(t, tempDir, func() {
				// Create command and capture output
				var buf bytes.Buffer

				cmd := newRootCmd()
				cmd.SetOut(&buf)
				cmd.SetErr(&buf)
				cmd.SetArgs(tt.args)

				if err := cmd.Execute(); err != nil {
					t.Fatalf("command failed: %v", err)
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
							t.Errorf("field %q = %v (%T), want %v (%T)", key, got, got, want, want)
						}
					}
				}
			})
		})
	}
}

func TestStatusNotARepo(t *testing.T) {
	// Create temp dir that's not a git repo
	tempDir := t.TempDir()

	runInDir(t, tempDir, func() {
		var buf bytes.Buffer

		cmd := newRootCmd()
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)
		cmd.SetArgs([]string{"status", "--json"})

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

func TestStatusHumanOutput(t *testing.T) {
	// Create a temp directory for test repo
	tempDir := t.TempDir()

	// Initialize a git repo
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

	runInDir(t, tempDir, func() {
		var buf bytes.Buffer

		cmd := newRootCmd()
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)
		cmd.SetArgs([]string{"status"}) // No --json flag = human output

		if err := cmd.Execute(); err != nil {
			t.Fatalf("command failed: %v", err)
		}

		output := buf.String()

		// Check for key information in human output
		checks := []string{
			filepath.Base(tempDir), // repo name
			"Branch",               // branch label
			"HEAD",                 // head label
			"Entries",              // entry count
		}

		for _, check := range checks {
			if !strings.Contains(output, check) {
				t.Errorf("human output missing %q\nOutput: %s", check, output)
			}
		}
	})
}

// runInDir changes to the given directory, runs testFunc, then restores the original directory.
func runInDir(t *testing.T, dir string, testFunc func()) {
	t.Helper()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir to %s: %v", dir, err)
	}
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Errorf("failed to restore dir: %v", err)
		}
	}()
	testFunc()
}

// runGit runs a git command in the given directory.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\nOutput: %s", args, err, out)
	}
}

// runGitOutput runs a git command and returns stdout.
func runGitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v failed: %v", args, err)
	}
	return string(out)
}
