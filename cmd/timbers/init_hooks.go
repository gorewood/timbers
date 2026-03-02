// Package main provides the entry point for the timbers CLI.
package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gorewood/timbers/internal/setup"
)

// executePostRewriteStep runs the post-rewrite hook installation step.
func executePostRewriteStep(state *initState, flags *initFlags) initStepResult {
	if !flags.hooks {
		return initStepResult{Name: "post_rewrite", Status: "skipped", Message: "not requested (use --hooks)"}
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
	if !flags.hooks {
		return initStepResult{Name: "post_commit", Status: "skipped", Message: "not requested (use --hooks)"}
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
