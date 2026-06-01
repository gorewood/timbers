// Package git provides Git operations via exec for the timbers CLI.
package git

import (
	"bytes"
	"context"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gorewood/timbers/internal/output"
)

// Commit represents a git commit with its metadata.
type Commit struct {
	SHA         string    // Full 40-character SHA
	Short       string    // Abbreviated SHA (typically 7 chars)
	Subject     string    // First line of commit message
	Body        string    // Rest of commit message (may be empty)
	Author      string    // Author name
	AuthorEmail string    // Author email, mailmap-resolved (.mailmap coalesces alternate emails for the same person)
	Date        time.Time // AuthorDate — when the commit was originally authored; preserved across rebase/amend
	CommitDate  time.Time // CommitDate — when the commit was recorded on the current DAG; advances on rebase/amend
	ParentCount int       // Number of parents (0=root, 1=normal, 2+=merge)
}

// IsMerge reports whether the commit is a merge commit (2+ parents).
func (c Commit) IsMerge() bool {
	return c.ParentCount >= 2
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
// Walks the full DAG — merge commits are visited and their second parents
// are followed, so commits brought in by a merge appear in the result.
func Log(fromRef, toRef string) ([]Commit, error) {
	return logRange(fromRef, toRef, false)
}

// LogFirstParent returns commits in the given range (fromRef..toRef) using
// first-parent traversal. The 'fromRef' ref is exclusive, 'toRef' is inclusive.
//
// First-parent traversal follows only the first parent of each merge commit,
// so commits brought in by a merge are NOT visited. This corresponds to the
// linear history of the current branch — useful for "what work happened on
// this branch?" without picking up commits authored elsewhere and merged in.
func LogFirstParent(fromRef, toRef string) ([]Commit, error) {
	return logRange(fromRef, toRef, true)
}

// logRange is the shared implementation for Log and LogFirstParent.
func logRange(fromRef, toRef string, firstParent bool) ([]Commit, error) {
	rangeSpec := fromRef + ".." + toRef
	args := []string{"log", "--pretty=format:" + commitFormat()}
	if firstParent {
		args = append(args, "--first-parent")
	}
	args = append(args, rangeSpec)

	out, err := Run(args...)
	if err != nil {
		return nil, output.NewSystemErrorWithCause("failed to get git log for range "+rangeSpec, err)
	}

	return parseCommits(out)
}

// commitFormat returns the git log --pretty=format string used by Log,
// LogFirstParent, and CommitsReachableFrom. Centralized so the field
// order stays in sync with parseCommitFields.
//
// Uses %aE (mailmap-resolved author email) so a repo's .mailmap coalesces
// alternate emails for the same operator — the canonical mechanism for
// "same person, multiple emails." Without mailmap, %aE returns the raw
// author email unchanged.
//
// Emits both %at (AuthorDate) and %ct (CommitDate). The two diverge on
// rebase and amend: AuthorDate is preserved, CommitDate advances. Callers
// that care about "when did this commit hit *this* DAG?" (provenance /
// session staleness) must use CommitDate, not AuthorDate.
func commitFormat() string {
	return strings.Join([]string{
		"%H",  // Full SHA
		"%h",  // Short SHA
		"%s",  // Subject
		"%b",  // Body
		"%an", // Author name
		"%aE", // Author email, mailmap-resolved
		"%at", // AuthorDate (Unix timestamp) — preserved across rebase/amend
		"%ct", // CommitDate (Unix timestamp) — advances on rebase/amend
		"%P",  // Parent SHAs (space-separated; empty for root commit)
	}, fieldSeparator) + commitSeparator
}

// CommitsReachableFrom returns all commits reachable from the given ref.
// Commits are returned in reverse chronological order (newest first).
func CommitsReachableFrom(sha string) ([]Commit, error) {
	out, err := Run("log", "--pretty=format:"+commitFormat(), sha)
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
//
// Field order must match commitFormat:
//
//	0: %H   full SHA
//	1: %h   short SHA
//	2: %s   subject
//	3: %b   body
//	4: %an  author name
//	5: %aE  author email (mailmap-resolved)
//	6: %at  AuthorDate (Unix)
//	7: %ct  CommitDate (Unix)
//	8: %P   parent SHAs
func parseCommitFields(commitStr string) (Commit, bool) {
	fields := strings.Split(commitStr, fieldSeparator)
	if len(fields) < 9 {
		return Commit{}, false
	}

	authorTS, err := strconv.ParseInt(strings.TrimSpace(fields[6]), 10, 64)
	if err != nil {
		authorTS = 0
	}
	commitTS, err := strconv.ParseInt(strings.TrimSpace(fields[7]), 10, 64)
	if err != nil {
		commitTS = 0
	}

	// Count parent SHAs (space-separated; empty string = 0 parents = root commit).
	parentField := strings.TrimSpace(fields[8])
	parentCount := 0
	if parentField != "" {
		parentCount = len(strings.Fields(parentField))
	}

	return Commit{
		SHA:         strings.TrimSpace(fields[0]),
		Short:       strings.TrimSpace(fields[1]),
		Subject:     strings.TrimSpace(fields[2]),
		Body:        strings.TrimSpace(fields[3]),
		Author:      strings.TrimSpace(fields[4]),
		AuthorEmail: strings.TrimSpace(fields[5]),
		Date:        time.Unix(authorTS, 0),
		CommitDate:  time.Unix(commitTS, 0),
		ParentCount: parentCount,
	}, true
}

// CommitFiles returns the list of files changed by the given commit.
func CommitFiles(sha string) ([]string, error) {
	out, err := Run("diff-tree", "--no-commit-id", "--name-only", "-r", sha)
	if err != nil {
		return nil, output.NewSystemErrorWithCause("failed to get files for commit "+sha, err)
	}
	if out == "" {
		return nil, nil
	}
	var files []string
	for line := range strings.SplitSeq(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

// CommitFilesMulti returns the files changed by each commit using a single git process.
// Uses git diff-tree --stdin for batch processing instead of one subprocess per commit.
// Returns a map from full SHA to file list. Commits with no changed files get a nil slice.
func CommitFilesMulti(shas []string) (map[string][]string, error) {
	if len(shas) == 0 {
		return make(map[string][]string), nil
	}

	input := strings.Join(shas, "\n") + "\n"
	cmd := exec.CommandContext(context.Background(), "git", "diff-tree", "-r", "--name-only", "--stdin")
	cmd.Stdin = strings.NewReader(input)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		return nil, output.NewSystemErrorWithCause("git diff-tree --stdin failed: "+errMsg, err)
	}

	// Build lookup set for input SHAs
	shaSet := make(map[string]bool, len(shas))
	for _, sha := range shas {
		shaSet[sha] = true
	}

	// Parse output: each commit SHA appears on its own line, followed by changed files.
	result := make(map[string][]string, len(shas))
	var current string
	for line := range strings.SplitSeq(strings.TrimSpace(stdout.String()), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if shaSet[line] {
			current = line
			if _, ok := result[current]; !ok {
				result[current] = nil
			}
		} else if current != "" {
			result[current] = append(result[current], line)
		}
	}

	return result, nil
}

// DiffNameOnly returns file paths changed between fromRef and toRef,
// optionally filtered to a path prefix.
// Uses git diff --name-only fromRef..toRef -- [pathPrefix].
func DiffNameOnly(fromRef, toRef, pathPrefix string) ([]string, error) {
	args := []string{"diff", "--name-only", fromRef + ".." + toRef}
	if pathPrefix != "" {
		args = append(args, "--", pathPrefix)
	}
	out, err := Run(args...)
	if err != nil {
		return nil, output.NewSystemErrorWithCause("failed to get diff for range "+fromRef+".."+toRef, err)
	}
	if out == "" {
		return nil, nil
	}
	var files []string
	for line := range strings.SplitSeq(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}
