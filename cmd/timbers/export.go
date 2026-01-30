// Package main provides the entry point for the timbers CLI.
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rbergman/timbers/internal/export"
	"github.com/rbergman/timbers/internal/git"
	"github.com/rbergman/timbers/internal/ledger"
	"github.com/rbergman/timbers/internal/output"
	"github.com/spf13/cobra"
)

// newExportCmd creates the export command.
func newExportCmd() *cobra.Command {
	return newExportCmdInternal(nil)
}

// newExportCmdInternal creates the export command with optional storage injection.
// If storage is nil, a real storage is created when the command runs.
func newExportCmdInternal(storage *ledger.Storage) *cobra.Command {
	var lastFlag string
	var sinceFlag string
	var untilFlag string
	var rangeFlag string
	var formatFlag string
	var outFlag string

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export entries to structured formats",
		Long: `Export entries to structured formats for pipelines.

Examples:
  timbers export --last 5 --json                    # Export last 5 as JSON array to stdout
  timbers export --since 24h                        # Export entries from last 24 hours
  timbers export --since 7d --format md             # Export last 7 days as markdown
  timbers export --since 2026-01-01 --until 2026-01-15  # Date range
  timbers export --last 5 --out ./exports/          # Export last 5 as JSON files to directory
  timbers export --range v1.0.0..v1.1.0 --json      # Export range as JSON
  timbers export --last 10 --format md --out ./notes/ # Export last 10 as markdown files`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runExport(cmd, storage, lastFlag, sinceFlag, untilFlag, rangeFlag, formatFlag, outFlag)
		},
	}

	cmd.Flags().StringVar(&lastFlag, "last", "", "Export last N entries")
	cmd.Flags().StringVar(&sinceFlag, "since", "", "Export entries since duration (24h, 7d) or date (2026-01-17)")
	cmd.Flags().StringVar(&untilFlag, "until", "", "Export entries until duration (24h, 7d) or date (2026-01-17)")
	cmd.Flags().StringVar(&rangeFlag, "range", "", "Export entries in commit range (A..B)")
	cmd.Flags().StringVar(&formatFlag, "format", "", "Output format: json or md (default: json for stdout, md for --out)")
	cmd.Flags().StringVar(&outFlag, "out", "", "Output directory (if omitted, writes to stdout)")

	return cmd
}

// runExport executes the export command.
func runExport(cmd *cobra.Command, storage *ledger.Storage, lastFlag, sinceFlag, untilFlag, rangeFlag, formatFlag, outFlag string) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), jsonFlag, output.IsTTY(cmd.OutOrStdout()))

	if err := validateExportFlags(printer, lastFlag, sinceFlag, untilFlag, rangeFlag); err != nil {
		return err
	}

	// Parse --since if provided
	var sinceCutoff time.Time
	if sinceFlag != "" {
		var parseErr error
		sinceCutoff, parseErr = parseSinceValue(sinceFlag)
		if parseErr != nil {
			err := output.NewUserError(parseErr.Error())
			printer.Error(err)
			return err
		}
	}

	// Parse --until if provided
	var untilCutoff time.Time
	if untilFlag != "" {
		var parseErr error
		untilCutoff, parseErr = parseUntilValue(untilFlag)
		if parseErr != nil {
			err := output.NewUserError(parseErr.Error())
			printer.Error(err)
			return err
		}
	}

	format := determineFormat(formatFlag, outFlag)
	if err := validateFormat(printer, format); err != nil {
		return err
	}

	storage, err := ensureStorage(printer, storage)
	if err != nil {
		return err
	}

	entries, err := getExportEntries(printer, storage, lastFlag, sinceCutoff, untilCutoff, rangeFlag)
	if err != nil {
		return err
	}

	return writeExportOutput(printer, entries, format, outFlag)
}

// validateExportFlags checks that required flags are provided.
func validateExportFlags(printer *output.Printer, lastFlag, sinceFlag, untilFlag, rangeFlag string) error {
	if lastFlag == "" && sinceFlag == "" && untilFlag == "" && rangeFlag == "" {
		err := output.NewUserError("specify --last N, --since <duration|date>, --until <duration|date>, or --range A..B to export entries")
		printer.Error(err)
		return err
	}
	return nil
}

// determineFormat returns the format to use based on flags.
func determineFormat(formatFlag, outFlag string) string {
	if formatFlag != "" {
		return formatFlag
	}
	// Default: json for stdout, md for --out
	if outFlag == "" {
		return "json"
	}
	return "md"
}

// validateFormat checks that the format is valid.
func validateFormat(printer *output.Printer, format string) error {
	if format != "json" && format != "md" {
		err := output.NewUserError("--format must be 'json' or 'md'")
		printer.Error(err)
		return err
	}
	return nil
}

// ensureStorage returns the storage, creating one if needed.
func ensureStorage(printer *output.Printer, storage *ledger.Storage) (*ledger.Storage, error) {
	if storage != nil {
		return storage, nil
	}

	if !git.IsRepo() {
		err := output.NewSystemError("not in a git repository")
		printer.Error(err)
		return nil, err
	}

	return ledger.NewStorage(nil), nil
}

