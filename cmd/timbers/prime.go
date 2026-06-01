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

// errNotGitRepo indicates prime was run outside a git repository.
var errNotGitRepo = errors.New("not in a git repository")

const (
	primeCompactMode = "compact v2"
	primeFullMode    = "full"
)

// primeResult holds the data for prime output.
type primeResult struct {
	Mode           string            `json:"mode"`
	Repo           string            `json:"repo"`
	Branch         string            `json:"branch"`
	Head           string            `json:"head"`
	TimbersDir     string            `json:"timbers_dir"`
	EntryCount     int               `json:"entry_count"`
	Pending        primePending      `json:"pending"`
	StaleAnchor    bool              `json:"stale_anchor,omitempty"`
	RecentEntries  []primeEntry      `json:"recent_entries"`
	Health         []primeHealthItem `json:"health,omitempty"`
	Workflow       string            `json:"workflow"`
	CustomWorkflow bool              `json:"custom_workflow,omitempty"`
}

// primePending holds pending commit information.
//
// Count is the IN-SESSION blocking count — the number an agent should
// drive to zero before session end. OutOfSession and Stale are sibling
// fields surfacing the provenance-classified commits the gate silently
// skips; they are visible-but-not-blocking so an operator can still see
// foreign work for ack/backfill flows, but agents that read Count and
// stop there get the right answer.
//
// Without this semantic split, an agent that sees Count=5 (raw total)
// will try to document 5 things — wasting tokens on commits whose
// reasoning context is no longer recoverable.
type primePending struct {
	Count        int             `json:"count"`
	OutOfSession int             `json:"out_of_session,omitempty"`
	Stale        int             `json:"stale,omitempty"`
	Commits      []commitSummary `json:"commits,omitempty"`
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
	var fullFlag bool
	var guideFlag bool
	var hookFlag bool
	var exportFlag bool

	cmd := &cobra.Command{
		Use:   "prime",
		Short: "Session bootstrapping context injection",
		Long: `Prime provides session context for starting a development session.

This command gathers repository info, recent ledger entries, and pending
commits to give agents and developers a quick overview of the current state.

The default output is compact for agent context injection. Use --full to include
the full workflow guide, which can be customized with .timbers/PRIME.md.

Examples:
  timbers prime              # Show compact session context with last 3 entries
  timbers prime --hook       # Show hook-optimized compact context
  timbers prime --last 5     # Show session context with last 5 entries
  timbers prime --verbose    # Include why/how in recent entries
  timbers prime --full       # Include full workflow guide
  timbers prime --json       # Output structured context as JSON
  timbers prime --export     # Output default workflow content for customization`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if exportFlag {
				cmd.Print(defaultWorkflowContent)
				return nil
			}
			full := fullFlag || guideFlag
			_ = hookFlag // --hook is an explicit name for the compact default.
			return runPrime(cmd, storage, lastFlag, verboseFlag, full)
		},
	}

	cmd.Flags().IntVar(&lastFlag, "last", 3, "Number of recent entries to show")
	cmd.Flags().BoolVar(&verboseFlag, "verbose", false, "Include why/how details in recent entries")
	cmd.Flags().BoolVar(&fullFlag, "full", false, "Include full workflow guide")
	cmd.Flags().BoolVar(&guideFlag, "guide", false, "Alias for --full")
	cmd.Flags().BoolVar(&hookFlag, "hook", false, "Output compact hook-friendly context")
	cmd.Flags().BoolVar(&exportFlag, "export", false, "Output default workflow content for customization")

	return cmd
}

// resolveStorage checks if we're in an initialized timbers repo and returns storage.
// Returns errNotInitialized if the repo is not initialized.
// Returns errNotGitRepo when run outside a git repository.
// Returns (storage, nil) on success.
func resolveStorage(storage *ledger.Storage) (*ledger.Storage, error) {
	if storage != nil {
		return storage, nil
	}

	if !git.IsRepo() {
		return nil, errNotGitRepo
	}

	root, err := git.RepoRoot()
	if err != nil {
		return nil, err
	}

	timbersDir := filepath.Join(root, ".timbers")
	if _, statErr := os.Stat(timbersDir); os.IsNotExist(statErr) {
		return nil, errNotInitialized
	}

	return ledger.NewDefaultStorage()
}

// runPrime executes the prime command.
func runPrime(cmd *cobra.Command, storage *ledger.Storage, lastN int, verbose bool, full bool) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), isJSONMode(cmd), useColor(cmd))

	resolved, err := resolveStorage(storage)
	if errors.Is(err, errNotInitialized) {
		return outputPrimeUnavailable(printer, "ledger not initialized", "timbers init")
	}
	if errors.Is(err, errNotGitRepo) {
		return outputPrimeUnavailable(printer, "not in a git repository", "cd <repo>")
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

	if full {
		result.Mode = primeFullMode
	}
	if printer.IsJSON() {
		return printer.WriteJSON(result)
	}
	if full {
		outputPrimeFullHuman(printer, result)
		return nil
	}
	outputPrimeCompactHuman(printer, result)
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
	staleAnchor := errors.Is(pendingErr, ledger.ErrStaleAnchor)
	if staleAnchor {
		pendingCommits = nil
	}

	// Bucket provenance reasons for the in-session vs out-of-session
	// breakdown that drives Count semantics. ExplainPending walks the
	// display range and classifies each commit; errors here are non-fatal
	// (we still ship the in-session count from GetPendingCommits).
	classified, _, _ := storage.ExplainPending()

	recentEntries, err := storage.GetLastNEntries(lastN)
	if err != nil {
		return nil, err
	}

	workflow, custom := loadWorkflowContent(root)
	health := runQuickHealthCheck()

	return &primeResult{
		Mode:           primeCompactMode,
		Repo:           repoName,
		Branch:         branch,
		Head:           head,
		TimbersDir:     filepath.Join(root, ".timbers"),
		EntryCount:     len(allEntries),
		Pending:        buildPrimePending(pendingCommits, classified),
		StaleAnchor:    staleAnchor,
		RecentEntries:  buildPrimeEntries(recentEntries, verbose),
		Health:         health,
		Workflow:       workflow,
		CustomWorkflow: custom,
	}, nil
}

// loadWorkflowContent loads workflow content from .timbers/PRIME.md.
// Returns (defaultWorkflowContent, false) when no override file exists,
// or (override, true) when .timbers/PRIME.md is present and readable.
func loadWorkflowContent(repoRoot string) (string, bool) {
	overridePath := filepath.Join(repoRoot, ".timbers", "PRIME.md")
	data, err := os.ReadFile(overridePath)
	if err != nil {
		return defaultWorkflowContent, false
	}
	return string(data), true
}

// outputPrimeFullHuman outputs the full guide in human-readable format.
func outputPrimeFullHuman(printer *output.Printer, result *primeResult) {
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
	switch {
	case result.StaleAnchor:
		printer.Println("  Pending: 0 actionable (stale anchor)")
	case result.Pending.Count == 0:
		printer.Println("  Pending: all work documented")
	default:
		printer.Print("  Pending: %d undocumented commit", result.Pending.Count)
		if result.Pending.Count != 1 {
			printer.Print("s")
		}
		printer.Println()
	}
	printer.Println()

	outputPrimeRecentWork(printer, result.RecentEntries)
	outputPrimeHealth(printer, result.Health)
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
