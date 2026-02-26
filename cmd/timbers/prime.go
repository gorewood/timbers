// Package main provides the entry point for the timbers CLI.
package main

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
	"github.com/gorewood/timbers/internal/output"
)

// errNotInitialized indicates timbers is not set up in this repo.
var errNotInitialized = errors.New("timbers not initialized")

// primeResult holds the data for prime output.
type primeResult struct {
	Repo          string       `json:"repo"`
	Branch        string       `json:"branch"`
	Head          string       `json:"head"`
	TimbersDir    string       `json:"timbers_dir"`
	EntryCount    int          `json:"entry_count"`
	Pending       primePending `json:"pending"`
	RecentEntries []primeEntry `json:"recent_entries"`
	Workflow      string       `json:"workflow"`
}

// primePending holds pending commit information.
type primePending struct {
	Count   int             `json:"count"`
	Commits []commitSummary `json:"commits,omitempty"`
}

// primeEntry is a simplified entry for prime output.
type primeEntry struct {
	ID        string `json:"id"`
	What      string `json:"what"`
	Why       string `json:"why,omitempty"`
	How       string `json:"how,omitempty"`
	Notes     string `json:"notes,omitempty"`
	CreatedAt string `json:"created_at"`
}

// newPrimeCmd creates the prime command.
func newPrimeCmd() *cobra.Command {
	return newPrimeCmdInternal(nil)
}

// newPrimeCmdInternal creates the prime command with optional storage injection.
// If storage is nil, a real storage is created when the command runs.
func newPrimeCmdInternal(storage *ledger.Storage) *cobra.Command {
	var lastFlag int
	var verboseFlag bool
	var exportFlag bool

	cmd := &cobra.Command{
		Use:   "prime",
		Short: "Session bootstrapping context injection",
		Long: `Prime provides session context for starting a development session.

This command gathers repository info, recent ledger entries, and pending
commits to give agents and developers a quick overview of the current state.

Workflow instructions are included to guide agents through the session close
protocol. These can be customized by creating .timbers/PRIME.md in the repo root.

Examples:
  timbers prime              # Show session context with last 3 entries
  timbers prime --last 5     # Show session context with last 5 entries
  timbers prime --verbose    # Include why/how in recent entries
  timbers prime --json       # Output structured context as JSON
  timbers prime --export     # Output default workflow content for customization`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if exportFlag {
				cmd.Print(defaultWorkflowContent)
				return nil
			}
			return runPrime(cmd, storage, lastFlag, verboseFlag)
		},
	}

	cmd.Flags().IntVar(&lastFlag, "last", 3, "Number of recent entries to show")
	cmd.Flags().BoolVar(&verboseFlag, "verbose", false, "Include why/how details in recent entries")
	cmd.Flags().BoolVar(&exportFlag, "export", false, "Output default workflow content for customization")

	return cmd
}

// resolveStorage checks if we're in an initialized timbers repo and returns storage.
// Returns (nil, nil) if the repo is not initialized (silent skip).
// Returns (nil, error) on failures.
// Returns (storage, nil) on success.
func resolveStorage(storage *ledger.Storage, verbose bool, printer *output.Printer) (*ledger.Storage, error) {
	if storage != nil {
		return storage, nil
	}

	if !git.IsRepo() {
		return nil, output.NewSystemError("not in a git repository")
	}

	root, err := git.RepoRoot()
	if err != nil {
		return nil, err
	}

	timbersDir := filepath.Join(root, ".timbers")
	if _, statErr := os.Stat(timbersDir); os.IsNotExist(statErr) {
		if verbose {
			printer.Stderr("timbers not initialized in this repo; run 'timbers init' to activate")
		}
		return nil, errNotInitialized
	}

	return ledger.NewDefaultStorage()
}

// runPrime executes the prime command.
func runPrime(cmd *cobra.Command, storage *ledger.Storage, lastN int, verbose bool) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), isJSONMode(cmd), useColor(cmd))

	resolved, err := resolveStorage(storage, verbose, printer)
	if errors.Is(err, errNotInitialized) {
		return nil // uninitiated repo, silent skip
	}
	if err != nil {
		printer.Error(err)
		return err
	}

	// Gather all context
	result, gatherErr := gatherPrimeContext(resolved, lastN, verbose)
	if gatherErr != nil {
		printer.Error(gatherErr)
		return gatherErr
	}

	if printer.IsJSON() {
		return printer.WriteJSON(result)
	}
	outputPrimeHuman(printer, result)
	return nil
}

