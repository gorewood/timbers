package main

import (
	"path/filepath"

	"github.com/gorewood/timbers/internal/setup"
)

// checkGitHooks checks if timbers is integrated with git hooks.
// Uses tier-based messaging. Never warns on hook absence.
func checkGitHooks(flags *doctorFlags) checkResult {
	env, err := setup.ClassifyHookEnv()
	if err != nil {
		return checkResult{
			Name:    "Git Hooks",
			Status:  checkWarn,
			Message: "could not classify hook environment: " + err.Error(),
		}
	}

	agentActive := len(setup.DetectedAgentEnvs()) > 0
	preCommitPath := filepath.Join(env.HooksDir, "pre-commit")

	// If timbers section is present, report active regardless of tier.
	if setup.HasTimbersSection(preCommitPath) {
		// Migrate old-format hooks to section-delimited on --fix.
		if flags.fix && setup.IsOldFormatHook(preCommitPath) {
			return migrateOldFormatHook(preCommitPath, agentActive)
		}
		return checkGitHooksActive(env, agentActive)
	}

	// Not installed — try --fix or emit informational message.
	if flags.fix {
		return fixGitHooks(env, preCommitPath, agentActive)
	}

	return checkGitHooksNotInstalled(env, agentActive)
}

// checkGitHooksActive returns the check result when hooks are installed.
func checkGitHooksActive(env setup.HookEnvInfo, agentActive bool) checkResult {
	var msg string

	switch env.Tier {
	case setup.HookEnvExistingHook:
		msg = "pre-commit hook active (alongside existing hook)"
	case setup.HookEnvKnownOverride:
		msg = "pre-commit hook active (in " + env.Owner + "-managed hooks)"
	case setup.HookEnvUncontested, setup.HookEnvUnknownOverride:
		msg = "pre-commit hook active (blocks commits; bypass with --no-verify)"
	}

	if agentActive {
		msg += ". Claude Code steering provides session-end enforcement."
	}

	return checkResult{Name: "Git Hooks", Status: checkPass, Message: msg}
}

// checkGitHooksNotInstalled returns informational results for absent hooks.
func checkGitHooksNotInstalled(
	env setup.HookEnvInfo, agentActive bool,
) checkResult {
	switch env.Tier {
	case setup.HookEnvUncontested:
		return checkGitHooksUncontested(agentActive)

	case setup.HookEnvExistingHook:
		return checkResult{
			Name:   "Git Hooks",
			Status: checkPass,
			Message: "pre-commit hook exists (not timbers)." +
				" Run `timbers hooks install` to add timbers alongside it.",
		}

	case setup.HookEnvKnownOverride:
		return checkResult{
			Name:   "Git Hooks",
			Status: checkPass,
			Message: "git hooks managed by " + env.Owner +
				". Run `timbers hooks install` to integrate.",
		}

	case setup.HookEnvUnknownOverride:
		return checkResult{
			Name:   "Git Hooks",
			Status: checkPass,
			Message: "core.hooksPath set to " + env.HooksDir +
				". Timbers defers to your configuration." +
				" Run `timbers hooks status` for details.",
		}

	default:
		return checkResult{
			Name: "Git Hooks", Status: checkPass,
			Message: "unknown hook environment",
		}
	}
}

// checkGitHooksUncontested returns the result for Tier 1 (no hooks).
func checkGitHooksUncontested(agentActive bool) checkResult {
	if agentActive {
		return checkResult{
			Name:   "Git Hooks",
			Status: checkPass,
			Message: "no pre-commit hook (optional)." +
				" Claude Code steering is active." +
				" Run `timbers init --git-hooks` for commit-time enforcement.",
		}
	}
	return checkResult{
		Name:   "Git Hooks",
		Status: checkPass,
		Message: "no pre-commit hook and no agent steering." +
			" Run `timbers init` for Claude Code integration," +
			" or `timbers init --git-hooks` for commit-time enforcement.",
	}
}

// migrateOldFormatHook replaces an old-format hook with the section-delimited format.
func migrateOldFormatHook(preCommitPath string, agentActive bool) checkResult {
	if err := setup.MigrateOldFormatHook(preCommitPath, preCommitSectionContent); err != nil {
		return checkResult{
			Name:    "Git Hooks",
			Status:  checkWarn,
			Message: "old-format hook migration failed: " + err.Error(),
		}
	}
	msg := "pre-commit hook migrated to section-delimited format"
	if agentActive {
		msg += ". Claude Code steering provides session-end enforcement."
	}
	return checkResult{Name: "Git Hooks", Status: checkPass, Message: msg}
}

// fixGitHooks attempts to install timbers hooks via AppendTimbersSection.
func fixGitHooks(
	env setup.HookEnvInfo, preCommitPath string, agentActive bool,
) checkResult {
	// Tier 4: no auto-fix.
	if env.Tier == setup.HookEnvUnknownOverride {
		return checkResult{
			Name:   "Git Hooks",
			Status: checkPass,
			Message: "core.hooksPath set to " + env.HooksDir +
				". Timbers defers to your configuration.",
			Hint: "Run `timbers hooks status` for details",
		}
	}

	// Check appendability for existing hooks.
	if setup.HookExists(preCommitPath) {
		appendable, reason := setup.IsAppendable(preCommitPath)
		if !appendable {
			return checkResult{
				Name:   "Git Hooks",
				Status: checkPass,
				Message: "hook is a " + reason +
					"; add `timbers hook run pre-commit \"$@\"` manually",
			}
		}
	}

	if err := setup.AppendTimbersSection(preCommitPath, preCommitSectionContent); err != nil {
		return checkResult{
			Name:    "Git Hooks",
			Status:  checkWarn,
			Message: "auto-fix failed: " + err.Error(),
		}
	}

	msg := "pre-commit hook installed (auto-fixed)"
	if agentActive {
		msg += ". Claude Code steering provides session-end enforcement."
	}
	return checkResult{
		Name:    "Git Hooks",
		Status:  checkPass,
		Message: msg,
	}
}

// checkPostCommitHook checks if a post-commit hook is installed to nudge logging.
// Uses HasTimbersSection for detection. Same tier-awareness. Never warns.
func checkPostCommitHook(flags *doctorFlags) checkResult {
	hooksDir, err := setup.GetHooksDir()
	if err != nil {
		return checkResult{
			Name:    "Post-commit Hook",
			Status:  checkWarn,
			Message: "could not determine hooks directory",
		}
	}

	hookPath := filepath.Join(hooksDir, "post-commit")
	if setup.HasTimbersSection(hookPath) {
		return checkResult{
			Name:    "Post-commit Hook",
			Status:  checkPass,
			Message: "timbers logging reminder installed",
		}
	}

	if flags.fix {
		// Use AppendTimbersSection for consistency.
		appendErr := setup.AppendTimbersSection(hookPath, postCommitSectionContent)
		if appendErr == nil {
			return checkResult{
				Name:    "Post-commit Hook",
				Status:  checkPass,
				Message: "timbers logging reminder installed (auto-fixed)",
			}
		}
	}

	return checkResult{
		Name:    "Post-commit Hook",
		Status:  checkPass,
		Message: "not installed (optional). Run `timbers hooks install` to add.",
	}
}
