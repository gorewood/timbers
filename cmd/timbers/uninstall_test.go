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
		cmd.SetArgs([]string{"uninstall", "--dry-run", "--json"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("command failed: %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, buf.String())
		}

		// Check required fields (without --binary, binary_path should not be present)
		if result["status"] != "dry_run" {
			t.Errorf("status = %v, want dry_run", result["status"])
		}
		if _, ok := result["binary_path"]; ok {
			t.Error("binary_path should not be present without --binary flag")
		}
		if result["in_repo"] != true {
			t.Errorf("in_repo = %v, want true", result["in_repo"])
		}
	})
}

func TestUninstallDryRunJSONWithBinary(t *testing.T) {
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
		cmd.SetArgs([]string{"uninstall", "--dry-run", "--binary", "--json"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("command failed: %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, buf.String())
		}

		// With --binary, binary_path should be present
		if result["status"] != "dry_run" {
			t.Errorf("status = %v, want dry_run", result["status"])
		}
		if _, ok := result["binary_path"]; !ok {
			t.Error("missing binary_path field with --binary flag")
		}
		if result["in_repo"] != true {
			t.Errorf("in_repo = %v, want true", result["in_repo"])
		}
	})
}

func TestUninstallDryRunHuman(t *testing.T) {
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
		cmd.SetArgs([]string{"uninstall", "--dry-run"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("command failed: %v", err)
		}

		output := buf.String()

		// Without --binary, should not mention binary removal
		if !strings.Contains(output, "Dry run") {
			t.Errorf("output missing 'Dry run'\nOutput: %s", output)
		}
		if strings.Contains(output, "Remove binary") {
			t.Errorf("output should not contain 'Remove binary' without --binary flag\nOutput: %s", output)
		}
	})
}

func TestUninstallDryRunHumanWithBinary(t *testing.T) {
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
		cmd.SetArgs([]string{"uninstall", "--dry-run", "--binary"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("command failed: %v", err)
		}

		output := buf.String()

		checks := []string{
			"Dry run",
			"Remove binary",
		}

		for _, check := range checks {
			if !strings.Contains(output, check) {
				t.Errorf("output missing %q\nOutput: %s", check, output)
			}
		}
	})
}

func TestUninstallDryRunWithNotesRef(t *testing.T) {
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

	// Add a timbers note
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
			t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, buf.String())
		}

		if result["notes_ref_exists"] != true {
			t.Errorf("notes_ref_exists = %v, want true", result["notes_ref_exists"])
		}
	})
}

func TestUninstallDryRunWithNotesConfig(t *testing.T) {
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

	// Add a remote and configure notes fetch
	runGit(t, tempDir, "remote", "add", "origin", "https://github.com/example/repo.git")
	runGit(t, tempDir, "config", "--add", "remote.origin.fetch", "+refs/notes/timbers:refs/notes/timbers")

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
			t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, buf.String())
		}

		configs, ok := result["configs_to_remove"].([]any)
		if !ok {
			t.Fatalf("configs_to_remove not a slice: %T", result["configs_to_remove"])
		}
		if len(configs) != 1 || configs[0] != "origin" {
			t.Errorf("configs_to_remove = %v, want [origin]", configs)
		}
	})
}

func TestUninstallDryRunNotInRepo(t *testing.T) {
	tempDir := t.TempDir()
	// No git init - just an empty directory

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
			t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, buf.String())
		}

		if result["in_repo"] != false {
			t.Errorf("in_repo = %v, want false", result["in_repo"])
		}
	})
}

func TestUninstallCancellation(t *testing.T) {
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
		var outBuf bytes.Buffer

		// Simulate user typing "n" for no
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
		"uninstall",
		"--dry-run",
		"--force",
		"--json",
		"binary",
		"notes",
	}

	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("help output missing %q\nOutput: %s", check, output)
		}
	}
}

func TestFindNotesConfigs(t *testing.T) {
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

	// Add multiple remotes
	runGit(t, tempDir, "remote", "add", "origin", "https://github.com/example/repo.git")
	runGit(t, tempDir, "remote", "add", "upstream", "https://github.com/example/upstream.git")

	// Configure notes fetch for both
	runGit(t, tempDir, "config", "--add", "remote.origin.fetch", "+refs/notes/timbers:refs/notes/timbers")
	runGit(t, tempDir, "config", "--add", "remote.upstream.fetch", "+refs/notes/timbers:refs/notes/timbers")

	runInDir(t, tempDir, func() {
		configs := findNotesConfigs()

		if len(configs) != 2 {
			t.Errorf("expected 2 configs, got %d: %v", len(configs), configs)
		}

		hasOrigin := false
		hasUpstream := false
		for _, c := range configs {
			if c == "origin" {
				hasOrigin = true
			}
			if c == "upstream" {
				hasUpstream = true
			}
		}

		if !hasOrigin {
			t.Error("missing origin in configs")
		}
		if !hasUpstream {
			t.Error("missing upstream in configs")
		}
	})
}
