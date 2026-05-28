package main

import (
	"errors"
	"strings"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
)

// getLogCommits retrieves the commits to include in the entry.
// Returns staleAnchor=true when the latest entry's anchor is missing from history.
func getLogCommits(storage *ledger.Storage, flags logFlags) ([]git.Commit, string, bool, error) {
	if flags.rangeStr != "" {
		parts := strings.SplitN(flags.rangeStr, "..", 2)
		fromRef := parts[0]
		toRef := parts[1]
		commits, err := storage.LogRange(fromRef, toRef)
		if err != nil {
			return nil, "", false, err
		}
		return commits, fromRef, false, nil
	}

	commits, _, err := storage.GetPendingCommits()
	stale := errors.Is(err, ledger.ErrStaleAnchor)
	if err != nil && !stale {
		return nil, "", false, err
	}

	// --anchor with no detected pending commits: honor the flag's promise
	// ("use this anchor") by logging that single commit, instead of refusing.
	// Covers states where detection legitimately finds nothing to anchor on
	// but the operator knows the commit they want documented.
	if len(commits) == 0 && flags.anchor != "" {
		single, rangeErr := storage.LogRange(flags.anchor+"^", flags.anchor)
		if rangeErr != nil {
			return nil, "", false, rangeErr
		}
		return single, flags.anchor + "^", false, nil
	}

	fromRef := ""
	if len(commits) > 0 {
		fromRef = commits[len(commits)-1].SHA + "^"
	}

	return commits, fromRef, stale, nil
}

// determineAnchor determines the anchor commit for the entry.
func determineAnchor(commits []git.Commit, anchorOverride string) string {
	if anchorOverride != "" {
		return anchorOverride
	}
	if len(commits) > 0 {
		return commits[0].SHA
	}
	return ""
}

// getDiffstatForRange gets the diffstat for a commit range.
func getDiffstatForRange(
	storage *ledger.Storage,
	fromRef, toRef string,
	commits []git.Commit,
) (git.Diffstat, error) {
	if fromRef == "" && len(commits) > 0 {
		fromRef = commits[len(commits)-1].SHA + "^"
	}
	if fromRef == "" {
		return git.Diffstat{}, nil
	}
	return storage.GetDiffstat(fromRef, toRef)
}
