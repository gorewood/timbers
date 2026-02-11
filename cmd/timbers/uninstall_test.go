package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func TestUninstallDryRunWithTimbersDir(t *testing.T) {
	tempDir := t.TempDir()

	runGit(t, tempDir, "init")
	runGit(t, tempDir, "config", "user.email", "test@test.com")
	runGit(t, tempDir, "config", "user.name", "Test User")

	timbersDir := filepath.Join(tempDir, ".timbers")
	if err := os.MkdirAll(timbersDir, 0o755); err != nil {
		t.Fatalf("failed to create .timbers dir: %v", err)
	}
	entryFile := filepath.Join(timbersDir, "tb_test.json")
	if err := os.WriteFile(entryFile, []byte(`{"id":"tb_test"}`), 0o600); err != nil {
		t.Fatalf("failed to write entry: %v", err)
	}

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

		if result["timbers_dir_exists"] != true {
			t.Errorf("timbers_dir_exists = %v, want true", result["timbers_dir_exists"])
		}
		if count, ok := result["entry_count"].(float64); !ok || count != 1 {
			t.Errorf("entry_count = %v, want 1", result["entry_count"])
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
		"--binary", "--keep-data", "hooks", "Claude",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("help output missing %q\nOutput: %s", check, output)
		}
	}
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

func TestUninstallKeepData(t *testing.T) {
	tempDir := t.TempDir()

	runGit(t, tempDir, "init")
	runGit(t, tempDir, "config", "user.email", "test@test.com")
	runGit(t, tempDir, "config", "user.name", "Test User")

	// Create .timbers/ with an entry
	timbersDir := filepath.Join(tempDir, ".timbers")
	if err := os.MkdirAll(timbersDir, 0o755); err != nil {
		t.Fatalf("failed to create .timbers dir: %v", err)
	}
	entryFile := filepath.Join(timbersDir, "tb_test.json")
	if err := os.WriteFile(entryFile, []byte(`{"id":"tb_test"}`), 0o600); err != nil {
		t.Fatalf("failed to write entry: %v", err)
	}

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
		cmd.SetArgs([]string{"uninstall", "--force", "--keep-data", "--json"})

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
		if result["timbers_dir_removed"] != false {
			t.Errorf("timbers_dir_removed = %v, want false", result["timbers_dir_removed"])
		}
		if result["keep_data"] != true {
			t.Errorf("keep_data = %v, want true", result["keep_data"])
		}

		if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
			t.Error("hook file should be removed")
		}

		// .timbers/ dir should still exist with --keep-data
		if _, err := os.Stat(entryFile); os.IsNotExist(err) {
			t.Error("entry file should still exist with --keep-data")
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
