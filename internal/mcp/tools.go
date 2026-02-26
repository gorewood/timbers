package mcp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
)

// --- Shared types ---

// CommitSummary is a simplified commit for output.
type CommitSummary struct {
	SHA     string `json:"sha"     jsonschema:"full commit SHA"`
	Short   string `json:"short"   jsonschema:"short SHA (7 chars)"`
	Subject string `json:"subject" jsonschema:"commit subject line"`
}

// EntryRef is a reference to a ledger entry.
type EntryRef struct {
	ID           string `json:"id"            jsonschema:"entry ID"`
	AnchorCommit string `json:"anchor_commit" jsonschema:"anchor commit SHA"`
	CreatedAt    string `json:"created_at"    jsonschema:"entry creation timestamp"`
}

// --- Pending tool ---

// PendingInput is the input for the pending tool (no parameters needed).
type PendingInput struct{}

// PendingOutput is the output for the pending tool.
type PendingOutput struct {
	Count     int             `json:"count"                jsonschema:"number of undocumented commits"`
	Commits   []CommitSummary `json:"commits,omitempty"    jsonschema:"list of undocumented commits"`
	LastEntry *EntryRef       `json:"last_entry,omitempty" jsonschema:"most recent ledger entry"`
	Warning   string          `json:"warning,omitempty"    jsonschema:"non-fatal warning message"`
}

func handlePending(storage *ledger.Storage) mcp.ToolHandlerFor[PendingInput, PendingOutput] {
	return func(_ context.Context, _ *mcp.CallToolRequest, _ PendingInput) (*mcp.CallToolResult, PendingOutput, error) {
		commits, latest, err := storage.GetPendingCommits()
		warning := ""
		if err != nil && !errors.Is(err, ledger.ErrStaleAnchor) {
			return nil, PendingOutput{}, fmt.Errorf("getting pending commits: %w", err)
		}
		if errors.Is(err, ledger.ErrStaleAnchor) {
			warning = "anchor commit not found in current history (likely squash merge or rebase); " +
				"showing all reachable commits — if the squash-merged branch had timbers entries, " +
				"this work is already documented; do not catch up; the anchor self-heals on your next timbers log"
		}

		out := PendingOutput{
			Count:   len(commits),
			Commits: toCommitSummaries(commits),
			Warning: warning,
		}

		if latest != nil {
			out.LastEntry = &EntryRef{
				ID:           latest.ID,
				AnchorCommit: latest.Workset.AnchorCommit,
				CreatedAt:    latest.CreatedAt.Format(time.RFC3339),
			}
		}

		return nil, out, nil
	}
}

// --- Prime tool ---

// PrimeInput is the input for the prime tool.
type PrimeInput struct {
	Last    int  `json:"last,omitempty"    jsonschema:"number of recent entries to include (default 3)"`
	Verbose bool `json:"verbose,omitempty" jsonschema:"include why/how in recent entries"`
}

// PrimeEntry is a simplified entry for prime output.
type PrimeEntry struct {
	ID        string `json:"id"                  jsonschema:"entry ID"`
	What      string `json:"what"                jsonschema:"what was done"`
	Why       string `json:"why,omitempty"       jsonschema:"why it was done (verbose only)"`
	How       string `json:"how,omitempty"       jsonschema:"how it was done (verbose only)"`
	CreatedAt string `json:"created_at"          jsonschema:"entry creation timestamp"`
}

// PrimePending holds pending commit information.
type PrimePending struct {
	Count   int             `json:"count"             jsonschema:"number of undocumented commits"`
	Commits []CommitSummary `json:"commits,omitempty" jsonschema:"list of undocumented commits"`
}

// PrimeOutput is the output for the prime tool.
type PrimeOutput struct {
	Repo          string       `json:"repo"           jsonschema:"repository name"`
	Branch        string       `json:"branch"         jsonschema:"current branch"`
	Head          string       `json:"head"           jsonschema:"HEAD commit SHA"`
	EntryCount    int          `json:"entry_count"    jsonschema:"total number of ledger entries"`
	Pending       PrimePending `json:"pending"        jsonschema:"pending commit information"`
	RecentEntries []PrimeEntry `json:"recent_entries" jsonschema:"recent ledger entries"`
	Workflow      string       `json:"workflow"       jsonschema:"workflow instructions text"`
}