// gatherPrimeContext collects all prime context information.
func gatherPrimeContext(storage *ledger.Storage, lastN int, verbose bool) (*primeResult, error) {
	root, err := git.RepoRoot()
	if err != nil {
		return nil, err
	}
	repoName := filepath.Base(root)

	branch, err := git.CurrentBranch()
	if err != nil {
		return nil, err
	}

	head, err := git.HEAD()
	if err != nil {
		return nil, err
	}

	allEntries, err := storage.ListEntries()
	if err != nil {
		return nil, err
	}

	pendingCommits, _, pendingErr := storage.GetPendingCommits()
	if pendingErr != nil && !errors.Is(pendingErr, ledger.ErrStaleAnchor) {
		return nil, pendingErr
	}

	recentEntries, err := storage.GetLastNEntries(lastN)
	if err != nil {
		return nil, err
	}

	workflow := loadWorkflowContent(root)

	return &primeResult{
		Repo:          repoName,
		Branch:        branch,
		Head:          head,
		TimbersDir:    filepath.Join(root, ".timbers"),
		EntryCount:    len(allEntries),
		Pending:       buildPrimePending(pendingCommits),
		RecentEntries: buildPrimeEntries(recentEntries, verbose),
		Workflow:      workflow,
	}, nil
}

// loadWorkflowContent loads workflow content from .timbers/PRIME.md or returns default.
func loadWorkflowContent(repoRoot string) string {
	overridePath := filepath.Join(repoRoot, ".timbers", "PRIME.md")
	data, err := os.ReadFile(overridePath)
	if err != nil {
		return defaultWorkflowContent
	}
	return string(data)
}

// buildPrimePending constructs the pending section from commits.
func buildPrimePending(commits []git.Commit) primePending {
	pending := primePending{
		Count:   len(commits),
		Commits: make([]commitSummary, 0, len(commits)),
	}

	for _, c := range commits {
		pending.Commits = append(pending.Commits, commitSummary{
			SHA:     c.SHA,
			Short:   c.Short,
			Subject: c.Subject,
		})
	}

	return pending
}

// buildPrimeEntries constructs the recent entries section.
// When verbose is true, includes why and how fields.
func buildPrimeEntries(entries []*ledger.Entry, verbose bool) []primeEntry {
	result := make([]primeEntry, 0, len(entries))

	for _, entry := range entries {
		prime := primeEntry{
			ID:        entry.ID,
			What:      entry.Summary.What,
			CreatedAt: entry.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
		if verbose {
			prime.Why = entry.Summary.Why
			prime.How = entry.Summary.How
			prime.Notes = truncateNotes(entry.Notes, 200)
		}
		result = append(result, prime)
	}

	return result
}

// truncateNotes truncates notes to maxLen characters, appending "..." if truncated.
func truncateNotes(notes string, maxLen int) string {
	if len(notes) <= maxLen {
		return notes
	}
	return notes[:maxLen] + "..."
}

// outputPrimeHuman outputs the result in human-readable format.
func outputPrimeHuman(printer *output.Printer, result *primeResult) {
	printer.Println("Timbers Session Context")
	printer.Println("=======================")
	printer.Println()
	shortHead := result.Head
	if len(shortHead) > 7 {
		shortHead = shortHead[:7]
	}
	printer.Print("Repository: %s (%s)\n", result.Repo, result.Branch)
	printer.Print("HEAD: %s\n", shortHead)
	printer.Println()
	printer.Println("Ledger Status")
	printer.Println("-------------")
	printer.Print("  Entries: %d\n", result.EntryCount)
	if result.Pending.Count == 0 {
		printer.Println("  Pending: all work documented")
	} else {
		printer.Print("  Pending: %d undocumented commit", result.Pending.Count)
		if result.Pending.Count != 1 {
			printer.Print("s")
		}
		printer.Println()
	}
	printer.Println()

	outputPrimeRecentWork(printer, result.RecentEntries)
	printer.Println(result.Workflow)
}

// outputPrimeRecentWork prints the recent entries section.
func outputPrimeRecentWork(printer *output.Printer, entries []primeEntry) {
	printer.Println("Recent Work")
	printer.Println("-----------")
	if len(entries) == 0 {
		printer.Println("  (no entries)")
	} else {
		for _, entry := range entries {
			printer.Print("  %s  %s\n", entry.ID, entry.What)
			if entry.Why != "" {
				printer.Print("    Why: %s\n", entry.Why)
			}
			if entry.How != "" {
				printer.Print("    How: %s\n", entry.How)
			}
			if entry.Notes != "" {
				printer.Print("    Notes: %s\n", entry.Notes)
			}
		}
	}
	printer.Println()
}
