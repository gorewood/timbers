package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorewood/timbers/internal/setup"
)

func TestUninstallDryRunJSON(t *testing.T) {
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
		cmd.SetArgs([]string{"uninstall", "--dry-run", "--json"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("command failed: %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("failed to parse JSON: %v\nOutput: %s", err, buf.String())
		}

		if result["status"] != "dry_run" {
			t.Errorf("status = %v, want dry_run", result["status"])
		}
		if _, ok := result["binary_path"]; ok {
			t.Error("binary_path should not be present without --binary flag")
		}
		if result["in_repo"] != true {
			t.Errorf("in_repo = %v, want true", result["in_repo"])
		}
		if _, ok := result["hooks_installed"]; !ok {
			t.Error("missing hooks_installed field")
		}
		if _, ok := result["claude_installed"]; !ok {
			t.Error("missing claude_installed field")
		}
	})
}

func TestUninstallDryRunJSONWithBinary(t *testing.T) {
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
		cmd.SetArgs([]string{"uninstall", "--dry-run", "--binary", "--json"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("command failed: %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("failed to parse JSON: %v\nOutput: %s", err, buf.String())
		}

		if result["status"] != "dry_run" {
			t.Errorf("status = %v, want dry_run", result["status"])
		}
		if _, ok := result["binary_path"]; !ok {
			t.Error("missing binary_path field with --binary flag")
		}
	})
}

func TestUninstallDryRunHuman(t *testing.T) {
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
		cmd.SetArgs([]string{"uninstall", "--dry-run"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("command failed: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "Dry run") {
			t.Errorf("output missing 'Dry run'\nOutput: %s", output)
		}
		if strings.Contains(output, "Remove binary") || strings.Contains(output, "Binary:") {
			t.Errorf("output should not contain binary info\nOutput: %s", output)
		}
	})
}

func TestUninstallDryRunWithNotesRef(t *testing.T) {
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
	runGit(t, tempDir, "notes", "--ref=refs/notes/timbers", "add", "-m", "test note", "HEAD")

	runInDir(t, tempDir, func() {
		var buf bytes.Buffer
		cmd := newRootCmd()
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)
		cmd.SetArgs([]string{"uninstall", "--dry-run", "--json"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("command failed: %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("failed to parse JSON: %v\nOutput: %s", err, buf.String())
		}

		if result["notes_ref_exists"] != true {
			t.Errorf("notes_ref_exists = %v, want true", result["notes_ref_exists"])
		}
		if count, ok := result["notes_entry_count"].(float64); !ok || count != 1 {
			t.Errorf("notes_entry_count = %v, want 1", result["notes_entry_count"])
		}
	})
}

func TestUninstallDryRunNotInRepo(t *testing.T) {
	tempDir := t.TempDir()

	runInDir(t, tempDir, func() {
		var buf bytes.Buffer
		cmd := newRootCmd()
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)
		cmd.SetArgs([]string{"uninstall", "--dry-run", "--json"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("command failed: %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("failed to parse JSON: %v\nOutput: %s", err, buf.String())
		}

		if result["in_repo"] != false {
			t.Errorf("in_repo = %v, want false", result["in_repo"])
		}
	})
}

func TestUninstallCancellation(t *testing.T) {
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
		var outBuf bytes.Buffer
		inBuf := strings.NewReader("n\n")

		cmd := newRootCmd()
		cmd.SetOut(&outBuf)
		cmd.SetErr(&outBuf)
		cmd.SetIn(inBuf)
		cmd.SetArgs([]string{"uninstall"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("command failed: %v", err)
		}

		output := outBuf.String()
		if !strings.Contains(output, "cancelled") {
			t.Errorf("output should contain 'cancelled'\nOutput: %s", output)
		}
	})
}

func TestUninstallHelpText(t *testing.T) {
	var buf bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"uninstall", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := buf.String()
	checks := []string{
		"uninstall", "--dry-run", "--force", "--json",
		"--binary", "--keep-notes", "hooks", "Claude",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("help output missing %q\nOutput: %s", check, output)
		}
	}
}

