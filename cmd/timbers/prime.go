// Package main provides the entry point for the timbers CLI.
package main

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
	"github.com/gorewood/timbers/internal/output"
)

// defaultWorkflowContent is the default workflow instructions for agent onboarding.
// This can be overridden by placing a .timbers/PRIME.md file in the repo root.
const defaultWorkflowContent = `# Session Close Protocol
- [ ] timbers pending (check for undocumented work)
- [ ] timbers log "what" --why "why" --how "how" (document work)
- [ ] timbers notes push (sync to remote)

# Core Rules
- Document work with what/why/how before session end
- Use ` + "`timbers pending`" + ` to check for undocumented commits
- Run ` + "`timbers notes push`" + ` to sync ledger to remote

# Essential Commands
### Recording Work
- ` + "`timbers log \"what\" --why \"why\" --how \"how\"`" + ` - Record development work
- ` + "`timbers pending`" + ` - Show undocumented commits

### Querying
- ` + "`timbers query --last 5`" + ` - Recent entries
- ` + "`timbers show <id>`" + ` - Single entry details

### Sync
- ` + "`timbers notes push`" + ` - Push notes to remote
- ` + "`timbers notes fetch`" + ` - Fetch notes from remote
`

// primeResult holds the data for prime output.
type primeResult struct {
	Repo            string       `json:"repo"`
	Branch          string       `json:"branch"`
	Head            string       `json:"head"`
	NotesRef        string       `json:"notes_ref"`
	NotesConfigured bool         `json:"notes_configured"`
	EntryCount      int          `json:"entry_count"`
	Pending         primePending `json:"pending"`
	RecentEntries   []primeEntry `json:"recent_entries"`
	Workflow        string       `json:"workflow"`
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
  timbers prime --json       # Output structured context as JSON
  timbers prime --export     # Output default workflow content for customization`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if exportFlag {
				cmd.Print(defaultWorkflowContent)
				return nil
			}
			return runPrime(cmd, storage, lastFlag)
		},
	}

	cmd.Flags().IntVar(&lastFlag, "last", 3, "Number of recent entries to show")
	cmd.Flags().BoolVar(&exportFlag, "export", false, "Output default workflow content for customization")

	return cmd
}

// runPrime executes the prime command.
func runPrime(cmd *cobra.Command, storage *ledger.Storage, lastN int) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), jsonFlag, output.IsTTY(cmd.OutOrStdout()))

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

	// Gather all context
	result, err := gatherPrimeContext(storage, lastN)
	if err != nil {
		printer.Error(err)
		return err
	}

	// Output based on mode
	if jsonFlag {
		return outputPrimeJSON(printer, result)
	}

	outputPrimeHuman(printer, result)
	return nil
}

// gatherPrimeContext collects all prime context information.
func gatherPrimeContext(storage *ledger.Storage, lastN int) (*primeResult, error) {
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

	// Get all entries (for count)
	allEntries, err := storage.ListEntries()
	if err != nil {
		return nil, err
	}

	// Get pending commits
	pendingCommits, _, err := storage.GetPendingCommits()
	if err != nil {
		return nil, err
	}

	// Get recent entries
	recentEntries, err := storage.GetLastNEntries(lastN)
	if err != nil {
		return nil, err
	}

	// Get workflow content (from override file or default)
	workflow := loadWorkflowContent(root)

	// Build result
	result := &primeResult{
		Repo:            repoName,
		Branch:          branch,
		Head:            head,
		NotesRef:        "refs/notes/timbers",
		NotesConfigured: notesConfigured,
		EntryCount:      len(allEntries),
		Pending:         buildPrimePending(pendingCommits),
		RecentEntries:   buildPrimeEntries(recentEntries),
		Workflow:        workflow,
	}

	return result, nil
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
func buildPrimeEntries(entries []*ledger.Entry) []primeEntry {
	result := make([]primeEntry, 0, len(entries))

	for _, e := range entries {
		result = append(result, primeEntry{
			ID:        e.ID,
			What:      e.Summary.What,
			CreatedAt: e.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	return result
}

// outputPrimeJSON outputs the result as JSON.
func outputPrimeJSON(printer *output.Printer, result *primeResult) error {
	return printer.WriteJSON(result)
}

// outputPrimeHuman outputs the result in human-readable format.
func outputPrimeHuman(printer *output.Printer, result *primeResult) {
	// Title
	printer.Println("Timbers Session Context")
	printer.Println("=======================")
	printer.Println()

	// Repository info
	shortHead := result.Head
	if len(shortHead) > 7 {
		shortHead = shortHead[:7]
	}
	printer.Print("Repository: %s (%s)\n", result.Repo, result.Branch)
	printer.Print("HEAD: %s\n", shortHead)
	printer.Println()

	// Ledger status
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

	// Recent work
	printer.Println("Recent Work")
	printer.Println("-----------")
	if len(result.RecentEntries) == 0 {
		printer.Println("  (no entries)")
	} else {
		for _, entry := range result.RecentEntries {
			printer.Print("  %s  %s\n", entry.ID, entry.What)
		}
	}
	printer.Println()

	// Workflow instructions
	printer.Println(result.Workflow)
}
