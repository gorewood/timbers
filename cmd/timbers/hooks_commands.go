package main

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/output"
	"github.com/gorewood/timbers/internal/setup"
)

// newHooksInstallCmd creates the hooks install subcommand.
func newHooksInstallCmd() *cobra.Command {
	var chain bool
	var force bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install timbers git hooks",
		Long: `Install timbers git hooks to .git/hooks/.

The pre-commit hook warns about undocumented commits but does not block them.
Use --chain to preserve existing hooks (runs them first).
Use --force to overwrite existing hooks without backup.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runHooksInstall(cmd, chain, force, dryRun)
		},
	}

	cmd.Flags().BoolVar(&chain, "chain", false, "Preserve existing hooks, run them first")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing hooks without backup")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without doing it")

	return cmd
}

// runHooksInstall executes the hooks install command.
func runHooksInstall(cmd *cobra.Command, chain, force, dryRun bool) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), isJSONMode(cmd), useColor(cmd))

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

	preCommitPath := filepath.Join(hooksDir, "pre-commit")
	existingHook := setup.HookExists(preCommitPath)

	if dryRun {
		return handleInstallDryRun(printer, preCommitPath, existingHook, chain, force)
	}

	return performInstall(printer, preCommitPath, existingHook, chain, force)
}

// performInstall does the actual hook installation.
func performInstall(printer *output.Printer, hookPath string, existingHook, chain, force bool) error {
	if existingHook && !force {
		if !chain {
			err := output.NewConflictError("hook already exists; use --chain to preserve or --force to overwrite")
			printer.Error(err)
			return err
		}
		if err := setup.BackupExistingHook(hookPath); err != nil {
			printer.Error(err)
			return err
		}
	}

	hookContent := setup.GeneratePreCommitHook(chain && existingHook)
	// #nosec G306 -- hook needs execute permission
	if err := os.WriteFile(hookPath, []byte(hookContent), 0o755); err != nil {
		sysErr := output.NewSystemErrorWithCause("failed to write hook", err)
		printer.Error(sysErr)
		return sysErr
	}

	return outputInstallSuccess(printer, chain && existingHook)
}

// outputInstallSuccess outputs the success message for install.
func outputInstallSuccess(printer *output.Printer, chained bool) error {
	if printer.IsJSON() {
		return printer.Success(map[string]any{
			"status":  "ok",
			"hook":    "pre-commit",
			"chained": chained,
		})
	}

	msg := "Installed pre-commit hook"
	if chained {
		msg += " (existing hook backed up and chained)"
	}
	return printer.Success(map[string]any{"message": msg})
}

// handleInstallDryRun handles dry-run output for install.
func handleInstallDryRun(printer *output.Printer, hookPath string, existingHook, chain, force bool) error {
	if printer.IsJSON() {
		return printer.Success(map[string]any{
			"status":          "dry_run",
			"hook":            "pre-commit",
			"exists":          existingHook,
			"would_chain":     chain && existingHook,
			"would_overwrite": force && existingHook,
		})
	}

	printer.Section("Dry Run")
	printer.KeyValue("Hook", "pre-commit")
	printer.KeyValue("Path", hookPath)
	printer.KeyValue("Action", setup.DescribeInstallAction(existingHook, chain, force))

	return nil
}

// newHooksUninstallCmd creates the hooks uninstall subcommand.
func newHooksUninstallCmd() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove timbers git hooks",
		Long:  `Remove timbers git hooks and restore any backups.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runHooksUninstall(cmd, dryRun)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without doing it")

	return cmd
}

// runHooksUninstall executes the hooks uninstall command.
func runHooksUninstall(cmd *cobra.Command, dryRun bool) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), isJSONMode(cmd), useColor(cmd))

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

	preCommitPath := filepath.Join(hooksDir, "pre-commit")
	backupPath := preCommitPath + ".backup"
	status := setup.CheckHookStatus(preCommitPath)
	hasBackup := setup.HookExists(backupPath)

	if dryRun {
		return handleUninstallDryRun(printer, preCommitPath, status.Installed, hasBackup)
	}

	return performUninstall(printer, preCommitPath, backupPath, status.Installed, hasBackup)
}

// performUninstall does the actual hook uninstallation.
func performUninstall(printer *output.Printer, hookPath, backupPath string, installed, hasBackup bool) error {
	if !installed {
		return outputNoHookInstalled(printer)
	}

	if err := os.Remove(hookPath); err != nil {
		sysErr := output.NewSystemErrorWithCause("failed to remove hook", err)
		printer.Error(sysErr)
		return sysErr
	}

	restored := false
	if hasBackup {
		if err := os.Rename(backupPath, hookPath); err != nil {
			sysErr := output.NewSystemErrorWithCause("failed to restore backup", err)
			printer.Error(sysErr)
			return sysErr
		}
		restored = true
	}

	return outputUninstallSuccess(printer, restored)
}

// outputNoHookInstalled outputs the message when no hook is installed.
func outputNoHookInstalled(printer *output.Printer) error {
	if printer.IsJSON() {
		return printer.Success(map[string]any{
			"status":  "ok",
			"message": "no timbers hook installed",
		})
	}
	return printer.Success(map[string]any{"message": "No timbers hook installed"})
}

// outputUninstallSuccess outputs the success message for uninstall.
func outputUninstallSuccess(printer *output.Printer, restored bool) error {
	if printer.IsJSON() {
		return printer.Success(map[string]any{
			"status":   "ok",
			"hook":     "pre-commit",
			"restored": restored,
		})
	}

	msg := "Removed pre-commit hook"
	if restored {
		msg += " and restored original"
	}
	return printer.Success(map[string]any{"message": msg})
}

// handleUninstallDryRun handles dry-run output for uninstall.
func handleUninstallDryRun(printer *output.Printer, hookPath string, installed, hasBackup bool) error {
	if printer.IsJSON() {
		return printer.Success(map[string]any{
			"status":        "dry_run",
			"hook":          "pre-commit",
			"installed":     installed,
			"has_backup":    hasBackup,
			"would_restore": hasBackup,
		})
	}

	printer.Section("Dry Run")
	printer.KeyValue("Hook", "pre-commit")
	printer.KeyValue("Path", hookPath)
	printer.KeyValue("Action", setup.DescribeUninstallAction(installed, hasBackup))

	return nil
}
