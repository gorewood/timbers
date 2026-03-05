package main

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/output"
	"github.com/gorewood/timbers/internal/setup"
)

// newHooksUninstallCmd creates the hooks uninstall subcommand.
func newHooksUninstallCmd() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove timbers git hooks",
		Long: `Remove timbers sections from all hook files (pre-commit, post-commit,
post-rewrite).

If a hook file becomes empty after section removal, the file is deleted.
Legacy .backup files from old chain installs are restored if present.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runHooksUninstall(cmd, dryRun)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false,
		"Show what would be done without doing it")

	return cmd
}

// runHooksUninstall executes the hooks uninstall command.
func runHooksUninstall(cmd *cobra.Command, dryRun bool) error {
	printer := output.NewPrinter(
		cmd.OutOrStdout(), isJSONMode(cmd), useColor(cmd),
	)

	if !git.IsRepo() {
		err := output.NewSystemError("not in a git repository")
		printer.Error(err)
		return err
	}

	hooksDir, err := setup.GetHooksDir()
	if err != nil {
		printer.Error(err)
		return err
	}

	if dryRun {
		return handleUninstallDryRun(printer, hooksDir)
	}

	return performUninstall(printer, hooksDir)
}

// allHookTypes is the list of hook types timbers manages.
var allHookTypes = []string{"pre-commit", "post-commit", "post-rewrite"}

// performUninstall removes timbers sections from all hook types.
func performUninstall(printer *output.Printer, hooksDir string) error {
	removed := make(map[string]string)
	anyFound := false

	for _, hookType := range allHookTypes {
		hookPath := filepath.Join(hooksDir, hookType)
		if setup.HasTimbersSection(hookPath) {
			anyFound = true
			if err := setup.RemoveTimbersSection(hookPath); err != nil {
				sysErr := output.NewSystemErrorWithCause(
					"failed to remove "+hookType+" section", err,
				)
				printer.Error(sysErr)
				return sysErr
			}
			removed[hookType] = "removed"
		} else {
			removed[hookType] = "not installed"
		}
	}

	// Handle legacy .backup files for pre-commit.
	restoredBackup := restoreLegacyBackup(hooksDir, removed)

	if !anyFound {
		return outputNoHookInstalled(printer)
	}

	return outputUninstallSuccess(printer, removed, restoredBackup)
}

// restoreLegacyBackup restores a legacy .backup file if present.
func restoreLegacyBackup(hooksDir string, removed map[string]string) bool {
	backupPath := filepath.Join(hooksDir, "pre-commit.backup")
	if _, err := os.Stat(backupPath); err != nil {
		return false
	}

	preCommitPath := filepath.Join(hooksDir, "pre-commit")
	// Only restore if pre-commit was removed or doesn't exist.
	if _, statErr := os.Stat(preCommitPath); os.IsNotExist(statErr) {
		if renameErr := os.Rename(backupPath, preCommitPath); renameErr == nil {
			removed["pre-commit"] = "removed and restored original from backup"
			return true
		}
	}
	return false
}

// outputNoHookInstalled outputs the message when no hook is installed.
func outputNoHookInstalled(printer *output.Printer) error {
	if printer.IsJSON() {
		return printer.Success(map[string]any{
			"status":  "ok",
			"message": "no timbers hooks installed",
		})
	}
	return printer.Success(map[string]any{
		"message": "No timbers hooks installed",
	})
}

// outputUninstallSuccess outputs the success message for uninstall.
func outputUninstallSuccess(
	printer *output.Printer, removed map[string]string, restoredBackup bool,
) error {
	if printer.IsJSON() {
		return printer.Success(map[string]any{
			"status":          "ok",
			"pre_commit":      removed["pre-commit"],
			"post_commit":     removed["post-commit"],
			"post_rewrite":    removed["post-rewrite"],
			"restored_backup": restoredBackup,
		})
	}

	for _, hookType := range allHookTypes {
		action := removed[hookType]
		if action != "not installed" {
			printer.Println(hookType + ": " + action)
		}
	}
	return printer.Success(map[string]any{"message": "Timbers hooks removed"})
}

// handleUninstallDryRun handles dry-run output for uninstall.
func handleUninstallDryRun(printer *output.Printer, hooksDir string) error {
	actions := make(map[string]string)

	for _, hookType := range allHookTypes {
		hookPath := filepath.Join(hooksDir, hookType)
		if setup.HasTimbersSection(hookPath) {
			actions[hookType] = "would remove timbers section"
		} else {
			actions[hookType] = "not installed (no-op)"
		}
	}

	// Check for legacy backup.
	backupPath := filepath.Join(hooksDir, "pre-commit.backup")
	hasBackup := false
	if _, err := os.Stat(backupPath); err == nil {
		hasBackup = true
		actions["pre-commit"] += " and restore backup"
	}

	if printer.IsJSON() {
		return printer.Success(map[string]any{
			"status":       "dry_run",
			"hooks_dir":    hooksDir,
			"pre_commit":   actions["pre-commit"],
			"post_commit":  actions["post-commit"],
			"post_rewrite": actions["post-rewrite"],
			"has_backup":   hasBackup,
		})
	}

	printer.Section("Dry Run")
	printer.KeyValue("Hooks dir", hooksDir)
	printer.Println()
	for _, hookType := range allHookTypes {
		printer.KeyValue("  "+hookType, actions[hookType])
	}
	if hasBackup {
		printer.KeyValue("  Legacy backup", "would restore pre-commit.backup")
	}

	return nil
}
