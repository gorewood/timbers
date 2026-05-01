package ledger

import (
	"regexp"
	"strings"

	"github.com/gorewood/timbers/internal/git"
)

// revertSubjectPrefix identifies commits created by `git revert`.
const revertSubjectPrefix = `Revert "`

// minRevertSHALen is the minimum SHA length we accept in a revert trailer.
// Git's default `git revert` writes the full 40-char SHA, but hand-edited
// revert messages occasionally use abbreviated SHAs. We refuse anything
// under 12 chars: a 7-char prefix has ~1-in-16M collision odds, which can
// realistically cause a false-positive cross-reference in busy repos and
// silently auto-skip an undocumented revert. 12 chars is git's default
// short-SHA width for large repositories.
const minRevertSHALen = 12

// revertedCommitRE matches the body trailer git revert produces. The SHA
// width window (12..40 hex chars) is enforced here so downstream matchers
// don't have to revalidate.
var revertedCommitRE = regexp.MustCompile(`(?m)^This reverts commit ([0-9a-f]{12,40})\b`)

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

// shaInSet returns true when needle matches any SHA in the set, either
// exactly or as a prefix of a full SHA. Needle width is constrained to
// [minRevertSHALen, 40] by the revert-trailer regex; that bound gives us
// negligible collision probability while tolerating abbreviated SHAs that
// hand-edited revert messages sometimes use.
func shaInSet(needle string, set map[string]bool) bool {
	if set[needle] {
		return true
	}
	if len(needle) >= 40 || len(needle) < minRevertSHALen {
		return false
	}
	for full := range set {
		if strings.HasPrefix(full, needle) {
			return true
		}
	}
	return false
}

// documentedSHASetFromEntries builds a set of every commit SHA covered by
// any ledger entry's Workset.Commits. Pure function — accepts a pre-loaded
// entry slice so callers can list entries once and reuse.
func documentedSHASetFromEntries(entries []*Entry) map[string]bool {
	if len(entries) == 0 {
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

// documentedSHASet lists entries from disk and builds the set. Convenience
// wrapper for callers that don't already have an entry slice in hand.
func (s *Storage) documentedSHASet() map[string]bool {
	entries, err := s.ListEntries()
	if err != nil {
		return nil
	}
	return documentedSHASetFromEntries(entries)
}
