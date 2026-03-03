// Package main provides the entry point for the timbers CLI.
package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorewood/timbers/internal/output"
	"github.com/gorewood/timbers/internal/setup"
)

func TestRunQuickHealthCheck(t *testing.T) {
	tempDir := t.TempDir()

	runGit(t, tempDir, "init")
	runGit(t, tempDir, "config", "user.email", "test@test.com")
	runGit(t, tempDir, "config", "user.name", "Test User")

	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0o600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	runGit(t, tempDir, "add", "test.txt")
	runGit(t, tempDir, "commit", "-m", "Initial commit")

	t.Run("reports missing post-commit hook", func(t *testing.T) {
		runInDir(t, tempDir, func() {
			issues := runQuickHealthCheck()

			found := false
			for _, item := range issues {
				if item.Name == "post_commit_hook" {
					found = true
					break
				}
			}
			if !found {
				t.Error("expected post_commit_hook issue")
			}
		})
	})

	t.Run("no issue when post-commit installed", func(t *testing.T) {
		hookPath := filepath.Join(tempDir, ".git", "hooks", "post-commit")
		if err := setup.InstallPostCommitHook(hookPath); err != nil {
			t.Fatalf("failed to install hook: %v", err)
		}

		runInDir(t, tempDir, func() {
			issues := runQuickHealthCheck()

			for _, item := range issues {
				if item.Name == "post_commit_hook" {
					t.Error("should not report post_commit_hook when installed")
				}
			}
		})
	})
}

func TestOutputPrimeHealth(t *testing.T) {
	t.Run("no issues - no output", func(t *testing.T) {
		var buf bytes.Buffer
		printer := output.NewPrinter(&buf, false, false)
		outputPrimeHealth(printer, nil)
		if buf.Len() != 0 {
			t.Errorf("expected no output, got: %s", buf.String())
		}
	})

	t.Run("with issues - shows health section", func(t *testing.T) {
		var buf bytes.Buffer
		printer := output.NewPrinter(&buf, false, false)
		issues := []primeHealthItem{
			{Name: "test_issue", Message: "something is wrong"},
		}
		outputPrimeHealth(printer, issues)
		out := buf.String()
		if !strings.Contains(out, "Health") {
			t.Error("expected Health heading")
		}
		if !strings.Contains(out, "something is wrong") {
			t.Error("expected issue message")
		}
		if !strings.Contains(out, "timbers doctor --fix") {
			t.Error("expected doctor --fix hint")
		}
	})
}
