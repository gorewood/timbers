package setup

import (
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

// GetHooksDir returns the path to the .git/hooks directory.
func GetHooksDir() (string, error) {
	root, err := git.RepoRoot()
	if err != nil {
		return "", err
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
func GeneratePreCommitHook(withChain bool) string {
	script := `#!/bin/sh
# timbers pre-commit hook
# Warns about undocumented commits (non-blocking)

if command -v timbers >/dev/null 2>&1; then
  timbers hook run pre-commit "$@"
fi
`

	if withChain {
		script += `
# Chain to original hook if it exists
if [ -x ".git/hooks/pre-commit.backup" ]; then
  exec .git/hooks/pre-commit.backup "$@"
fi
`
	}

	return script
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
