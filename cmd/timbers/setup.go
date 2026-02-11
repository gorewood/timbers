package main

import (
	"github.com/spf13/cobra"

	"github.com/gorewood/timbers/internal/output"
	"github.com/gorewood/timbers/internal/setup"
)

// integrationInfo describes an available integration.
type integrationInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Installed   bool   `json:"installed"`
	Scope       string `json:"scope,omitempty"`
	Location    string `json:"location,omitempty"`
}

// newSetupCmd creates the setup parent command with subcommands.
func newSetupCmd() *cobra.Command {
	var listFlag bool

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Configure editor and tool integrations",
		Long: `Configure timbers integrations with editors and development tools.

Subcommands:
  claude    Install Claude Code integration

Flags:
  --list    List available integrations and their status

Examples:
  timbers setup --list           # List available integrations
  timbers setup claude           # Install Claude integration for this project
  timbers setup claude --global  # Install globally (~/.claude/settings.json)
  timbers setup claude --check   # Check installation status
  timbers setup claude --remove  # Remove integration`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if listFlag {
				return runSetupList(cmd)
			}
			return cmd.Help()
		},
	}

	cmd.Flags().BoolVar(&listFlag, "list", false, "List available integrations and their status")

	cmd.AddCommand(newSetupClaudeCmd())
	return cmd
}

// newSetupClaudeCmd creates the claude subcommand for setup.
func newSetupClaudeCmd() *cobra.Command {
	var (
		globalFlag bool
		checkFlag  bool
		removeFlag bool
		dryRunFlag bool
	)

	cmd := &cobra.Command{
		Use:   "claude",
		Short: "Install Claude Code integration",
		Long: `Install timbers integration with Claude Code.

Adds hooks to multiple Claude Code events for comprehensive workflow integration:
  - SessionStart: injects development context at session start
  - PreCompact: re-injects context before context compression
  - Stop: warns about undocumented commits at session end
  - PostToolUse (Bash): nudges to run 'timbers log' after git commits

By default, installs to the current project's .claude/settings.local.json.
Use --global to install to ~/.claude/settings.json instead.

Re-running this command safely upgrades existing installations by adding
missing hooks without duplicating existing ones.

Examples:
  timbers setup claude           # Install for this project
  timbers setup claude --global  # Install globally
  timbers setup claude --check   # Check if installed
  timbers setup claude --remove  # Uninstall
  timbers setup claude --dry-run # Show what would be done`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runSetupClaude(cmd, !globalFlag, checkFlag, removeFlag, dryRunFlag)
		},
	}

	cmd.Flags().BoolVar(&globalFlag, "global", false, "Install globally to ~/.claude/settings.json")
	cmd.Flags().BoolVar(&checkFlag, "check", false, "Check installation status without changes")
	cmd.Flags().BoolVar(&removeFlag, "remove", false, "Remove the integration")
	cmd.Flags().BoolVar(&dryRunFlag, "dry-run", false, "Show what would be done without doing it")

	return cmd
}

// runSetupClaude executes the setup claude command.
func runSetupClaude(cmd *cobra.Command, project, check, remove, dryRun bool) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), isJSONMode(cmd), output.IsTTY(cmd.OutOrStdout()))

	hookPath, scope, err := setup.ResolveClaudeSettingsPath(project)
	if err != nil {
		printer.Error(err)
		return err
	}

	if check {
		return runSetupClaudeCheck(printer, hookPath, scope)
	}

	if remove {
		return runSetupClaudeRemove(printer, hookPath, scope, dryRun)
	}

	return runSetupClaudeInstall(printer, hookPath, scope, dryRun)
}

// runSetupList lists available integrations and their status.
func runSetupList(cmd *cobra.Command) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), isJSONMode(cmd), output.IsTTY(cmd.OutOrStdout()))

	globalHookPath, _, _ := setup.ResolveClaudeSettingsPath(false)
	projectHookPath, _, _ := setup.ResolveClaudeSettingsPath(true)

	globalInstalled := setup.IsTimbersSectionInstalled(globalHookPath)
	projectInstalled := setup.IsTimbersSectionInstalled(projectHookPath)

	var claudeScope, claudeLocation string
	var claudeInstalled bool
	if projectInstalled {
		claudeInstalled = true
		claudeScope = "project"
		claudeLocation = projectHookPath
	} else if globalInstalled {
		claudeInstalled = true
		claudeScope = "global"
		claudeLocation = globalHookPath
	}

	integrations := []integrationInfo{
		{
			Name:        "claude",
			Description: "Claude Code session context injection",
			Installed:   claudeInstalled,
			Scope:       claudeScope,
			Location:    claudeLocation,
		},
	}

	if printer.IsJSON() {
		return printer.Success(map[string]any{
			"integrations": integrations,
		})
	}

	printer.Section("Available Integrations")
	headers := []string{"Name", "Description", "Status", "Scope"}
	rows := make([][]string, 0, len(integrations))
	for _, integ := range integrations {
		status := "not installed"
		if integ.Installed {
			status = "installed"
		}
		scope := "-"
		if integ.Scope != "" {
			scope = integ.Scope
		}
		rows = append(rows, []string{integ.Name, integ.Description, status, scope})
	}
	printer.Table(headers, rows)
	return nil
}
