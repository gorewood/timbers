// Package main provides the entry point for the timbers CLI.
package main

import (
	"path/filepath"

	"github.com/rbergman/timbers/internal/git"
	"github.com/rbergman/timbers/internal/output"
	"github.com/spf13/cobra"
)

// statusResult holds the data for status output.
type statusResult struct {
	Repo            string `json:"repo"`
	Branch          string `json:"branch"`
	Head            string `json:"head"`
	NotesRef        string `json:"notes_ref"`
	NotesConfigured bool   `json:"notes_configured"`
	EntryCount      int    `json:"entry_count"`
}

// newStatusCmd creates the status command.
func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show repository and notes state",
		Long: `Show the current state of the repository and timbers notes configuration.

Displays repository info (name, branch, HEAD), notes ref status, whether
notes fetch is configured for the remote, and total entry count.

Examples:
  timbers status         # Show human-readable status
  timbers status --json  # Output status as JSON for scripting`,
		RunE: runStatus,
	}
	return cmd
}

// runStatus executes the status command.
func runStatus(cmd *cobra.Command, _ []string) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), jsonFlag, output.IsTTY(cmd.OutOrStdout()))

	// Check if we're in a git repo
	if !git.IsRepo() {
		err := output.NewSystemError("not in a git repository")
		printer.Error(err)
		return err
	}

	// Gather status information
	result, err := gatherStatus()
	if err != nil {
		printer.Error(err)
		return err
	}

	// Output based on mode
	if jsonFlag {
		return printer.Success(map[string]any{
			"repo":             result.Repo,
			"branch":           result.Branch,
			"head":             result.Head,
			"notes_ref":        result.NotesRef,
			"notes_configured": result.NotesConfigured,
			"entry_count":      result.EntryCount,
		})
	}

	// Human-readable output
	printHumanStatus(printer, result)
	return nil
}

// gatherStatus collects all status information.
func gatherStatus() (*statusResult, error) {
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

	// Count entries (commits with notes)
	commits, err := git.ListNotedCommits()
	if err != nil {
		return nil, err
	}

	return &statusResult{
		Repo:            repoName,
		Branch:          branch,
		Head:            head,
		NotesRef:        "refs/notes/timbers",
		NotesConfigured: notesConfigured,
		EntryCount:      len(commits),
	}, nil
}

// printHumanStatus outputs status in human-readable format.
func printHumanStatus(printer *output.Printer, status *statusResult) {
	printer.Println("Repository Status")
	printer.Println("─────────────────")
	printer.Print("  Repo:    %s\n", status.Repo)
	printer.Print("  Branch:  %s\n", status.Branch)
	printer.Print("  HEAD:    %s\n", status.Head[:min(12, len(status.Head))])
	printer.Println()
	printer.Println("Timbers Notes")
	printer.Println("─────────────")
	printer.Print("  Ref:        %s\n", status.NotesRef)
	printer.Print("  Configured: %s\n", formatBool(status.NotesConfigured))
	printer.Print("  Entries:    %d\n", status.EntryCount)
}

// formatBool returns a human-readable boolean string.
func formatBool(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
