package main

import (
	"fmt"

	"github.com/gorewood/timbers/internal/git"
	"github.com/gorewood/timbers/internal/ledger"
)

// checkSessionIdentity reports the git config user.email value that the
// cross-agent debt classifier uses to identify in-session work. An empty
// or unset value is a high-severity warning — under safe-degradation
// rules, an empty UserEmail disables the foreign-author skip entirely,
// which means the gate falls back to "treat every commit as in-session"
// and a misconfigured environment can silently route work past the
// classifier without the operator noticing.
func checkSessionIdentity() checkResult {
	email := git.ConfigUserEmail()
	if email == "" {
		return checkResult{
			Name:    "Session Identity",
			Status:  checkWarn,
			Message: "git config user.email is unset",
			Hint: "Run: git config user.email \"you@example.com\". " +
				"Without this, the cross-agent debt classifier cannot identify " +
				"in-session work and falls back to treating every commit as " +
				"in-session — the gate becomes a no-op for foreign-author detection.",
		}
	}
	return checkResult{
		Name:    "Session Identity",
		Status:  checkPass,
		Message: email,
	}
}

// checkSessionWindow reports the staleness window the cross-agent debt
// classifier is using. A present-but-malformed .timbersignore
// session-window: directive surfaces as a warning so the operator sees
// that what they configured did not take. A missing directive (default
// 24h in force) reports pass with the value.
func checkSessionWindow() checkResult {
	root, err := git.RepoRoot()
	if err != nil {
		return checkResult{
			Name:    "Session Window",
			Status:  checkPass,
			Message: fmt.Sprintf("%s (default, not in a git repo)", ledger.DefaultSessionWindow),
		}
	}
	result := ledger.LoadSessionWindow(root)
	if result.ParseErr != nil {
		return checkResult{
			Name:    "Session Window",
			Status:  checkWarn,
			Message: fmt.Sprintf("malformed session-window: %q — using default %s", result.Raw, ledger.DefaultSessionWindow),
			Hint: "Format: Go time.ParseDuration grammar. " +
				"Accepts \"4h\", \"2h30m\", \"15m\", \"90m\". " +
				"Day suffixes (\"1d\") and capitalized hours (\"4H\") are NOT supported.",
		}
	}
	if result.Raw == "" {
		return checkResult{
			Name:    "Session Window",
			Status:  checkPass,
			Message: fmt.Sprintf("%s (default)", result.Window),
		}
	}
	return checkResult{
		Name:    "Session Window",
		Status:  checkPass,
		Message: fmt.Sprintf("%s (.timbersignore: %s)", result.Window, result.Raw),
	}
}
