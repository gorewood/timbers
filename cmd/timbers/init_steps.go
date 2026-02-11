// Package main provides the entry point for the timbers CLI.
package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/output"
	"github.com/gorewood/timbers/internal/setup"
)

// buildDryRunSteps constructs the list of dry-run step results.
func buildDryRunSteps(state *initState, flags *initFlags) []initStepResult {
	steps := make([]initStepResult, 0, 6)
	steps = append(steps, buildTimbersDirStep(state))
	steps = append(steps, buildGitattributesStep(state))
	steps = append(steps, buildRemoteConfigStep(state))
	steps = append(steps, buildHooksStep(state, flags))
	steps = append(steps, buildPostRewriteStep(state, flags))
	steps = append(steps, buildClaudeStep(state, flags))
	return steps
}

// buildTimbersDirStep creates the dry-run step for .timbers/ directory.
func buildTimbersDirStep(state *initState) initStepResult {
	if state.timbersDirExists {
		return initStepResult{Name: "timbers_dir", Status: "skipped", Message: "already exists"}
	}
	return initStepResult{Name: "timbers_dir", Status: "dry_run", Message: "would create .timbers/ directory"}
}

// buildGitattributesStep creates the dry-run step for .gitattributes.
func buildGitattributesStep(state *initState) initStepResult {
	if state.gitattributesHasEntry {
		return initStepResult{Name: "gitattributes", Status: "skipped", Message: "already configured"}
	}
	return initStepResult{Name: "gitattributes", Status: "dry_run", Message: "would add linguist-generated entry"}
}

// buildRemoteConfigStep creates the dry-run step for remote config.
func buildRemoteConfigStep(state *initState) initStepResult {
	if state.remoteConfigured {
		return initStepResult{Name: "remote_config", Status: "skipped", Message: "already configured"}
	}
	return initStepResult{Name: "remote_config", Status: "dry_run", Message: "would configure notes fetch for origin"}
}

// buildHooksStep creates the dry-run step for hooks.
func buildHooksStep(state *initState, flags *initFlags) initStepResult {
	switch {
	case !flags.hooks:
		return initStepResult{Name: "hooks", Status: "skipped", Message: "not requested (use --hooks)"}
	case state.hooksInstalled:
		return initStepResult{Name: "hooks", Status: "skipped", Message: "already installed"}
	default:
		return initStepResult{Name: "hooks", Status: "dry_run", Message: "would install pre-commit hook"}
	}
}

// buildPostRewriteStep creates the dry-run step for the post-rewrite hook.
func buildPostRewriteStep(state *initState, flags *initFlags) initStepResult {
	switch {
	case !flags.hooks:
		return initStepResult{Name: "post_rewrite", Status: "skipped", Message: "not requested (use --hooks)"}
	case state.postRewriteInstalled:
		return initStepResult{Name: "post_rewrite", Status: "skipped", Message: "already installed"}
	default:
		return initStepResult{Name: "post_rewrite", Status: "dry_run", Message: "would install post-rewrite hook"}
	}
}

// buildClaudeStep creates the dry-run step for Claude integration.
func buildClaudeStep(state *initState, flags *initFlags) initStepResult {
	switch {
	case flags.noClaude:
		return initStepResult{Name: "claude", Status: "skipped", Message: "disabled via --no-claude"}
	case state.claudeInstalled:
		return initStepResult{Name: "claude", Status: "skipped", Message: "already installed"}
	case flags.yes:
		return initStepResult{Name: "claude", Status: "dry_run", Message: "would install Claude integration"}
	default:
		return initStepResult{Name: "claude", Status: "dry_run", Message: "would prompt Claude integration"}
	}
}

// executeInitSteps runs all initialization steps and returns results.
func executeInitSteps(
	cmd *cobra.Command, printer *output.Printer, styles initStyleSet,
	state *initState, flags *initFlags,
) []initStepResult {
	steps := make([]initStepResult, 0, 6)

	for _, stepFn := range []func() initStepResult{
		func() initStepResult { return performTimbersDirInit(state) },
		func() initStepResult { return performGitattributesInit(state) },
		func() initStepResult { return performRemoteConfig(state) },
		func() initStepResult { return executeHooksStep(state, flags) },
		func() initStepResult { return executePostRewriteStep(state, flags) },
		func() initStepResult { return executeClaudeStep(cmd, printer, styles, state, flags) },
	} {
		step := stepFn()
		steps = append(steps, step)
		if !printer.IsJSON() {
			printStepResult(printer, styles, step)
		}
	}

	return steps
}

// executeHooksStep runs the hooks installation step.
func executeHooksStep(state *initState, flags *initFlags) initStepResult {
	if !flags.hooks {
		return initStepResult{Name: "hooks", Status: "skipped", Message: "not requested (use --hooks)"}
	}
	return performHooksInstall(state)
}

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

// executeClaudeStep runs the Claude integration step.
func executeClaudeStep(
	cmd *cobra.Command, printer *output.Printer, styles initStyleSet,
	state *initState, flags *initFlags,
) initStepResult {
	if flags.noClaude {
		return initStepResult{Name: "claude", Status: "skipped", Message: "disabled via --no-claude"}
	}
	return performClaudeSetup(cmd, printer, styles, state, flags)
}

