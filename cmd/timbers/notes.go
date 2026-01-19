// Package main provides the entry point for the timbers CLI.
package main

import (
	"github.com/rbergman/timbers/internal/git"
	"github.com/rbergman/timbers/internal/output"
	"github.com/spf13/cobra"
)

// newNotesCmd creates the notes parent command with subcommands.
func newNotesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "notes",
		Short: "Manage timbers notes sync",
		Long: `Manage timbers notes synchronization with remote repositories.

Timbers stores ledger entries in Git notes (refs/notes/timbers). These notes
must be explicitly configured to sync with remotes.

Subcommands:
  init    Configure notes fetch for a remote
  push    Push notes to a remote
  fetch   Fetch notes from a remote
  status  Show notes sync state

Examples:
  timbers notes init             # Configure notes fetch for origin
  timbers notes push             # Push notes to origin
  timbers notes fetch            # Fetch notes from origin
  timbers notes status           # Show sync state`,
	}

	cmd.AddCommand(newNotesInitCmd())
	cmd.AddCommand(newNotesPushCmd())
	cmd.AddCommand(newNotesFetchCmd())
	cmd.AddCommand(newNotesStatusCmd())
	return cmd
}

// newNotesInitCmd creates the notes init subcommand.
func newNotesInitCmd() *cobra.Command {
	var remote string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Configure notes fetch for a remote",
		Long: `Configure git to fetch timbers notes from a remote repository.

This adds a fetch refspec for refs/notes/timbers to your git config,
enabling 'git fetch' to pull notes automatically.

Examples:
  timbers notes init                   # Configure for origin
  timbers notes init --remote upstream # Configure for upstream
  timbers notes init --dry-run         # Show what would be configured`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runNotesInit(cmd, remote, dryRun)
		},
	}

	cmd.Flags().StringVar(&remote, "remote", "origin", "Remote name to configure")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without doing it")

	return cmd
}

// runNotesInit executes the notes init command.
func runNotesInit(cmd *cobra.Command, remote string, dryRun bool) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), jsonFlag, output.IsTTY(cmd.OutOrStdout()))

	if !git.IsRepo() {
		err := output.NewSystemError("not in a git repository")
		printer.Error(err)
		return err
	}

	wasConfigured := git.NotesConfigured(remote)

	if dryRun {
		if jsonFlag {
			return printer.Success(map[string]any{
				"status":             "dry_run",
				"remote":             remote,
				"already_configured": wasConfigured,
				"would_configure":    !wasConfigured,
			})
		}
		if wasConfigured {
			printer.Print("Dry run: Notes fetch already configured for remote '%s' (no changes needed)\n", remote)
		} else {
			printer.Print("Dry run: Would configure notes fetch for remote '%s'\n", remote)
		}
		return nil
	}

	if err := git.ConfigureNotesFetch(remote); err != nil {
		sysErr := output.NewSystemErrorWithCause("failed to configure notes fetch", err)
		printer.Error(sysErr)
		return sysErr
	}

	if jsonFlag {
		return printer.Success(map[string]any{
			"status":     "ok",
			"remote":     remote,
			"configured": true,
		})
	}

	if wasConfigured {
		printer.Print("Notes fetch already configured for remote '%s'\n", remote)
	} else {
		printer.Print("Configured notes fetch for remote '%s'\n", remote)
	}
	return nil
}

// newNotesPushCmd creates the notes push subcommand.
func newNotesPushCmd() *cobra.Command {
	var remote string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "push",
		Short: "Push notes to a remote",
		Long: `Push timbers notes to a remote repository.

Pushes refs/notes/timbers to the specified remote, making your ledger
entries available to collaborators.

Examples:
  timbers notes push                   # Push to origin
  timbers notes push --remote upstream # Push to upstream
  timbers notes push --dry-run         # Show what would be pushed`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runNotesPush(cmd, remote, dryRun)
		},
	}

	cmd.Flags().StringVar(&remote, "remote", "origin", "Remote name to push to")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without doing it")

	return cmd
}

// runNotesPush executes the notes push command.
func runNotesPush(cmd *cobra.Command, remote string, dryRun bool) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), jsonFlag, output.IsTTY(cmd.OutOrStdout()))

	if !git.IsRepo() {
		err := output.NewSystemError("not in a git repository")
		printer.Error(err)
		return err
	}

	if dryRun {
		commits, _ := git.ListNotedCommits()
		entryCount := len(commits)

		if jsonFlag {
			return printer.Success(map[string]any{
				"status":      "dry_run",
				"remote":      remote,
				"entry_count": entryCount,
			})
		}
		printer.Print("Dry run: Would push %d entries to remote '%s'\n", entryCount, remote)
		return nil
	}

	if err := git.PushNotes(remote); err != nil {
		sysErr := output.NewSystemErrorWithCause("failed to push notes", err)
		printer.Error(sysErr)
		return sysErr
	}

	if jsonFlag {
		return printer.Success(map[string]any{
			"status": "ok",
			"remote": remote,
		})
	}

	printer.Print("Pushed notes to remote '%s'\n", remote)
	return nil
}

