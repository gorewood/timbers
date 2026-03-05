package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/output"
)

// HookEnvTier classifies the hook environment by conflict level.
type HookEnvTier int

const (
	// HookEnvUncontested means no core.hooksPath override and no existing hook.
	HookEnvUncontested HookEnvTier = iota
	// HookEnvExistingHook means a hook exists at the standard .git/hooks path.
	HookEnvExistingHook
	// HookEnvKnownOverride means core.hooksPath is set by a recognized tool.
	HookEnvKnownOverride
	// HookEnvUnknownOverride means core.hooksPath is set to an unrecognized path.
	HookEnvUnknownOverride
)

// HookEnvInfo describes the classified hook environment.
type HookEnvInfo struct {
	Tier       HookEnvTier
	HooksDir   string // Resolved absolute path to hooks directory
	Owner      string // "beads", "husky", "" if unknown or N/A
	HasHook    bool   // Pre-commit hook file exists
	HasTimbers bool   // Timbers integration present (old or new format)
}

// knownOwner maps a path pattern to a tool name.
type knownOwner struct {
	Pattern string
	Owner   string
}

// knownOwners is the registry of recognized core.hooksPath patterns.
// Order matters: more specific patterns should come first.
var knownOwners = []knownOwner{
	{".beads/hooks", "beads"},
	{".husky/_", "husky"},
	{".husky", "husky"},
}

// matchKnownOwner returns the owner name if coreHooksPath matches a known pattern.
func matchKnownOwner(coreHooksPath string) string {
	for _, ko := range knownOwners {
		if strings.Contains(coreHooksPath, ko.Pattern) {
			return ko.Owner
		}
	}
	return ""
}

// ClassifyHookEnv determines the hook environment tier by inspecting git
// configuration and the filesystem. This is the convenience wrapper that
// performs git and filesystem calls.
func ClassifyHookEnv() (HookEnvInfo, error) {
	hooksDir, err := GetHooksDir()
	if err != nil {
		return HookEnvInfo{}, fmt.Errorf("getting hooks dir: %w", err)
	}

	coreHooksPath, _ := git.Run("config", "core.hooksPath")
	coreHooksPath = strings.TrimSpace(coreHooksPath)

	preCommitPath := filepath.Join(hooksDir, "pre-commit")
	hookExists := false
	hookContent := ""

	if data, readErr := os.ReadFile(preCommitPath); readErr == nil {
		hookExists = true
		hookContent = string(data)
	}

	info := classifyHookEnvFrom(coreHooksPath, hooksDir, hookExists, hookContent)
	return info, nil
}

// classifyHookEnvFrom is a pure classification function that takes resolved
// values instead of performing git or filesystem calls. This is the function
// tests call directly.
func classifyHookEnvFrom(coreHooksPath, hooksDir string, hookExists bool, hookContent string) HookEnvInfo {
	info := HookEnvInfo{
		HooksDir: hooksDir,
		HasHook:  hookExists,
	}

	if hookExists {
		info.HasTimbers = hasSectionDelimiters(hookContent) || hasOldFormatTimbers(hookContent)
	}

	// Tier 3 or 4: core.hooksPath is set
	if coreHooksPath != "" {
		owner := matchKnownOwner(coreHooksPath)
		if owner != "" {
			info.Tier = HookEnvKnownOverride
			info.Owner = owner
			return info
		}
		info.Tier = HookEnvUnknownOverride
		return info
	}

	// Tier 1 or 2: standard hooks path
	if hookExists {
		info.Tier = HookEnvExistingHook
		return info
	}

	info.Tier = HookEnvUncontested
	return info
}

// IsAppendable checks if a file is a regular text file that can be safely
// appended to. Returns (true, "") if appendable, or (false, reason) where
// reason is "symlink", "binary", or "not found".
func IsAppendable(hookPath string) (bool, string) {
	fi, err := os.Lstat(hookPath)
	if err != nil {
		return false, "not found"
	}

	if fi.Mode()&os.ModeSymlink != 0 {
		return false, "symlink"
	}

	// Read first 512 bytes to check for binary content (null bytes).
	hookFile, err := os.Open(hookPath)
	if err != nil {
		return false, "not found"
	}
	defer hookFile.Close() //nolint:errcheck // best-effort close on read-only file

	buf := make([]byte, 512)
	bytesRead, _ := hookFile.Read(buf)
	for i := range bytesRead {
		if buf[i] == 0 {
			return false, "binary"
		}
	}

	return true, ""
}

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
