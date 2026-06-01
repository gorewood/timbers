package ledger

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
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

// messageLinePrefix marks a .timbersignore line as a commit-subject glob
// (matched against commit.Subject via filepath.Match) instead of a path
// pattern. Skips housekeeping commits whose files aren't all path-skippable
// — e.g. a release changelog commit that also touches a version-badge file.
const messageLinePrefix = "msg:"

// sessionWindowLinePrefix marks a .timbersignore line that overrides the
// cross-agent debt classifier's staleness window. Single-valued (last
// occurrence wins). Format: "session-window: 4h" — the value is parsed
// via Go's time.ParseDuration so the supported grammar is "4h", "2h30m",
// "15m", "90m". Day suffixes ("1d") and capitalized hour suffixes ("4H")
// are NOT supported; a parse failure falls back to DefaultSessionWindow
// with a diagnostic surfaced via timbers doctor.
const sessionWindowLinePrefix = "session-window:"

// loadSkipConfig returns the effective skip-rule set, author-glob set, and
// commit-subject-glob set for a given repo root, parsed from
// <repoRoot>/.timbersignore. A missing file is not an error. The built-in
// default rules are always merged into the result; user-supplied patterns
// extend the defaults.
//
// File format: one entry per line. Lines starting with "#" are comments;
// inline " #" trailers are stripped. Three entry shapes:
//   - "<pattern>"        — path skip rule (existing behavior)
//   - "author:<glob>"    — author skip rule (matched against email + name)
//   - "msg:<glob>"       — commit-subject skip rule (matched against Subject)
//
// Path patterns follow the skipRule grammar: trailing "/" = directory
// prefix, leading "*" = suffix match, otherwise exact path. Author and
// message globs use filepath.Match semantics (*, ?, character classes).
//
// Edge case: a path that literally starts with "author:" or "msg:" (e.g. a
// file named "msg:notes.txt") cannot be expressed as a path rule because
// the prefix is reserved. Such filenames are exceedingly rare in real
// repos (Windows forbids ':' in filenames; most tooling treats ':' as
// special), and the syntax was chosen so common paths like "msg/" do NOT
// collide (the prefix requires a literal ':' after "msg"). If you hit this,
// use a parent-directory rule instead.
func loadSkipConfig(repoRoot string) ([]skipRule, []string, []string, error) {
	rules := make([]skipRule, 0, len(compiledDefaultSkipRules))
	rules = append(rules, compiledDefaultSkipRules...)

	if repoRoot == "" {
		return rules, nil, nil, nil
	}

	patterns, authors, messages, err := readTimbersIgnore(filepath.Join(repoRoot, timbersIgnoreFilename))
	if err != nil {
		return rules, nil, nil, err
	}
	rules = append(rules, compileSkipRules(patterns)...)
	return rules, authors, messages, nil
}

// LoadIgnoreGlobs returns the author and message globs configured in
// <repoRoot>/.timbersignore. The built-in defaults carry none, so an empty
// result means the repo has no author/message rules. A missing file is not an
// error. Exposed for diagnostics (e.g. timbers doctor's glob lint).
func LoadIgnoreGlobs(repoRoot string) (authors, messages []string, err error) {
	_, authors, messages, err = loadSkipConfig(repoRoot)
	return authors, messages, err
}

// SessionWindowResult reports the staleness window for the cross-agent debt
// classifier as configured in <repoRoot>/.timbersignore. The Window field is
// the value the classifier should use — when no directive is present or the
// directive value is malformed, Window is DefaultSessionWindow and the
// caller does not need a separate fallback. Raw and ParseErr are populated
// for doctor-style diagnostics so the operator can see what they configured
// and why it didn't take.
type SessionWindowResult struct {
	Window   time.Duration // effective window (default if missing/malformed)
	Raw      string        // exact directive value as authored, "" if no directive
	ParseErr error         // non-nil when Raw was supplied but failed to parse
}

