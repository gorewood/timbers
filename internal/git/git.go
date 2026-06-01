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

// IsAncestorOf returns true if ancestor is an ancestor of (or equal to) descendant.
// Returns false if either SHA doesn't exist or ancestor is not in descendant's history.
// Useful for detecting anchors that exist in the object store but were rewritten
// by rebase or squash merge and are no longer reachable from HEAD.
func IsAncestorOf(ancestor, descendant string) bool {
	if ancestor == "" || descendant == "" {
		return false
	}
	_, err := Run("merge-base", "--is-ancestor", ancestor, descendant)
	return err == nil
}

// IsOnFirstParentLine reports whether sha is reachable from head via
// first-parent traversal. The current branch's first-parent line is the
// "spine" — commits made directly on this branch, ignoring side branches
// brought in via merge. A side-branch anchor (e.g., a commit from a
// merged-in PR's branch) is reachable from head via the DAG but NOT via
// first-parent — that distinction is what this helper exposes.
//
// Bounded at 5000 commits so a pathologically deep history doesn't
// stall pending detection. For shallower histories the walk terminates
// at the root. Returns false on any git error or when either SHA is
// empty, so callers degrade gracefully (no diagnostic, no false
// positive).
func IsOnFirstParentLine(sha, head string) bool {
	if sha == "" || head == "" {
		return false
	}
	out, err := Run("rev-list", "--first-parent", "--max-count=5000", head)
	if err != nil {
		return false
	}
	for line := range strings.SplitSeq(out, "\n") {
		if strings.TrimSpace(line) == sha {
			return true
		}
	}
	return false
}

// IsPushedToUpstream returns true if the given SHA is reachable from the
// current branch's upstream (origin/<branch> via @{u}). Returns false when
// there is no upstream configured, when HEAD is detached, or when any git
// call fails — the caller treats those as "don't warn" so missing config
// never produces a spurious warning.
//
// Used by `timbers log` to detect the push-before-log race: if the commit
// the user just documented is already pushed but the entry's auto-commit
// isn't, the entry is stranded locally and the user needs to push again.
func IsPushedToUpstream(sha string) bool {
	if sha == "" {
		return false
	}
	upstream, err := Run("rev-parse", "--symbolic-full-name", "@{u}")
	if err != nil || upstream == "" {
		return false
	}
	return IsAncestorOf(sha, upstream)
}

// HasUncommittedChanges returns true if the working tree has staged or unstaged changes.
func HasUncommittedChanges() bool {
	out, err := Run("status", "--porcelain")
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) != ""
}

// HasStagedChanges reports whether the index differs from HEAD. The pre-commit
// hook uses this to tell the user "your staged changes are still there" when
// the gate aborted their commit — leaving the index untouched is what makes
// the timbers log → phantom-entry path so easy to stumble into.
//
// Returns false on any git failure: this is a diagnostic helper for hook
// messaging, not a correctness check, and the hook must not leak errors.
func HasStagedChanges() bool {
	out, err := Run("diff", "--cached", "--name-only")
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
