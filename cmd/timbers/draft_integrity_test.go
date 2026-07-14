package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDraftFailsClosedOnCorruptEntry(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init")
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "config", "user.name", "Test User")
	timbersDir := filepath.Join(repo, ".timbers")
	if err := os.MkdirAll(timbersDir, 0o755); err != nil {
		t.Fatal(err)
	}
	badPath := filepath.Join(timbersDir, "bad.json")
	if err := os.WriteFile(badPath, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	runInDir(t, repo, func() {
		cmd := newDraftCmd()
		cmd.SilenceUsage = true
		cmd.SetArgs([]string{"release-notes", "--last", "1"})
		var stdout, stderr strings.Builder
		cmd.SetOut(&stdout)
		cmd.SetErr(&stderr)
		err := cmd.Execute()
		if err == nil {
			t.Fatal("draft succeeded with a malformed ledger entry")
		}
		if !strings.Contains(err.Error(), badPath) {
			t.Fatalf("error = %q, want corrupt path %q", err, badPath)
		}
		if stdout.Len() != 0 {
			t.Fatalf("stdout = %q, want no partial report", stdout.String())
		}
	})
}