// LoadSessionWindow scans <repoRoot>/.timbersignore for a session-window:
// directive and returns the result. A missing file or missing directive
// returns Window = DefaultSessionWindow with empty Raw and nil ParseErr —
// the caller treats that as "use the default, no diagnostic." A present-
// but-malformed directive returns Window = DefaultSessionWindow with Raw
// set and ParseErr non-nil — the caller should still use Window for the
// classifier but surface Raw/ParseErr through doctor.
//
// Last occurrence wins if the file lists multiple session-window lines.
// This matches how git config handles repeated keys (we expect operators
// to interleave overrides above defaults in the same file rarely).
func LoadSessionWindow(repoRoot string) SessionWindowResult {
	result := SessionWindowResult{Window: DefaultSessionWindow}
	if repoRoot == "" {
		return result
	}
	file, openErr := os.Open(filepath.Join(repoRoot, timbersIgnoreFilename)) //nolint:gosec // path is composed from trusted root
	if openErr != nil {
		return result
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		raw, ok := extractSessionWindowDirective(scanner.Text())
		if !ok {
			continue
		}
		result.Raw = raw
		parsed, parseErr := time.ParseDuration(raw)
		if parseErr != nil || parsed <= 0 {
			result.ParseErr = parseErr
			result.Window = DefaultSessionWindow
			continue
		}
		result.Window = parsed
		result.ParseErr = nil
	}
	return result
}

// extractSessionWindowDirective parses a single .timbersignore line and
// returns the directive value (trimmed) when the line is a session-window
// directive. Comments, blanks, and non-directive lines return ok=false.
func extractSessionWindowDirective(raw string) (string, bool) {
	line := strings.TrimSpace(raw)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", false
	}
	if idx := indexInlineComment(line); idx >= 0 {
		line = strings.TrimSpace(line[:idx])
	}
	rest, ok := strings.CutPrefix(line, sessionWindowLinePrefix)
	if !ok {
		return "", false
	}
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return "", false
	}
	return rest, true
}

// readTimbersIgnore reads and parses a .timbersignore file at the given path.
// Returns (paths, authors, messages, error). Author entries are lines prefixed
// with "author:", message entries with "msg:"; everything else is a path
// pattern. Returns empty slices (no error) if the file does not exist.
func readTimbersIgnore(path string) (paths, authors, messages []string, err error) {
	file, openErr := os.Open(path) //nolint:gosec // path is composed from trusted .timbers/ root
	if openErr != nil {
		if errors.Is(openErr, fs.ErrNotExist) {
			return nil, nil, nil, nil
		}
		return nil, nil, nil, fmt.Errorf("read %s: %w", timbersIgnoreFilename, openErr)
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
		case ignoreLineMessage:
			messages = append(messages, value)
		}
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return nil, nil, nil, fmt.Errorf("scan %s: %w", timbersIgnoreFilename, scanErr)
	}
	return paths, authors, messages, nil
}

// ignoreLineKind tags how a .timbersignore line should be consumed.
type ignoreLineKind int

const (
	ignoreLineSkip    ignoreLineKind = iota // blank, comment, empty after trim, or malformed entry
	ignoreLinePath                          // path pattern (no prefix)
	ignoreLineAuthor                        // "author:<glob>" entry
	ignoreLineMessage                       // "msg:<glob>" entry
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
	if rest, isAuthor := strings.CutPrefix(line, authorLinePrefix); isAuthor {
		return classifyGlobLine(ignoreLineAuthor, rest)
	}
	if rest, isMsg := strings.CutPrefix(line, messageLinePrefix); isMsg {
		return classifyGlobLine(ignoreLineMessage, rest)
	}
	// session-window: directives are owned by LoadSessionWindow (a separate
	// pass over the file). Recognize the prefix here so the line is not
	// misread as a path skip rule, but skip past it without adding to any
	// glob list.
	if strings.HasPrefix(line, sessionWindowLinePrefix) {
		return ignoreLineSkip, ""
	}
	return ignoreLinePath, line
}

// classifyGlobLine validates a glob-bearing line (author: or msg:) and
// returns the given kind with the cleaned glob, or ignoreLineSkip if the
// glob is empty or rejected by filepath.Match. Dropping malformed globs
// means a bad entry can never break pending detection — same posture as
// path rules.
func classifyGlobLine(kind ignoreLineKind, rest string) (ignoreLineKind, string) {
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return ignoreLineSkip, ""
	}
	if _, matchErr := filepath.Match(rest, ""); matchErr != nil {
		return ignoreLineSkip, ""
	}
	return kind, rest
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
