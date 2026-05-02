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

// loadSkipRules returns the effective skip-rule set for a given repo root.
// It returns the built-in defaults merged with any patterns declared in
// <repoRoot>/.timbersignore (if the file exists). A missing file is not an error.
//
// File format: one pattern per line. Lines starting with "#" are comments.
// Trailing whitespace and blank lines are ignored. Patterns follow the
// skipRule grammar: trailing "/" = directory prefix, leading "*" = suffix
// match, otherwise exact path.
func loadSkipRules(repoRoot string) ([]skipRule, error) {
	rules := make([]skipRule, 0, len(compiledDefaultSkipRules))
	rules = append(rules, compiledDefaultSkipRules...)

	if repoRoot == "" {
		return rules, nil
	}

	patterns, err := readTimbersIgnore(filepath.Join(repoRoot, timbersIgnoreFilename))
	if err != nil {
		return rules, err
	}
	rules = append(rules, compileSkipRules(patterns)...)
	return rules, nil
}

// readTimbersIgnore reads and parses a .timbersignore file at the given path.
// Returns an empty slice (no error) if the file does not exist.
func readTimbersIgnore(path string) ([]string, error) {
	file, err := os.Open(path) //nolint:gosec // path is composed from trusted .timbers/ root
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", timbersIgnoreFilename, err)
	}
	defer func() { _ = file.Close() }()

	var patterns []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Strip inline comments (everything after a # preceded by whitespace).
		// This keeps "*.lock" patterns intact while allowing "vendor/  # libs" trailers.
		if idx := indexInlineComment(line); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}
		if line == "" {
			continue
		}
		patterns = append(patterns, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", timbersIgnoreFilename, err)
	}
	return patterns, nil
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
