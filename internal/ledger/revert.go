package ledger

import (
	"regexp"
	"strings"

	"github.com/gorewood/timbers/internal/git"
)

// revertSubjectPrefix identifies commits created by `git revert`.
const revertSubjectPrefix = `Revert "`

// revertedCommitRE matches the body trailer git revert produces.
// We tolerate variable SHA width (7..40 hex chars) for short-SHA edge cases,
// even though git's default format uses the full 40-char SHA.
var revertedCommitRE = regexp.MustCompile(`(?m)^This reverts commit ([0-9a-f]{7,40})\b`)

// parseRevertedSHAs extracts every "This reverts commit <sha>" reference
// from a commit body. Returns nil if none are found.
func parseRevertedSHAs(body string) []string {
	matches := revertedCommitRE.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		return nil
	}
	shas := make([]string, 0, len(matches))
	for _, m := range matches {
		shas = append(shas, m[1])
	}
	return shas
}

// isRevertCommit reports whether a commit looks like a `git revert` output.
// Both the subject prefix and a body trailer are required — subject alone
// is not enough (an agent could write "Revert \"X\"" by hand without the
// machine-readable trailer).
func isRevertCommit(c git.Commit) bool {
	if !strings.HasPrefix(c.Subject, revertSubjectPrefix) {
		return false
	}
	return len(parseRevertedSHAs(c.Body)) > 0
}

// isDocumentedRevert returns true when c is a revert AND every SHA it claims
// to revert is already covered by some ledger entry. Conservative: if any
// referenced SHA is undocumented, the revert remains pending so the user
// can decide whether to write a fresh entry.
func isDocumentedRevert(c git.Commit, documented map[string]bool) bool {
	if !isRevertCommit(c) {
		return false
	}
	shas := parseRevertedSHAs(c.Body)
	for _, sha := range shas {
		if !shaInSet(sha, documented) {
			return false
		}
	}
	return true
}

// shaInSet returns true when needle (which may be a short SHA) matches any
// SHA in the set by prefix. The set keys are full 40-char SHAs from entry
// worksets; the needle may be 7..40 chars from a revert trailer.
func shaInSet(needle string, set map[string]bool) bool {
	if set[needle] {
		return true
	}
	if len(needle) >= 40 {
		return false
	}
	for full := range set {
		if strings.HasPrefix(full, needle) {
			return true
		}
	}
	return false
}

// documentedSHASet builds a set of every commit SHA covered by any ledger
// entry's Workset.Commits. Used to short-circuit revert auto-skipping.
func (s *Storage) documentedSHASet() map[string]bool {
	entries, err := s.ListEntries()
	if err != nil || len(entries) == 0 {
		return nil
	}
	set := make(map[string]bool)
	for _, e := range entries {
		for _, sha := range e.Workset.Commits {
			set[sha] = true
		}
	}
	return set
}
