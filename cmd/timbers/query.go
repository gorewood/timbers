// Package main provides the entry point for the timbers CLI.
package main

import (
	"sort"
	"strconv"
	"time"

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
	var sinceFlag string
	var onelineFlag bool

	cmd := &cobra.Command{
		Use:   "query",
		Short: "Retrieve ledger entries with filters",
		Long: `Retrieve ledger entries with filters like --last N or --since.

Examples:
  timbers query --last 5           # Show last 5 entries
  timbers query --since 24h        # Show entries from last 24 hours
  timbers query --since 7d         # Show entries from last 7 days
  timbers query --since 2026-01-15 # Show entries since date
  timbers query --last 10 --json   # Show last 10 as JSON
  timbers query --last 3 --oneline # Show last 3 in compact format`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runQuery(cmd, storage, lastFlag, sinceFlag, onelineFlag)
		},
	}

	cmd.Flags().StringVar(&lastFlag, "last", "", "Retrieve last N entries")
	cmd.Flags().StringVar(&sinceFlag, "since", "", "Retrieve entries since duration (24h, 7d) or date (2026-01-17)")
	cmd.Flags().BoolVar(&onelineFlag, "oneline", false, "Show compact format: <id>  <what>")

	return cmd
}

// queryParams holds parsed query parameters.
type queryParams struct {
	count       int
	sinceCutoff time.Time
}

// runQuery executes the query command.
func runQuery(cmd *cobra.Command, storage *ledger.Storage, lastFlag, sinceFlag string, onelineFlag bool) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), jsonFlag, output.IsTTY(cmd.OutOrStdout()))

	// Parse and validate flags
	params, err := parseQueryFlags(lastFlag, sinceFlag)
	if err != nil {
		printer.Error(err)
		return err
	}

	// Initialize storage
	storage, err = initQueryStorage(storage, printer)
	if err != nil {
		return err
	}

	// Get entries based on filters
	entries, err := getQueryEntries(storage, params.count, params.sinceCutoff)
	if err != nil {
		printer.Error(err)
		return err
	}

	// Output based on mode
	return outputQueryResults(printer, entries, onelineFlag)
}

// parseQueryFlags validates and parses the query flags.
func parseQueryFlags(lastFlag, sinceFlag string) (*queryParams, error) {
	if lastFlag == "" && sinceFlag == "" {
		return nil, output.NewUserError("specify --last N or --since <duration|date> to retrieve entries")
	}

	params := &queryParams{}

	if sinceFlag != "" {
		cutoff, err := parseSinceValue(sinceFlag)
		if err != nil {
			return nil, output.NewUserError(err.Error())
		}
		params.sinceCutoff = cutoff
	}

	if lastFlag != "" {
		count, err := strconv.Atoi(lastFlag)
		if err != nil || count <= 0 {
			return nil, output.NewUserError("--last must be a positive integer")
		}
		params.count = count
	}

	return params, nil
}

// initQueryStorage initializes storage, checking for git repo if needed.
func initQueryStorage(storage *ledger.Storage, printer *output.Printer) (*ledger.Storage, error) {
	if storage == nil && !git.IsRepo() {
		err := output.NewSystemError("not in a git repository")
		printer.Error(err)
		return nil, err
	}

	if storage == nil {
		storage = ledger.NewStorage(nil)
	}

	return storage, nil
}

// outputQueryResults outputs entries based on the output mode.
func outputQueryResults(printer *output.Printer, entries []*ledger.Entry, onelineFlag bool) error {
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

// getQueryEntries retrieves entries based on --last and --since filters.
func getQueryEntries(storage *ledger.Storage, count int, sinceCutoff time.Time) ([]*ledger.Entry, error) {
	// If only --last is specified, use the optimized path
	if sinceCutoff.IsZero() && count > 0 {
		return storage.GetLastNEntries(count)
	}

	// Otherwise, get all entries and filter
	entries, err := storage.ListEntries()
	if err != nil {
		return nil, err
	}

	// Filter by --since if specified
	if !sinceCutoff.IsZero() {
		entries = filterEntriesSince(entries, sinceCutoff)
	}

	// Sort by created_at descending (most recent first)
	sortEntriesByCreatedAt(entries)

	// Apply --last limit if specified
	if count > 0 && len(entries) > count {
		entries = entries[:count]
	}

	return entries, nil
}

// filterEntriesSince filters entries to those created after the cutoff.
func filterEntriesSince(entries []*ledger.Entry, cutoff time.Time) []*ledger.Entry {
	var result []*ledger.Entry
	for _, entry := range entries {
		if entry.CreatedAt.After(cutoff) || entry.CreatedAt.Equal(cutoff) {
			result = append(result, entry)
		}
	}
	return result
}

// sortEntriesByCreatedAt sorts entries by created_at descending (most recent first).
func sortEntriesByCreatedAt(entries []*ledger.Entry) {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].CreatedAt.After(entries[j].CreatedAt)
	})
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
