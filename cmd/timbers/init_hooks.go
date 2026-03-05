// Package main provides the entry point for the timbers CLI.
package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gorewood/timbers/internal/output"
	"github.com/gorewood/timbers/internal/setup"
)

// preCommitSectionContent is the timbers section content for the pre-commit hook.
// Does not include delimiters — AppendTimbersSection adds those.
const preCommitSectionContent = `if command -v timbers >/dev/null 2>&1; then
  timbers hook run pre-commit "$@"
  rc=$?
  if [ $rc -ne 0 ]; then exit $rc; fi
fi
`

// performHooksInstallWithTier installs hooks using tier-based logic.
func performHooksInstallWithTier(env setup.HookEnvInfo, state *initState) initStepResult {
	switch env.Tier {
	case setup.HookEnvUncontested:
		// Tier 1: Create hook file with timbers section.
		hookPath := filepath.Join(env.HooksDir, "pre-commit")
		if err := setup.AppendTimbersSection(hookPath, preCommitSectionContent); err != nil {
			return initStepResult{
				Name: "hooks", Status: "failed",
				Message: "failed to install: " + err.Error(),
			}
		}
		state.hooksInstalled = true
		return initStepResult{Name: "hooks", Status: "ok", Message: "installed"}

	case setup.HookEnvExistingHook:
		if env.HasTimbers {
			return initStepResult{
				Name: "hooks", Status: "skipped",
				Message: "timbers hooks already installed",
			}
		}
		return appendPreCommitSection(env, state)

	case setup.HookEnvKnownOverride:
		if env.HasTimbers {
			return initStepResult{
				Name: "hooks", Status: "skipped",
				Message: "timbers hooks already installed",
			}
		}
		return appendPreCommitSection(env, state)

	case setup.HookEnvUnknownOverride:
		msg := "core.hooksPath is set to " + env.HooksDir +
			"; timbers won't modify hooks it doesn't recognize." +
			" Use `timbers hooks install --force` to override."
		return initStepResult{Name: "hooks", Status: "skipped", Message: msg}

	default:
		return initStepResult{
			Name: "hooks", Status: "skipped",
			Message: "unknown hook environment tier",
		}
	}
}

// appendPreCommitSection appends the timbers section to an existing hook.
func appendPreCommitSection(
	env setup.HookEnvInfo, state *initState,
) initStepResult {
	hookPath := filepath.Join(env.HooksDir, "pre-commit")
	appendable, reason := setup.IsAppendable(hookPath)
	if !appendable {
		msg := "hook at " + hookPath + " is a " + reason
		if env.Owner != "" {
			msg += " managed by " + env.Owner
		}
		msg += "; add `timbers hook run pre-commit \"$@\"` manually"
		return initStepResult{Name: "hooks", Status: "skipped", Message: msg}
	}

	if err := setup.AppendTimbersSection(hookPath, preCommitSectionContent); err != nil {
		return initStepResult{
			Name: "hooks", Status: "failed",
			Message: "failed to append: " + err.Error(),
		}
	}
	state.hooksInstalled = true

	msg := "installed"
	if env.Owner != "" {
		msg += " (alongside " + env.Owner + " hook)"
	} else {
		msg += " (appended to existing hook)"
	}
	return initStepResult{Name: "hooks", Status: "ok", Message: msg}
}

// informHookOpportunity emits info messages about hook availability.
// Called when --git-hooks is not specified.
func informHookOpportunity(env setup.HookEnvInfo, printer *output.Printer) {
	switch env.Tier {
	case setup.HookEnvUncontested:
		printer.Stderr("%s\n",
			"Git hooks available for commit-time enforcement."+
				" Run `timbers init --git-hooks` to install.")
	case setup.HookEnvExistingHook:
		if env.HasTimbers {
			return
		}
		printer.Stderr("%s %s\n",
			"Pre-commit hook exists."+
				" Claude Code steering provides session-end enforcement.",
			"Run `timbers hooks install` to also add commit-time checks.")
	case setup.HookEnvKnownOverride:
		if env.HasTimbers {
			return
		}
		printer.Stderr("Git hooks managed by %s. %s\n", env.Owner,
			"Claude Code steering provides session-end enforcement."+
				" Run `timbers hooks install` to integrate.")
	case setup.HookEnvUnknownOverride:
		printer.Stderr(
			"core.hooksPath is set to %s."+
				" Timbers defers to your configuration."+
				" Run `timbers doctor` for details.\n",
			env.HooksDir)
	}
}

