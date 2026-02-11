package setup

import (
	"fmt"
	"os"
	"path/filepath"

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
	EntryCount          int
	TimbersDirPath      string
	TimbersDirExists    bool
	TimbersDirRemoved   bool
	BinaryRemoved       bool
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

// GatherRepoInfo collects repository-level state: name, .timbers dir, entry count.
func GatherRepoInfo(info *UninstallInfo) {
	root, err := git.RepoRoot()
	if err != nil {
		return
	}
	info.RepoName = filepath.Base(root)
	timbersDir := filepath.Join(root, ".timbers")
	info.TimbersDirPath = timbersDir

	dirInfo, statErr := os.Stat(timbersDir)
	if statErr != nil || !dirInfo.IsDir() {
		return
	}
	info.TimbersDirExists = true

	_ = filepath.WalkDir(timbersDir, func(_ string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && filepath.Ext(d.Name()) == ".json" {
			info.EntryCount++
		}
		return nil
	})
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

// RemoveTimbersDirContents removes all JSON entry files from .timbers/ recursively,
// then removes any empty subdirectories.
func RemoveTimbersDirContents(dirPath string) error {
	// Remove all JSON files recursively
	err := filepath.WalkDir(dirPath, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() && filepath.Ext(d.Name()) == ".json" {
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("removing %s: %w", path, err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walking %s: %w", dirPath, err)
	}

	// Collect subdirectories and remove empty ones bottom-up
	var dirs []string
	_ = filepath.WalkDir(dirPath, func(path string, d os.DirEntry, err error) error {
		if err == nil && d.IsDir() && path != dirPath {
			dirs = append(dirs, path)
		}
		return nil
	})
	for i := len(dirs) - 1; i >= 0; i-- {
		_ = os.Remove(dirs[i]) // succeeds only if empty
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
