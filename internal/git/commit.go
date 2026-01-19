// Package git provides Git operations via exec for the timbers CLI.
package git

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rbergman/timbers/internal/output"
)

// Commit represents a git commit with its metadata.
type Commit struct {
	SHA         string    // Full 40-character SHA
	Short       string    // Abbreviated SHA (typically 7 chars)
	Subject     string    // First line of commit message
	Body        string    // Rest of commit message (may be empty)
	Author      string    // Author name
	AuthorEmail string    // Author email
	Date        time.Time // Commit date
}

// Diffstat represents the change statistics for a range of commits.
type Diffstat struct {
	Files      int // Number of files changed
	Insertions int // Number of lines inserted
	Deletions  int // Number of lines deleted
}

// commitSeparator is used to delimit commits in log output.
const commitSeparator = "---COMMIT-BOUNDARY---"

// fieldSeparator is used to delimit fields within a commit.
const fieldSeparator = "---FIELD---"

// Log returns commits in the given range (fromRef..toRef).
// The 'fromRef' ref is exclusive, 'toRef' is inclusive.
func Log(fromRef, toRef string) ([]Commit, error) {
	// Use custom format to parse commits reliably
	// Format: SHA, Short, Subject, Body, Author, AuthorEmail, Date (Unix timestamp)
	format := strings.Join([]string{
		"%H",  // Full SHA
		"%h",  // Short SHA
		"%s",  // Subject
		"%b",  // Body
		"%an", // Author name
		"%ae", // Author email
		"%at", // Unix timestamp
	}, fieldSeparator) + commitSeparator

	rangeSpec := fromRef + ".." + toRef
	out, err := Run("log", "--pretty=format:"+format, rangeSpec)
	if err != nil {
		return nil, output.NewSystemErrorWithCause("failed to get git log for range "+rangeSpec, err)
	}

	return parseCommits(out)
}

// CommitsReachableFrom returns all commits reachable from the given ref.
// Commits are returned in reverse chronological order (newest first).
func CommitsReachableFrom(sha string) ([]Commit, error) {
	format := strings.Join([]string{
		"%H",  // Full SHA
		"%h",  // Short SHA
		"%s",  // Subject
		"%b",  // Body
		"%an", // Author name
		"%ae", // Author email
		"%at", // Unix timestamp
	}, fieldSeparator) + commitSeparator

	out, err := Run("log", "--pretty=format:"+format, sha)
	if err != nil {
		return nil, output.NewSystemErrorWithCause("failed to get commits from "+sha, err)
	}

	return parseCommits(out)
}

// parseCommits parses the custom formatted git log output into Commit structs.
func parseCommits(out string) ([]Commit, error) {
	if out == "" {
		return nil, nil
	}

	// Split by commit boundary
	commitStrs := strings.Split(out, commitSeparator)
	var commits []Commit

	for _, commitStr := range commitStrs {
		commitStr = strings.TrimSpace(commitStr)
		if commitStr == "" {
			continue
		}

		commit, ok := parseCommitFields(commitStr)
		if ok {
			commits = append(commits, commit)
		}
	}

	return commits, nil
}

// parseCommitFields parses a single commit string into a Commit struct.
// Returns the commit and true if successful, zero value and false otherwise.
func parseCommitFields(commitStr string) (Commit, bool) {
	fields := strings.Split(commitStr, fieldSeparator)
	if len(fields) < 7 {
		return Commit{}, false
	}

	// Parse Unix timestamp
	timestamp, err := strconv.ParseInt(strings.TrimSpace(fields[6]), 10, 64)
	if err != nil {
		timestamp = 0
	}

	return Commit{
		SHA:         strings.TrimSpace(fields[0]),
		Short:       strings.TrimSpace(fields[1]),
		Subject:     strings.TrimSpace(fields[2]),
		Body:        strings.TrimSpace(fields[3]),
		Author:      strings.TrimSpace(fields[4]),
		AuthorEmail: strings.TrimSpace(fields[5]),
		Date:        time.Unix(timestamp, 0),
	}, true
}

// diffstatLineRegex matches the summary line of git diff --stat
// Example: " 3 files changed, 45 insertions(+), 12 deletions(-)"
var diffstatLineRegex = regexp.MustCompile(`(\d+)\s+files?\s+changed(?:,\s+(\d+)\s+insertions?\(\+\))?(?:,\s+(\d+)\s+deletions?\(-\))?`)

// emptyTreeSHA is the SHA of git's empty tree object.
// Used when diffing from a root commit (which has no parent).
const emptyTreeSHA = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

// GetDiffstat returns the change statistics for the given commit range.
// The 'fromRef' ref is exclusive, 'toRef' is inclusive.
// If fromRef doesn't exist (e.g., parent of root commit), uses empty tree.
func GetDiffstat(fromRef, toRef string) (Diffstat, error) {
	resolvedFrom := resolveRefOrEmptyTree(fromRef)
	rangeSpec := resolvedFrom + ".." + toRef
	out, err := Run("diff", "--stat", rangeSpec)
	if err != nil {
		return Diffstat{}, output.NewSystemErrorWithCause("failed to get diffstat for range "+rangeSpec, err)
	}

	return parseDiffstat(out), nil
}

// resolveRefOrEmptyTree resolves a ref, returning empty tree SHA if it doesn't exist.
// This handles the case of "SHA^" for root commits.
func resolveRefOrEmptyTree(ref string) string {
	if ref == "" {
		return emptyTreeSHA
	}
	_, err := Run("rev-parse", "--verify", "--quiet", ref)
	if err != nil {
		return emptyTreeSHA
	}
	return ref
}

// parseDiffstat extracts file, insertion, and deletion counts from git diff --stat output.
func parseDiffstat(out string) Diffstat {
	summaryLine := findSummaryLine(out)
	if summaryLine == "" {
		return Diffstat{}
	}

	return extractDiffstatFromSummary(summaryLine)
}

// findSummaryLine finds the last non-empty line in the diff stat output.
func findSummaryLine(out string) string {
	lines := strings.Split(out, "\n")
	for idx := len(lines) - 1; idx >= 0; idx-- {
		line := strings.TrimSpace(lines[idx])
		if line != "" {
			return line
		}
	}
	return ""
}

// parseMatchInt extracts an int from a regex match group, returning 0 on error.
func parseMatchInt(matches []string, idx int) int {
	if idx >= len(matches) || matches[idx] == "" {
		return 0
	}
	val, err := strconv.Atoi(matches[idx])
	if err != nil {
		return 0
	}
	return val
}

// extractDiffstatFromSummary parses the diffstat summary line using regex.
func extractDiffstatFromSummary(summaryLine string) Diffstat {
	matches := diffstatLineRegex.FindStringSubmatch(summaryLine)
	if matches == nil {
		return Diffstat{}
	}

	return Diffstat{
		Files:      parseMatchInt(matches, 1),
		Insertions: parseMatchInt(matches, 2),
		Deletions:  parseMatchInt(matches, 3),
	}
}
