package ledger

import "strings"

// skipRule classifies a single path-matching rule used to filter
// housekeeping commits from pending detection.
type skipRule struct {
	pattern string
	kind    skipKind
}

type skipKind int

const (
	// skipPrefix matches any path under a directory (pattern ends with "/").
	skipPrefix skipKind = iota
	// skipExact matches a literal path (no wildcards).
	skipExact
	// skipSuffix matches paths ending with the literal after "*" (pattern starts with "*").
	skipSuffix
)

// parseSkipRule classifies a pattern. Trailing "/" → directory prefix.
// Leading "*" → suffix match. Otherwise exact path.
func parseSkipRule(pattern string) skipRule {
	switch {
	case strings.HasSuffix(pattern, "/"):
		return skipRule{pattern: pattern, kind: skipPrefix}
	case strings.HasPrefix(pattern, "*"):
		return skipRule{pattern: pattern[1:], kind: skipSuffix}
	default:
		return skipRule{pattern: pattern, kind: skipExact}
	}
}

// match reports whether the rule matches the given path.
func (r skipRule) match(path string) bool {
	switch r.kind {
	case skipPrefix:
		return strings.HasPrefix(path, r.pattern)
	case skipSuffix:
		return strings.HasSuffix(path, r.pattern)
	case skipExact:
		return path == r.pattern
	default:
		return false
	}
}

// defaultSkipPatterns are housekeeping path patterns that timbers auto-skips
// from pending detection without configuration. These are the cases where a
// commit touching only these files carries zero design intent: ledger churn
// (.timbers/), beads sync (.beads/), tooling metadata (.gitignore family,
// .editorconfig), narrowly-scoped GitHub metadata files, dependency-bot
// configuration, and lockfiles for the major package ecosystems.
//
// .github/ as a directory is intentionally NOT included — .github/workflows/
// changes are substantive.
//
// Lockfile patterns use suffix matching (leading "*"). When a lockfile is
// part of a meaningful dependency change, the manifest (package.json, go.mod,
// Cargo.toml, etc.) is in the same commit and the file-level filter keeps
// it pending. Only isolated lockfile-only commits — usually manual conflict
// resolution or auto-rebases — get auto-skipped.
var defaultSkipPatterns = []string{
	".timbers/",
	".beads/",
	".gitignore",
	".gitattributes",
	".editorconfig",
	".github/dependabot.yml",
	".github/CODEOWNERS",
	".github/FUNDING.yml",
	".github/pull_request_template.md",
	"renovate.json",
	"dependabot.yml",
	"*package-lock.json",
	"*pnpm-lock.yaml",
	"*yarn.lock",
	"*go.sum",
	"*Cargo.lock",
	"*Gemfile.lock",
}

// compiledDefaultSkipRules is the parsed default ruleset, computed once.
var compiledDefaultSkipRules = compileSkipRules(defaultSkipPatterns)

// compileSkipRules parses a slice of patterns into rules.
func compileSkipRules(patterns []string) []skipRule {
	rules := make([]skipRule, 0, len(patterns))
	for _, p := range patterns {
		if p == "" {
			continue
		}
		rules = append(rules, parseSkipRule(p))
	}
	return rules
}

// matchAny reports whether any rule in the set matches path.
func matchAny(rules []skipRule, path string) bool {
	for _, r := range rules {
		if r.match(path) {
			return true
		}
	}
	return false
}
