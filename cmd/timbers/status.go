// Package main provides the entry point for the timbers CLI.
package main

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
	"github.com/gorewood/timbers/internal/output"
)

// statusResult holds the data for status output.
type statusResult struct {
	Repo            string `json:"repo"`
	Branch          string `json:"branch"`
	Head            string `json:"head"`
	NotesRef        string `json:"notes_ref"`
	NotesConfigured bool   `json:"notes_configured"`
	EntryCount      int    `json:"entry_count"`
	NotesTotal      int    `json:"notes_total,omitempty"`
	NotesSkipped    int    `json:"notes_skipped,omitempty"`
	NotTimbers      int    `json:"not_timbers,omitempty"`
	ParseErrors     int    `json:"parse_errors,omitempty"`
}

var verboseFlag bool

// newStatusCmd creates the status command.
func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show repository and notes state",
		Long: `Show the current state of the repository and timbers notes configuration.

Displays repository info (name, branch, HEAD), notes ref status, whether
notes fetch is configured for the remote, and total entry count.

Examples:
  timbers status            # Show human-readable status
  timbers status --verbose  # Show detailed notes statistics
  timbers status --json     # Output status as JSON for scripting`,
		RunE: runStatus,
	}
	cmd.Flags().BoolVarP(&verboseFlag, "verbose", "v", false, "Show detailed notes statistics")
	return cmd
}

// runStatus executes the status command.
func runStatus(cmd *cobra.Command, _ []string) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), isJSONMode(cmd), output.IsTTY(cmd.OutOrStdout()))

	// Check if we're in a git repo
	if !git.IsRepo() {
		err := output.NewSystemError("not in a git repository")
		printer.Error(err)
		return err
	}

	// Gather status information
	result, err := gatherStatus(verboseFlag)
	if err != nil {
		printer.Error(err)
		return err
	}

	// Output based on mode
	if printer.IsJSON() {
		data := map[string]any{
			"repo":             result.Repo,
			"branch":           result.Branch,
			"head":             result.Head,
			"notes_ref":        result.NotesRef,
			"notes_configured": result.NotesConfigured,
			"entry_count":      result.EntryCount,
		}
		// Add verbose stats if present
		if verboseFlag {
			data["notes_total"] = result.NotesTotal
			data["notes_skipped"] = result.NotesSkipped
			data["not_timbers"] = result.NotTimbers
			data["parse_errors"] = result.ParseErrors
		}
		// Add suggested commands based on state
		var suggestions []string
		if !result.NotesConfigured {
			suggestions = append(suggestions, "timbers notes init")
		}
		suggestions = append(suggestions, "timbers pending")
		data["suggested_commands"] = suggestions
		return printer.Success(data)
	}

	// Human-readable output
	printHumanStatus(printer, result, verboseFlag)
	return nil
}

// gatherStatus collects all status information.
func gatherStatus(verbose bool) (*statusResult, error) {
	// Get repo root and extract name
	root, err := git.RepoRoot()
	if err != nil {
		return nil, err
	}
	repoName := filepath.Base(root)

	// Get current branch
	branch, err := git.CurrentBranch()
	if err != nil {
		return nil, err
	}

	// Get HEAD commit
	head, err := git.HEAD()
	if err != nil {
		return nil, err
	}

	// Check notes configuration
	notesConfigured := git.NotesConfigured("origin")

	result := &statusResult{
		Repo:            repoName,
		Branch:          branch,
		Head:            head,
		NotesRef:        "refs/notes/timbers",
		NotesConfigured: notesConfigured,
	}

	// Get entry count with stats if verbose
	if verbose {
		store := ledger.NewStorage(nil)
		entries, stats, statsErr := store.ListEntriesWithStats()
		if statsErr != nil {
			return nil, statsErr
		}
		result.EntryCount = len(entries)
		result.NotesTotal = stats.Total
		result.NotesSkipped = stats.Skipped
		result.NotTimbers = stats.NotTimbers
		result.ParseErrors = stats.ParseErrors
	} else {
		// Simple count (commits with notes)
		commits, listErr := git.ListNotedCommits()
		if listErr != nil {
			return nil, listErr
		}
		result.EntryCount = len(commits)
	}

	return result, nil
}

// printHumanStatus outputs status in human-readable format.
func printHumanStatus(printer *output.Printer, status *statusResult, verbose bool) {
	printer.Section("Repository")
	printer.KeyValue("Repo", status.Repo)
	printer.KeyValue("Branch", status.Branch)
	printer.KeyValue("HEAD", status.Head[:min(12, len(status.Head))])

	printer.Section("Timbers Notes")
	printer.KeyValue("Ref", status.NotesRef)
	printer.KeyValue("Configured", formatBool(status.NotesConfigured))

	if verbose {
		printer.KeyValue("Notes Total", strconv.Itoa(status.NotesTotal))
		printer.KeyValue("Entries", strconv.Itoa(status.EntryCount))
		if status.NotesSkipped > 0 {
			skippedStr := fmt.Sprintf("%d (%d not timbers, %d parse error)",
				status.NotesSkipped, status.NotTimbers, status.ParseErrors)
			printer.KeyValue("Skipped", skippedStr)
		}
	} else {
		printer.KeyValue("Entries", strconv.Itoa(status.EntryCount))
	}
}

// formatBool returns a human-readable boolean string.
func formatBool(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
