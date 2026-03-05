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

Timbers installs hooks as delimited sections appended to existing hook files,
preserving any existing hooks. The pre-commit hook blocks commits when
undocumented commits exist (bypass with --no-verify).

Subcommands:
  install    Install timbers git hooks (pre-commit, post-commit, post-rewrite)
  uninstall  Remove timbers sections from all hook files
  list       Show status of hooks
  status     Show hook environment and integration details

Examples:
  timbers hooks list              # Show hook status
  timbers hooks status            # Show environment tier and integration details
  timbers hooks install           # Install hooks (appends to existing)
  timbers hooks install --force   # Install even in unknown hook environments
  timbers hooks uninstall         # Remove timbers sections from all hooks`,
	}

	cmd.AddCommand(newHooksListCmd())
	cmd.AddCommand(newHooksStatusCmd())
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
	printer := output.NewPrinter(cmd.OutOrStdout(), isJSONMode(cmd), useColor(cmd))

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
