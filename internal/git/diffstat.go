// Package git — diffstat extraction and root-commit handling.
// Split out of commit.go to keep that file under the file-length-limit.
package git

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/gorewood/timbers/internal/output"
)

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
