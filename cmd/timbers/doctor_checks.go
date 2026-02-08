// Package main provides the entry point for the timbers CLI.
package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
)

// runCoreChecks performs core infrastructure checks.
func runCoreChecks(flags *doctorFlags) []checkResult {
	checks := make([]checkResult, 0, 3)
	checks = append(checks, checkNotesRefExists())
	checks = append(checks, checkRemoteConfigured(flags))
	checks = append(checks, checkBinaryInPath())
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
	storage := ledger.NewStorage(nil)
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
func runIntegrationChecks() []checkResult {
	checks := make([]checkResult, 0, 2)
	checks = append(checks, checkGitHooks())
	checks = append(checks, checkClaudeIntegration())
	return checks
}

// checkGitHooks checks if timbers is integrated with git hooks.
func checkGitHooks() checkResult {
	root, err := git.RepoRoot()
	if err != nil {
		return checkResult{
			Name:    "Git Hooks",
			Status:  checkWarn,
			Message: "could not determine repo root",
		}
	}

	preCommitPath := filepath.Join(root, ".git", "hooks", "pre-commit")
	content, readErr := os.ReadFile(preCommitPath)
	if readErr != nil {
		if os.IsNotExist(readErr) {
			return checkResult{
				Name:    "Git Hooks",
				Status:  checkWarn,
				Message: "pre-commit hook not installed",
				Hint:    "Consider adding timbers to your pre-commit workflow",
			}
		}
		return checkResult{
			Name:    "Git Hooks",
			Status:  checkWarn,
			Message: "could not read pre-commit hook",
		}
	}

	if strings.Contains(string(content), "timbers") {
		return checkResult{
			Name:    "Git Hooks",
			Status:  checkPass,
			Message: "timbers integrated in pre-commit hook",
		}
	}

	return checkResult{
		Name:    "Git Hooks",
		Status:  checkWarn,
		Message: "pre-commit hook exists but does not reference timbers",
		Hint:    "Consider adding timbers to your pre-commit workflow",
	}
}

// checkClaudeIntegration checks if Claude Code hooks are configured.
func checkClaudeIntegration() checkResult {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return checkResult{
			Name:    "Claude Integration",
			Status:  checkWarn,
			Message: "could not determine home directory",
		}
	}

	claudeHooksDir := filepath.Join(homeDir, ".claude", "hooks")
	if _, statErr := os.Stat(claudeHooksDir); os.IsNotExist(statErr) {
		return checkResult{
			Name:    "Claude Integration",
			Status:  checkWarn,
			Message: "Claude Code hooks not configured",
			Hint:    "Run 'timbers setup claude' to install (if available)",
		}
	}

	entries, readErr := os.ReadDir(claudeHooksDir)
	if readErr != nil {
		return checkResult{
			Name:    "Claude Integration",
			Status:  checkWarn,
			Message: "could not read Claude hooks directory",
		}
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		content, fileErr := os.ReadFile(filepath.Join(claudeHooksDir, entry.Name()))
		if fileErr != nil {
			continue
		}
		if strings.Contains(string(content), "timbers") {
			return checkResult{
				Name:    "Claude Integration",
				Status:  checkPass,
				Message: "timbers configured in Claude hooks",
			}
		}
	}

	return checkResult{
		Name:    "Claude Integration",
		Status:  checkWarn,
		Message: "Claude hooks directory exists but no timbers integration found",
		Hint:    "Run 'timbers setup claude' to install (if available)",
	}
}
