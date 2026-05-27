package main

import (
	"strings"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
)

// checkTimbersignoreGlobs lints .timbersignore author:/msg: globs for the
// literal-bracket footgun — e.g. "author:dependabot[bot]", where filepath.Match
// reads [bot] as a character class so the rule silently matches nothing. The
// fix is the wildcard form ("author:dependabot*"). Non-blocking warning.
func checkTimbersignoreGlobs() checkResult {
	const name = ".timbersignore Globs"

	root, err := git.RepoRoot()
	if err != nil {
		return checkResult{Name: name, Status: checkPass, Message: "skipped: " + err.Error()}
	}

	authors, messages, loadErr := ledger.LoadIgnoreGlobs(root)
	if loadErr != nil {
		return checkResult{Name: name, Status: checkPass, Message: "skipped: " + loadErr.Error()}
	}

	var suspect []string
	for _, g := range authors {
		if ledger.LooksLikeLiteralBracket(g) {
			suspect = append(suspect, "author:"+g)
		}
	}
	for _, g := range messages {
		if ledger.LooksLikeLiteralBracket(g) {
			suspect = append(suspect, "msg:"+g)
		}
	}

	total := len(authors) + len(messages)
	if len(suspect) == 0 {
		if total == 0 {
			return checkResult{Name: name, Status: checkPass, Message: "no author:/msg: globs configured"}
		}
		return checkResult{Name: name, Status: checkPass, Message: "author:/msg: globs look well-formed"}
	}

	return checkResult{
		Name:    name,
		Status:  checkWarn,
		Message: "glob(s) use a literal-looking [..] class that won't match a literal bracket: " + strings.Join(suspect, ", "),
		Hint: "filepath.Match reads [..] as a character class — e.g. 'author:dependabot[bot]' matches nothing. " +
			"Use a wildcard instead: 'author:dependabot*' (or 'author:*dependabot*').",
	}
}
