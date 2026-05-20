package ledger

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// timbersIgnoreFilename is the per-repo skip-rule extension file. It lives
// at the repo root (peer to .timbers/) following the convention established
// by .gitignore, .dockerignore, .npmignore, etc.
const timbersIgnoreFilename = ".timbersignore"

// authorLinePrefix marks a .timbersignore line as an author glob (matched
// against commit.AuthorEmail and commit.Author via filepath.Match)
// instead of a path pattern. Chosen for explicit self-documentation —
// readers can tell at a glance which lines target paths vs. authors.
const authorLinePrefix = "author:"

// loadSkipConfig returns the effective skip-rule set and author-glob set
// for a given repo root, parsed from <repoRoot>/.timbersignore. A missing
// file is not an error. The built-in default rules are always merged into
// the result; user-supplied patterns extend the defaults.
//
// File format: one entry per line. Lines starting with "#" are comments;
// inline " #" trailers are stripped. Two entry shapes:
//   - "<pattern>"        — path skip rule (existing behavior)
//   - "author:<glob>"    — author skip rule (matched against email + name)
//
// Path patterns follow the skipRule grammar: trailing "/" = directory
// prefix, leading "*" = suffix match, otherwise exact path. Author globs
// use filepath.Match semantics (*, ?, character classes).
//
// Edge case: a path that literally starts with "author:" (e.g. a file
// named "author:notes.txt") cannot be expressed as a path rule because
// the prefix is reserved. Such filenames are exceedingly rare in real
// repos (Windows forbids ':' in filenames; most tooling treats ':' as
// special), and the syntax was chosen so common paths like "author/" do
// NOT collide (the prefix requires a literal ':' after "author"). If you
// hit this, use a parent-directory rule instead.
func loadSkipConfig(repoRoot string) ([]skipRule, []string, error) {
	rules := make([]skipRule, 0, len(compiledDefaultSkipRules))
	rules = append(rules, compiledDefaultSkipRules...)

	if repoRoot == "" {
		return rules, nil, nil
	}

	patterns, authors, err := readTimbersIgnore(filepath.Join(repoRoot, timbersIgnoreFilename))
	if err != nil {
		return rules, nil, err
	}
	rules = append(rules, compileSkipRules(patterns)...)
	return rules, authors, nil
}

// readTimbersIgnore reads and parses a .timbersignore file at the given path.
// Returns (paths, authors, error). Author entries are lines prefixed with
// "author:"; everything else is a path pattern. Returns empty slices (no
// error) if the file does not exist.
func readTimbersIgnore(path string) (paths, authors []string, err error) {
	file, openErr := os.Open(path) //nolint:gosec // path is composed from trusted .timbers/ root
	if openErr != nil {
		if errors.Is(openErr, fs.ErrNotExist) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("read %s: %w", timbersIgnoreFilename, openErr)
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		kind, value := classifyTimbersIgnoreLine(scanner.Text())
		switch kind {
		case ignoreLineSkip:
			// comment, blank, or malformed — drop silently
		case ignoreLinePath:
			paths = append(paths, value)
		case ignoreLineAuthor:
			authors = append(authors, value)
		}
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return nil, nil, fmt.Errorf("scan %s: %w", timbersIgnoreFilename, scanErr)
	}
	return paths, authors, nil
}

// ignoreLineKind tags how a .timbersignore line should be consumed.
type ignoreLineKind int

const (
	ignoreLineSkip   ignoreLineKind = iota // blank, comment, empty after trim, or malformed entry
	ignoreLinePath                         // path pattern (no prefix)
	ignoreLineAuthor                       // "author:<glob>" entry
)

// classifyTimbersIgnoreLine parses a single .timbersignore line and
// returns its kind plus the cleaned value (pattern or glob). Comments,
// blanks, and malformed entries collapse to ignoreLineSkip with an empty
// value. Extracted so readTimbersIgnore stays under the cognitive-
// complexity budget and so the per-line classification rules are
// independently testable.
func classifyTimbersIgnoreLine(raw string) (ignoreLineKind, string) {
	line := strings.TrimSpace(raw)
	if line == "" || strings.HasPrefix(line, "#") {
		return ignoreLineSkip, ""
	}
	// Strip inline comments (everything after a # preceded by whitespace).
	// Keeps "*.lock" intact while allowing "vendor/  # libs" trailers.
	if idx := indexInlineComment(line); idx >= 0 {
		line = strings.TrimSpace(line[:idx])
	}
	if line == "" {
		return ignoreLineSkip, ""
	}
	rest, isAuthor := strings.CutPrefix(line, authorLinePrefix)
	if !isAuthor {
		return ignoreLinePath, line
	}
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return ignoreLineSkip, ""
	}
	// Drop globs that filepath.Match rejects so a malformed entry can
	// never break pending detection — same posture as path rules.
	if _, matchErr := filepath.Match(rest, ""); matchErr != nil {
		return ignoreLineSkip, ""
	}
	return ignoreLineAuthor, rest
}

// indexInlineComment returns the index of the first '#' preceded by whitespace,
// or -1 if no inline comment is present. Patterns themselves cannot contain '#'.
func indexInlineComment(s string) int {
	for i := 1; i < len(s); i++ {
		if s[i] == '#' && (s[i-1] == ' ' || s[i-1] == '\t') {
			return i
		}
	}
	return -1
}
