// Package main provides the entry point for the timbers CLI.
package main

import (
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
	"github.com/gorewood/timbers/internal/output"
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
	var untilFlag string
	var onelineFlag bool

	cmd := &cobra.Command{
		Use:   "query",
		Short: "Retrieve ledger entries with filters",
		Long: `Retrieve ledger entries with filters like --last N, --since, or --until.

Examples:
  timbers query --last 5                      # Show last 5 entries
  timbers query --since 24h                   # Show entries from last 24 hours
  timbers query --since 7d                    # Show entries from last 7 days
  timbers query --since 2026-01-15            # Show entries since date
  timbers query --since 2026-01-01 --until 2026-01-15  # Date range
  timbers query --last 10 --json              # Show last 10 as JSON
  timbers query --last 3 --oneline            # Show last 3 in compact format`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runQuery(cmd, storage, lastFlag, sinceFlag, untilFlag, onelineFlag)
		},
	}

	cmd.Flags().StringVar(&lastFlag, "last", "", "Retrieve last N entries")
	cmd.Flags().StringVar(&sinceFlag, "since", "", "Retrieve entries since duration (24h, 7d) or date (2026-01-17)")
	cmd.Flags().StringVar(&untilFlag, "until", "", "Retrieve entries until duration (24h, 7d) or date (2026-01-17)")
	cmd.Flags().BoolVar(&onelineFlag, "oneline", false, "Show compact format: <id>  <what>")

	return cmd
}

// queryParams holds parsed query parameters.
type queryParams struct {
	count       int
	sinceCutoff time.Time
	untilCutoff time.Time
}

// runQuery executes the query command.
func runQuery(cmd *cobra.Command, storage *ledger.Storage, lastFlag, sinceFlag, untilFlag string, onelineFlag bool) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), isJSONMode(cmd), output.IsTTY(cmd.OutOrStdout()))

	// Parse and validate flags
	params, err := parseQueryFlags(lastFlag, sinceFlag, untilFlag)
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
	entries, err := getQueryEntries(storage, params.count, params.sinceCutoff, params.untilCutoff)
	if err != nil {
		printer.Error(err)
		return err
	}

	// Output based on mode
	return outputQueryResults(printer, entries, onelineFlag)
}

// parseQueryFlags validates and parses the query flags.
func parseQueryFlags(lastFlag, sinceFlag, untilFlag string) (*queryParams, error) {
	if lastFlag == "" && sinceFlag == "" && untilFlag == "" {
		return nil, output.NewUserError("specify --last N, --since <duration|date>, or --until <duration|date> to retrieve entries")
	}

	params := &queryParams{}

	if err := parseQuerySinceFlag(sinceFlag, params); err != nil {
		return nil, err
	}
	if err := parseQueryUntilFlag(untilFlag, params); err != nil {
		return nil, err
	}
	if err := parseQueryLastFlag(lastFlag, params); err != nil {
		return nil, err
	}

	return params, nil
}

// parseQuerySinceFlag parses the --since flag into params.
func parseQuerySinceFlag(sinceFlag string, params *queryParams) error {
	if sinceFlag == "" {
		return nil
	}
	cutoff, err := parseSinceValue(sinceFlag)
	if err != nil {
		return output.NewUserError(err.Error())
	}
	params.sinceCutoff = cutoff
	return nil
}

// parseQueryUntilFlag parses the --until flag into params.
func parseQueryUntilFlag(untilFlag string, params *queryParams) error {
	if untilFlag == "" {
		return nil
	}
	cutoff, err := parseUntilValue(untilFlag)
	if err != nil {
		return output.NewUserError(err.Error())
	}
	params.untilCutoff = cutoff
	return nil
}

// parseQueryLastFlag parses the --last flag into params.
func parseQueryLastFlag(lastFlag string, params *queryParams) error {
	if lastFlag == "" {
		return nil
	}
	count, err := strconv.Atoi(lastFlag)
	if err != nil || count <= 0 {
		return output.NewUserError("--last must be a positive integer")
	}
	params.count = count
	return nil
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
	if printer.IsJSON() {
		return outputQueryJSON(printer, entries)
	}

	if onelineFlag {
		outputQueryOneline(printer, entries)
		return nil
	}

	outputQueryHuman(printer, entries)
	return nil
}

// getQueryEntries retrieves entries based on --last, --since, and --until filters.
func getQueryEntries(storage *ledger.Storage, count int, sinceCutoff, untilCutoff time.Time) ([]*ledger.Entry, error) {
	// If only --last is specified, use the optimized path
	if sinceCutoff.IsZero() && untilCutoff.IsZero() && count > 0 {
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

	// Filter by --until if specified
	if !untilCutoff.IsZero() {
		entries = filterEntriesUntil(entries, untilCutoff)
	}

	// Sort by created_at descending (most recent first)
	sortEntriesByCreatedAt(entries)

	// Apply --last limit if specified
	if count > 0 && len(entries) > count {
		entries = entries[:count]
	}

	return entries, nil
}

// outputQueryJSON outputs the entries as JSON array.
func outputQueryJSON(printer *output.Printer, entries []*ledger.Entry) error {
	return printer.WriteJSON(entries)
}

// outputQueryOneline outputs entries in compact table format: ID | Date | What
func outputQueryOneline(printer *output.Printer, entries []*ledger.Entry) {
	headers := []string{"ID", "Date", "What"}
	rows := make([][]string, 0, len(entries))

	for _, entry := range entries {
		date := entry.CreatedAt.Format("2006-01-02")
		rows = append(rows, []string{entry.ID, date, entry.Summary.What})
	}

	printer.Table(headers, rows)
}

// outputQueryHuman outputs entries in human-readable format.
func outputQueryHuman(printer *output.Printer, entries []*ledger.Entry) {
	if len(entries) == 0 {
		printer.Println("No entries found")
		return
	}

	for i, entry := range entries {
		if i > 0 {
			printer.Println("────────────────────────────────────────")
		}
		outputQueryEntry(printer, entry)
	}
}

// outputQueryEntry outputs a single entry in human-readable format.
func outputQueryEntry(printer *output.Printer, entry *ledger.Entry) {
	printer.Section(entry.ID)
	printer.KeyValue("What", entry.Summary.What)
	printer.KeyValue("Why", entry.Summary.Why)
	printer.KeyValue("How", entry.Summary.How)
	printer.KeyValue("Anchor", shortSHA(entry.Workset.AnchorCommit))
	printer.KeyValue("Created", entry.CreatedAt.Format("2006-01-02 15:04:05 UTC"))

	if len(entry.Tags) > 0 {
		printer.KeyValue("Tags", strings.Join(entry.Tags, ", "))
	}
}
