package main

import (
	"time"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
	"github.com/gorewood/timbers/internal/output"
)

// buildBatchEntry constructs a ledger entry from a commit group.
func buildBatchEntry(
	storage *ledger.Storage, group commitGroup, tags, who []string,
) (*ledger.Entry, error) {
	what, why, how := extractAutoContent(group.commits)
	workItems := extractWorkItemsFromKey(group.key)
	anchor := pickBatchAnchor(group.commits)
	diffstat := getBatchDiffstat(storage, group.commits, anchor)
	now := time.Now().UTC()
	contributors, err := ledger.ResolveContributors(group.commits, who)
	if err != nil {
		return nil, output.NewUserError(err.Error())
	}

	return &ledger.Entry{
		Schema:    ledger.SchemaVersion,
		Kind:      ledger.KindEntry,
		ID:        ledger.GenerateID(anchor, now),
		CreatedAt: now,
		UpdatedAt: now,
		Workset: ledger.Workset{
			AnchorCommit: anchor,
			Commits:      extractCommitSHAs(group.commits),
			Range:        buildCommitRange(group.commits),
			Diffstat: &ledger.Diffstat{
				Files:      diffstat.Files,
				Insertions: diffstat.Insertions,
				Deletions:  diffstat.Deletions,
			},
		},
		Summary: ledger.Summary{
			What: what,
			Why:  why,
			How:  how,
		},
		Tags:         tags,
		WorkItems:    workItems,
		Contributors: contributors,
	}, nil
}

func extractWorkItemsFromKey(key string) []ledger.WorkItem {
	if !isWorkItemKey(key) {
		return nil
	}
	system, id, err := parseWorkItem(key)
	if err != nil {
		return nil
	}
	return []ledger.WorkItem{{System: system, ID: id}}
}

func getBatchDiffstat(storage *ledger.Storage, commits []git.Commit, anchor string) git.Diffstat {
	if len(commits) == 0 {
		return git.Diffstat{}
	}
	diffstat, err := storage.GetDiffstat(commits[len(commits)-1].SHA+"^", anchor)
	if err != nil {
		return git.Diffstat{}
	}
	return diffstat
}

func extractCommitSHAs(commits []git.Commit) []string {
	shas := make([]string, len(commits))
	for idx, commit := range commits {
		shas[idx] = commit.SHA
	}
	return shas
}

func buildCommitRange(commits []git.Commit) string {
	if len(commits) <= 1 {
		return ""
	}
	return commits[len(commits)-1].Short + ".." + commits[0].Short
}
