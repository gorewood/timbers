// Package main provides the entry point for the timbers CLI.
package main

import (
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/output"
)

// initFlags holds the command-line flags for the init command.
type initFlags struct {
	yes      bool
	noHooks  bool
	noClaude bool
	dryRun   bool
}

// initStepResult tracks the result of a single initialization step.
type initStepResult struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "ok", "skipped", "failed", "dry_run"
	Message string `json:"message,omitempty"`
}

// initState holds the current state of timbers setup.
type initState struct {
	notesRefExists   bool
	remoteConfigured bool
	hooksInstalled   bool
	claudeInstalled  bool
}

// newInitCmd creates the init command.
func newInitCmd() *cobra.Command {
	flags := &initFlags{}

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize timbers in the current repository",
		Long: `Initialize timbers in the current repository.

This command sets up everything needed to use timbers:
  - Creates the Git notes ref (refs/notes/timbers)
  - Configures remote for notes push/fetch
  - Installs Git hooks (optional)
  - Sets up Claude Code integration (optional)

The command is idempotent - safe to run multiple times.

Examples:
  timbers init              # Interactive setup
  timbers init --yes        # Accept all defaults, no prompts
  timbers init --no-hooks   # Skip git hook installation
  timbers init --no-claude  # Skip Claude integration
  timbers init --dry-run    # Show what would be done`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runInit(cmd, flags)
		},
	}

	cmd.Flags().BoolVarP(&flags.yes, "yes", "y", false, "Accept all defaults, no prompts")
	cmd.Flags().BoolVar(&flags.noHooks, "no-hooks", false, "Skip git hook installation")
	cmd.Flags().BoolVar(&flags.noClaude, "no-claude", false, "Skip Claude integration prompt")
	cmd.Flags().BoolVar(&flags.dryRun, "dry-run", false, "Show what would be done without doing it")

	return cmd
}

// runInit executes the init command.
func runInit(cmd *cobra.Command, flags *initFlags) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), jsonFlag, output.IsTTY(cmd.OutOrStdout()))

	if !git.IsRepo() {
		err := output.NewSystemError("not in a git repository")
		printer.Error(err)
		return err
	}

	repoName := getRepoName()
	state := gatherInitState()

	if flags.dryRun {
		return handleInitDryRun(printer, repoName, state, flags)
	}

	return performInit(cmd, printer, repoName, state, flags)
}

// gatherInitState checks the current timbers setup state.
func gatherInitState() *initState {
	state := &initState{
		notesRefExists:   git.NotesRefExists(),
		remoteConfigured: git.NotesConfigured("origin"),
	}

	if hooksDir, err := getHooksDir(); err == nil {
		preCommitPath := filepath.Join(hooksDir, "pre-commit")
		hookStatus := checkHookStatus(preCommitPath)
		state.hooksInstalled = hookStatus.Installed
	}

	globalHookPath, _, _ := resolveClaudeHookPath(false)
	projectHookPath, _, _ := resolveClaudeHookPath(true)
	state.claudeInstalled = isTimbersSectionInstalled(globalHookPath) || isTimbersSectionInstalled(projectHookPath)

	return state
}

// handleInitDryRun outputs what would be done without making changes.
func handleInitDryRun(printer *output.Printer, repoName string, state *initState, flags *initFlags) error {
	steps := buildDryRunSteps(state, flags)

	if jsonFlag {
		return printer.Success(map[string]any{
			"status":    "dry_run",
			"repo_name": repoName,
			"steps":     steps,
		})
	}

	outputDryRunHumanInit(printer, repoName, steps)
	return nil
}

// outputDryRunHumanInit prints dry-run output in human format.
func outputDryRunHumanInit(printer *output.Printer, repoName string, steps []initStepResult) {
	printer.Println()
	printer.Print("Dry run: timbers init in %s\n", repoName)
	printer.Println()

	for _, step := range steps {
		icon := dryRunIcon(step.Status)
		printer.Print("  %s %s: %s\n", icon, step.Name, step.Message)
	}
}

// dryRunIcon returns the icon for a dry-run step status.
func dryRunIcon(status string) string {
	switch status {
	case "skipped":
		return "-"
	case "dry_run":
		return ">"
	default:
		return "?"
	}
}

