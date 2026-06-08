// Package main provides the entry point for the timbers CLI.
package main

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
	"github.com/gorewood/timbers/internal/output"
)

// newShowCmd creates the show command.
func newShowCmd() *cobra.Command {
	return newShowCmdInternal(nil)
}

// newShowCmdInternal creates the show command with optional storage injection.
// If storage is nil, a real storage is created when the command runs.
func newShowCmdInternal(storage *ledger.Storage) *cobra.Command {
	var latestFlag bool

	cmd := &cobra.Command{
		Use:   "show [<id>]",
		Short: "Display a single ledger entry",
		Long: `Display a single ledger entry by ID or show the most recent entry.

Examples:
  timbers show tb_2026-01-15T15:04:05Z_8f2c1a  # Show specific entry
  timbers show --latest                        # Show most recent entry
  timbers show --latest --json                 # Show as JSON`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShow(cmd, storage, args, latestFlag)
		},
	}

	cmd.Flags().BoolVar(&latestFlag, "latest", false, "Show the most recent entry")

	return cmd
}

// runShow executes the show command.
func runShow(cmd *cobra.Command, storage *ledger.Storage, args []string, latestFlag bool) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), isJSONMode(cmd), useColor(cmd)).
		WithWidth(output.TerminalWidth(cmd.OutOrStdout(), 80))

	if err := validateShowArgs(args, latestFlag); err != nil {
		printer.Error(err)
		return err
	}

	storage, err := resolveShowStorage(storage)
	if err != nil {
		printer.Error(err)
		return err
	}

	// Get the entry
	entry, err := getShowEntry(storage, args, latestFlag)
	if err != nil {
		printer.Error(err)
		return err
	}

	// Output based on mode
	if printer.IsJSON() {
		return outputShowJSON(printer, entry)
	}

	outputShowHuman(printer, entry)
	return nil
}

// validateShowArgs checks that the arguments are valid.
func validateShowArgs(args []string, latestFlag bool) error {
	if len(args) == 0 && !latestFlag {
		return output.NewUserError("specify an entry ID or use --latest")
	}
	if len(args) > 0 && latestFlag {
		return output.NewUserError("cannot use both ID argument and --latest flag")
	}
	return nil
}

// resolveShowStorage returns the injected storage or creates a default one.
func resolveShowStorage(storage *ledger.Storage) (*ledger.Storage, error) {
	if storage != nil {
		return storage, nil
	}
	if !git.IsRepo() {
		return nil, output.NewSystemError("not in a git repository")
	}
	return ledger.NewDefaultStorage()
}

// getShowEntry retrieves the entry based on arguments.
func getShowEntry(storage *ledger.Storage, args []string, latestFlag bool) (*ledger.Entry, error) {
	if latestFlag {
		entry, err := storage.GetLatestEntry()
		if err != nil {
			if errors.Is(err, ledger.ErrNoEntries) {
				return nil, output.NewUserError("no entries found in ledger")
			}
			return nil, err
		}
		return entry, nil
	}

	return storage.GetEntryByID(args[0])
}

// outputShowJSON outputs the entry as JSON.
func outputShowJSON(printer *output.Printer, entry *ledger.Entry) error {
	return printer.WriteJSON(entry)
}

// outputShowHuman outputs the entry as an aligned panel: the ID is the title
// (the thing you copy), substance (what/why/how/notes/tags/work) leads, and
// workset bookkeeping trails after a separator. Rounded box at a TTY,
// borderless plain text when piped.
func outputShowHuman(printer *output.Printer, entry *ledger.Entry) {
	printer.FieldsBox(entry.ID, showFields(entry))
}

// shaExistsFunc is the function used to check if a SHA exists in the repo.
// Overridable in tests to avoid requiring a real git repo.
var shaExistsFunc = git.SHAExists

// anchorDisplay returns the display string for an anchor SHA.
// If the SHA does not exist in the current git history, it appends an annotation.
func anchorDisplay(sha string) string {
	display := shortSHA(sha)
	if sha != "" && !shaExistsFunc(sha) {
		display += " (not in current history)"
	}
	return display
}

// shortSHA returns a shortened SHA (first 7 characters).
func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
