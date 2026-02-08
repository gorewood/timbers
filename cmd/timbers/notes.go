// Package main provides the entry point for the timbers CLI.
package main

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/output"
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
		Long:  `Configure git to fetch timbers notes from a remote. Adds a fetch refspec for refs/notes/timbers to your git config.`,
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
	printer := output.NewPrinter(cmd.OutOrStdout(), isJSONMode(cmd), output.IsTTY(cmd.OutOrStdout()))

	if !git.IsRepo() {
		err := output.NewSystemError("not in a git repository")
		printer.Error(err)
		return err
	}

	wasConfigured := git.NotesConfigured(remote)

	if dryRun {
		if printer.IsJSON() {
			return printer.Success(map[string]any{
				"status":             "dry_run",
				"remote":             remote,
				"already_configured": wasConfigured,
				"would_configure":    !wasConfigured,
			})
		}
		printer.Section("Dry Run")
		printer.KeyValue("Remote", remote)
		if wasConfigured {
			printer.KeyValue("Status", "already configured (no changes needed)")
		} else {
			printer.KeyValue("Action", "would configure notes fetch")
		}
		return nil
	}

	if err := git.ConfigureNotesFetch(remote); err != nil {
		sysErr := output.NewSystemErrorWithCause("failed to configure notes fetch", err)
		printer.Error(sysErr)
		return sysErr
	}

	if printer.IsJSON() {
		return printer.Success(map[string]any{
			"status":     "ok",
			"remote":     remote,
			"configured": true,
		})
	}

	if wasConfigured {
		return printer.Success(map[string]any{
			"message": "Notes fetch already configured for remote '" + remote + "'",
		})
	}
	return printer.Success(map[string]any{
		"message": "Configured notes fetch for remote '" + remote + "'",
	})
}

// newNotesPushCmd creates the notes push subcommand.
func newNotesPushCmd() *cobra.Command {
	var remote string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "push",
		Short: "Push notes to a remote",
		Long:  `Push timbers notes (refs/notes/timbers) to a remote, making ledger entries available to collaborators.`,
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
	printer := output.NewPrinter(cmd.OutOrStdout(), isJSONMode(cmd), output.IsTTY(cmd.OutOrStdout()))

	if !git.IsRepo() {
		err := output.NewSystemError("not in a git repository")
		printer.Error(err)
		return err
	}

	if dryRun {
		commits, _ := git.ListNotedCommits()
		entryCount := len(commits)

		if printer.IsJSON() {
			return printer.Success(map[string]any{
				"status":      "dry_run",
				"remote":      remote,
				"entry_count": entryCount,
			})
		}
		printer.Section("Dry Run")
		printer.KeyValue("Remote", remote)
		printer.KeyValue("Entries", strconv.Itoa(entryCount))
		printer.KeyValue("Action", "would push notes")
		return nil
	}

	if err := git.PushNotes(remote); err != nil {
		sysErr := output.NewSystemErrorWithCause("failed to push notes", err)
		printer.Error(sysErr)
		return sysErr
	}

	if printer.IsJSON() {
		return printer.Success(map[string]any{
			"status": "ok",
			"remote": remote,
		})
	}

	return printer.Success(map[string]any{
		"message": "Pushed notes to remote '" + remote + "'",
	})
}

// newNotesFetchCmd creates the notes fetch subcommand.
func newNotesFetchCmd() *cobra.Command {
	var remote string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "fetch",
		Short: "Fetch notes from a remote",
		Long:  `Fetch timbers notes (refs/notes/timbers) from a remote, pulling in ledger entries from collaborators.`,
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
	printer := output.NewPrinter(cmd.OutOrStdout(), isJSONMode(cmd), output.IsTTY(cmd.OutOrStdout()))

	if !git.IsRepo() {
		err := output.NewSystemError("not in a git repository")
		printer.Error(err)
		return err
	}

	if dryRun {
		configured := git.NotesConfigured(remote)
		if printer.IsJSON() {
			return printer.Success(map[string]any{
				"status":     "dry_run",
				"remote":     remote,
				"configured": configured,
			})
		}
		printer.Section("Dry Run")
		printer.KeyValue("Remote", remote)
		printer.KeyValue("Configured", formatBool(configured))
		if configured {
			printer.KeyValue("Action", "would fetch notes")
		} else {
			printer.KeyValue("Action", "would fetch notes (not configured; run 'timbers notes init' first)")
		}
		return nil
	}

	if err := git.FetchNotes(remote); err != nil {
		sysErr := output.NewSystemErrorWithCause("failed to fetch notes", err)
		printer.Error(sysErr)
		return sysErr
	}

	if printer.IsJSON() {
		return printer.Success(map[string]any{
			"status": "ok",
			"remote": remote,
		})
	}

	return printer.Success(map[string]any{
		"message": "Fetched notes from remote '" + remote + "'",
	})
}

// newNotesStatusCmd creates the notes status subcommand.
func newNotesStatusCmd() *cobra.Command {
	var remote string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show notes sync state",
		Long:  `Show timbers notes sync state: ref existence, fetch configuration, and entry count.`,
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
	printer := output.NewPrinter(cmd.OutOrStdout(), isJSONMode(cmd), output.IsTTY(cmd.OutOrStdout()))

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

	if printer.IsJSON() {
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
	printer.Section("Notes Sync Status")
	printer.KeyValue("Remote", status.Remote)
	printer.KeyValue("Ref exists", formatBool(status.RefExists))
	printer.KeyValue("Configured", formatBool(status.Configured))
	printer.KeyValue("Entries", strconv.Itoa(status.EntryCount))
}
