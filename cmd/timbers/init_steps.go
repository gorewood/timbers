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
)

// buildDryRunSteps constructs the list of dry-run step results.
func buildDryRunSteps(state *initState, flags *initFlags) []initStepResult {
	steps := make([]initStepResult, 0, 4)
	steps = append(steps, buildNotesRefStep(state))
	steps = append(steps, buildRemoteConfigStep(state))
	steps = append(steps, buildHooksStep(state, flags))
	steps = append(steps, buildClaudeStep(state, flags))
	return steps
}

// buildNotesRefStep creates the dry-run step for notes ref.
func buildNotesRefStep(state *initState) initStepResult {
	if state.notesRefExists {
		return initStepResult{Name: "notes_ref", Status: "skipped", Message: "already exists"}
	}
	return initStepResult{Name: "notes_ref", Status: "dry_run", Message: "would create refs/notes/timbers"}
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
	case flags.noHooks:
		return initStepResult{Name: "hooks", Status: "skipped", Message: "disabled via --no-hooks"}
	case state.hooksInstalled:
		return initStepResult{Name: "hooks", Status: "skipped", Message: "already installed"}
	default:
		return initStepResult{Name: "hooks", Status: "dry_run", Message: "would install pre-commit hook"}
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
func executeInitSteps(cmd *cobra.Command, printer *output.Printer, state *initState, flags *initFlags) []initStepResult {
	steps := make([]initStepResult, 0, 3)

	step := performNotesInit(state)
	steps = append(steps, step)
	if !printer.IsJSON() {
		printStepResult(printer, step)
	}

	step = executeHooksStep(state, flags)
	steps = append(steps, step)
	if !printer.IsJSON() {
		printStepResult(printer, step)
	}

	step = executeClaudeStep(cmd, printer, state, flags)
	steps = append(steps, step)
	if !printer.IsJSON() {
		printStepResult(printer, step)
	}

	return steps
}

// executeHooksStep runs the hooks installation step.
func executeHooksStep(state *initState, flags *initFlags) initStepResult {
	if flags.noHooks {
		return initStepResult{Name: "hooks", Status: "skipped", Message: "disabled via --no-hooks"}
	}
	return performHooksInstall(state)
}

// executeClaudeStep runs the Claude integration step.
func executeClaudeStep(cmd *cobra.Command, printer *output.Printer, state *initState, flags *initFlags) initStepResult {
	if flags.noClaude {
		return initStepResult{Name: "claude", Status: "skipped", Message: "disabled via --no-claude"}
	}
	return performClaudeSetup(cmd, printer, state, flags)
}

// performNotesInit configures notes fetch for origin.
func performNotesInit(state *initState) initStepResult {
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

	hooksDir, err := getHooksDir()
	if err != nil {
		return initStepResult{Name: "hooks", Status: "failed", Message: err.Error()}
	}

	preCommitPath := filepath.Join(hooksDir, "pre-commit")
	existingHook := hookExists(preCommitPath)

	if existingHook {
		if err := backupExistingHook(preCommitPath); err != nil {
			return initStepResult{Name: "hooks", Status: "failed", Message: "failed to backup: " + err.Error()}
		}
	}

	hookContent := generatePreCommitHook(existingHook)
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
func performClaudeSetup(cmd *cobra.Command, printer *output.Printer, state *initState, flags *initFlags) initStepResult {
	if state.claudeInstalled {
		return initStepResult{Name: "claude", Status: "skipped", Message: "already installed"}
	}

	if flags.yes {
		return installClaudeIntegration()
	}

	if !printer.IsJSON() && output.IsTTY(cmd.OutOrStdout()) {
		return promptClaudeInstall(printer)
	}

	return initStepResult{Name: "claude", Status: "skipped", Message: "non-interactive mode"}
}

// promptClaudeInstall prompts the user for Claude integration.
func promptClaudeInstall(printer *output.Printer) initStepResult {
	printer.Println()
	printer.Print("Optional integrations:\n")
	printer.Print("  Install Claude Code integration? [Y/n] ")

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

// installClaudeIntegration installs the Claude hook globally.
func installClaudeIntegration() initStepResult {
	hookPath, _, err := resolveClaudeHookPath(false)
	if err != nil {
		return initStepResult{Name: "claude", Status: "failed", Message: err.Error()}
	}

	if err := installTimbersSection(hookPath); err != nil {
		return initStepResult{Name: "claude", Status: "failed", Message: err.Error()}
	}

	return initStepResult{Name: "claude", Status: "ok", Message: "installed globally"}
}
