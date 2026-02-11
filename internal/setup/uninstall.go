package setup

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/output"
)

// UninstallInfo holds the state gathered before an uninstall operation.
// Each field captures whether a component is present and its location.
type UninstallInfo struct {
	BinaryPath          string
	ClaudeScope         string
	ClaudeHookPath      string
	RepoName            string
	PreCommitHookPath   string
	PreCommitBackupPath string
	ConfigsRemoved      []string
	NotesEntryCount     int
	BinaryRemoved       bool
	NotesRefRemoved     bool
	HooksRemoved        bool
	HooksRestored       bool
	ClaudeRemoved       bool
	InRepo              bool
	HooksInstalled      bool
	ClaudeInstalled     bool
	HooksHasBackup      bool
}

// GatherBinaryPath resolves the current executable path.
func GatherBinaryPath() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", output.NewSystemErrorWithCause("failed to determine binary location", err)
	}
	return execPath, nil
}

// GatherRepoInfo collects repository-level state: name, notes ref, entry count.
func GatherRepoInfo(info *UninstallInfo) {
	if root, err := git.RepoRoot(); err == nil {
		info.RepoName = filepath.Base(root)
	}
	if git.NotesRefExists() {
		info.NotesRefRemoved = true
		if commits, err := git.ListNotedCommits(); err == nil {
			info.NotesEntryCount = len(commits)
		}
	}
	info.ConfigsRemoved = FindNotesConfigs()
}

// GatherHookInfo collects pre-commit hook state.
func GatherHookInfo(info *UninstallInfo) {
	hooksDir, err := GetHooksDir()
	if err != nil {
		return
	}
	p := filepath.Join(hooksDir, "pre-commit")
	info.HooksInstalled = CheckHookStatus(p).Installed
	info.HooksHasBackup = HookExists(p + ".backup")
	info.PreCommitHookPath = p
	info.PreCommitBackupPath = p + ".backup"
}

// GatherClaudeInfo detects Claude integration in project or global scope.
func GatherClaudeInfo(info *UninstallInfo) {
	globalPath, _, _ := ResolveClaudeSettingsPath(false)
	projectPath, _, _ := ResolveClaudeSettingsPath(true)
	if IsTimbersSectionInstalled(projectPath) {
		info.ClaudeInstalled = true
		info.ClaudeScope = "project"
		info.ClaudeHookPath = projectPath
	} else if IsTimbersSectionInstalled(globalPath) {
		info.ClaudeInstalled = true
		info.ClaudeScope = "global"
		info.ClaudeHookPath = globalPath
	}
}

// FindNotesConfigs returns remotes that have timbers notes fetch configured.
func FindNotesConfigs() []string {
	var configs []string
	remotesOut, err := git.Run("remote")
	if err != nil {
		return configs
	}
	for remote := range strings.SplitSeq(strings.TrimSpace(remotesOut), "\n") {
		if remote = strings.TrimSpace(remote); remote != "" && git.NotesConfigured(remote) {
			configs = append(configs, remote)
		}
	}
	return configs
}

// RemoveClaudeIntegration removes the timbers section from a Claude hook file.
func RemoveClaudeIntegration(hookPath string) error {
	return RemoveTimbersSectionFromHook(hookPath)
}

// RemoveGitHook removes the pre-commit hook and optionally restores a backup.
// Returns whether the hook was removed and whether the backup was restored.
func RemoveGitHook(hookPath string, hasBackup bool, backupPath string) (removed, restored bool, err error) {
	if err := os.Remove(hookPath); err != nil && !os.IsNotExist(err) {
		return false, false, output.NewSystemErrorWithCause("failed to remove hook", err)
	}
	if !hasBackup {
		return true, false, nil
	}
	if err := os.Rename(backupPath, hookPath); err != nil {
		return true, false, output.NewSystemErrorWithCause("failed to restore backup hook", err)
	}
	return true, true, nil
}

// RemoveNotesRef deletes the timbers notes ref from the repository.
func RemoveNotesRef() error {
	_, err := git.Run("update-ref", "-d", "refs/notes/timbers")
	return err
}

// RemoveNotesConfig removes the timbers notes fetch refspec from a remote.
// Returns nil if no config exists for the remote.
func RemoveNotesConfig(remote string) error {
	configKey := "remote." + remote + ".fetch"
	out, err := git.Run("config", "--get-all", configKey)
	if err != nil {
		return nil // No config exists for this remote
	}
	for line := range strings.SplitSeq(out, "\n") {
		if line = strings.TrimSpace(line); strings.Contains(line, "refs/notes/timbers") {
			if _, err := git.Run("config", "--unset", configKey, line); err != nil {
				return err
			}
		}
	}
	return nil
}

// RemoveBinary removes the timbers binary at the given path.
func RemoveBinary(path string) error {
	if err := os.Remove(path); err != nil {
		return output.NewSystemErrorWithCause("failed to remove binary", err)
	}
	return nil
}
