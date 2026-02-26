package mcp

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
)

// toCommitSummaries converts git commits to CommitSummary slice.
func toCommitSummaries(commits []git.Commit) []CommitSummary {
	result := make([]CommitSummary, 0, len(commits))
	for _, commit := range commits {
		result = append(result, CommitSummary{
			SHA:     commit.SHA,
			Short:   commit.Short,
			Subject: commit.Subject,
		})
	}
	return result
}

// buildPrimePending constructs PrimePending from commits.
func buildPrimePending(commits []git.Commit) PrimePending {
	return PrimePending{
		Count:   len(commits),
		Commits: toCommitSummaries(commits),
	}
}

// buildPrimeEntries constructs PrimeEntry slice from entries.
func buildPrimeEntries(entries []*ledger.Entry, verbose bool) []PrimeEntry {
	result := make([]PrimeEntry, 0, len(entries))
	for _, entry := range entries {
		primeEntry := PrimeEntry{
			ID:        entry.ID,
			What:      entry.Summary.What,
			CreatedAt: entry.CreatedAt.Format(time.RFC3339),
		}
		if verbose {
			primeEntry.Why = entry.Summary.Why
			primeEntry.How = entry.Summary.How
		}
		result = append(result, primeEntry)
	}
	return result
}

// defaultWorkflowContent is the fallback workflow text when no PRIME.md exists.
// NOTE: Keep in sync with cmd/timbers/prime_workflow.go defaultWorkflowContent.
const defaultWorkflowContent = `<protocol>
# Session Protocol

After each git commit, run timbers log to document what you committed.
Entries reference commit SHAs, so the commit must exist before the entry.
Document each commit individually — batching loses commit-level granularity.

Session checklist:
- [ ] git add && git commit (commit code first)
- [ ] timbers log "what" --why "why" --how "how" (document committed work)
- [ ] timbers pending (should be zero before session end)
- [ ] git push (timbers log auto-commits entries, push to sync)
</protocol>

<stale-anchor>
# Stale Anchor After Squash Merge

If timbers warns that the anchor commit is missing from history, this typically
means a branch was squash-merged or rebased. The pending list may show commits
that are already documented by entries from the original branch.

What to do:
- Do NOT try to catch up or re-document these commits
- If the squash-merged branch had timbers entries, the work is already covered
- Just proceed with your normal work — the anchor self-heals the next time you
  run timbers log after a real commit
</stale-anchor>
`

// loadWorkflowContent loads workflow content from .timbers/PRIME.md or returns default.
func loadWorkflowContent(repoRoot string) string {
	overridePath := filepath.Join(repoRoot, ".timbers", "PRIME.md")
	data, err := os.ReadFile(overridePath)
	if err != nil {
		return defaultWorkflowContent
	}
	return string(data)
}

// parseDurationOrDate parses a duration string (24h, 7d) or ISO date into a time.
func parseDurationOrDate(value string) (time.Time, error) {
	// Try parsing as a Go duration first
	if duration, err := time.ParseDuration(value); err == nil {
		return time.Now().UTC().Add(-duration), nil
	}

	// Try day-based duration (e.g. "7d")
	if len(value) > 1 && value[len(value)-1] == 'd' {
		var days int
		if _, err := fmt.Sscanf(value, "%dd", &days); err == nil && days > 0 {
			return time.Now().UTC().AddDate(0, 0, -days), nil
		}
	}

	// Try ISO date
	if parsed, err := time.Parse("2006-01-02", value); err == nil {
		return parsed, nil
	}

	// Try RFC3339
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed, nil
	}

	return time.Time{}, fmt.Errorf("cannot parse %q as duration or date", value)
}

// parseWorkItem parses a "system:id" string into a WorkItem.
func parseWorkItem(value string) (ledger.WorkItem, error) {
	for idx, char := range value {
		if char == ':' && idx > 0 && idx < len(value)-1 {
			return ledger.WorkItem{
				System: value[:idx],
				ID:     value[idx+1:],
			}, nil
		}
	}
	return ledger.WorkItem{}, errors.New(
		"work_item must be in system:id format (e.g. beads:abc123)",
	)
}
