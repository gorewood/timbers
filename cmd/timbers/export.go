// Package main provides the entry point for the timbers CLI.
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

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
	var rangeFlag string
	var formatFlag string
	var outFlag string

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export entries to structured formats",
		Long: `Export entries to structured formats for pipelines.

Examples:
  timbers export --last 5 --json                    # Export last 5 as JSON array to stdout
  timbers export --last 5 --out ./exports/          # Export last 5 as JSON files to directory
  timbers export --range v1.0.0..v1.1.0 --json      # Export range as JSON
  timbers export --last 10 --format md --out ./notes/ # Export last 10 as markdown files`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runExport(cmd, storage, lastFlag, rangeFlag, formatFlag, outFlag)
		},
	}

	cmd.Flags().StringVar(&lastFlag, "last", "", "Export last N entries")
	cmd.Flags().StringVar(&rangeFlag, "range", "", "Export entries in commit range (A..B)")
	cmd.Flags().StringVar(&formatFlag, "format", "", "Output format: json or md (default: json for stdout, md for --out)")
	cmd.Flags().StringVar(&outFlag, "out", "", "Output directory (if omitted, writes to stdout)")

	return cmd
}

// runExport executes the export command.
func runExport(cmd *cobra.Command, storage *ledger.Storage, lastFlag, rangeFlag, formatFlag, outFlag string) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), jsonFlag, output.IsTTY(cmd.OutOrStdout()))

	if err := validateExportFlags(printer, lastFlag, rangeFlag); err != nil {
		return err
	}

	format := determineFormat(formatFlag, outFlag)
	if err := validateFormat(printer, format); err != nil {
		return err
	}

	storage, err := ensureStorage(printer, storage)
	if err != nil {
		return err
	}

	entries, err := getExportEntries(printer, storage, lastFlag, rangeFlag)
	if err != nil {
		return err
	}

	return writeExportOutput(printer, entries, format, outFlag)
}

// validateExportFlags checks that required flags are provided.
func validateExportFlags(printer *output.Printer, lastFlag, rangeFlag string) error {
	if lastFlag == "" && rangeFlag == "" {
		err := output.NewUserError("specify --last N or --range A..B to export entries")
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

// getExportEntries retrieves entries based on --last or --range flags.
func getExportEntries(printer *output.Printer, storage *ledger.Storage, lastFlag, rangeFlag string) ([]*ledger.Entry, error) {
	if lastFlag != "" {
		return getEntriesByLast(printer, storage, lastFlag)
	}
	return getEntriesByRange(printer, storage, rangeFlag)
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

	if !jsonFlag {
		printer.Print("Exported %d entries to %s\n", len(entries), outFlag)
	}
	return nil
}