// newNotesFetchCmd creates the notes fetch subcommand.
func newNotesFetchCmd() *cobra.Command {
	var remote string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "fetch",
		Short: "Fetch notes from a remote",
		Long: `Fetch timbers notes from a remote repository.

Fetches refs/notes/timbers from the specified remote, pulling in ledger
entries created by collaborators.

Examples:
  timbers notes fetch                   # Fetch from origin
  timbers notes fetch --remote upstream # Fetch from upstream
  timbers notes fetch --dry-run         # Show what would be fetched`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runNotesFetch(cmd, remote, dryRun)
		},
	}

	cmd.Flags().StringVar(&remote, "remote", "origin", "Remote name to fetch from")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without doing it")

	return cmd
}

// runNotesFetch executes the notes fetch command.
func runNotesFetch(cmd *cobra.Command, remote string, dryRun bool) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), jsonFlag, output.IsTTY(cmd.OutOrStdout()))

	if !git.IsRepo() {
		err := output.NewSystemError("not in a git repository")
		printer.Error(err)
		return err
	}

	if dryRun {
		configured := git.NotesConfigured(remote)
		if jsonFlag {
			return printer.Success(map[string]any{
				"status":     "dry_run",
				"remote":     remote,
				"configured": configured,
			})
		}
		if configured {
			printer.Print("Dry run: Would fetch notes from remote '%s'\n", remote)
		} else {
			printer.Print("Dry run: Would fetch notes from remote '%s' (note: fetch not configured; run 'timbers notes init' first)\n", remote)
		}
		return nil
	}

	if err := git.FetchNotes(remote); err != nil {
		sysErr := output.NewSystemErrorWithCause("failed to fetch notes", err)
		printer.Error(sysErr)
		return sysErr
	}

	if jsonFlag {
		return printer.Success(map[string]any{
			"status": "ok",
			"remote": remote,
		})
	}

	printer.Print("Fetched notes from remote '%s'\n", remote)
	return nil
}

// newNotesStatusCmd creates the notes status subcommand.
func newNotesStatusCmd() *cobra.Command {
	var remote string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show notes sync state",
		Long: `Show the current state of timbers notes synchronization.

Displays whether the notes ref exists, if fetch is configured for the
remote, and the current entry count.

Examples:
  timbers notes status                   # Check status for origin
  timbers notes status --remote upstream # Check status for upstream
  timbers notes status --json            # Output as JSON`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runNotesStatus(cmd, remote)
		},
	}

	cmd.Flags().StringVar(&remote, "remote", "origin", "Remote name to check")

	return cmd
}

// notesStatusResult holds the data for notes status output.
type notesStatusResult struct {
	RefExists  bool   `json:"ref_exists"`
	Configured bool   `json:"configured"`
	EntryCount int    `json:"entry_count"`
	Remote     string `json:"remote"`
}

// runNotesStatus executes the notes status command.
func runNotesStatus(cmd *cobra.Command, remote string) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), jsonFlag, output.IsTTY(cmd.OutOrStdout()))

	if !git.IsRepo() {
		err := output.NewSystemError("not in a git repository")
		printer.Error(err)
		return err
	}

	result, err := gatherNotesStatus(remote)
	if err != nil {
		printer.Error(err)
		return err
	}

	if jsonFlag {
		return printer.Success(map[string]any{
			"ref_exists":  result.RefExists,
			"configured":  result.Configured,
			"entry_count": result.EntryCount,
			"remote":      result.Remote,
		})
	}

	printHumanNotesStatus(printer, result)
	return nil
}

// gatherNotesStatus collects notes status information.
func gatherNotesStatus(remote string) (*notesStatusResult, error) {
	refExists := git.NotesRefExists()
	configured := git.NotesConfigured(remote)

	commits, err := git.ListNotedCommits()
	if err != nil {
		return nil, output.NewSystemErrorWithCause("failed to list notes", err)
	}

	return &notesStatusResult{
		RefExists:  refExists,
		Configured: configured,
		EntryCount: len(commits),
		Remote:     remote,
	}, nil
}

// printHumanNotesStatus outputs notes status in human-readable format.
func printHumanNotesStatus(printer *output.Printer, status *notesStatusResult) {
	printer.Println("Notes Sync Status")
	printer.Println("-----------------")
	printer.Print("  Remote:     %s\n", status.Remote)
	printer.Print("  Ref exists: %s\n", formatBool(status.RefExists))
	printer.Print("  Configured: %s\n", formatBool(status.Configured))
	printer.Print("  Entries:    %d\n", status.EntryCount)
}
