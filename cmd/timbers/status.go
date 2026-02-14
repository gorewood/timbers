// Package main provides the entry point for the timbers CLI.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
	"github.com/gorewood/timbers/internal/output"
)

// statusResult holds the data for status output.
type statusResult struct {
	Repo         string `json:"repo"`
	Branch       string `json:"branch"`
	Head         string `json:"head"`
	TimbersDir   string `json:"timbers_dir"`
	DirExists    bool   `json:"dir_exists"`
	EntryCount   int    `json:"entry_count"`
	FilesTotal   int    `json:"files_total,omitempty"`
	FilesSkipped int    `json:"files_skipped,omitempty"`
	NotTimbers   int    `json:"not_timbers,omitempty"`
	ParseErrors  int    `json:"parse_errors,omitempty"`
}

// newStatusCmd creates the status command.
func newStatusCmd() *cobra.Command {
	var verboseFlag bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show repository and ledger state",
		Long: `Show the current state of the repository and timbers ledger.

Displays repository info (name, branch, HEAD), .timbers/ directory status,
and total entry count.

Examples:
  timbers status            # Show human-readable status
  timbers status --verbose  # Show detailed storage statistics
  timbers status --json     # Output status as JSON for scripting`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(cmd, args, verboseFlag)
		},
	}
	cmd.Flags().BoolVarP(&verboseFlag, "verbose", "v", false, "Show detailed entry statistics")
	return cmd
}

// runStatus executes the status command.
func runStatus(cmd *cobra.Command, _ []string, verbose bool) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), isJSONMode(cmd), useColor(cmd))

	// Check if we're in a git repo
	if !git.IsRepo() {
		err := output.NewSystemError("not in a git repository")
		printer.Error(err)
		return err
	}

	// Gather status information
	result, err := gatherStatus(verbose)
	if err != nil {
		printer.Error(err)
		return err
	}

	// Output based on mode
	if printer.IsJSON() {
		data := map[string]any{
			"repo":        result.Repo,
			"branch":      result.Branch,
			"head":        result.Head,
			"timbers_dir": result.TimbersDir,
			"dir_exists":  result.DirExists,
			"entry_count": result.EntryCount,
		}
		// Add verbose stats if present
		if verbose {
			data["files_total"] = result.FilesTotal
			data["files_skipped"] = result.FilesSkipped
			data["not_timbers"] = result.NotTimbers
			data["parse_errors"] = result.ParseErrors
		}
		// Add suggested commands based on state
		data["suggested_commands"] = []string{"timbers pending"}
		return printer.Success(data)
	}

	// Human-readable output
	printHumanStatus(printer, result, verbose)
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

	// Check .timbers/ directory
	timbersDir := filepath.Join(root, ".timbers")
	dirInfo, statErr := os.Stat(timbersDir)
	dirExists := statErr == nil && dirInfo.IsDir()

	result := &statusResult{
		Repo:       repoName,
		Branch:     branch,
		Head:       head,
		TimbersDir: timbersDir,
		DirExists:  dirExists,
	}

	// Get entry count
	store, storeErr := ledger.NewDefaultStorage()
	if storeErr != nil {
		return nil, storeErr
	}

	if verbose {
		entries, stats, statsErr := store.ListEntriesWithStats()
		if statsErr != nil {
			return nil, statsErr
		}
		result.EntryCount = len(entries)
		result.FilesTotal = stats.Total
		result.FilesSkipped = stats.Skipped
		result.NotTimbers = stats.NotTimbers
		result.ParseErrors = stats.ParseErrors
	} else {
		entries, listErr := store.ListEntries()
		if listErr != nil {
			return nil, listErr
		}
		result.EntryCount = len(entries)
	}

	return result, nil
}

// printHumanStatus outputs status in human-readable format.
func printHumanStatus(printer *output.Printer, status *statusResult, verbose bool) {
	printer.Section("Repository")
	printer.KeyValue("Repo", status.Repo)
	printer.KeyValue("Branch", status.Branch)
	printer.KeyValue("HEAD", status.Head[:min(12, len(status.Head))])

	printer.Section("Timbers Storage")
	printer.KeyValue("Directory", status.TimbersDir)
	printer.KeyValue("Initialized", formatBool(status.DirExists))

	if verbose {
		printer.KeyValue("Files Total", strconv.Itoa(status.FilesTotal))
		printer.KeyValue("Entries", strconv.Itoa(status.EntryCount))
		if status.FilesSkipped > 0 {
			skippedStr := fmt.Sprintf("%d (%d not timbers, %d parse error)",
				status.FilesSkipped, status.NotTimbers, status.ParseErrors)
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
