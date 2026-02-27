// Package main provides the entry point for the timbers CLI.
package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/output"
	"github.com/gorewood/timbers/internal/setup"
)

// initFlags holds the command-line flags for the init command.
type initFlags struct {
	yes     bool
	hooks   bool
	noAgent bool
	dryRun  bool
}

// initStepResult tracks the result of a single initialization step.
type initStepResult struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "ok", "skipped", "failed", "dry_run"
	Message string `json:"message,omitempty"`
}

// initState holds the current state of timbers setup.
type initState struct {
	timbersDirExists      bool
	gitattributesHasEntry bool
	hooksInstalled        bool
	postRewriteInstalled  bool
	agentEnvInstalled     bool // true if any agent env integration is present
}

// initStyleSet holds lipgloss styles for init output.
type initStyleSet struct {
	heading lipgloss.Style
	pass    lipgloss.Style
	skip    lipgloss.Style
	fail    lipgloss.Style
	dim     lipgloss.Style
	accent  lipgloss.Style
}

// initStyles returns a TTY-aware style set.
func initStyles(isTTY bool) initStyleSet {
	if !isTTY {
		return initStyleSet{}
	}
	return initStyleSet{
		heading: lipgloss.NewStyle().Bold(true),
		pass:    lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "10", Dark: "10"}),
		skip:    lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "8", Dark: "7"}),
		fail:    lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "9", Dark: "9"}),
		dim:     lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "8", Dark: "7"}),
		accent:  lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "12", Dark: "12"}),
	}
}

// newInitCmd creates the init command.
func newInitCmd() *cobra.Command {
	flags := &initFlags{}

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize timbers in the current repository",
		Long: `Initialize timbers in the current repository.

This command sets up everything needed to use timbers:
  - Creates the .timbers/ directory for entry storage
  - Adds .gitattributes entry to collapse timbers files in diffs
  - Configures .gitattributes for diff collapsing
  - Installs Git hooks (optional, includes post-rewrite for rebase safety)
  - Sets up agent environment integration (optional, e.g. Claude Code)

The command is idempotent - safe to run multiple times.

Examples:
  timbers init              # Interactive setup
  timbers init --yes        # Accept all defaults, no prompts
  timbers init --hooks      # Also install git hooks
  timbers init --no-agent   # Skip agent environment integration
  timbers init --dry-run    # Show what would be done`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runInit(cmd, flags)
		},
	}

	cmd.Flags().BoolVarP(&flags.yes, "yes", "y", false, "Accept all defaults, no prompts")
	cmd.Flags().BoolVar(&flags.hooks, "hooks", false, "Install git hooks (pre-commit)")
	cmd.Flags().BoolVar(&flags.noAgent, "no-agent", false, "Skip agent environment integration")
	cmd.Flags().BoolVar(&flags.dryRun, "dry-run", false, "Show what would be done without doing it")

	// Deprecated alias for backward compatibility.
	cmd.Flags().Bool("no-claude", false, "Alias for --no-agent (deprecated)")
	_ = cmd.Flags().MarkHidden("no-claude")

	return cmd
}

// runInit executes the init command.
func runInit(cmd *cobra.Command, flags *initFlags) error {
	// Support deprecated --no-claude as alias for --no-agent.
	if nc, _ := cmd.Flags().GetBool("no-claude"); nc {
		flags.noAgent = true
	}

	printer := output.NewPrinter(cmd.OutOrStdout(), isJSONMode(cmd), useColor(cmd))
	styles := initStyles(printer.IsTTY())

	if !git.IsRepo() {
		err := output.NewSystemError("not in a git repository")
		printer.Error(err)
		return err
	}

	repoName := getRepoName()
	state := gatherInitState()

	if flags.dryRun {
		return handleInitDryRun(printer, styles, repoName, state, flags)
	}

	return performInit(cmd, printer, styles, repoName, state, flags)
}

// gatherInitState checks the current timbers setup state.
func gatherInitState() *initState {
	state := &initState{}

	// Check .timbers/ directory
	if root, err := git.RepoRoot(); err == nil {
		timbersDir := filepath.Join(root, ".timbers")
		info, statErr := os.Stat(timbersDir)
		state.timbersDirExists = statErr == nil && info.IsDir()

		state.gitattributesHasEntry = checkGitattributesEntry(root)
	}

	if hooksDir, err := setup.GetHooksDir(); err == nil {
		preCommitPath := filepath.Join(hooksDir, "pre-commit")
		hookStatus := setup.CheckHookStatus(preCommitPath)
		state.hooksInstalled = hookStatus.Installed

		postRewritePath := filepath.Join(hooksDir, "post-rewrite")
		state.postRewriteInstalled = checkPostRewriteHook(postRewritePath)
	}

	state.agentEnvInstalled = len(setup.DetectedAgentEnvs()) > 0

	return state
}

