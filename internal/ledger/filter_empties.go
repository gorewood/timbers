package ledger

import "github.com/gorewood/timbers/internal/git"

// dropEmptyFileChanges removes commits whose file map entry is nil or empty.
// Used by the gate's strict path. Order-preserving.
func dropEmptyFileChanges(commits []git.Commit, fileMap map[string][]string) []git.Commit {
	out := make([]git.Commit, 0, len(commits))
	for _, commit := range commits {
		if len(fileMap[commit.SHA]) == 0 {
			continue
		}
		out = append(out, commit)
	}
	return out
}

// dropEmptyMerges removes merge commits (2+ parents) whose combined diff
// returned no files — i.e., the merge added nothing on this branch's
// first-parent line. Single-parent commits are preserved even when their
// file list is empty (--allow-empty marker commits stay visible in
// display output). Order-preserving.
func dropEmptyMerges(commits []git.Commit, fileMap map[string][]string) []git.Commit {
	out := make([]git.Commit, 0, len(commits))
	for _, commit := range commits {
		if commit.IsMerge() && len(fileMap[commit.SHA]) == 0 {
			continue
		}
		out = append(out, commit)
	}
	return out
}