func handlePrime(storage *ledger.Storage) mcp.ToolHandlerFor[PrimeInput, PrimeOutput] {
	return func(_ context.Context, _ *mcp.CallToolRequest, input PrimeInput) (*mcp.CallToolResult, PrimeOutput, error) {
		lastN := input.Last
		if lastN <= 0 {
			lastN = 3
		}

		root, err := git.RepoRoot()
		if err != nil {
			return nil, PrimeOutput{}, fmt.Errorf("getting repo root: %w", err)
		}

		branch, err := git.CurrentBranch()
		if err != nil {
			return nil, PrimeOutput{}, fmt.Errorf("getting current branch: %w", err)
		}

		head, err := git.HEAD()
		if err != nil {
			return nil, PrimeOutput{}, fmt.Errorf("getting HEAD: %w", err)
		}

		allEntries, err := storage.ListEntries()
		if err != nil {
			return nil, PrimeOutput{}, fmt.Errorf("listing entries: %w", err)
		}

		pendingCommits, _, pendingErr := storage.GetPendingCommits()
		if pendingErr != nil && !errors.Is(pendingErr, ledger.ErrStaleAnchor) {
			return nil, PrimeOutput{}, fmt.Errorf("getting pending commits: %w", pendingErr)
		}

		recentEntries, err := storage.GetLastNEntries(lastN)
		if err != nil {
			return nil, PrimeOutput{}, fmt.Errorf("getting recent entries: %w", err)
		}

		workflow := loadWorkflowContent(root)

		out := PrimeOutput{
			Repo:          filepath.Base(root),
			Branch:        branch,
			Head:          head,
			EntryCount:    len(allEntries),
			Pending:       buildPrimePending(pendingCommits),
			RecentEntries: buildPrimeEntries(recentEntries, input.Verbose),
			Workflow:      workflow,
		}

		return nil, out, nil
	}
}

// --- Show tool ---

// ShowInput is the input for the show tool.
type ShowInput struct {
	ID     string `json:"id,omitempty"     jsonschema:"entry ID to display"`
	Latest bool   `json:"latest,omitempty" jsonschema:"show the most recent entry"`
}

// ShowOutput is the output for the show tool.
type ShowOutput struct {
	Entry *ledger.Entry `json:"entry" jsonschema:"the ledger entry"`
}

func handleShow(storage *ledger.Storage) mcp.ToolHandlerFor[ShowInput, ShowOutput] {
	return func(_ context.Context, _ *mcp.CallToolRequest, input ShowInput) (*mcp.CallToolResult, ShowOutput, error) {
		if input.ID == "" && !input.Latest {
			return nil, ShowOutput{}, errors.New("specify id or set latest=true")
		}
		if input.ID != "" && input.Latest {
			return nil, ShowOutput{}, errors.New("cannot use both id and latest")
		}

		var entry *ledger.Entry
		var err error

		if input.Latest {
			entry, err = storage.GetLatestEntry()
			if errors.Is(err, ledger.ErrNoEntries) {
				return nil, ShowOutput{}, errors.New("no entries found in ledger")
			}
		} else {
			entry, err = storage.GetEntryByID(input.ID)
		}

		if err != nil {
			return nil, ShowOutput{}, fmt.Errorf("getting entry: %w", err)
		}

		return nil, ShowOutput{Entry: entry}, nil
	}
}

// --- Status tool ---

// StatusInput is the input for the status tool (no parameters needed).
type StatusInput struct{}

// StatusOutput is the output for the status tool.
type StatusOutput struct {
	Repo       string `json:"repo"        jsonschema:"repository name"`
	Branch     string `json:"branch"      jsonschema:"current branch"`
	Head       string `json:"head"        jsonschema:"HEAD commit SHA"`
	TimbersDir string `json:"timbers_dir" jsonschema:"path to .timbers directory"`
	DirExists  bool   `json:"dir_exists"  jsonschema:"whether .timbers directory exists"`
	EntryCount int    `json:"entry_count" jsonschema:"total number of ledger entries"`
}

func handleStatus(storage *ledger.Storage) mcp.ToolHandlerFor[StatusInput, StatusOutput] {
	return func(_ context.Context, _ *mcp.CallToolRequest, _ StatusInput) (*mcp.CallToolResult, StatusOutput, error) {
		root, err := git.RepoRoot()
		if err != nil {
			return nil, StatusOutput{}, fmt.Errorf("getting repo root: %w", err)
		}

		branch, err := git.CurrentBranch()
		if err != nil {
			return nil, StatusOutput{}, fmt.Errorf("getting current branch: %w", err)
		}

		head, err := git.HEAD()
		if err != nil {
			return nil, StatusOutput{}, fmt.Errorf("getting HEAD: %w", err)
		}

		timbersDir := filepath.Join(root, ".timbers")
		dirInfo, statErr := os.Stat(timbersDir)
		dirExists := statErr == nil && dirInfo.IsDir()

		entries, err := storage.ListEntries()
		if err != nil {
			return nil, StatusOutput{}, fmt.Errorf("listing entries: %w", err)
		}

		out := StatusOutput{
			Repo:       filepath.Base(root),
			Branch:     branch,
			Head:       head,
			TimbersDir: timbersDir,
			DirExists:  dirExists,
			EntryCount: len(entries),
		}

		return nil, out, nil
	}
}