// getExportEntries retrieves entries based on --last, --since, --until, or --range flags.
func getExportEntries(
	printer *output.Printer, storage *ledger.Storage, lastFlag string, sinceCutoff, untilCutoff time.Time, rangeFlag string,
) ([]*ledger.Entry, error) {
	// If --range is specified, use commit-based filtering
	if rangeFlag != "" {
		entries, err := getEntriesByRange(printer, storage, rangeFlag)
		if err != nil {
			return nil, err
		}
		// Apply --since filter if also specified
		if !sinceCutoff.IsZero() {
			entries = filterEntriesSince(entries, sinceCutoff)
		}
		// Apply --until filter if also specified
		if !untilCutoff.IsZero() {
			entries = filterEntriesUntil(entries, untilCutoff)
		}
		return entries, nil
	}

	// If --since or --until is specified, filter by time
	if !sinceCutoff.IsZero() || !untilCutoff.IsZero() {
		return getEntriesByTimeRange(printer, storage, sinceCutoff, untilCutoff, lastFlag)
	}

	// Otherwise use --last
	return getEntriesByLast(printer, storage, lastFlag)
}

// getEntriesByTimeRange retrieves entries within the time range, with optional limit.
func getEntriesByTimeRange(printer *output.Printer, storage *ledger.Storage, sinceCutoff, untilCutoff time.Time, lastFlag string) ([]*ledger.Entry, error) {
	entries, err := storage.ListEntries()
	if err != nil {
		printer.Error(err)
		return nil, err
	}

	// Filter by since cutoff
	if !sinceCutoff.IsZero() {
		entries = filterEntriesSince(entries, sinceCutoff)
	}

	// Filter by until cutoff
	if !untilCutoff.IsZero() {
		entries = filterEntriesUntil(entries, untilCutoff)
	}

	// Sort by created_at descending
	sortEntriesByCreatedAt(entries)

	// Apply --last limit if specified
	if lastFlag != "" {
		count, parseErr := strconv.Atoi(lastFlag)
		if parseErr == nil && count > 0 && len(entries) > count {
			entries = entries[:count]
		}
	}

	return entries, nil
}

// getEntriesByLast retrieves the last N entries.
func getEntriesByLast(printer *output.Printer, storage *ledger.Storage, lastFlag string) ([]*ledger.Entry, error) {
	count, parseErr := strconv.Atoi(lastFlag)
	if parseErr != nil || count <= 0 {
		err := output.NewUserError("--last must be a positive integer")
		printer.Error(err)
		return nil, err
	}

	entries, err := storage.GetLastNEntries(count)
	if err != nil {
		printer.Error(err)
		return nil, err
	}
	return entries, nil
}

// getEntriesByRange retrieves entries whose commits fall within the given range.
func getEntriesByRange(printer *output.Printer, storage *ledger.Storage, rangeFlag string) ([]*ledger.Entry, error) {
	parts := strings.Split(rangeFlag, "..")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		err := output.NewUserError("--range must be in format A..B")
		printer.Error(err)
		return nil, err
	}

	fromRef, toRef := parts[0], parts[1]

	allEntries, err := storage.ListEntries()
	if err != nil {
		printer.Error(err)
		return nil, err
	}

	commits, err := storage.LogRange(fromRef, toRef)
	if err != nil {
		printer.Error(err)
		return nil, err
	}

	commitSet := make(map[string]bool, len(commits))
	for _, commit := range commits {
		commitSet[commit.SHA] = true
	}

	return filterEntriesByCommits(allEntries, commitSet), nil
}

// filterEntriesByCommits returns entries that have at least one commit in the given set.
func filterEntriesByCommits(allEntries []*ledger.Entry, commitSet map[string]bool) []*ledger.Entry {
	var entries []*ledger.Entry
	for _, entry := range allEntries {
		if entryInCommitSet(entry, commitSet) {
			entries = append(entries, entry)
		}
	}
	return entries
}

// entryInCommitSet checks if any commit in entry's workset is in the set.
func entryInCommitSet(entry *ledger.Entry, commitSet map[string]bool) bool {
	for _, commitSHA := range entry.Workset.Commits {
		if commitSet[commitSHA] {
			return true
		}
	}
	return false
}

// writeExportOutput writes entries to stdout or directory based on flags.
func writeExportOutput(printer *output.Printer, entries []*ledger.Entry, format, outFlag string) error {
	if outFlag == "" {
		return writeToStdout(printer, entries, format)
	}
	return writeToDirectory(printer, entries, format, outFlag)
}

// writeToStdout writes entries to stdout in the specified format.
func writeToStdout(printer *output.Printer, entries []*ledger.Entry, format string) error {
	if format == "json" {
		return export.FormatJSON(printer, entries)
	}
	// Markdown to stdout: output each entry separated by ---
	for i, entry := range entries {
		if i > 0 {
			printer.Println("---")
		}
		printer.Print("%s", export.FormatMarkdown(entry))
	}
	return nil
}

// writeToDirectory writes entries to files in the specified directory.
func writeToDirectory(printer *output.Printer, entries []*ledger.Entry, format, outFlag string) error {
	if err := os.MkdirAll(outFlag, 0755); err != nil {
		sysErr := output.NewSystemError(fmt.Sprintf("failed to create output directory: %v", err))
		printer.Error(sysErr)
		return sysErr
	}

	var writeErr error
	if format == "json" {
		writeErr = export.WriteJSONFiles(entries, outFlag)
	} else {
		writeErr = export.WriteMarkdownFiles(entries, outFlag)
	}

	if writeErr != nil {
		printer.Error(writeErr)
		return writeErr
	}

	// Output confirmation - JSON or human-readable
	if jsonFlag {
		entryIDs := make([]string, len(entries))
		for i, e := range entries {
			entryIDs[i] = e.ID
		}
		return printer.Success(map[string]any{
			"status":     "ok",
			"count":      len(entries),
			"format":     format,
			"output_dir": outFlag,
			"entry_ids":  entryIDs,
		})
	}

	printer.Print("Exported %d entries to %s\n", len(entries), outFlag)
	return nil
}
