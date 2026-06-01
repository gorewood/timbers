// Package main — prime data-builders extracted from prime.go to keep that
// file under the file-length limit. Pure functions; no I/O.
package main

import (
	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
)

// buildPrimePending constructs the pending section.
//
// commits is the filtered in-session set (from GetPendingCommits) — these
// populate Count and the displayed Commits slice, the surface agents act on.
// classified is the unfiltered classify-everything output from ExplainPending;
// we bucket it to fill the OutOfSession and Stale visibility fields without
// surfacing the foreign commits themselves (agents that see subject lines
// will hallucinate documentation for them).
func buildPrimePending(commits []git.Commit, classified []ledger.ClassifiedCommit) primePending {
	pending := primePending{
		Count:   len(commits),
		Commits: make([]commitSummary, 0, len(commits)),
	}
	for _, commit := range commits {
		pending.Commits = append(pending.Commits, commitSummary{
			SHA:     commit.SHA,
			Short:   commit.Short,
			Subject: commit.Subject,
		})
	}
	for _, item := range classified {
		switch item.Reason {
		case "foreign-author", "foreign-author+stale":
			pending.OutOfSession++
		case "stale":
			pending.Stale++
		}
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
