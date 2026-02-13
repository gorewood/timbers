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
