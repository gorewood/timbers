package git

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestCommit_MailmapResolvesAuthorEmail confirms that the parse format uses
// %aE (mailmap-resolved author email), not %ae (raw). This is the load-
// bearing fix for repos where one operator commits under multiple emails:
// without mailmap resolution, the proposed strict-email skip heuristic in
// the cross-agent debt classifier silently misclassifies the operator's
// own work as foreign-author. .mailmap is git's canonical mechanism for
// this case and timbers must honor it.
//
// Scenario: a temp repo with a .mailmap mapping <alt@example.com> to
// <canonical@example.com>. Configure git to author commits as alt@; run
// Log; assert AuthorEmail is canonical@.
func TestCommit_MailmapResolvesAuthorEmail(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	run := func(args ...string) {
		t.Helper()
		out, err := Run(args...)
		if err != nil {
			t.Fatalf("git %v failed: %v (output: %s)", args, err, out)
		}
	}
	run("init")
	run("config", "user.email", "alt@example.com")
	run("config", "user.name", "Author Name")

	// Write a .mailmap that coalesces alt → canonical.
	mailmap := "Author Name <canonical@example.com> <alt@example.com>\n"
	if err := os.WriteFile(filepath.Join(dir, ".mailmap"), []byte(mailmap), 0o600); err != nil {
		t.Fatalf("write .mailmap: %v", err)
	}
	run("add", ".mailmap")
	run("commit", "-m", "add mailmap")

	if err := os.WriteFile(filepath.Join(dir, "work.txt"), []byte("work\n"), 0o600); err != nil {
		t.Fatalf("write work.txt: %v", err)
	}
	run("add", "work.txt")
	run("commit", "-m", "feat: work commit by alt identity")

	commits, err := Log("HEAD~1", "HEAD")
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if len(commits) != 1 {
		t.Fatalf("expected 1 commit, got %d", len(commits))
	}
	if got, want := commits[0].AuthorEmail, "canonical@example.com"; got != want {
		t.Errorf("AuthorEmail = %q, want %q (mailmap-resolved); raw alt@example.com indicates timbers is using %%ae instead of %%aE", got, want)
	}
}

// TestCommit_AuthorDateAndCommitDateDiverge is the regression test that
// locks in the rebase/amend correctness fix. AuthorDate is preserved
// across rebase and `git commit --amend`; CommitDate advances. The cross-
// agent debt classifier's staleness check uses CommitDate so that work
// the user just touched in-session (rebased or amended from an older
// authorship) does NOT silently auto-skip. If a future refactor reverts
// CommitDate to read from %at, this test catches it.
func TestCommit_AuthorDateAndCommitDateDiverge(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	run := func(args ...string) {
		t.Helper()
		out, err := Run(args...)
		if err != nil {
			t.Fatalf("git %v failed: %v (output: %s)", args, err, out)
		}
	}
	run("init")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")

	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("a\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	run("add", "f.txt")
	// Author commit at a fixed point in the past so we can verify divergence.
	pastAuthorDate := "2020-01-15T10:00:00Z"
	t.Setenv("GIT_AUTHOR_DATE", pastAuthorDate)
	run("commit", "-m", "original commit with old author date")

	// Now amend with a fresh CommitDate but preserved AuthorDate. Setting
	// only GIT_COMMITTER_DATE (clearing GIT_AUTHOR_DATE) tells git to leave
	// the author date alone and stamp a new committer date on the amend.
	t.Setenv("GIT_AUTHOR_DATE", "")
	nowCommitDate := time.Now().UTC().Format(time.RFC3339)
	t.Setenv("GIT_COMMITTER_DATE", nowCommitDate)
	run("commit", "--amend", "--no-edit")

	commits, err := CommitsReachableFrom("HEAD")
	if err != nil {
		t.Fatalf("CommitsReachableFrom: %v", err)
	}
	if len(commits) != 1 {
		t.Fatalf("expected 1 commit, got %d", len(commits))
	}
	commit := commits[0]
	if commit.Date.IsZero() || commit.CommitDate.IsZero() {
		t.Fatalf("expected both Date and CommitDate populated; got Date=%v CommitDate=%v",
			commit.Date, commit.CommitDate)
	}
	// Date (AuthorDate) should be in 2020. CommitDate should be recent.
	// A simple year check is enough — the test cares about the divergence,
	// not exact wall-clock precision.
	if commit.Date.Year() != 2020 {
		t.Errorf("Date (AuthorDate) = %v, expected year 2020 (the original author date)", commit.Date)
	}
	if commit.CommitDate.Year() < 2024 {
		t.Errorf("CommitDate = %v, expected a recent year (advanced by --amend)", commit.CommitDate)
	}
	if !commit.CommitDate.After(commit.Date) {
		t.Errorf("CommitDate (%v) should be after Date (%v) on amended commit",
			commit.CommitDate, commit.Date)
	}
}
