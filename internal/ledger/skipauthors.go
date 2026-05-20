package ledger

import "path/filepath"

// matchesSkipAuthor reports whether any glob in the set matches either
// the AuthorEmail or Author (name) of the commit. Empty glob list returns
// false. Globs are loaded from .timbersignore lines prefixed "author:"
// and use filepath.Match semantics (*, ?, character classes — so '[' and
// ']' have glob meaning, not literal). To match GitHub bot names like
// "dependabot[bot]" use a prefix wildcard: "dependabot*".
//
// Match failures from filepath.Match are treated as non-matches so a
// malformed glob can never break pending detection.
func matchesSkipAuthor(globs []string, authorEmail, authorName string) bool {
	if len(globs) == 0 {
		return false
	}
	for _, glob := range globs {
		if authorEmail != "" {
			if ok, _ := filepath.Match(glob, authorEmail); ok {
				return true
			}
		}
		if authorName != "" {
			if ok, _ := filepath.Match(glob, authorName); ok {
				return true
			}
		}
	}
	return false
}