// performInit runs the actual initialization steps.
func performInit(cmd *cobra.Command, printer *output.Printer, repoName string, state *initState, flags *initFlags) error {
	if isAlreadyInitialized(state, flags) {
		return outputAlreadyInitialized(printer, repoName)
	}

	if !jsonFlag {
		printer.Println()
		printer.Print("Initializing timbers in %s...\n", repoName)
		printer.Println()
	}

	steps := executeInitSteps(cmd, printer, state, flags)
	return outputInitResult(printer, repoName, state, steps)
}

// isAlreadyInitialized checks if timbers is fully initialized.
func isAlreadyInitialized(state *initState, flags *initFlags) bool {
	return state.notesRefExists &&
		state.remoteConfigured &&
		(flags.noHooks || state.hooksInstalled) &&
		(flags.noClaude || state.claudeInstalled)
}

// outputAlreadyInitialized handles the already-initialized case.
func outputAlreadyInitialized(printer *output.Printer, repoName string) error {
	if jsonFlag {
		return printer.Success(map[string]any{
			"status":              "ok",
			"already_initialized": true,
			"repo_name":           repoName,
		})
	}
	printer.Println()
	printer.Print("Timbers is already initialized in %s\n", repoName)
	printer.Println()
	printer.Print("Run 'timbers doctor' to check health.\n")
	return nil
}

// outputInitResult outputs the final initialization result.
func outputInitResult(printer *output.Printer, repoName string, state *initState, steps []initStepResult) error {
	remoteConfigured := stepHasStatus(steps, "remote_config", "ok")
	hooksInstalled := stepHasStatus(steps, "hooks", "ok")
	claudeInstalled := stepHasStatus(steps, "claude", "ok")

	if jsonFlag {
		return printer.Success(map[string]any{
			"status":              "ok",
			"repo_name":           repoName,
			"notes_created":       state.notesRefExists || remoteConfigured,
			"remote_configured":   remoteConfigured,
			"hooks_installed":     hooksInstalled,
			"claude_installed":    claudeInstalled,
			"already_initialized": false,
			"steps":               steps,
		})
	}

	printNextSteps(printer)
	return nil
}

// stepHasStatus checks if a step with the given name has the given status.
func stepHasStatus(steps []initStepResult, name, status string) bool {
	for _, s := range steps {
		if s.Name == name && s.Status == status {
			return true
		}
	}
	return false
}

// printNextSteps outputs the next steps message.
func printNextSteps(printer *output.Printer) {
	printer.Println()
	printer.Print("Timbers initialized!\n")
	printer.Println()
	printer.Print("Next steps:\n")
	printer.Print("  1. Add the timbers snippet to CLAUDE.md:\n")
	printer.Print("     timbers onboard >> CLAUDE.md\n")
	printer.Println()
	printer.Print("  2. Start documenting work:\n")
	printer.Print("     timbers log \"what\" --why \"why\" --how \"how\"\n")
	printer.Println()
	printer.Print("  3. Verify setup:\n")
	printer.Print("     timbers doctor\n")
}

// printStepResult prints a single step result in human format.
func printStepResult(printer *output.Printer, step initStepResult) {
	icon := stepIcon(step.Status)
	name := formatStepName(step.Name)
	printer.Print("  %s %s", icon, name)
	if step.Message != "" {
		printer.Print(" (%s)", step.Message)
	}
	printer.Println()
}

// stepIcon returns the icon for a step status.
func stepIcon(status string) string {
	switch status {
	case "ok":
		return "ok"
	case "skipped":
		return "--"
	case "failed":
		return "XX"
	default:
		return "??"
	}
}

// formatStepName converts internal step names to display names.
func formatStepName(name string) string {
	switch name {
	case "notes_ref":
		return "Git notes ref"
	case "remote_config":
		return "Remote configured"
	case "hooks":
		return "Git hooks"
	case "claude":
		return "Claude integration"
	default:
		return name
	}
}

// getRepoName returns the name of the current repository.
func getRepoName() string {
	root, err := git.RepoRoot()
	if err != nil {
		return "unknown"
	}
	return filepath.Base(root)
}
