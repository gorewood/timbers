package setup

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/output"
)

// AgentEnvState captures the installation state of a single agent environment.
type AgentEnvState struct {
	Name    string // agent env name (e.g. "claude")
	Display string // display name (e.g. "Claude Code")
	Path    string // settings file path
	Scope   string // "project" or "global"
	Removed bool   // set after successful removal
}

// UninstallInfo holds the state gathered before an uninstall operation.
// Each field captures whether a component is present and its location.
type UninstallInfo struct {
	BinaryPath          string
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
	InRepo              bool
	HooksInstalled      bool
	HooksHasBackup      bool

	// Agent environment integrations detected during gather.
	AgentEnvs []AgentEnvState
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

// GatherAgentEnvInfo detects all registered agent environment integrations.
func GatherAgentEnvInfo(info *UninstallInfo) {
	for _, env := range AllAgentEnvs() {
		path, scope, installed := env.Detect()
		if installed {
			info.AgentEnvs = append(info.AgentEnvs, AgentEnvState{
				Name:    env.Name(),
				Display: env.DisplayName(),
				Path:    path,
				Scope:   scope,
			})
		}
	}
}

// HasAgentEnvs returns true if any agent environment integrations were detected.
func (info *UninstallInfo) HasAgentEnvs() bool {
	return len(info.AgentEnvs) > 0
}

// RemoveAgentEnvs removes timbers from all detected agent environments.
func RemoveAgentEnvs(info *UninstallInfo) []string {
	var errs []string
	for i := range info.AgentEnvs {
		env := GetAgentEnv(info.AgentEnvs[i].Name)
		if env == nil {
			errs = append(errs, info.AgentEnvs[i].Name+": unknown agent environment")
			continue
		}
		// Detect which scope is installed and remove from that scope.
		project := info.AgentEnvs[i].Scope == "project"
		if err := env.Remove(project); err != nil {
			errs = append(errs, info.AgentEnvs[i].Name+": "+err.Error())
			continue
		}
		info.AgentEnvs[i].Removed = true
	}
	return errs
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
