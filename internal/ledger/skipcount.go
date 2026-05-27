package ledger

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gorewood/timbers/internal/git"
)

// debugEnvVar enables structured trace output from pending-detection
// paths when set to a truthy value. Useful for diagnosing "why is this
// commit pending / not pending" without modifying the call site.
const debugEnvVar = "TIMBERS_DEBUG"

// debugEnabled reports whether TIMBERS_DEBUG is set to a recognized truthy
// value (1, true, yes, on — case-insensitive, whitespace-trimmed).
func debugEnabled() bool {
	val := strings.TrimSpace(strings.ToLower(os.Getenv(debugEnvVar)))
	switch val {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// debugWriter returns the stderr target for debug traces, indirected so
// tests can substitute a buffer. Real production code uses os.Stderr.
var debugWriter = func() io.Writer { return os.Stderr }

// CountInfraSkippedSinceLatest returns the number of commits between the
// latest entry's anchor and HEAD that pending detection auto-skips: both
// infrastructure-only commits (.timbers/, .beads/, .timbersignore matches)
// and reverts of already-documented commits. The count uses the same
// predicates as filterByRules so the visibility surface and the gate stay
// in sync.
//
// Returns 0 when no entries exist, when the anchor is stale, or when any
// underlying git call fails — those cases are not actionable signal for
// the caller. Status visibility is best-effort by design.
func (s *Storage) CountInfraSkippedSinceLatest() (int, error) {
	commits, ok, err := s.commitsSinceLatest()
	if err != nil || !ok {
		return 0, err
	}
	return s.countAutoSkipped(commits), nil
}

// commitsSinceLatest returns commits in (latestAnchor..HEAD) when both ends
// are valid and reachable. ok=false signals "no actionable signal" for any
// non-error degraded case (no entries, stale anchor, log failure).
func (s *Storage) commitsSinceLatest() ([]git.Commit, bool, error) {
	latest, err := s.GetLatestEntry()
	if err != nil {
		if errors.Is(err, ErrNoEntries) {
			return nil, false, nil
		}
		return nil, false, err
	}

	head, err := s.git.HEAD()
	if err != nil {
		return nil, false, err
	}

	anchor := latest.Workset.AnchorCommit
	if !s.git.IsAncestorOf(anchor, head) {
		return nil, false, nil
	}

	commits, err := s.git.Log(anchor, head)
	if err != nil {
		return nil, false, nil //nolint:nilerr // stale-anchor counts are best-effort
	}
	return commits, true, nil
}

// countAutoSkipped returns how many of the given commits would be filtered
// from pending — covers infrastructure-only commits, author-glob matches,
// acked commits, and documented reverts. Uses the same predicate set as
// filterByRules.
func (s *Storage) countAutoSkipped(commits []git.Commit) int {
	if len(commits) == 0 {
		return 0
	}
	fileMap, err := s.git.CommitFilesMulti(commitSHAs(commits))
	if err != nil {
		return 0
	}
	rules := s.rulesOrDefault()
	docSet := s.documentedSHASet()
	ackedSet := s.AckedSet()
	var count int
	for _, commit := range commits {
		if isInfrastructureOnlyCommit(rules, fileMap[commit.SHA]) {
			count++
			continue
		}
		if matchesSkipAuthor(s.skipAuthors, commit.AuthorEmail, commit.Author) {
			count++
			continue
		}
		if docSet[commit.SHA] {
			count++
			continue
		}
		if ackedSet[commit.SHA] {
			count++
			continue
		}
		if isDocumentedRevert(commit, docSet) {
			count++
		}
	}
	return count
}

// commitSHAs extracts the SHA from each commit for batch lookups.
func commitSHAs(commits []git.Commit) []string {
	shas := make([]string, len(commits))
	for i, commit := range commits {
		shas[i] = commit.SHA
	}
	return shas
}

// isInfrastructureOnlyCommit returns true if every file in the list matches
// a skip rule (built-in defaults plus any patterns from .timbersignore).
// Returns false for empty lists (unknown = don't filter).
func isInfrastructureOnlyCommit(rules []skipRule, files []string) bool {
	if len(files) == 0 {
		return false
	}
	for _, f := range files {
		if !matchAny(rules, f) {
			return false
		}
	}
	return true
}

// rulesOrDefault returns the storage's skip rules, falling back to the
// built-in defaults when none have been loaded (e.g., bare-bones tests).
func (s *Storage) rulesOrDefault() []skipRule {
	if len(s.skipRules) == 0 {
		return compiledDefaultSkipRules
	}
	return s.skipRules
}

// filterByRules removes infrastructure-only commits, author-glob-matched
// commits, acked commits, and documented reverts from the input list,
// preserving order. The docSet and ackedSet are supplied by the caller
// so callers can build them once and reuse — avoids re-scanning the
// ledger on every invocation. Pass nil docSet to disable revert
// auto-skipping; pass nil ackedSet to disable ack-based skipping.
func (s *Storage) filterByRules(
	commits []git.Commit,
	fileMap map[string][]string,
	docSet map[string]bool,
	ackedSet map[string]bool,
) []git.Commit {
	rules := s.rulesOrDefault()
	filtered := make([]git.Commit, 0, len(commits))
	for _, commit := range commits {
		// A commit is dropped if it's infrastructure-only (all files match
		// skip rules) or matches any identity-based skip (author/message
		// glob, direct docSet membership, ack, documented revert).
		// classifyByIdentity encodes that chain; reusing it keeps this loop
		// flat and the skip semantics single-sourced with the debug trace.
		if isInfrastructureOnlyCommit(rules, fileMap[commit.SHA]) {
			continue
		}
		if classifyByIdentity(commit, docSet, ackedSet, s.skipAuthors, s.skipMessages) != "" {
			continue
		}
		filtered = append(filtered, commit)
	}
	return filtered
}

// classifyCommit reports the first applicable skip reason for the commit
// under the current rules, or "" if the commit is kept. Used for both
// filtering (when the caller doesn't care about the reason) and the
// TIMBERS_DEBUG trace (where the reason is the whole point).
func (s *Storage) classifyCommit(
	commit git.Commit,
	fileMap map[string][]string,
	docSet, ackedSet map[string]bool,
	gateStrict bool,
) string {
	rules := s.rulesOrDefault()
	files := fileMap[commit.SHA]
	if isInfrastructureOnlyCommit(rules, files) {
		return "infra"
	}
	if reason := classifyByIdentity(commit, docSet, ackedSet, s.skipAuthors, s.skipMessages); reason != "" {
		return reason
	}
	return classifyByContent(commit, files, gateStrict)
}

// classifyByIdentity checks author globs, commit-subject globs, direct
// docSet membership, ack records, and revert relationships. Returns the
// first matching reason or "" if none match.
func classifyByIdentity(
	commit git.Commit,
	docSet, ackedSet map[string]bool,
	skipAuthors, skipMessages []string,
) string {
	if matchesSkipAuthor(skipAuthors, commit.AuthorEmail, commit.Author) {
		return "author"
	}
	if matchesSkipMessage(skipMessages, commit.Subject) {
		return "message"
	}
	if docSet != nil && docSet[commit.SHA] {
		return "documented"
	}
	if ackedSet != nil && ackedSet[commit.SHA] {
		return "ack"
	}
	if docSet != nil && isDocumentedRevert(commit, docSet) {
		return "revert"
	}
	return ""
}

// classifyByContent checks for empty-file-list commits (clean merges,
// --allow-empty markers). gateStrict controls whether non-merge empty
// commits are dropped (gate path) or kept visible (display path).
func classifyByContent(commit git.Commit, files []string, gateStrict bool) string {
	if len(files) > 0 {
		return ""
	}
	if commit.IsMerge() {
		return "merge-empty"
	}
	if gateStrict {
		return "empty"
	}
	return ""
}

// traceFilterDecisions prints per-commit keep/skip classification to
// stderr when TIMBERS_DEBUG is enabled. Called from filterCommits after
// the fileMap has been fetched so the trace and the production filter
// see the same data.
func (s *Storage) traceFilterDecisions(
	commits []git.Commit,
	fileMap map[string][]string,
	docSet map[string]bool,
	ackedSet map[string]bool,
	gateStrict bool,
) {
	if !debugEnabled() {
		return
	}
	path := "display"
	if gateStrict {
		path = "gate"
	}
	w := debugWriter()
	_, _ = fmt.Fprintf(w, "[timbers] debug: path=%s raw=%d\n", path, len(commits))
	counts := map[string]int{}
	for _, commit := range commits {
		reason := s.classifyCommit(commit, fileMap, docSet, ackedSet, gateStrict)
		if reason == "" {
			_, _ = fmt.Fprintf(w, "[timbers] debug:   %s keep\n", commit.Short)
		} else {
			_, _ = fmt.Fprintf(w, "[timbers] debug:   %s skip %s\n", commit.Short, reason)
			counts[reason]++
		}
	}
	if len(counts) > 0 {
		_, _ = fmt.Fprintf(w, "[timbers] debug: dropped=%s\n", formatDropCounts(counts))
	}
}

// formatDropCounts renders a map of reason→count as "reason:N,reason:N"
// in a stable order for parseability.
func formatDropCounts(counts map[string]int) string {
	order := []string{"infra", "author", "message", "documented", "ack", "revert", "merge-empty", "empty"}
	parts := make([]string, 0, len(counts))
	for _, k := range order {
		if n, ok := counts[k]; ok && n > 0 {
			parts = append(parts, fmt.Sprintf("%s:%d", k, n))
		}
	}
	return strings.Join(parts, ",")
}

// filterCommits removes commits that don't represent pending work:
//   - Infrastructure-only commits (.timbers/, .beads/, .timbersignore matches)
//   - Reverts of already-documented commits (parsed from "This reverts commit <sha>")
//   - Clean merge commits (2+ parents AND empty combined diff) — added by
//     a routine `git merge --no-ff branch-y` that brought in sibling work
//     but added nothing on this branch's first-parent line. Dropped from
//     both gate and display paths so pending output matches the gate.
//   - When gateStrict is true: also non-merge commits with empty file lists
//     (i.e., --allow-empty marker commits). The display keeps these so
//     intentionally-empty commits remain visible to the user; the gate
//     drops them because they're not actionable debt.
//
// On git lookup error, returns all commits unfiltered (safe default).
// docSet is supplied by the caller so a single ListEntries scan can feed
// both latest-entry resolution and revert auto-skipping.
func (s *Storage) filterCommits(commits []git.Commit, docSet, ackedSet map[string]bool, gateStrict bool) []git.Commit {
	if len(commits) == 0 {
		return commits
	}
	fileMap, err := s.git.CommitFilesMulti(commitSHAs(commits))
	if err != nil {
		return commits
	}
	s.traceFilterDecisions(commits, fileMap, docSet, ackedSet, gateStrict)
	filtered := s.filterByRules(commits, fileMap, docSet, ackedSet)
	if gateStrict {
		// Gate: drop ALL empty-file commits (merges with clean combined
		// diff AND non-merge --allow-empty commits). Strict — no content
		// means no actionable debt for the current branch.
		return dropEmptyFileChanges(filtered, fileMap)
	}
	// Display: drop only empty-file MERGE commits. Single-parent empty
	// commits (--allow-empty) are preserved because they're intentional
	// and the user may want to see them in pending.
	return dropEmptyMerges(filtered, fileMap)
}
