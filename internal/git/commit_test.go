// Package git provides Git operations via exec for the timbers CLI.
package git

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/gorewood/timbers/internal/output"
)

func TestCommitStruct(t *testing.T) {
	// Verify Commit struct has expected fields
	commit := Commit{
		SHA:         "abc123def456abc123def456abc123def456abc1",
		Short:       "abc123d",
		Subject:     "Fix authentication bug",
		Body:        "Detailed description here",
		Author:      "Test Author",
		AuthorEmail: "test@example.com",
		Date:        time.Now(),
	}

	if commit.SHA == "" {
		t.Error("Commit.SHA should not be empty")
	}
	if commit.Short == "" {
		t.Error("Commit.Short should not be empty")
	}
	if commit.Subject == "" {
		t.Error("Commit.Subject should not be empty")
	}
	if commit.Body == "" {
		t.Error("Commit.Body should not be empty")
	}
	if commit.Author == "" {
		t.Error("Commit.Author should not be empty")
	}
	if commit.AuthorEmail == "" {
		t.Error("Commit.AuthorEmail should not be empty")
	}
	if commit.Date.IsZero() {
		t.Error("Commit.Date should not be zero")
	}
}

func TestDiffstatStruct(t *testing.T) {
	diffstat := Diffstat{
		Files:      3,
		Insertions: 45,
		Deletions:  12,
	}

	if diffstat.Files != 3 {
		t.Errorf("Diffstat.Files = %d, want 3", diffstat.Files)
	}
	if diffstat.Insertions != 45 {
		t.Errorf("Diffstat.Insertions = %d, want 45", diffstat.Insertions)
	}
	if diffstat.Deletions != 12 {
		t.Errorf("Diffstat.Deletions = %d, want 12", diffstat.Deletions)
	}
}

func TestLog(t *testing.T) {
	t.Run("in git repo with commits", func(t *testing.T) {
		origDir, getWdErr := os.Getwd()
		if getWdErr != nil {
			t.Fatalf("failed to get current dir: %v", getWdErr)
		}
		defer func() { _ = os.Chdir(origDir) }()

		if chdirErr := os.Chdir("/Users/bob/Projects/agent/timbers"); chdirErr != nil {
			t.Skipf("cannot change to test repo: %v", chdirErr)
		}

		// Get last 3 commits using HEAD~2..HEAD
		commits, logErr := Log("HEAD~2", "HEAD")
		if logErr != nil {
			t.Errorf("Log() error = %v, expected nil", logErr)
			return
		}
		// Should get at least some commits (exact count depends on repo state)
		if len(commits) == 0 {
			t.Error("Log() returned 0 commits, expected at least one")
		}
		// Verify commit fields are populated
		for idx, commit := range commits {
			if commit.SHA == "" {
				t.Errorf("commits[%d].SHA is empty", idx)
			}
			if commit.Short == "" {
				t.Errorf("commits[%d].Short is empty", idx)
			}
			if commit.Subject == "" {
				t.Errorf("commits[%d].Subject is empty", idx)
			}
			if commit.Author == "" {
				t.Errorf("commits[%d].Author is empty", idx)
			}
			if commit.Date.IsZero() {
				t.Errorf("commits[%d].Date is zero", idx)
			}
		}
	})

	t.Run("not in git repo", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, getWdErr := os.Getwd()
		if getWdErr != nil {
			t.Fatalf("failed to get current dir: %v", getWdErr)
		}
		defer func() { _ = os.Chdir(origDir) }()

		if chdirErr := os.Chdir(tmpDir); chdirErr != nil {
			t.Fatalf("failed to change to temp dir: %v", chdirErr)
		}

		_, logErr := Log("HEAD~1", "HEAD")
		if logErr == nil {
			t.Error("Log() expected error outside git repo")
			return
		}

		var exitErr *output.ExitError
		if !errors.As(logErr, &exitErr) {
			t.Errorf("Log() error should be *output.ExitError, got %T", logErr)
		}
	})

	t.Run("invalid range", func(t *testing.T) {
		origDir, getWdErr := os.Getwd()
		if getWdErr != nil {
			t.Fatalf("failed to get current dir: %v", getWdErr)
		}
		defer func() { _ = os.Chdir(origDir) }()

		if chdirErr := os.Chdir("/Users/bob/Projects/agent/timbers"); chdirErr != nil {
			t.Skipf("cannot change to test repo: %v", chdirErr)
		}

		_, logErr := Log("nonexistent-ref", "HEAD")
		if logErr == nil {
			t.Error("Log() expected error for invalid ref")
		}
	})
}

