// Package main provides the entry point for the timbers CLI.
package main

import (
	"bufio"
	"os"
	"strings"

	"github.com/rbergman/timbers/internal/git"
	"github.com/rbergman/timbers/internal/output"
	"github.com/spf13/cobra"
)

// uninstallResult holds the data for uninstall output.
type uninstallResult struct {
	BinaryPath      string   `json:"binary_path"`
	BinaryRemoved   bool     `json:"binary_removed"`
	NotesRefRemoved bool     `json:"notes_ref_removed"`
	ConfigsRemoved  []string `json:"configs_removed"`
	InRepo          bool     `json:"in_repo"`
}

// newUninstallCmd creates the uninstall command.
func newUninstallCmd() *cobra.Command {
	var dryRun bool
	var force bool

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove timbers from the system",
		Long: `Remove timbers binary, git notes, and git config from the system.

This command performs a clean removal of timbers:
  - Removes the timbers binary
  - Removes git notes refs (refs/notes/timbers) from the current repo
  - Removes git config for timbers notes fetch/push

Examples:
  timbers uninstall              # Uninstall with confirmation prompts
  timbers uninstall --dry-run    # Show what would be removed
  timbers uninstall --force      # Skip confirmation prompts
  timbers uninstall --json       # Output as JSON`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runUninstall(cmd, dryRun, force)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be removed without doing it")
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompts")

	return cmd
}

// runUninstall executes the uninstall command.
func runUninstall(cmd *cobra.Command, dryRun bool, force bool) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), jsonFlag, output.IsTTY(cmd.OutOrStdout()))

	// Gather what would be removed
	result, err := gatherUninstallInfo()
	if err != nil {
		printer.Error(err)
		return err
	}

	if dryRun {
		return outputDryRunUninstall(printer, result)
	}

	// Confirm unless --force
	if !force && !jsonFlag {
		if !confirmUninstall(cmd, result) {
			printer.Println("Uninstall cancelled.")
			return nil
		}
	}

	// Perform uninstall
	return executeUninstall(printer, result)
}

// gatherUninstallInfo collects information about what would be removed.
func gatherUninstallInfo() (*uninstallResult, error) {
	result := &uninstallResult{
		ConfigsRemoved: []string{},
	}

	// Get binary path
	execPath, err := os.Executable()
	if err != nil {
		return nil, output.NewSystemErrorWithCause("failed to determine binary location", err)
	}
	result.BinaryPath = execPath

	// Check if we're in a git repo
	result.InRepo = git.IsRepo()

	if result.InRepo {
		// Check for notes ref
		if git.NotesRefExists() {
			result.NotesRefRemoved = true
		}

		// Check for notes config on common remotes
		result.ConfigsRemoved = findNotesConfigs()
	}

	return result, nil
}

// findNotesConfigs finds all remotes with timbers notes configuration.
func findNotesConfigs() []string {
	var configs []string

	// Get list of remotes
	remotesOut, err := git.Run("remote")
	if err != nil {
		return configs
	}

	for remote := range strings.SplitSeq(strings.TrimSpace(remotesOut), "\n") {
		remote = strings.TrimSpace(remote)
		if remote == "" {
			continue
		}
		if git.NotesConfigured(remote) {
			configs = append(configs, remote)
		}
	}

	return configs
}

// outputDryRunUninstall outputs what would be removed in dry-run mode.
func outputDryRunUninstall(printer *output.Printer, result *uninstallResult) error {
	if jsonFlag {
		return printer.Success(map[string]any{
			"status":            "dry_run",
			"binary_path":       result.BinaryPath,
			"would_remove":      true,
			"notes_ref_exists":  result.NotesRefRemoved,
			"configs_to_remove": result.ConfigsRemoved,
			"in_repo":           result.InRepo,
		})
	}

	printer.Println("Dry run: Would perform the following actions:")
	printer.Println()
	printer.Print("  Remove binary: %s\n", result.BinaryPath)

	if result.InRepo {
		if result.NotesRefRemoved {
			printer.Println("  Remove notes ref: refs/notes/timbers")
		}
		if len(result.ConfigsRemoved) > 0 {
			for _, remote := range result.ConfigsRemoved {
				printer.Print("  Remove notes config for remote: %s\n", remote)
			}
		}
	} else {
		printer.Println("  (Not in a git repository - skipping notes cleanup)")
	}

	return nil
}

