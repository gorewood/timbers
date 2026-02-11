package main

import (
	"strconv"
	"strings"
	"time"

	"github.com/gorewood/timbers/internal/ledger"
	"github.com/gorewood/timbers/internal/output"
)

const catchupSystemPrompt = `You are a development documentation assistant. ` +
	`Given git commits, generate a concise what/why/how summary.

Output EXACTLY in this format (3 lines, no extra text):
WHAT: <one sentence describing what was done>
WHY: <one sentence explaining the motivation>
HOW: <one sentence describing the approach>

Rules: Be concise (<100 chars each). Use active voice. Infer reason if unclear.`

func buildCatchupPrompt(group commitGroup) string {
	var b strings.Builder
	b.WriteString("Group: " + group.key + "\nCommits: " + strconv.Itoa(len(group.commits)) + "\n\n")
	for idx, c := range group.commits {
		b.WriteString("--- Commit " + strconv.Itoa(idx+1) + " ---\nSHA: " + c.Short)
		b.WriteString("\nDate: " + c.Date.Format("2006-01-02 15:04") + "\nSubject: " + c.Subject + "\n")
		if c.Body != "" {
			b.WriteString("Body:\n" + c.Body + "\n")
		}
		b.WriteString("\n")
	}
	return b.String()
}

func parseCatchupResponse(response string) (what, why, how string) {
	for line := range strings.SplitSeq(response, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "WHAT:"):
			what = strings.TrimSpace(strings.TrimPrefix(line, "WHAT:"))
		case strings.HasPrefix(line, "WHY:"):
			why = strings.TrimSpace(strings.TrimPrefix(line, "WHY:"))
		case strings.HasPrefix(line, "HOW:"):
			how = strings.TrimSpace(strings.TrimPrefix(line, "HOW:"))
		}
	}
	if what == "" {
		what = "Auto-documented commits"
	}
	if why == "" {
		why = "Historical documentation"
	}
	if how == "" {
		how = "See commit messages for details"
	}
	return what, why, how
}

func buildCatchupEntry(
	storage *ledger.Storage, group commitGroup, what, why, how string, tags []string,
) *ledger.Entry {
	anchor := group.commits[0].SHA
	now := time.Now().UTC()
	return &ledger.Entry{
		Schema: ledger.SchemaVersion, Kind: ledger.KindEntry,
		ID: ledger.GenerateID(anchor, now), CreatedAt: now, UpdatedAt: now,
		Workset: ledger.Workset{
			AnchorCommit: anchor, Commits: extractCommitSHAs(group.commits),
			Range: buildCommitRange(group.commits),
			Diffstat: func() *ledger.Diffstat {
				d := getBatchDiffstat(storage, group.commits, anchor)
				return &ledger.Diffstat{Files: d.Files, Insertions: d.Insertions, Deletions: d.Deletions}
			}(),
		},
		Summary:   ledger.Summary{What: what, Why: why, How: how},
		Tags:      tags,
		WorkItems: extractWorkItemsFromKey(group.key),
	}
}

func outputCatchupResult(printer *output.Printer, entries []catchupEntryRef, flags catchupFlags) error {
	status := "created"
	if flags.dryRun {
		status = "dry_run"
	}
	if printer.IsJSON() {
		return printer.WriteJSON(catchupResult{Status: status, Count: len(entries), Entries: entries})
	}
	if flags.dryRun {
		printer.Print("Dry run - would create %d entries:\n\n", len(entries))
	} else {
		printer.Print("Created %d entries:\n\n", len(entries))
	}
	for _, e := range entries {
		printer.Print("  %s [%s]\n    What: %s\n    Why:  %s\n    How:  %s\n\n",
			e.ID, e.GroupKey, truncateString(e.What, 70), truncateString(e.Why, 70), truncateString(e.How, 70))
	}
	return nil
}
