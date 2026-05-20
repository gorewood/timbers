// Package git provides Git operations via exec for the timbers CLI.
package git

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorewood/timbers/internal/output"
)

// chdirToRepoRoot changes to the git repo root and returns a cleanup function.
// Skips the test if not running inside a git repository.
func chdirToRepoRoot(t *testing.T) {
	t.Helper()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	out, err := exec.CommandContext(context.Background(), "git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		t.Skip("not running inside a git repository")
	}
	root := strings.TrimSpace(string(out))
	if err := os.Chdir(root); err != nil {
		t.Skipf("cannot change to repo root: %v", err)
	}
}

func TestRun(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		wantErr       bool
		wantErrMsg    string
		checkExitCode int
	}{
		{
			name:    "git version succeeds",
			args:    []string{"version"},
			wantErr: false,
		},
		{
			name:          "invalid git command",
			args:          []string{"invalid-command-that-does-not-exist"},
			wantErr:       true,
			wantErrMsg:    "git command failed",
			checkExitCode: output.ExitSystemError,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			out, runErr := Run(testCase.args...)
			if testCase.wantErr {
				if runErr == nil {
					t.Errorf("Run() expected error, got nil")
					return
				}
				var exitErr *output.ExitError
				if !errors.As(runErr, &exitErr) {
					t.Errorf("Run() error should be *output.ExitError, got %T", runErr)
					return
				}
				if testCase.checkExitCode != 0 && exitErr.Code != testCase.checkExitCode {
					t.Errorf("Run() exit code = %d, want %d", exitErr.Code, testCase.checkExitCode)
				}
			} else {
				if runErr != nil {
					t.Errorf("Run() unexpected error: %v", runErr)
					return
				}
				if out == "" {
					t.Error("Run() expected non-empty output for 'git version'")
				}
			}
		})
	}
}

func TestIsRepo(t *testing.T) {
	// Test in the current directory (which should be a git repo based on context)
	t.Run("in git repo", func(t *testing.T) {
		chdirToRepoRoot(t)

		if !IsRepo() {
			t.Error("IsRepo() = false, expected true in git repo")
		}
	})

	t.Run("not in git repo", func(t *testing.T) {
		// Create temp dir that is not a git repo
		tmpDir := t.TempDir()
		origDir, getWdErr := os.Getwd()
		if getWdErr != nil {
			t.Fatalf("failed to get current dir: %v", getWdErr)
		}
		defer func() { _ = os.Chdir(origDir) }()

		if chdirErr := os.Chdir(tmpDir); chdirErr != nil {
			t.Fatalf("failed to change to temp dir: %v", chdirErr)
		}

		if IsRepo() {
			t.Error("IsRepo() = true, expected false outside git repo")
		}
	})
}

