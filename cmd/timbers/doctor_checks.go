// Package main provides the entry point for the timbers CLI.
package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
	"github.com/gorewood/timbers/internal/setup"
)

// runCoreChecks performs core infrastructure checks.
func runCoreChecks() []checkResult {
	checks := make([]checkResult, 0, 4)
	checks = append(checks, checkTimbersDirExists())
	checks = append(checks, checkBinaryInPath())
	checks = append(checks, checkVersion())
	checks = append(checks, checkGitattributes())
	return checks
}

// checkTimbersDirExists checks if the .timbers/ directory exists.
func checkTimbersDirExists() checkResult {
	root, err := git.RepoRoot()
	if err != nil {
		return checkResult{
			Name:    "Timbers Directory",
			Status:  checkWarn,
			Message: "could not determine repo root: " + err.Error(),
		}
	}

	timbersDir := filepath.Join(root, ".timbers")
	info, statErr := os.Stat(timbersDir)
	if statErr == nil && info.IsDir() {
		return checkResult{
			Name:    "Timbers Directory",
			Status:  checkPass,
			Message: ".timbers/ directory exists",
		}
	}

	return checkResult{
		Name:    "Timbers Directory",
		Status:  checkWarn,
		Message: ".timbers/ directory not found",
		Hint:    "Run 'timbers init' to initialize",
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
	storage, storageErr := ledger.NewDefaultStorage()
	if storageErr != nil {
		return checkResult{
			Name:    "Recent Entries",
			Status:  checkWarn,
			Message: "could not check: " + storageErr.Error(),
		}
	}

	entries, err := storage.ListEntries()
	if err != nil {
		return checkResult{
			Name:    "Recent Entries",
			Status:  checkWarn,
			Message: "could not list entries: " + err.Error(),
		}
	}

	count := len(entries)
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

// checkGitattributes checks if .gitattributes has the linguist-generated rule for .timbers/.
func checkGitattributes() checkResult {
	root, err := git.RepoRoot()
	if err != nil {
		return checkResult{
			Name:    "Gitattributes",
			Status:  checkWarn,
			Message: "could not determine repo root: " + err.Error(),
		}
	}

	data, readErr := os.ReadFile(filepath.Join(root, ".gitattributes"))
	if readErr != nil {
		return checkResult{
			Name:    "Gitattributes",
			Status:  checkWarn,
			Message: ".gitattributes missing linguist-generated rule",
			Hint:    "Run 'timbers init' to configure",
		}
	}

	if strings.Contains(string(data), "/.timbers/**") && strings.Contains(string(data), "linguist-generated") {
		return checkResult{
			Name:    "Gitattributes",
			Status:  checkPass,
			Message: ".gitattributes configured for timbers",
		}
	}

	return checkResult{
		Name:    "Gitattributes",
		Status:  checkWarn,
		Message: ".gitattributes missing linguist-generated rule",
		Hint:    "Run 'timbers init' to configure",
	}
}

// runIntegrationChecks performs integration-related checks.
func runIntegrationChecks(flags *doctorFlags) []checkResult {
	checks := make([]checkResult, 0, 2)
	checks = append(checks, checkGitHooks())
	checks = append(checks, checkAgentIntegrations(flags)...)
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

// checkAgentIntegrations checks all registered agent environments.
func checkAgentIntegrations(flags *doctorFlags) []checkResult {
	envs := setup.AllAgentEnvs()
	results := make([]checkResult, 0, len(envs))
	for _, env := range envs {
		results = append(results, checkAgentEnv(env, flags))
	}
	return results
}

// checkAgentEnv checks a single agent environment's integration status.
func checkAgentEnv(env setup.AgentEnv, flags *doctorFlags) checkResult {
	name := env.DisplayName() + " Integration"

	_, _, installed := env.Detect()
	if installed {
		return checkResult{
			Name:    name,
			Status:  checkPass,
			Message: "timbers configured in " + env.DisplayName() + " hooks",
		}
	}

	if flags.fix {
		if _, err := env.Install(true); err == nil {
			return checkResult{
				Name:    name,
				Status:  checkPass,
				Message: "timbers configured in " + env.DisplayName() + " hooks (auto-fixed)",
			}
		}
	}

	return checkResult{
		Name:    name,
		Status:  checkWarn,
		Message: env.DisplayName() + " hooks not configured for timbers",
		Hint:    "Run 'timbers setup " + env.Name() + "' or 'timbers doctor --fix'",
	}
}
