package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNotesInitCommand(t *testing.T) {
	tempDir := t.TempDir()

	// Initialize a git repo
	runGit(t, tempDir, "init")
	runGit(t, tempDir, "config", "user.email", "test@test.com")
	runGit(t, tempDir, "config", "user.name", "Test User")

	// Create a file and commit (needed for notes ref to work)
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
	}{
		{
			name: "JSON output contains all fields",
			args: []string{"notes", "init", "--json"},
			wantFields: map[string]any{
				"status":     "ok",
				"remote":     "origin",
				"configured": true,
			},
		},
		{
			name: "JSON output with custom remote",
			args: []string{"notes", "init", "--remote", "upstream", "--json"},
			wantFields: map[string]any{
				"status":     "ok",
				"remote":     "upstream",
				"configured": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runInDir(t, tempDir, func() {
				var buf bytes.Buffer

				cmd := newRootCmd()
				cmd.SetOut(&buf)
				cmd.SetErr(&buf)
				cmd.SetArgs(tt.args)

				if err := cmd.Execute(); err != nil {
					t.Fatalf("command failed: %v", err)
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
			})
		})
	}
}

func TestNotesInitIdempotent(t *testing.T) {
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
		// First call
		var buf1 bytes.Buffer
		cmd1 := newRootCmd()
		cmd1.SetOut(&buf1)
		cmd1.SetErr(&buf1)
		cmd1.SetArgs([]string{"notes", "init", "--json"})

		if err := cmd1.Execute(); err != nil {
			t.Fatalf("first init failed: %v", err)
		}

		// Second call (should succeed - idempotent)
		var buf2 bytes.Buffer
		cmd2 := newRootCmd()
		cmd2.SetOut(&buf2)
		cmd2.SetErr(&buf2)
		cmd2.SetArgs([]string{"notes", "init", "--json"})

		if err := cmd2.Execute(); err != nil {
			t.Fatalf("second init failed: %v", err)
		}

		// Both should have status "ok"
		var result map[string]any
		if err := json.Unmarshal(buf2.Bytes(), &result); err != nil {
			t.Fatalf("failed to parse JSON output: %v", err)
		}
		if result["status"] != "ok" {
			t.Errorf("second init status = %v, want ok", result["status"])
		}
	})
}

func TestNotesInitHumanOutput(t *testing.T) {
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
		// First call - should say "Configured"
		var buf1 bytes.Buffer
		cmd1 := newRootCmd()
		cmd1.SetOut(&buf1)
		cmd1.SetErr(&buf1)
		cmd1.SetArgs([]string{"notes", "init"})

		if err := cmd1.Execute(); err != nil {
			t.Fatalf("first init failed: %v", err)
		}

		output1 := buf1.String()
		if !strings.Contains(output1, "Configured") {
			t.Errorf("first init output missing 'Configured'\nOutput: %s", output1)
		}

		// Second call - should say "already configured"
		var buf2 bytes.Buffer
		cmd2 := newRootCmd()
		cmd2.SetOut(&buf2)
		cmd2.SetErr(&buf2)
		cmd2.SetArgs([]string{"notes", "init"})

		if err := cmd2.Execute(); err != nil {
			t.Fatalf("second init failed: %v", err)
		}

		output2 := buf2.String()
		if !strings.Contains(output2, "already configured") {
			t.Errorf("second init output missing 'already configured'\nOutput: %s", output2)
		}
	})
}

func TestNotesStatusCommand(t *testing.T) {
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

	tests := []struct {
		name       string
		args       []string
		wantFields map[string]any
	}{
		{
			name: "JSON output contains all fields",
			args: []string{"notes", "status", "--json"},
			wantFields: map[string]any{
				"ref_exists":  false,
				"configured":  false,
				"entry_count": float64(0),
				"remote":      "origin",
			},
		},
		{
			name: "JSON output with custom remote",
			args: []string{"notes", "status", "--remote", "upstream", "--json"},
			wantFields: map[string]any{
				"remote": "upstream",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runInDir(t, tempDir, func() {
				var buf bytes.Buffer

				cmd := newRootCmd()
				cmd.SetOut(&buf)
				cmd.SetErr(&buf)
				cmd.SetArgs(tt.args)

				if err := cmd.Execute(); err != nil {
					t.Fatalf("command failed: %v", err)
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
			})
		})
	}
}

func TestNotesStatusHumanOutput(t *testing.T) {
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

		cmd := newRootCmd()
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)
		cmd.SetArgs([]string{"notes", "status"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("command failed: %v", err)
		}

		output := buf.String()

		checks := []string{
			"Remote",
			"Configured",
			"Entries",
		}

		for _, check := range checks {
			if !strings.Contains(output, check) {
				t.Errorf("human output missing %q\nOutput: %s", check, output)
			}
		}
	})
}

func TestNotesNotARepo(t *testing.T) {
	tempDir := t.TempDir()

	subcommands := []string{"init", "status", "push", "fetch"}

	for _, subcmd := range subcommands {
		t.Run(subcmd, func(t *testing.T) {
			runInDir(t, tempDir, func() {
				var buf bytes.Buffer

				cmd := newRootCmd()
				cmd.SetOut(&buf)
				cmd.SetErr(&buf)
				cmd.SetArgs([]string{"notes", subcmd, "--json"})

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