func TestRepoRoot(t *testing.T) {
	t.Run("in git repo", func(t *testing.T) {
		chdirToRepoRoot(t)

		root, rootErr := RepoRoot()
		if rootErr != nil {
			t.Errorf("RepoRoot() error = %v, expected nil", rootErr)
			return
		}
		if root == "" {
			t.Error("RepoRoot() returned empty string")
		}
		if !filepath.IsAbs(root) {
			t.Errorf("RepoRoot() = %q, expected absolute path", root)
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

		_, rootErr := RepoRoot()
		if rootErr == nil {
			t.Error("RepoRoot() expected error outside git repo")
			return
		}

		var exitErr *output.ExitError
		if !errors.As(rootErr, &exitErr) {
			t.Errorf("RepoRoot() error should be *output.ExitError, got %T", rootErr)
			return
		}
		if exitErr.Code != output.ExitSystemError {
			t.Errorf("RepoRoot() exit code = %d, want %d", exitErr.Code, output.ExitSystemError)
		}
	})
}

func TestCurrentBranch(t *testing.T) {
	t.Run("in git repo", func(t *testing.T) {
		chdirToRepoRoot(t)

		branch, branchErr := CurrentBranch()
		if branchErr != nil {
			t.Errorf("CurrentBranch() error = %v, expected nil", branchErr)
			return
		}
		if branch == "" {
			t.Error("CurrentBranch() returned empty string")
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

		_, branchErr := CurrentBranch()
		if branchErr == nil {
			t.Error("CurrentBranch() expected error outside git repo")
		}
	})
}

func TestHEAD(t *testing.T) {
	t.Run("in git repo", func(t *testing.T) {
		chdirToRepoRoot(t)

		sha, headErr := HEAD()
		if headErr != nil {
			t.Errorf("HEAD() error = %v, expected nil", headErr)
			return
		}
		if len(sha) != 40 {
			t.Errorf("HEAD() returned SHA of length %d, expected 40", len(sha))
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

		_, headErr := HEAD()
		if headErr == nil {
			t.Error("HEAD() expected error outside git repo")
		}
	})
}

func TestSHAExists(t *testing.T) {
	tests := []struct {
		name string
		sha  string
		want bool
	}{
		{
			name: "empty SHA returns false",
			sha:  "",
			want: false,
		},
		{
			name: "nonexistent SHA returns false",
			sha:  "0000000000000000000000000000000000000000",
			want: false,
		},
		{
			name: "garbage SHA returns false",
			sha:  "not-a-real-sha",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SHAExists(tt.sha); got != tt.want {
				t.Errorf("SHAExists(%q) = %v, want %v", tt.sha, got, tt.want)
			}
		})
	}

	t.Run("HEAD SHA exists", func(t *testing.T) {
		chdirToRepoRoot(t)

		headSHA, err := HEAD()
		if err != nil {
			t.Fatalf("HEAD() error: %v", err)
		}

		if !SHAExists(headSHA) {
			t.Errorf("SHAExists(HEAD) = false, expected true for %s", headSHA)
		}
	})
}

// TestIsPushedToUpstream verifies the push-before-log race detector. The
// helper has to fail safely when there's no upstream, when HEAD is detached,
// or when git is unhappy for any reason — the gate's job is to warn, not
// to surprise users with false positives.
func TestIsPushedToUpstream(t *testing.T) {
	t.Run("returns false for empty SHA", func(t *testing.T) {
		if IsPushedToUpstream("") {
			t.Error("expected false for empty SHA")
		}
	})

	t.Run("returns false when no upstream configured", func(t *testing.T) {
		dir := t.TempDir()
		origDir, _ := os.Getwd()
		defer func() { _ = os.Chdir(origDir) }()
		if err := os.Chdir(dir); err != nil {
			t.Fatalf("chdir: %v", err)
		}
		mustRun := func(args ...string) {
			t.Helper()
			if _, err := Run(args...); err != nil {
				t.Fatalf("git %v failed: %v", args, err)
			}
		}
		mustRun("init")
		mustRun("config", "user.email", "test@test.com")
		mustRun("config", "user.name", "Test")
		if err := os.WriteFile("a.txt", []byte("a"), 0o600); err != nil {
			t.Fatal(err)
		}
		mustRun("add", "a.txt")
		mustRun("commit", "-m", "first")

		sha, err := HEAD()
		if err != nil {
			t.Fatalf("HEAD: %v", err)
		}
		// No upstream configured — must not warn.
		if IsPushedToUpstream(sha) {
			t.Error("expected false when no upstream is configured")
		}
	})

	t.Run("returns true when SHA is reachable from upstream", func(t *testing.T) {
		// Two repos wired up: 'upstream' as a bare remote, 'local' tracking it.
		// Mimics the push-before-log race: agent pushed the content SHA to
		// origin, then ran timbers log locally.
		root := t.TempDir()
		upstream := filepath.Join(root, "upstream.git")
		local := filepath.Join(root, "local")
		origDir, _ := os.Getwd()
		defer func() { _ = os.Chdir(origDir) }()

		// Create bare upstream
		if err := os.Chdir(root); err != nil {
			t.Fatalf("chdir root: %v", err)
		}
		if _, err := Run("init", "--bare", upstream); err != nil {
			t.Fatalf("init upstream: %v", err)
		}

		// Create local clone
		if _, err := Run("clone", upstream, local); err != nil {
			t.Fatalf("clone: %v", err)
		}
		if err := os.Chdir(local); err != nil {
			t.Fatalf("chdir local: %v", err)
		}
		mustRun := func(args ...string) {
			t.Helper()
			if _, err := Run(args...); err != nil {
				t.Fatalf("git %v failed: %v", args, err)
			}
		}
		mustRun("config", "user.email", "test@test.com")
		mustRun("config", "user.name", "Test")
		if err := os.WriteFile("a.txt", []byte("a"), 0o600); err != nil {
			t.Fatal(err)
		}
		mustRun("add", "a.txt")
		mustRun("commit", "-m", "first")
		// Determine the default branch and push it so upstream tracking is set.
		branch, err := CurrentBranch()
		if err != nil {
			t.Fatalf("current branch: %v", err)
		}
		mustRun("push", "-u", "origin", branch)

		pushedSHA, err := HEAD()
		if err != nil {
			t.Fatalf("HEAD after push: %v", err)
		}

		// The pushed SHA is reachable from @{u} — this is the race condition.
		if !IsPushedToUpstream(pushedSHA) {
			t.Error("expected true: pushed SHA should be reachable from upstream")
		}

		// Make a follow-up commit locally without pushing — mimics the
		// auto-committed timbers entry. The unpushed SHA must NOT report
		// as pushed.
		if writeErr := os.WriteFile("b.txt", []byte("b"), 0o600); writeErr != nil {
			t.Fatal(writeErr)
		}
		mustRun("add", "b.txt")
		mustRun("commit", "-m", "second (unpushed)")
		localSHA, headErr := HEAD()
		if headErr != nil {
			t.Fatalf("HEAD after local commit: %v", headErr)
		}
		if IsPushedToUpstream(localSHA) {
			t.Error("expected false: unpushed local SHA should not be reachable from upstream")
		}
	})
}

func TestIsInteractiveGitOp(t *testing.T) {
	t.Run("normal repo is not mid-operation", func(t *testing.T) {
		chdirToRepoRoot(t)

		if IsInteractiveGitOp() {
			t.Error("IsInteractiveGitOp() = true in normal repo, expected false")
		}
	})

	t.Run("detects rebase-merge directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		setupGitRepoWithCommit(t, tmpDir)

		gitDir := filepath.Join(tmpDir, ".git", "rebase-merge")
		if err := os.MkdirAll(gitDir, 0o755); err != nil {
			t.Fatalf("failed to create rebase-merge dir: %v", err)
		}

		if !IsInteractiveGitOp() {
			t.Error("IsInteractiveGitOp() = false with rebase-merge dir, expected true")
		}
	})

	t.Run("detects rebase-apply directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		setupGitRepoWithCommit(t, tmpDir)

		gitDir := filepath.Join(tmpDir, ".git", "rebase-apply")
		if err := os.MkdirAll(gitDir, 0o755); err != nil {
			t.Fatalf("failed to create rebase-apply dir: %v", err)
		}

		if !IsInteractiveGitOp() {
			t.Error("IsInteractiveGitOp() = false with rebase-apply dir, expected true")
		}
	})

	t.Run("detects MERGE_HEAD file", func(t *testing.T) {
		tmpDir := t.TempDir()
		setupGitRepoWithCommit(t, tmpDir)

		mergeHead := filepath.Join(tmpDir, ".git", "MERGE_HEAD")
		if err := os.WriteFile(mergeHead, []byte("abc123\n"), 0o600); err != nil {
			t.Fatalf("failed to create MERGE_HEAD: %v", err)
		}

		if !IsInteractiveGitOp() {
			t.Error("IsInteractiveGitOp() = false with MERGE_HEAD, expected true")
		}
	})

	t.Run("detects CHERRY_PICK_HEAD file", func(t *testing.T) {
		tmpDir := t.TempDir()
		setupGitRepoWithCommit(t, tmpDir)

		cpHead := filepath.Join(tmpDir, ".git", "CHERRY_PICK_HEAD")
		if err := os.WriteFile(cpHead, []byte("abc123\n"), 0o600); err != nil {
			t.Fatalf("failed to create CHERRY_PICK_HEAD: %v", err)
		}

		if !IsInteractiveGitOp() {
			t.Error("IsInteractiveGitOp() = false with CHERRY_PICK_HEAD, expected true")
		}
	})

	t.Run("detects REVERT_HEAD file", func(t *testing.T) {
		tmpDir := t.TempDir()
		setupGitRepoWithCommit(t, tmpDir)

		revertHead := filepath.Join(tmpDir, ".git", "REVERT_HEAD")
		if err := os.WriteFile(revertHead, []byte("abc123\n"), 0o600); err != nil {
			t.Fatalf("failed to create REVERT_HEAD: %v", err)
		}

		if !IsInteractiveGitOp() {
			t.Error("IsInteractiveGitOp() = false with REVERT_HEAD, expected true")
		}
	})
}

// setupGitRepoWithCommit creates a temporary git repo with one commit and chdirs to it.
func setupGitRepoWithCommit(t *testing.T, dir string) {
	t.Helper()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "init"},
	}
	for _, args := range cmds {
		cmd := exec.CommandContext(context.Background(), args[0], args[1:]...) //nolint:gosec // test helper with fixed commands
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("cmd %v failed: %v\n%s", args, err, out)
		}
	}
}