// checkGitattributesEntry checks if .gitattributes contains the timbers linguist-generated line.
func checkGitattributesEntry(repoRoot string) bool {
	path := filepath.Join(repoRoot, ".gitattributes")
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return containsTimbersGitattribute(string(data))
}

// containsTimbersGitattribute returns true if the content contains the timbers linguist-generated line.
func containsTimbersGitattribute(content string) bool {
	for line := range strings.SplitSeq(content, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "/.timbers/** linguist-generated" {
			return true
		}
	}
	return false
}

// checkPostRewriteHook checks if a post-rewrite hook contains timbers SHA remapping.
func checkPostRewriteHook(hookPath string) bool {
	data, err := os.ReadFile(hookPath)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), ".timbers")
}

// handleInitDryRun outputs what would be done without making changes.
func handleInitDryRun(printer *output.Printer, styles initStyleSet, repoName string, state *initState, flags *initFlags) error {
	steps := buildDryRunSteps(state, flags)

	if printer.IsJSON() {
		return printer.Success(map[string]any{
			"status":    "dry_run",
			"repo_name": repoName,
			"steps":     steps,
		})
	}

	outputDryRunHumanInit(printer, styles, repoName, steps)
	return nil
}

// performInit runs the actual initialization steps.
func performInit(
	cmd *cobra.Command, printer *output.Printer, styles initStyleSet,
	repoName string, state *initState, flags *initFlags,
) error {
	if isAlreadyInitialized(state, flags) {
		return outputAlreadyInitialized(printer, styles, repoName)
	}

	if !printer.IsJSON() {
		printer.Println()
		printer.Print("%s %s...\n", styles.heading.Render("Initializing timbers in"), styles.dim.Render(repoName))
		printer.Println()
	}

	steps := executeInitSteps(cmd, printer, styles, state, flags)
	return outputInitResult(printer, styles, repoName, state, steps)
}

// isAlreadyInitialized checks if timbers is fully initialized.
func isAlreadyInitialized(state *initState, flags *initFlags) bool {
	return state.timbersDirExists &&
		state.gitattributesHasEntry &&
		(!flags.hooks || (state.hooksInstalled && state.postRewriteInstalled)) &&
		(flags.noAgent || state.agentEnvInstalled)
}

// outputAlreadyInitialized handles the already-initialized case.
func outputAlreadyInitialized(printer *output.Printer, styles initStyleSet, repoName string) error {
	if printer.IsJSON() {
		return printer.Success(map[string]any{
			"status":              "ok",
			"already_initialized": true,
			"repo_name":           repoName,
		})
	}
	printer.Println()
	printer.Print("%s %s\n", styles.pass.Render("Timbers is already initialized in"), repoName)
	printer.Println()
	printer.Print("Run '%s' to check health.\n", styles.accent.Render("timbers doctor"))
	return nil
}

// outputInitResult outputs the final initialization result.
func outputInitResult(printer *output.Printer, styles initStyleSet, repoName string, _ *initState, steps []initStepResult) error {
	hooksInstalled := stepSucceeded(steps, "hooks")
	agentInstalled := stepSucceeded(steps, "agent_env")
	timbersDirCreated := stepSucceeded(steps, "timbers_dir")

	if printer.IsJSON() {
		return printer.Success(map[string]any{
			"status":              "ok",
			"repo_name":           repoName,
			"timbers_dir_created": timbersDirCreated,
			"hooks_installed":     hooksInstalled,
			"claude_installed":    agentInstalled, // backward compat key
			"already_initialized": false,
			"steps":               steps,
		})
	}

	printNextSteps(printer, styles)
	return nil
}

// stepSucceeded checks if a step with the given name completed with "ok" status.
func stepSucceeded(steps []initStepResult, name string) bool {
	for _, s := range steps {
		if s.Name == name && s.Status == "ok" {
			return true
		}
	}
	return false
}

// getRepoName returns the name of the current repository.
func getRepoName() string {
	root, err := git.RepoRoot()
	if err != nil {
		return "unknown"
	}
	return filepath.Base(root)
}
