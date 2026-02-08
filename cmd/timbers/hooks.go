package main

import (
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/output"
	"github.com/gorewood/timbers/internal/setup"
)

// hookStatus represents the status of a single hook.
type hookStatus = setup.HookStatus

// hooksListResult holds the data for hooks list output.
type hooksListResult struct {
	PreCommit hookStatus `json:"pre_commit"`
}

// newHooksCmd creates the hooks parent command with subcommands.
func newHooksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hooks",
		Short: "Manage git hooks for timbers",
		Long: `Manage git hooks that integrate timbers into your workflow.

Timbers can install a pre-commit hook that warns about undocumented commits.
The warning is non-blocking - commits still proceed, but you'll see a reminder
when you have pending work to document.

Subcommands:
  install    Install timbers git hooks
  uninstall  Remove timbers git hooks
  list       Show status of hooks

Examples:
  timbers hooks list              # Show hook status
  timbers hooks install           # Install pre-commit hook
  timbers hooks install --chain   # Install and preserve existing hook
  timbers hooks uninstall         # Remove hooks, restore backups`,
	}

	cmd.AddCommand(newHooksListCmd())
	cmd.AddCommand(newHooksInstallCmd())
	cmd.AddCommand(newHooksUninstallCmd())
	return cmd
}

// newHooksListCmd creates the hooks list subcommand.
func newHooksListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Show status of git hooks",
		Long:  `Show the installation status of each timbers hook.`,
		RunE:  runHooksList,
	}
}

// runHooksList executes the hooks list command.
func runHooksList(cmd *cobra.Command, _ []string) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), isJSONMode(cmd), output.IsTTY(cmd.OutOrStdout()))

	if !git.IsRepo() {
		err := output.NewSystemError("not in a git repository")
		printer.Error(err)
		return err
	}

	result, err := gatherHooksStatus()
	if err != nil {
		printer.Error(err)
		return err
	}

	if printer.IsJSON() {
		return printer.Success(map[string]any{
			"pre_commit": map[string]any{
				"installed": result.PreCommit.Installed,
				"chained":   result.PreCommit.Chained,
			},
		})
	}

	printHumanHooksList(printer, result)
	return nil
}

// gatherHooksStatus collects hook status information.
func gatherHooksStatus() (*hooksListResult, error) {
	hooksDir, err := setup.GetHooksDir()
	if err != nil {
		return nil, err
	}

	result := &hooksListResult{}
	preCommitPath := filepath.Join(hooksDir, "pre-commit")
	result.PreCommit = setup.CheckHookStatus(preCommitPath)

	return result, nil
}

// printHumanHooksList outputs hooks status in human-readable format.
func printHumanHooksList(printer *output.Printer, result *hooksListResult) {
	printer.Section("Git Hooks")

	statusStr := "not installed"
	if result.PreCommit.Installed {
		statusStr = "installed"
		if result.PreCommit.Chained {
			statusStr += " (chained)"
		}
	}
	printer.KeyValue("pre-commit", statusStr)
}
