// Package main provides the entry point for the timbers CLI.
package main

import (
	"github.com/spf13/cobra"

	"github.com/gorewood/timbers/internal/output"
)

// Onboard snippet templates - both targets use the same content currently.
const onboardSnippet = `## Development Ledger

This project uses **timbers** to capture development context as structured entries in ` + "`.timbers/`" + `.

**Before using timbers commands**, verify it is installed: ` + "`command -v timbers`" + `.
If not installed, warn the user once and continue working normally.
Install: ` + "`curl -fsSL https://raw.githubusercontent.com/gorewood/timbers/main/install.sh | bash`" + `

**Quick reference:**
- ` + "`timbers prime`" + ` - Get workflow context at session start
- ` + "`timbers pending`" + ` - Check for undocumented commits before session end
- ` + "`timbers log \"what\" --why \"why\" --how \"how\"`" + ` - Document committed work
- Entries are files in .timbers/ — sync via ` + "`git push`"

// newOnboardCmd creates the onboard command.
func newOnboardCmd() *cobra.Command {
	var formatFlag string
	var targetFlag string

	cmd := &cobra.Command{
		Use:   "onboard",
		Short: "Output a minimal snippet for CLAUDE.md or AGENTS.md",
		Long: `Output a minimal onboarding snippet for agent instruction files.

The snippet provides just enough context to point agents to 'timbers prime'
for full workflow details, keeping documentation DRY.

Examples:
  timbers onboard                    # Output markdown snippet for CLAUDE.md
  timbers onboard --target agents    # Output snippet for AGENTS.md
  timbers onboard --json             # Output snippet wrapped in JSON`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runOnboard(cmd, formatFlag, targetFlag)
		},
	}
	cmd.Flags().StringVar(&formatFlag, "format", "md", "Output format: md (default), json")
	cmd.Flags().StringVar(&targetFlag, "target", "claude", "Target file: claude (default), agents")
	return cmd
}

// runOnboard executes the onboard command.
func runOnboard(cmd *cobra.Command, formatFlag, targetFlag string) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), isJSONMode(cmd), useColor(cmd))

	// Validate target flag
	if targetFlag != "claude" && targetFlag != "agents" {
		err := output.NewUserError("invalid target: must be 'claude' or 'agents'")
		printer.Error(err)
		return err
	}

	// Validate format flag
	if formatFlag != "md" && formatFlag != "json" {
		err := output.NewUserError("invalid format: must be 'md' or 'json'")
		printer.Error(err)
		return err
	}

	// JSON output (either --json flag or --format json)
	if printer.IsJSON() || formatFlag == "json" {
		return printer.WriteJSON(map[string]string{
			"target":  targetFlag,
			"format":  formatFlag,
			"snippet": onboardSnippet,
		})
	}

	// Human-readable: just output the snippet directly
	printer.Println(onboardSnippet)
	return nil
}
