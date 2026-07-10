package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorewood/timbers/internal/ledger"
)

// newLogAnchorRepo creates a temp git repo with two commits and no ledger
// entries, then returns the repo dir. It is the fixture for --anchor resolution
// tests: a real repo is required because anchor resolution shells to
// `git rev-parse`, which the mock GitOps can't exercise.
func newLogAnchorRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test User")
	writeAndCommit(t, dir, "README.md", "seed\n", "initial")
	writeAndCommit(t, dir, "feature.go", "package main\n", "feat: work")
	return dir
}

// writeAndCommit writes content to relPath under dir and commits it.
func writeAndCommit(t *testing.T, dir, relPath, content, msg string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, relPath), []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", relPath, err)
	}
	runGit(t, dir, "add", relPath)
	runGit(t, dir, "commit", "-m", msg)
}

// runLogCmd runs `timbers log` with the given args in dir and returns output + error.
func runLogCmd(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	var out strings.Builder
	var execErr error
	runInDir(t, dir, func() {
		cmd := newRootCmd()
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		cmd.SetArgs(append([]string{"log"}, args...))
		execErr = cmd.Execute()
	})
	return out.String(), execErr
}

// onlyEntryInDir reads the single ledger entry under dir, failing if the count
// is not exactly one.
func onlyEntryInDir(t *testing.T, dir string) *ledger.Entry {
	t.Helper()
	var entries []*ledger.Entry
	walkJSONFiles(dir, func(_ string, data []byte) {
		entry, err := ledger.FromJSON(data)
		if err != nil {
			t.Fatalf("parse entry: %v", err)
		}
		entries = append(entries, entry)
	})
	if len(entries) != 1 {
		t.Fatalf("expected exactly one entry, found %d", len(entries))
	}
	return entries[0]
}

// TestLogResolvesSymbolicAnchor is the regression for the _HEAD-anchor defect:
// `timbers log --anchor HEAD` stored the literal string "HEAD" as the anchor and
// produced an id suffixed "_HEAD". A symbolic anchor changes meaning per-commit
// and per-worktree and defeats the since-anchor model. The anchor must be
// resolved to a full SHA at write time.
func TestLogResolvesSymbolicAnchor(t *testing.T) {
	dir := newLogAnchorRepo(t)
	headSHA := strings.TrimSpace(runGitOutput(t, dir, "rev-parse", "HEAD"))

	out, err := runLogCmd(t, dir, "documented via HEAD",
		"--why", "testing anchor resolution", "--how", "passed --anchor HEAD",
		"--anchor", "HEAD")
	if err != nil {
		t.Fatalf("timbers log --anchor errored: %v\noutput: %s", err, out)
	}

	entry := onlyEntryInDir(t, filepath.Join(dir, ".timbers"))
	if entry.Workset.AnchorCommit == "HEAD" {
		t.Fatalf("anchor stored as literal symbolic ref %q; want a resolved SHA", entry.Workset.AnchorCommit)
	}
	if entry.Workset.AnchorCommit != headSHA {
		t.Errorf("anchor = %q, want resolved HEAD %q", entry.Workset.AnchorCommit, headSHA)
	}
	if strings.HasSuffix(entry.ID, "_HEAD") {
		t.Errorf("id %q suffixed with symbolic ref; want a short-sha suffix", entry.ID)
	}
}

// TestLogRejectsUnknownAnchor verifies a non-existent ref passed to --anchor
// fails cleanly (user error) instead of writing a phantom entry anchored on an
// unresolvable ref.
func TestLogRejectsUnknownAnchor(t *testing.T) {
	dir := newLogAnchorRepo(t)

	out, err := runLogCmd(t, dir, "documented via bad ref",
		"--why", "y", "--how", "z", "--anchor", "no-such-ref-xyz")
	if err == nil {
		t.Fatalf("expected error for unknown --anchor ref; got nil\noutput: %s", out)
	}
	if countJSONFilesInDir(filepath.Join(dir, ".timbers")) != 0 {
		t.Errorf("no entry should be written when --anchor is unresolvable")
	}
}
