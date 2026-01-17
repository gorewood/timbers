// Package main provides the entry point for the timbers CLI.
package main

import (
	"strconv"

	"github.com/rbergman/timbers/internal/git"
	"github.com/rbergman/timbers/internal/ledger"
	"github.com/rbergman/timbers/internal/output"
	"github.com/spf13/cobra"
)

// newQueryCmd creates the query command.
func newQueryCmd() *cobra.Command {
	return newQueryCmdInternal(nil)
}

// newQueryCmdInternal creates the query command with optional storage injection.
// If storage is nil, a real storage is created when the command runs.
func newQueryCmdInternal(storage *ledger.Storage) *cobra.Command {
	var lastFlag string
	var onelineFlag bool

	cmd := &cobra.Command{
		Use:   "query",
		Short: "Retrieve ledger entries with filters",
		Long: `Retrieve ledger entries with filters like --last N.

Examples:
  timbers query --last 5           # Show last 5 entries
  timbers query --last 10 --json   # Show last 10 as JSON
  timbers query --last 3 --oneline # Show last 3 in compact format`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runQuery(cmd, storage, lastFlag, onelineFlag)
		},
	}

	cmd.Flags().StringVar(&lastFlag, "last", "", "Retrieve last N entries")
	cmd.Flags().BoolVar(&onelineFlag, "oneline", false, "Show compact format: <id>  <what>")

	return cmd
}

// runQuery executes the query command.
func runQuery(cmd *cobra.Command, storage *ledger.Storage, lastFlag string, onelineFlag bool) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), jsonFlag, output.IsTTY(cmd.OutOrStdout()))

	// Validate --last flag is provided
	if lastFlag == "" {
		err := output.NewUserError("specify --last N to retrieve entries")
		printer.Error(err)
		return err
	}

	// Parse --last value
	count, err := strconv.Atoi(lastFlag)
	if err != nil || count <= 0 {
		parseErr := output.NewUserError("--last must be a positive integer")
		printer.Error(parseErr)
		return parseErr
	}

	// Check if we're in a git repo (only when using real git)
	if storage == nil && !git.IsRepo() {
		err := output.NewSystemError("not in a git repository")
		printer.Error(err)
		return err
	}

	// Create storage if not injected
	if storage == nil {
		storage = ledger.NewStorage(nil)
	}

	// Get last N entries
	entries, err := storage.GetLastNEntries(count)
	if err != nil {
		printer.Error(err)
		return err
	}

	// Output based on mode
	if jsonFlag {
		return outputQueryJSON(printer, entries)
	}

	if onelineFlag {
		outputQueryOneline(printer, entries)
		return nil
	}

	outputQueryHuman(printer, entries)
	return nil
}

// outputQueryJSON outputs the entries as JSON array.
func outputQueryJSON(printer *output.Printer, entries []*ledger.Entry) error {
	return printer.WriteJSON(entries)
}

// outputQueryOneline outputs entries in compact format: <id>  <what>
func outputQueryOneline(printer *output.Printer, entries []*ledger.Entry) {
	for _, entry := range entries {
		printer.Print("%s  %s\n", entry.ID, entry.Summary.What)
	}
}

// outputQueryHuman outputs entries in human-readable format.
func outputQueryHuman(printer *output.Printer, entries []*ledger.Entry) {
	if len(entries) == 0 {
		printer.Println("No entries found")
		return
	}

	for i, entry := range entries {
		if i > 0 {
			printer.Println()
		}
		outputQueryEntry(printer, entry)
	}
}

// outputQueryEntry outputs a single entry in human-readable format.
func outputQueryEntry(printer *output.Printer, entry *ledger.Entry) {
	outputShowHeader(printer, entry)
	outputShowSummary(printer, entry)
	outputShowWorkset(printer, entry)
	outputShowTags(printer, entry)
	outputShowWorkItems(printer, entry)
	outputShowTimestamps(printer, entry)
}
