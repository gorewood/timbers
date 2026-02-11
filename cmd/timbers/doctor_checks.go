// Package main provides the entry point for the timbers CLI.
package main

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
	"github.com/gorewood/timbers/internal/setup"
)

// runCoreChecks performs core infrastructure checks.
func runCoreChecks(flags *doctorFlags) []checkResult {
	checks := make([]checkResult, 0, 4)
	checks = append(checks, checkNotesRefExists())
	checks = append(checks, checkRemoteConfigured(flags))
	checks = append(checks, checkBinaryInPath())
	checks = append(checks, checkVersion())
	return checks
}

// checkNotesRefExists checks if refs/notes/timbers exists.
func checkNotesRefExists() checkResult {
	if git.NotesRefExists() {
		return checkResult{
			Name:    "Git Notes Ref",
			Status:  checkPass,
			Message: "refs/notes/timbers exists",
		}
	}

	return checkResult{
		Name:    "Git Notes Ref",
		Status:  checkWarn,
		Message: "refs/notes/timbers does not exist (will be created on first log)",
		Hint:    "Run 'timbers log' to create the first entry",
	}
}

// checkRemoteConfigured checks if notes fetch is configured for origin.
func checkRemoteConfigured(flags *doctorFlags) checkResult {
	remote := "origin"

	if git.NotesConfigured(remote) {
		return checkResult{
			Name:    "Remote Configured",
			Status:  checkPass,
			Message: "fetch/push configured for " + remote,
		}
	}

	// Attempt auto-fix if requested
	if flags.fix {
		if err := git.ConfigureNotesFetch(remote); err == nil {
			return checkResult{
				Name:    "Remote Configured",
				Status:  checkPass,
				Message: "fetch/push configured for " + remote + " (auto-fixed)",
			}
		}
	}

	return checkResult{
		Name:    "Remote Configured",
		Status:  checkWarn,
		Message: "notes fetch not configured for " + remote,
		Hint:    "Run 'timbers notes init' or 'timbers doctor --fix'",
	}
}

// checkBinaryInPath checks if timbers binary is in PATH.
func checkBinaryInPath() checkResult {
	execPath, err := os.Executable()
	if err != nil {
		return checkResult{
			Name:    "Binary in PATH",
			Status:  checkWarn,
			Message: "could not determine executable path",
		}
	}

	resolvedPath, resolveErr := filepath.EvalSymlinks(execPath)
	if resolveErr != nil {
		return checkResult{
			Name:    "Binary in PATH",
			Status:  checkWarn,
			Message: "could not resolve executable path",
		}
	}

	return checkResult{
		Name:    "Binary in PATH",
		Status:  checkPass,
		Message: resolvedPath,
	}
}

// runWorkflowChecks performs workflow-related checks.
func runWorkflowChecks() []checkResult {
	checks := make([]checkResult, 0, 2)
	checks = append(checks, checkPendingCommits())
	checks = append(checks, checkRecentEntries())
	return checks
}

// checkPendingCommits checks for undocumented commits.
func checkPendingCommits() checkResult {
	storage, storageErr := ledger.NewDefaultStorage()
	if storageErr != nil {
		return checkResult{
			Name:    "Pending Commits",
			Status:  checkWarn,
			Message: "could not check: " + storageErr.Error(),
		}
	}
	commits, _, err := storage.GetPendingCommits()
	if err != nil {
		return checkResult{
			Name:    "Pending Commits",
			Status:  checkWarn,
			Message: "could not check pending commits: " + err.Error(),
		}
	}

	count := len(commits)
	if count == 0 {
		return checkResult{
			Name:    "Pending Commits",
			Status:  checkPass,
			Message: "no undocumented commits",
		}
	}

	return checkResult{
		Name:    "Pending Commits",
		Status:  checkWarn,
		Message: strconv.Itoa(count) + " undocumented commit(s)",
		Hint:    "Run 'timbers pending' to review, then 'timbers log' to document",
	}
}

// checkRecentEntries checks if any ledger entries exist.
func checkRecentEntries() checkResult {
	commits, err := git.ListNotedCommits()
	if err != nil {
		return checkResult{
			Name:    "Recent Entries",
			Status:  checkWarn,
			Message: "could not list entries: " + err.Error(),
		}
	}

	count := len(commits)
	if count == 0 {
		return checkResult{
			Name:    "Recent Entries",
			Status:  checkWarn,
			Message: "no ledger entries found",
			Hint:    "Run 'timbers log' to create your first entry",
		}
	}

	return checkResult{
		Name:    "Recent Entries",
		Status:  checkPass,
		Message: strconv.Itoa(count) + " entry(ies) in ledger",
	}
}

// runIntegrationChecks performs integration-related checks.
func runIntegrationChecks(flags *doctorFlags) []checkResult {
	checks := make([]checkResult, 0, 2)
	checks = append(checks, checkGitHooks())
	checks = append(checks, checkClaudeIntegration(flags))
	return checks
}

// checkGitHooks checks if timbers is integrated with git hooks.
func checkGitHooks() checkResult {
	hooksDir, err := setup.GetHooksDir()
	if err != nil {
		return checkResult{
			Name:    "Git Hooks",
			Status:  checkWarn,
			Message: "could not determine hooks directory",
		}
	}

	preCommitPath := filepath.Join(hooksDir, "pre-commit")
	if !setup.HookExists(preCommitPath) {
		return checkResult{
			Name:    "Git Hooks",
			Status:  checkPass,
			Message: "not installed (optional, use 'timbers init --hooks')",
		}
	}

	status := setup.CheckHookStatus(preCommitPath)
	if status.Installed {
		return checkResult{
			Name:    "Git Hooks",
			Status:  checkPass,
			Message: "timbers integrated in pre-commit hook",
		}
	}

	return checkResult{
		Name:    "Git Hooks",
		Status:  checkPass,
		Message: "pre-commit hook present (no timbers integration)",
	}
}

// checkClaudeIntegration checks if Claude Code hooks are configured.
func checkClaudeIntegration(flags *doctorFlags) checkResult {
	// Check project-scope Claude hook first, then global.
	for _, projectScope := range []bool{true, false} {
		hookPath, _, err := setup.ResolveClaudeSettingsPath(projectScope)
		if err != nil {
			continue
		}
		if setup.IsTimbersSectionInstalled(hookPath) {
			return checkResult{
				Name:    "Claude Integration",
				Status:  checkPass,
				Message: "timbers configured in Claude hooks",
			}
		}
	}

	if flags.fix {
		hookPath, _, err := setup.ResolveClaudeSettingsPath(true) // project-level
		if err == nil {
			if err := setup.InstallTimbersSection(hookPath); err == nil {
				return checkResult{
					Name:    "Claude Integration",
					Status:  checkPass,
					Message: "timbers configured in Claude hooks (auto-fixed)",
				}
			}
		}
	}

	return checkResult{
		Name:    "Claude Integration",
		Status:  checkWarn,
		Message: "Claude hooks not configured for timbers",
		Hint:    "Run 'timbers setup claude' or 'timbers doctor --fix'",
	}
}