// confirmUninstall prompts the user for confirmation.
func confirmUninstall(cmd *cobra.Command, result *uninstallResult) bool {
	printer := output.NewPrinter(cmd.OutOrStdout(), false, output.IsTTY(cmd.OutOrStdout()))

	printer.Println("This will remove:")
	printer.Print("  Binary: %s\n", result.BinaryPath)

	if result.InRepo {
		if result.NotesRefRemoved {
			printer.Println("  Notes ref: refs/notes/timbers")
		}
		if len(result.ConfigsRemoved) > 0 {
			for _, remote := range result.ConfigsRemoved {
				printer.Print("  Notes config for remote: %s\n", remote)
			}
		}
	}

	printer.Println()
	printer.Print("Continue? [y/N]: ")

	reader := bufio.NewReader(cmd.InOrStdin())
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

// executeUninstall performs the actual uninstall operations.
func executeUninstall(printer *output.Printer, result *uninstallResult) error {
	var errors []string

	// Remove notes ref if in a repo
	if result.InRepo && result.NotesRefRemoved {
		if err := removeNotesRef(); err != nil {
			errors = append(errors, "notes ref: "+err.Error())
			result.NotesRefRemoved = false
		}
	}

	// Remove notes configs
	removedConfigs := []string{}
	for _, remote := range result.ConfigsRemoved {
		if err := removeNotesConfig(remote); err != nil {
			errors = append(errors, "config for "+remote+": "+err.Error())
		} else {
			removedConfigs = append(removedConfigs, remote)
		}
	}
	result.ConfigsRemoved = removedConfigs

	// Remove binary
	if err := os.Remove(result.BinaryPath); err != nil {
		errors = append(errors, "binary: "+err.Error())
		result.BinaryRemoved = false
	} else {
		result.BinaryRemoved = true
	}

	// Report results
	if len(errors) > 0 {
		errMsg := "uninstall completed with errors: " + strings.Join(errors, "; ")
		if jsonFlag {
			return printer.Success(map[string]any{
				"status":          "partial",
				"binary_removed":  result.BinaryRemoved,
				"notes_removed":   result.NotesRefRemoved,
				"configs_removed": result.ConfigsRemoved,
				"errors":          errors,
				"recovery_hint":   "Some components could not be removed. Check permissions and try again.",
			})
		}
		printer.Print("Warning: %s\n", errMsg)
		return nil
	}

	if jsonFlag {
		return printer.Success(map[string]any{
			"status":          "ok",
			"binary_removed":  result.BinaryRemoved,
			"notes_removed":   result.NotesRefRemoved,
			"configs_removed": result.ConfigsRemoved,
		})
	}

	printer.Println("Timbers uninstalled successfully.")
	return nil
}

// removeNotesRef removes the timbers notes ref from the repository.
func removeNotesRef() error {
	_, err := git.Run("update-ref", "-d", "refs/notes/timbers")
	return err
}

// removeNotesConfig removes the timbers notes fetch config for a remote.
func removeNotesConfig(remote string) error {
	// Get all fetch configs for this remote
	configKey := "remote." + remote + ".fetch"
	out, err := git.Run("config", "--get-all", configKey)
	if err != nil {
		// Config key doesn't exist or git error - either way, nothing to remove
		return nil //nolint:nilerr // expected when no config exists
	}

	// Find and remove the timbers notes refspec
	for line := range strings.SplitSeq(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "refs/notes/timbers") {
			// Use --unset with value pattern to remove only the matching entry
			if _, err := git.Run("config", "--unset", configKey, line); err != nil {
				return err
			}
		}
	}

	return nil
}