func TestFindNotesConfigs(t *testing.T) {
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
	runGit(t, tempDir, "remote", "add", "origin", "https://github.com/example/repo.git")
	runGit(t, tempDir, "config", "--add", "remote.origin.fetch", "+refs/notes/timbers:refs/notes/timbers")

	runInDir(t, tempDir, func() {
		configs := setup.FindNotesConfigs()
		if len(configs) != 1 || configs[0] != "origin" {
			t.Errorf("configs = %v, want [origin]", configs)
		}
	})
}

func TestUninstallWithHooks(t *testing.T) {
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

	hooksDir := filepath.Join(tempDir, ".git", "hooks")
	hookPath := filepath.Join(hooksDir, "pre-commit")
	hookContent := "#!/bin/sh\n# timbers pre-commit hook\ntimbers hook run pre-commit\n"
	if err := os.WriteFile(hookPath, []byte(hookContent), 0o755); err != nil { //nolint:gosec
		t.Fatalf("failed to write hook: %v", err)
	}

	runInDir(t, tempDir, func() {
		var buf bytes.Buffer
		cmd := newRootCmd()
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)
		cmd.SetArgs([]string{"uninstall", "--force", "--json"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("command failed: %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("failed to parse JSON: %v\nOutput: %s", err, buf.String())
		}

		if result["hooks_removed"] != true {
			t.Errorf("hooks_removed = %v, want true", result["hooks_removed"])
		}

		if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
			t.Error("hook file should be removed")
		}
	})
}

func TestUninstallKeepNotes(t *testing.T) {
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
	runGit(t, tempDir, "notes", "--ref=refs/notes/timbers", "add", "-m", "test note", "HEAD")

	hooksDir := filepath.Join(tempDir, ".git", "hooks")
	hookPath := filepath.Join(hooksDir, "pre-commit")
	hookContent := "#!/bin/sh\n# timbers pre-commit hook\ntimbers hook run pre-commit\n"
	if err := os.WriteFile(hookPath, []byte(hookContent), 0o755); err != nil { //nolint:gosec
		t.Fatalf("failed to write hook: %v", err)
	}

	runInDir(t, tempDir, func() {
		var buf bytes.Buffer
		cmd := newRootCmd()
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)
		cmd.SetArgs([]string{"uninstall", "--force", "--keep-notes", "--json"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("command failed: %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("failed to parse JSON: %v\nOutput: %s", err, buf.String())
		}

		if result["hooks_removed"] != true {
			t.Errorf("hooks_removed = %v, want true", result["hooks_removed"])
		}
		if result["notes_removed"] != false {
			t.Errorf("notes_removed = %v, want false", result["notes_removed"])
		}
		if result["keep_notes"] != true {
			t.Errorf("keep_notes = %v, want true", result["keep_notes"])
		}

		if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
			t.Error("hook file should be removed")
		}

		out := runGitOutput(t, tempDir, "show-ref", "--verify", "refs/notes/timbers")
		if out == "" {
			t.Error("notes ref should still exist with --keep-notes")
		}
	})
}

func TestUninstallIdempotent(t *testing.T) {
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

	for i := range 2 {
		runInDir(t, tempDir, func() {
			var buf bytes.Buffer
			cmd := newRootCmd()
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs([]string{"uninstall", "--force", "--json"})

			if err := cmd.Execute(); err != nil {
				t.Fatalf("uninstall %d failed: %v", i+1, err)
			}

			var result map[string]any
			if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
				t.Fatalf("failed to parse JSON: %v\nOutput: %s", err, buf.String())
			}

			if result["status"] != "ok" {
				t.Errorf("status = %v, want ok", result["status"])
			}
		})
	}
}

func TestFormatEntryCount(t *testing.T) {
	tests := []struct {
		count int
		want  string
	}{
		{0, "0 entries"},
		{1, "1 entry"},
		{2, "2 entries"},
		{10, "10 entries"},
	}

	for _, tt := range tests {
		got := formatEntryCount(tt.count)
		if got != tt.want {
			t.Errorf("formatEntryCount(%d) = %q, want %q", tt.count, got, tt.want)
		}
	}
}
