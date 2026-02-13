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
	steps := make([]initStepResult, 0, 5)
	steps = append(steps, buildTimbersDirStep(state))
	steps = append(steps, buildGitattributesStep(state))
	steps = append(steps, buildHooksStep(state, flags))
	steps = append(steps, buildPostRewriteStep(state, flags))
	steps = append(steps, buildAgentEnvStep(state, flags))
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

// buildAgentEnvStep creates the dry-run step for agent environment integration.
func buildAgentEnvStep(state *initState, flags *initFlags) initStepResult {
	switch {
	case flags.noAgent:
		return initStepResult{Name: "agent_env", Status: "skipped", Message: "disabled via --no-agent"}
	case state.agentEnvInstalled:
		return initStepResult{Name: "agent_env", Status: "skipped", Message: "already installed"}
	case flags.yes:
		return initStepResult{Name: "agent_env", Status: "dry_run", Message: "would install agent integration"}
	default:
		return initStepResult{Name: "agent_env", Status: "dry_run", Message: "would prompt agent integration"}
	}
}

// executeInitSteps runs all initialization steps and returns results.
func executeInitSteps(
	cmd *cobra.Command, printer *output.Printer, styles initStyleSet,
	state *initState, flags *initFlags,
) []initStepResult {
	steps := make([]initStepResult, 0, 5)

	for _, stepFn := range []func() initStepResult{
		func() initStepResult { return performTimbersDirInit(state) },
		func() initStepResult { return performGitattributesInit(state) },
		func() initStepResult { return executeHooksStep(state, flags) },
		func() initStepResult { return executePostRewriteStep(state, flags) },
		func() initStepResult { return executeAgentEnvStep(cmd, printer, styles, state, flags) },
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

// executeAgentEnvStep runs the agent environment integration step.
func executeAgentEnvStep(
	cmd *cobra.Command, printer *output.Printer, styles initStyleSet,
	state *initState, flags *initFlags,
) initStepResult {
	if flags.noAgent {
		return initStepResult{Name: "agent_env", Status: "skipped", Message: "disabled via --no-agent"}
	}
	return performAgentEnvSetup(cmd, printer, styles, state, flags)
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

// performAgentEnvSetup handles agent environment integration setup.
// Currently installs Claude Code integration (the only registered agent env).
func performAgentEnvSetup(
	cmd *cobra.Command, printer *output.Printer, styles initStyleSet,
	state *initState, flags *initFlags,
) initStepResult {
	if state.agentEnvInstalled {
		return initStepResult{Name: "agent_env", Status: "skipped", Message: "already installed"}
	}

	// Default to Claude Code (first registered agent env).
	env := setup.GetAgentEnv("claude")
	if env == nil {
		return initStepResult{Name: "agent_env", Status: "skipped", Message: "no agent environments available"}
	}

	if flags.yes {
		return installAgentEnv(env)
	}

	if !printer.IsJSON() && output.IsTTY(cmd.OutOrStdout()) {
		return promptAgentEnvInstall(printer, styles, env)
	}

	return initStepResult{Name: "agent_env", Status: "skipped", Message: "non-interactive mode"}
}

// promptAgentEnvInstall prompts the user for agent env integration.
func promptAgentEnvInstall(printer *output.Printer, styles initStyleSet, env setup.AgentEnv) initStepResult {
	printer.Println()
	printer.Print("%s\n", styles.dim.Render("Optional integrations:"))
	printer.Print("  Install %s? [Y/n] ", styles.accent.Render(env.DisplayName()+" integration"))

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return initStepResult{Name: "agent_env", Status: "skipped", Message: "could not read input"}
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response == "" || response == "y" || response == "yes" {
		return installAgentEnv(env)
	}

	return initStepResult{Name: "agent_env", Status: "skipped", Message: "user declined"}
}

// installAgentEnv installs an agent environment integration at project level.
func installAgentEnv(env setup.AgentEnv) initStepResult {
	path, err := env.Install(true) // project scope
	if err != nil {
		return initStepResult{Name: "agent_env", Status: "failed", Message: err.Error()}
	}

	location := filepath.Base(filepath.Dir(path)) + "/" + filepath.Base(path)
	msg := "installed " + env.DisplayName() + " at " + location
	return initStepResult{Name: "agent_env", Status: "ok", Message: msg}
}