// performTimbersDirInit creates the .timbers/ directory if it doesn't exist.
func performTimbersDirInit(state *initState) initStepResult {
	if state.timbersDirExists {
		return initStepResult{Name: "timbers_dir", Status: "skipped", Message: "already exists"}
	}

	root, err := git.RepoRoot()
	if err != nil {
		return initStepResult{Name: "timbers_dir", Status: "failed", Message: err.Error()}
	}

	timbersDir := filepath.Join(root, ".timbers")
	if err := os.MkdirAll(timbersDir, 0o755); err != nil {
		return initStepResult{Name: "timbers_dir", Status: "failed", Message: err.Error()}
	}

	state.timbersDirExists = true
	return initStepResult{Name: "timbers_dir", Status: "ok", Message: "created .timbers/"}
}

// performGitattributesInit ensures .gitattributes contains the timbers linguist-generated line.
func performGitattributesInit(state *initState) initStepResult {
	if state.gitattributesHasEntry {
		return initStepResult{Name: "gitattributes", Status: "skipped", Message: "already configured"}
	}

	root, err := git.RepoRoot()
	if err != nil {
		return initStepResult{Name: "gitattributes", Status: "failed", Message: err.Error()}
	}

	path := filepath.Join(root, ".gitattributes")
	line := "/.timbers/** linguist-generated"

	existing, readErr := os.ReadFile(path)
	var content string
	if readErr == nil {
		content = string(existing)
		if !strings.HasSuffix(content, "\n") && len(content) > 0 {
			content += "\n"
		}
		content += line + "\n"
	} else {
		content = line + "\n"
	}

	// #nosec G306 -- .gitattributes is a tracked file, needs standard perms
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return initStepResult{Name: "gitattributes", Status: "failed", Message: err.Error()}
	}

	state.gitattributesHasEntry = true
	return initStepResult{Name: "gitattributes", Status: "ok", Message: "added linguist-generated entry"}
}

// performRemoteConfig configures notes fetch for origin.
func performRemoteConfig(state *initState) initStepResult {
	if state.remoteConfigured {
		return initStepResult{Name: "remote_config", Status: "skipped", Message: "already configured"}
	}

	if err := git.ConfigureNotesFetch("origin"); err != nil {
		return initStepResult{Name: "remote_config", Status: "failed", Message: err.Error()}
	}

	return initStepResult{Name: "remote_config", Status: "ok", Message: "configured for origin"}
}

// performHooksInstall installs the pre-commit hook.
func performHooksInstall(state *initState) initStepResult {
	if state.hooksInstalled {
		return initStepResult{Name: "hooks", Status: "skipped", Message: "already installed"}
	}

	hooksDir, err := setup.GetHooksDir()
	if err != nil {
		return initStepResult{Name: "hooks", Status: "failed", Message: err.Error()}
	}

	preCommitPath := filepath.Join(hooksDir, "pre-commit")
	existingHook := setup.HookExists(preCommitPath)

	if existingHook {
		if err := setup.BackupExistingHook(preCommitPath); err != nil {
			return initStepResult{Name: "hooks", Status: "failed", Message: "failed to backup: " + err.Error()}
		}
	}

	hookContent := setup.GeneratePreCommitHook(existingHook)
	// #nosec G306 -- hook needs execute permission
	if err := os.WriteFile(preCommitPath, []byte(hookContent), 0o755); err != nil {
		return initStepResult{Name: "hooks", Status: "failed", Message: "failed to write: " + err.Error()}
	}

	msg := "installed"
	if existingHook {
		msg = "installed (chained)"
	}
	return initStepResult{Name: "hooks", Status: "ok", Message: msg}
}

// performClaudeSetup handles Claude integration setup.
func performClaudeSetup(
	cmd *cobra.Command, printer *output.Printer, styles initStyleSet,
	state *initState, flags *initFlags,
) initStepResult {
	if state.claudeInstalled {
		return initStepResult{Name: "claude", Status: "skipped", Message: "already installed"}
	}

	if flags.yes {
		return installClaudeIntegration()
	}

	if !printer.IsJSON() && output.IsTTY(cmd.OutOrStdout()) {
		return promptClaudeInstall(printer, styles)
	}

	return initStepResult{Name: "claude", Status: "skipped", Message: "non-interactive mode"}
}

// promptClaudeInstall prompts the user for Claude integration.
func promptClaudeInstall(printer *output.Printer, styles initStyleSet) initStepResult {
	printer.Println()
	printer.Print("%s\n", styles.dim.Render("Optional integrations:"))
	printer.Print("  Install %s? [Y/n] ", styles.accent.Render("Claude Code integration"))

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return initStepResult{Name: "claude", Status: "skipped", Message: "could not read input"}
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response == "" || response == "y" || response == "yes" {
		return installClaudeIntegration()
	}

	return initStepResult{Name: "claude", Status: "skipped", Message: "user declined"}
}

// installClaudeIntegration installs the Claude hook at project level.
func installClaudeIntegration() initStepResult {
	hookPath, _, err := setup.ResolveClaudeSettingsPath(true)
	if err != nil {
		return initStepResult{Name: "claude", Status: "failed", Message: err.Error()}
	}

	if err := setup.InstallTimbersSection(hookPath); err != nil {
		return initStepResult{Name: "claude", Status: "failed", Message: err.Error()}
	}

	return initStepResult{Name: "claude", Status: "ok", Message: "installed in .claude/settings.local.json"}
}
