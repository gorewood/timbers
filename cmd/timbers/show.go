// Package main provides the entry point for the timbers CLI.
package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
	"github.com/gorewood/timbers/internal/output"
	"github.com/spf13/cobra"
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
	printer := output.NewPrinter(cmd.OutOrStdout(), jsonFlag, output.IsTTY(cmd.OutOrStdout()))

	// Validate arguments
	if len(args) == 0 && !latestFlag {
		err := output.NewUserError("specify an entry ID or use --latest")
		printer.Error(err)
		return err
	}
	if len(args) > 0 && latestFlag {
		err := output.NewUserError("cannot use both ID argument and --latest flag")
		printer.Error(err)
		return err
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

	// Get the entry
	entry, err := getShowEntry(storage, args, latestFlag)
	if err != nil {
		printer.Error(err)
		return err
	}

	// Output based on mode
	if jsonFlag {
		return outputShowJSON(printer, entry)
	}

	outputShowHuman(printer, entry)
	return nil
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

// outputShowHuman outputs the entry in human-readable format.
func outputShowHuman(printer *output.Printer, entry *ledger.Entry) {
	outputShowHeader(printer, entry)
	outputShowSummary(printer, entry)
	outputShowWorkset(printer, entry)
	outputShowTags(printer, entry)
	outputShowWorkItems(printer, entry)
	outputShowTimestamps(printer, entry)
}

// outputShowHeader prints the entry ID.
func outputShowHeader(printer *output.Printer, entry *ledger.Entry) {
	printer.Println(entry.ID)
}

// outputShowSummary prints the what/why/how summary.
func outputShowSummary(printer *output.Printer, entry *ledger.Entry) {
	printer.Section("Summary")
	printer.KeyValue("What", entry.Summary.What)
	printer.KeyValue("Why", entry.Summary.Why)
	printer.KeyValue("How", entry.Summary.How)
}

// outputShowWorkset prints the workset information.
func outputShowWorkset(printer *output.Printer, entry *ledger.Entry) {
	printer.Section("Workset")
	printer.KeyValue("Anchor", shortSHA(entry.Workset.AnchorCommit))
	if len(entry.Workset.Commits) > 0 {
		commitValue := strconv.Itoa(len(entry.Workset.Commits))
		if entry.Workset.Range != "" {
			commitValue = strconv.Itoa(len(entry.Workset.Commits)) + " (" + entry.Workset.Range + ")"
		}
		printer.KeyValue("Commits", commitValue)
	}
	if entry.Workset.Diffstat != nil {
		files := entry.Workset.Diffstat.Files
		suffix := "s"
		if files == 1 {
			suffix = ""
		}
		changedValue := fmt.Sprintf("%d file%s, +%d/-%d lines",
			files, suffix, entry.Workset.Diffstat.Insertions, entry.Workset.Diffstat.Deletions)
		printer.KeyValue("Changed", changedValue)
	}
}

// outputShowTags prints the tags if present.
func outputShowTags(printer *output.Printer, entry *ledger.Entry) {
	if len(entry.Tags) == 0 {
		return
	}
	printer.Println()
	printer.KeyValue("Tags", strings.Join(entry.Tags, ", "))
}

// outputShowWorkItems prints work items if present.
func outputShowWorkItems(printer *output.Printer, entry *ledger.Entry) {
	if len(entry.WorkItems) == 0 {
		return
	}
	items := make([]string, len(entry.WorkItems))
	for i, wi := range entry.WorkItems {
		items[i] = fmt.Sprintf("%s:%s", wi.System, wi.ID)
	}
	printer.Println()
	printer.KeyValue("Work Items", strings.Join(items, ", "))
}

// outputShowTimestamps prints the created timestamp.
func outputShowTimestamps(printer *output.Printer, entry *ledger.Entry) {
	printer.Println()
	printer.KeyValue("Created", entry.CreatedAt.Format("2006-01-02 15:04:05 UTC"))
}

// shortSHA returns a shortened SHA (first 7 characters).
func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
