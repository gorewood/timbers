// Package main provides the entry point for the timbers CLI.
package main

import (
	"errors"

	"github.com/rbergman/timbers/internal/git"
	"github.com/rbergman/timbers/internal/ledger"
	"github.com/rbergman/timbers/internal/output"
	"github.com/spf13/cobra"
)

// newShowCmd creates the show command.
func newShowCmd() *cobra.Command {
	return newShowCmdInternal(nil)
}

// newShowCmdInternal creates the show command with optional storage injection.
// If storage is nil, a real storage is created when the command runs.
func newShowCmdInternal(storage *ledger.Storage) *cobra.Command {
	var lastFlag bool

	cmd := &cobra.Command{
		Use:   "show [<id>]",
		Short: "Display a single ledger entry",
		Long: `Display a single ledger entry by ID or show the most recent entry.

Examples:
  timbers show tb_2026-01-15T15:04:05Z_8f2c1a  # Show specific entry
  timbers show --last                          # Show most recent entry
  timbers show --last --json                   # Show as JSON`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShow(cmd, storage, args, lastFlag)
		},
	}

	cmd.Flags().BoolVar(&lastFlag, "last", false, "Show the most recent entry")

	return cmd
}

// runShow executes the show command.
func runShow(cmd *cobra.Command, storage *ledger.Storage, args []string, lastFlag bool) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), jsonFlag, output.IsTTY(cmd.OutOrStdout()))

	// Validate arguments
	if len(args) == 0 && !lastFlag {
		err := output.NewUserError("specify an entry ID or use --last")
		printer.Error(err)
		return err
	}
	if len(args) > 0 && lastFlag {
		err := output.NewUserError("cannot use both ID argument and --last flag")
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
	entry, err := getShowEntry(storage, args, lastFlag)
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
func getShowEntry(storage *ledger.Storage, args []string, lastFlag bool) (*ledger.Entry, error) {
	if lastFlag {
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
	printer.Println()
}

// outputShowSummary prints the what/why/how summary.
func outputShowSummary(printer *output.Printer, entry *ledger.Entry) {
	printer.Print("What: %s\n", entry.Summary.What)
	printer.Print("Why:  %s\n", entry.Summary.Why)
	printer.Print("How:  %s\n", entry.Summary.How)
	printer.Println()
}

// outputShowWorkset prints the workset information.
func outputShowWorkset(printer *output.Printer, entry *ledger.Entry) {
	printer.Print("Anchor: %s\n", shortSHA(entry.Workset.AnchorCommit))
	if len(entry.Workset.Commits) > 0 {
		printer.Print("Commits: %d", len(entry.Workset.Commits))
		if entry.Workset.Range != "" {
			printer.Print(" (%s)", entry.Workset.Range)
		}
		printer.Println()
	}
	if entry.Workset.Diffstat != nil {
		files := entry.Workset.Diffstat.Files
		suffix := "s"
		if files == 1 {
			suffix = ""
		}
		printer.Print("Changed: %d file%s, +%d/-%d lines\n",
			files, suffix, entry.Workset.Diffstat.Insertions, entry.Workset.Diffstat.Deletions)
	}
}

// outputShowTags prints the tags if present.
func outputShowTags(printer *output.Printer, entry *ledger.Entry) {
	if len(entry.Tags) == 0 {
		return
	}
	printer.Println()
	printer.Print("Tags: ")
	for i, tag := range entry.Tags {
		if i > 0 {
			printer.Print(", ")
		}
		printer.Print("%s", tag)
	}
	printer.Println()
}

// outputShowWorkItems prints work items if present.
func outputShowWorkItems(printer *output.Printer, entry *ledger.Entry) {
	if len(entry.WorkItems) == 0 {
		return
	}
	printer.Println()
	printer.Print("Work items: ")
	for i, wi := range entry.WorkItems {
		if i > 0 {
			printer.Print(", ")
		}
		printer.Print("%s:%s", wi.System, wi.ID)
	}
	printer.Println()
}

// outputShowTimestamps prints the created timestamp.
func outputShowTimestamps(printer *output.Printer, entry *ledger.Entry) {
	printer.Println()
	printer.Print("Created: %s\n", entry.CreatedAt.Format("2006-01-02 15:04:05 UTC"))
}

// shortSHA returns a shortened SHA (first 7 characters).
func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
