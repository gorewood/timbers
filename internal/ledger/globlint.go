package ledger

import "strings"

// LooksLikeLiteralBracket reports whether a glob contains a `[...]` group whose
// body looks like an intended literal rather than a character class. This is
// the "author:dependabot[bot]" footgun: filepath.Match treats `[bot]` as a
// class matching one of b/o/t, so the glob silently fails to match a literal
// "[bot]". The heuristic flags a bracket group whose body is 2+ word-ish
// characters with no range (`-`) or negation (`!`/`^`) — the shape of a
// mistaken literal, not a deliberate class like `[a-z]` or `[!x]`.
//
// It is a lint hint, not a correctness check: a deliberate multi-char class
// (e.g. `[abc]`) will also be flagged, but those are rare in author/subject
// globs and the warning is non-blocking and self-explanatory.
func LooksLikeLiteralBracket(glob string) bool {
	rest := glob
	for {
		open := strings.IndexByte(rest, '[')
		if open < 0 {
			return false
		}
		rest = rest[open+1:]
		closeIdx := strings.IndexByte(rest, ']')
		if closeIdx < 0 {
			return false // unterminated '[' — not our concern here
		}
		if isLiteralLookingClass(rest[:closeIdx]) {
			return true
		}
		rest = rest[closeIdx+1:]
	}
}

// isLiteralLookingClass reports whether a bracket body (the chars between `[`
// and `]`) looks like a mistaken literal: 2+ characters, every one a letter or
// digit, and no range or negation markers.
func isLiteralLookingClass(body string) bool {
	if len(body) < 2 {
		return false
	}
	for _, r := range body {
		isWord := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
		if !isWord {
			return false // '-', '!', '^', etc. → deliberate class, not a literal
		}
	}
	return true
}