func TestCommitsReachableFrom(t *testing.T) {
	t.Run("in git repo", func(t *testing.T) {
		origDir, getWdErr := os.Getwd()
		if getWdErr != nil {
			t.Fatalf("failed to get current dir: %v", getWdErr)
		}
		defer func() { _ = os.Chdir(origDir) }()

		if chdirErr := os.Chdir("/Users/bob/Projects/agent/timbers"); chdirErr != nil {
			t.Skipf("cannot change to test repo: %v", chdirErr)
		}

		// Get all commits from HEAD
		commits, reachErr := CommitsReachableFrom("HEAD")
		if reachErr != nil {
			t.Errorf("CommitsReachableFrom() error = %v, expected nil", reachErr)
			return
		}
		if len(commits) == 0 {
			t.Error("CommitsReachableFrom() returned 0 commits")
		}
	})

	t.Run("not in git repo", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, getWdErr := os.Getwd()
		if getWdErr != nil {
			t.Fatalf("failed to get current dir: %v", getWdErr)
		}
		defer func() { _ = os.Chdir(origDir) }()

		if chdirErr := os.Chdir(tmpDir); chdirErr != nil {
			t.Fatalf("failed to change to temp dir: %v", chdirErr)
		}

		_, reachErr := CommitsReachableFrom("HEAD")
		if reachErr == nil {
			t.Error("CommitsReachableFrom() expected error outside git repo")
		}
	})
}

func TestDiffstat(t *testing.T) {
	t.Run("in git repo", func(t *testing.T) {
		origDir, getWdErr := os.Getwd()
		if getWdErr != nil {
			t.Fatalf("failed to get current dir: %v", getWdErr)
		}
		defer func() { _ = os.Chdir(origDir) }()

		if chdirErr := os.Chdir("/Users/bob/Projects/agent/timbers"); chdirErr != nil {
			t.Skipf("cannot change to test repo: %v", chdirErr)
		}

		// Get diffstat for a small range
		stat, diffErr := GetDiffstat("HEAD~1", "HEAD")
		if diffErr != nil {
			t.Errorf("GetDiffstat() error = %v, expected nil", diffErr)
			return
		}
		// Just verify we got valid data (values could be 0 if no changes)
		if stat.Files < 0 || stat.Insertions < 0 || stat.Deletions < 0 {
			t.Error("GetDiffstat() returned negative values")
		}
	})

	t.Run("not in git repo", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, getWdErr := os.Getwd()
		if getWdErr != nil {
			t.Fatalf("failed to get current dir: %v", getWdErr)
		}
		defer func() { _ = os.Chdir(origDir) }()

		if chdirErr := os.Chdir(tmpDir); chdirErr != nil {
			t.Fatalf("failed to change to temp dir: %v", chdirErr)
		}

		_, diffErr := GetDiffstat("HEAD~1", "HEAD")
		if diffErr == nil {
			t.Error("GetDiffstat() expected error outside git repo")
		}
	})
}

func TestResolveRefOrEmptyTree(t *testing.T) {
	origDir, getWdErr := os.Getwd()
	if getWdErr != nil {
		t.Fatalf("failed to get current dir: %v", getWdErr)
	}
	defer func() { _ = os.Chdir(origDir) }()

	if chdirErr := os.Chdir("/Users/bob/Projects/agent/timbers"); chdirErr != nil {
		t.Skipf("cannot change to test repo: %v", chdirErr)
	}

	tests := []struct {
		name     string
		ref      string
		wantTree bool // true if we expect empty tree SHA
	}{
		{
			name:     "empty ref returns empty tree",
			ref:      "",
			wantTree: true,
		},
		{
			name:     "valid ref returns ref",
			ref:      "HEAD",
			wantTree: false,
		},
		{
			name:     "nonexistent ref returns empty tree",
			ref:      "nonexistent-ref-abc123",
			wantTree: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveRefOrEmptyTree(tt.ref)
			if tt.wantTree && got != emptyTreeSHA {
				t.Errorf("resolveRefOrEmptyTree(%q) = %q, want empty tree SHA", tt.ref, got)
			}
			if !tt.wantTree && got == emptyTreeSHA {
				t.Errorf("resolveRefOrEmptyTree(%q) = empty tree, want resolved ref", tt.ref)
			}
		})
	}
}

func TestGetDiffstatRootCommit(t *testing.T) {
	origDir, getWdErr := os.Getwd()
	if getWdErr != nil {
		t.Fatalf("failed to get current dir: %v", getWdErr)
	}
	defer func() { _ = os.Chdir(origDir) }()

	if chdirErr := os.Chdir("/Users/bob/Projects/agent/timbers"); chdirErr != nil {
		t.Skipf("cannot change to test repo: %v", chdirErr)
	}

	// Find the root commit
	rootSHA, err := Run("rev-list", "--max-parents=0", "HEAD")
	if err != nil {
		t.Fatalf("failed to find root commit: %v", err)
	}

	// Try to get diffstat using root^ (which doesn't exist)
	// This should fall back to empty tree and succeed
	stat, diffErr := GetDiffstat(rootSHA+"^", rootSHA)
	if diffErr != nil {
		t.Errorf("GetDiffstat(root^, root) error = %v, expected nil", diffErr)
		return
	}

	// Root commit should have some files added
	if stat.Files == 0 && stat.Insertions == 0 {
		t.Log("Warning: root commit diffstat shows 0 files/insertions - may be an edge case")
	}
}
