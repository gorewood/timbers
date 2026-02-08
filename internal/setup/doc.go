// Package setup provides business logic for installing and managing
// timbers integrations: git hooks and Claude Code session hooks.
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
// # Claude Integration
//
// Claude Code hook operations (install, remove, check):
//
//	path, scope, err := setup.ResolveClaudeHookPath(false)
//	installed := setup.IsTimbersSectionInstalled(path)
//	err := setup.InstallTimbersSection(path)
//	err := setup.RemoveTimbersSectionFromHook(path)
package setup
