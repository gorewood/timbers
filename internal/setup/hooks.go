package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/output"
)

// HookStatus represents the status of a single git hook.
type HookStatus struct {
	Installed bool
	Chained   bool
}

// GetHooksDir returns the active git hooks directory.
// Respects core.hooksPath if configured; defaults to .git/hooks.
func GetHooksDir() (string, error) {
	root, err := git.RepoRoot()
	if err != nil {
		return "", err
	}

	// Check core.hooksPath (set by beads, husky, etc.)
	hooksPath, configErr := git.Run("config", "core.hooksPath")
	if configErr == nil && hooksPath != "" {
		if filepath.IsAbs(hooksPath) {
			return hooksPath, nil
		}
		return filepath.Join(root, hooksPath), nil
	}

	return filepath.Join(root, ".git", "hooks"), nil
}

// HookExists checks if a hook file exists at the given path.
func HookExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// CheckHookStatus checks if a hook is installed and whether it chains to a backup.
func CheckHookStatus(hookPath string) HookStatus {
	status := HookStatus{}

	content, err := os.ReadFile(hookPath)
	if err != nil {
		return status // Not installed
	}

	contentStr := string(content)
	if strings.Contains(contentStr, "timbers hook run") {
		status.Installed = true
		status.Chained = strings.Contains(contentStr, ".backup")
	}

	return status
}

// GeneratePreCommitHook generates the pre-commit hook script content.
// If withChain is true, the hook chains to the backed-up original hook.
// The hooksDir parameter sets the backup path for chaining; pass "" for default.
func GeneratePreCommitHook(withChain bool, hooksDir string) string {
	script := `#!/bin/sh
# timbers pre-commit hook
# Blocks commits when undocumented commits exist (use --no-verify to bypass)

if command -v timbers >/dev/null 2>&1; then
  timbers hook run pre-commit "$@"
  rc=$?
  if [ $rc -ne 0 ]; then exit $rc; fi
fi
`

	if withChain {
		backupPath := ".git/hooks/pre-commit.backup"
		if hooksDir != "" {
			backupPath = filepath.Join(hooksDir, "pre-commit.backup")
		}
		script += fmt.Sprintf(`
# Chain to original hook if it exists
if [ -x %q ]; then
  exec %q "$@"
fi
`, backupPath, backupPath)
	}

	return script
}

// GeneratePostCommitHook generates the post-commit hook script content.
// The hook reminds users/agents to document their work after each commit.
func GeneratePostCommitHook() string {
	return `#!/bin/sh
# timbers post-commit hook
# Reminds you to document commits (non-blocking)

if command -v timbers >/dev/null 2>&1; then
  timbers hook run post-commit "$@"
fi
`
}

// CheckPostCommitHookStatus checks if a post-commit hook contains timbers integration.
func CheckPostCommitHookStatus(hookPath string) HookStatus {
	status := HookStatus{}

	content, err := os.ReadFile(hookPath)
	if err != nil {
		return status
	}

	contentStr := string(content)
	if strings.Contains(contentStr, "timbers hook run post-commit") {
		status.Installed = true
	}

	return status
}

// InstallPostCommitHook installs the post-commit hook at the given path.
// If a hook already exists, appends the timbers section. Returns nil on success.
func InstallPostCommitHook(hookPath string) error {
	if HookExists(hookPath) {
		existing, err := os.ReadFile(hookPath)
		if err != nil {
			return fmt.Errorf("reading post-commit hook: %w", err)
		}
		content := string(existing)
		if strings.Contains(content, "timbers hook run post-commit") {
			return nil // already installed
		}
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		content += postCommitSection()
		// #nosec G306 -- hook needs execute permission
		if err := os.WriteFile(hookPath, []byte(content), 0o755); err != nil {
			return fmt.Errorf("writing post-commit hook: %w", err)
		}
		return nil
	}
	// #nosec G306 -- hook needs execute permission
	if err := os.WriteFile(hookPath, []byte(GeneratePostCommitHook()), 0o755); err != nil {
		return fmt.Errorf("writing post-commit hook: %w", err)
	}
	return nil
}

// postCommitSection returns the timbers section to append to an existing hook.
func postCommitSection() string {
	return `
# timbers post-commit hook
if command -v timbers >/dev/null 2>&1; then
  timbers hook run post-commit "$@"
fi
`
}

// BackupExistingHook moves an existing hook to a .backup location.
func BackupExistingHook(hookPath string) error {
	backupPath := hookPath + ".backup"
	if err := os.Rename(hookPath, backupPath); err != nil {
		return output.NewSystemErrorWithCause("failed to backup existing hook", err)
	}
	return nil
}

// DescribeInstallAction returns a human-readable description of what the
// install operation would do given the current state.
func DescribeInstallAction(existingHook, chain, force bool) string {
	if !existingHook {
		return "would install"
	}
	switch {
	case force:
		return "would overwrite existing hook"
	case chain:
		return "would backup and chain existing hook"
	default:
		return "would fail (hook exists, use --chain or --force)"
	}
}

// DescribeUninstallAction returns a human-readable description of what the
// uninstall operation would do given the current state.
func DescribeUninstallAction(installed, hasBackup bool) string {
	switch {
	case !installed:
		return "no timbers hook installed"
	case hasBackup:
		return "would remove and restore backup"
	default:
		return "would remove"
	}
}
