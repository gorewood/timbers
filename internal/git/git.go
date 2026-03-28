// Package git provides Git operations via exec for the timbers CLI.
package git

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gorewood/timbers/internal/output"
)

// Run executes a git command with the given arguments.
// It captures stdout and returns it as a trimmed string.
// Returns an *output.ExitError on failure with appropriate exit code.
func Run(args ...string) (string, error) {
	return RunContext(context.Background(), args...)
}

// RunContext executes a git command with the given context and arguments.
// It captures stdout and returns it as a trimmed string.
// Returns an *output.ExitError on failure with appropriate exit code.
func RunContext(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Check if git is not found
		var execErr *exec.Error
		if errors.As(err, &execErr) {
			return "", output.NewSystemError("git not found: ensure git is installed and in PATH")
		}

		// Git command failed - include stderr in message
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		return "", output.NewSystemErrorWithCause("git command failed: "+errMsg, err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

// IsRepo checks if the current directory is inside a git repository.
func IsRepo() bool {
	_, err := Run("rev-parse", "--git-dir")
	return err == nil
}

// RepoRoot returns the root directory of the current git repository.
// Returns an error if not in a git repository.
func RepoRoot() (string, error) {
	root, err := Run("rev-parse", "--show-toplevel")
	if err != nil {
		return "", output.NewSystemErrorWithCause("not in a git repository", err)
	}
	return root, nil
}

// CurrentBranch returns the name of the current branch.
// Returns an error if not in a git repository or HEAD is detached.
func CurrentBranch() (string, error) {
	branch, err := Run("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", output.NewSystemErrorWithCause("failed to get current branch", err)
	}
	return branch, nil
}

// HEAD returns the full SHA of the current HEAD commit.
// Returns an error if not in a git repository or no commits exist.
func HEAD() (string, error) {
	sha, err := Run("rev-parse", "HEAD")
	if err != nil {
		return "", output.NewSystemErrorWithCause("failed to get HEAD", err)
	}
	return sha, nil
}

// SHAExists checks if a SHA exists in the current repository.
// Returns true if the SHA resolves to a known git object, false otherwise.
// Useful for detecting stale references after squash merges or history rewrites.
func SHAExists(sha string) bool {
	if sha == "" {
		return false
	}
	_, err := Run("cat-file", "-t", sha)
	return err == nil
}

// HasUncommittedChanges returns true if the working tree has staged or unstaged changes.
func HasUncommittedChanges() bool {
	out, err := Run("status", "--porcelain")
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) != ""
}

// IsInteractiveGitOp returns true when git is in the middle of a rebase,
// merge, cherry-pick, or revert. Hooks should suppress blocking behavior
// during these operations because:
//   - Rebased commits are replayed, not new work — don't nag per-commit
//   - timbers log can't commit entries mid-rebase (working tree is locked)
//   - Pending counts are unreliable until the operation completes
func IsInteractiveGitOp() bool {
	gitDir, err := Run("rev-parse", "--git-dir")
	if err != nil {
		return false
	}

	// rev-parse --git-dir returns a relative path (".git") in normal repos
	// but absolute paths in worktrees. Resolve relative paths so file checks
	// work regardless of the process's current working directory.
	if !filepath.IsAbs(gitDir) {
		root, rootErr := RepoRoot()
		if rootErr != nil {
			return false
		}
		gitDir = filepath.Join(root, gitDir)
	}

	// git rebase (interactive or standard)
	for _, dir := range []string{"rebase-merge", "rebase-apply"} {
		if isDir(filepath.Join(gitDir, dir)) {
			return true
		}
	}

	// git merge, cherry-pick, revert
	for _, file := range []string{"MERGE_HEAD", "CHERRY_PICK_HEAD", "REVERT_HEAD"} {
		if fileExists(filepath.Join(gitDir, file)) {
			return true
		}
	}

	return false
}

// isDir reports whether path is an existing directory.
func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// fileExists reports whether path exists as a regular file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
