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
	var removeBinary bool

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove timbers from the current repository",
		Long: `Remove timbers git notes and config from the current repository.

This command performs a clean removal of timbers from a repo:
  - Removes git notes refs (refs/notes/timbers)
  - Removes git config for timbers notes fetch/push

Use --binary to also remove the timbers binary itself.

Examples:
  timbers uninstall              # Remove from repo with confirmation
  timbers uninstall --dry-run    # Show what would be removed
  timbers uninstall --force      # Skip confirmation prompts
  timbers uninstall --binary     # Also remove the binary
  timbers uninstall --json       # Output as JSON`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runUninstall(cmd, dryRun, force, removeBinary)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be removed without doing it")
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompts")
	cmd.Flags().BoolVar(&removeBinary, "binary", false, "Also remove the timbers binary")

	return cmd
}

// runUninstall executes the uninstall command.
func runUninstall(cmd *cobra.Command, dryRun bool, force bool, removeBinary bool) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), jsonFlag, output.IsTTY(cmd.OutOrStdout()))

	// Gather what would be removed
	result, err := gatherUninstallInfo(removeBinary)
	if err != nil {
		printer.Error(err)
		return err
	}

	if dryRun {
		return outputDryRunUninstall(printer, result, removeBinary)
	}

	// Confirm unless --force
	if !force && !jsonFlag {
		if !confirmUninstall(cmd, result, removeBinary) {
			printer.Println("Uninstall cancelled.")
			return nil
		}
	}

	// Perform uninstall
	return executeUninstall(printer, result, removeBinary)
}

// gatherUninstallInfo collects information about what would be removed.
func gatherUninstallInfo(includeBinary bool) (*uninstallResult, error) {
	result := &uninstallResult{
		ConfigsRemoved: []string{},
	}

	// Get binary path if requested
	if includeBinary {
		execPath, err := os.Executable()
		if err != nil {
			return nil, output.NewSystemErrorWithCause("failed to determine binary location", err)
		}
		result.BinaryPath = execPath
	}

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
func outputDryRunUninstall(printer *output.Printer, result *uninstallResult, includeBinary bool) error {
	if jsonFlag {
		return outputDryRunJSON(printer, result, includeBinary)
	}
	outputDryRunHuman(printer, result, includeBinary)
	return nil
}

func outputDryRunJSON(printer *output.Printer, result *uninstallResult, includeBinary bool) error {
	data := map[string]any{
		"status":            "dry_run",
		"notes_ref_exists":  result.NotesRefRemoved,
		"configs_to_remove": result.ConfigsRemoved,
		"in_repo":           result.InRepo,
	}
	if includeBinary {
		data["binary_path"] = result.BinaryPath
	}
	return printer.Success(data)
}

func outputDryRunHuman(printer *output.Printer, result *uninstallResult, includeBinary bool) {
	printer.Println("Dry run: Would perform the following actions:")
	printer.Println()

	switch {
	case !result.InRepo:
		printer.Println("  (Not in a git repository - nothing to remove)")
	case !result.NotesRefRemoved && len(result.ConfigsRemoved) == 0:
		printer.Println("  (No timbers data found in this repository)")
	default:
		if result.NotesRefRemoved {
			printer.Println("  Remove notes ref: refs/notes/timbers")
		}
		for _, remote := range result.ConfigsRemoved {
			printer.Print("  Remove notes config for remote: %s\n", remote)
		}
	}

	if includeBinary {
		printer.Print("  Remove binary: %s\n", result.BinaryPath)
	}
}

// confirmUninstall prompts the user for confirmation.
func confirmUninstall(cmd *cobra.Command, result *uninstallResult, includeBinary bool) bool {
	printer := output.NewPrinter(cmd.OutOrStdout(), false, output.IsTTY(cmd.OutOrStdout()))

	printer.Println("This will remove:")

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

	if includeBinary {
		printer.Print("  Binary: %s\n", result.BinaryPath)
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
func executeUninstall(printer *output.Printer, result *uninstallResult, includeBinary bool) error {
	errors := performUninstallOperations(result, includeBinary)
	return reportUninstallResult(printer, result, includeBinary, errors)
}

func performUninstallOperations(result *uninstallResult, includeBinary bool) []string {
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

	// Remove binary if requested
	if includeBinary {
		if err := os.Remove(result.BinaryPath); err != nil {
			errors = append(errors, "binary: "+err.Error())
			result.BinaryRemoved = false
		} else {
			result.BinaryRemoved = true
		}
	}

	return errors
}

func reportUninstallResult(printer *output.Printer, result *uninstallResult, includeBinary bool, errors []string) error {
	if len(errors) > 0 {
		return reportPartialUninstall(printer, result, includeBinary, errors)
	}
	return reportSuccessfulUninstall(printer, result, includeBinary)
}

func reportPartialUninstall(printer *output.Printer, result *uninstallResult, includeBinary bool, errors []string) error {
	if jsonFlag {
		data := map[string]any{
			"status":          "partial",
			"notes_removed":   result.NotesRefRemoved,
			"configs_removed": result.ConfigsRemoved,
			"errors":          errors,
			"recovery_hint":   "Some components could not be removed. Check permissions and try again.",
		}
		if includeBinary {
			data["binary_removed"] = result.BinaryRemoved
		}
		return printer.Success(data)
	}
	printer.Print("Warning: uninstall completed with errors: %s\n", strings.Join(errors, "; "))
	return nil
}

func reportSuccessfulUninstall(printer *output.Printer, result *uninstallResult, includeBinary bool) error {
	if jsonFlag {
		data := map[string]any{
			"status":          "ok",
			"notes_removed":   result.NotesRefRemoved,
			"configs_removed": result.ConfigsRemoved,
		}
		if includeBinary {
			data["binary_removed"] = result.BinaryRemoved
		}
		return printer.Success(data)
	}

	if includeBinary {
		printer.Println("Timbers uninstalled successfully (including binary).")
	} else {
		printer.Println("Timbers removed from this repository.")
	}
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
