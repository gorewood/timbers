// Package setup provides business logic for installing and managing
// timbers integrations: git hooks and agent coding environment hooks.
//
// This package contains pure functions for hook generation, installation,
// backup, and removal. Command-layer adapters in cmd/timbers/ handle
// CLI concerns (flags, output formatting, cobra wiring) and delegate
// to this package for the actual work.
//
// # Git Hooks
//
// Git hook operations (pre-commit install, uninstall, backup, status):
//
//	status := setup.CheckHookStatus(hookPath)
//	content := setup.GeneratePreCommitHook(true)
//	err := setup.BackupExistingHook(hookPath)
//
// # Agent Environment Integration
//
// Agent environments (Claude Code, Gemini CLI, Codex, etc.) are handled
// through the AgentEnv interface. Each implementation manages detection,
// installation, and removal of timbers hooks for its specific tool.
//
//	envs := setup.AllAgentEnvs()          // all registered environments
//	env := setup.GetAgentEnv("claude")    // specific environment
//	detected := setup.DetectedAgentEnvs() // environments with timbers installed
package setup