// executePostRewriteStep runs the post-rewrite hook installation step.
func executePostRewriteStep(state *initState, flags *initFlags) initStepResult {
	if flags.noGitHooks {
		return initStepResult{Name: "post_rewrite", Status: "skipped", Message: "disabled via --no-git-hooks"}
	}
	if !flags.gitHooks {
		return initStepResult{Name: "post_rewrite", Status: "skipped", Message: "not requested (use --git-hooks)"}
	}
	return performPostRewriteInstall(state)
}

// performPostRewriteInstall installs the post-rewrite hook for SHA remapping after rebase.
func performPostRewriteInstall(state *initState) initStepResult {
	if state.postRewriteInstalled {
		return initStepResult{Name: "post_rewrite", Status: "skipped", Message: "already installed"}
	}

	hooksDir, err := setup.GetHooksDir()
	if err != nil {
		return initStepResult{Name: "post_rewrite", Status: "failed", Message: err.Error()}
	}

	hookPath := filepath.Join(hooksDir, "post-rewrite")
	hookContent := generatePostRewriteHook()

	existingHook := setup.HookExists(hookPath)
	if existingHook {
		// Read existing hook and append timbers section
		existing, readErr := os.ReadFile(hookPath)
		if readErr != nil {
			return initStepResult{Name: "post_rewrite", Status: "failed", Message: readErr.Error()}
		}
		content := string(existing)
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		content += "\n" + postRewriteTimbersSection()
		// #nosec G306 -- hook needs execute permission
		if err := os.WriteFile(hookPath, []byte(content), 0o755); err != nil {
			return initStepResult{Name: "post_rewrite", Status: "failed", Message: err.Error()}
		}
		return initStepResult{Name: "post_rewrite", Status: "ok", Message: "installed (chained)"}
	}

	// #nosec G306 -- hook needs execute permission
	if err := os.WriteFile(hookPath, []byte(hookContent), 0o755); err != nil {
		return initStepResult{Name: "post_rewrite", Status: "failed", Message: err.Error()}
	}

	return initStepResult{Name: "post_rewrite", Status: "ok", Message: "installed"}
}

// executePostCommitStep runs the post-commit hook installation step.
func executePostCommitStep(state *initState, flags *initFlags) initStepResult {
	if flags.noGitHooks {
		return initStepResult{Name: "post_commit", Status: "skipped", Message: "disabled via --no-git-hooks"}
	}
	if !flags.gitHooks {
		return initStepResult{Name: "post_commit", Status: "skipped", Message: "not requested (use --git-hooks)"}
	}
	return performPostCommitInstall(state)
}

// performPostCommitInstall installs the post-commit hook for logging reminders.
func performPostCommitInstall(state *initState) initStepResult {
	if state.postCommitInstalled {
		return initStepResult{Name: "post_commit", Status: "skipped", Message: "already installed"}
	}

	hooksDir, err := setup.GetHooksDir()
	if err != nil {
		return initStepResult{Name: "post_commit", Status: "failed", Message: err.Error()}
	}

	hookPath := filepath.Join(hooksDir, "post-commit")
	existed := setup.HookExists(hookPath)

	if err := setup.InstallPostCommitHook(hookPath); err != nil {
		return initStepResult{Name: "post_commit", Status: "failed", Message: err.Error()}
	}

	state.postCommitInstalled = true
	msg := "installed"
	if existed {
		msg = "installed (appended)"
	}
	return initStepResult{Name: "post_commit", Status: "ok", Message: msg}
}
