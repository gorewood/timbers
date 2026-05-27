package main

import "github.com/spf13/cobra"

const timbersignoreGuide = `.timbersignore — exempt commits from pending detection

A per-repo file (repo root, peer to .timbers/) that tells 'timbers pending'
and the pre-commit gate which commits don't warrant a ledger entry. Committed
to git, so every clone and the agent stop-hook share it. Retroactive: pending
re-evaluates the current range against the current file.

Three entry shapes (one per line; '#' starts a comment):

  <path>          Path rule. A commit is exempt only if EVERY changed file
                  matches a path pattern. Forms:
                    vendor/        directory prefix (trailing /)
                    *.lock         suffix match (leading *)
                    go.work        exact path
  author:<glob>   Author rule. Exempts a commit when its author NAME or EMAIL
                  matches the glob (filepath.Match) — regardless of files.
  msg:<glob>      Subject rule. Exempts a commit when its first commit-message
                  line matches the glob (filepath.Match, whole-line).

Globs use filepath.Match: * matches any run of non-/ chars, ? one char,
[...] a character class.

CAVEAT — the [bot] footgun. filepath.Match reads [bot] as a character class
(one of b/o/t), so 'author:dependabot[bot]' matches NOTHING. Use a wildcard:

  author:dependabot*        # exempt all Dependabot commits (matches the name)
  author:renovate*          # exempt all Renovate commits
  author:*dependabot*       # belt-and-suspenders (matches name and email)

For bots, an author rule is the complete answer: it exempts every commit by
that author, including version bumps that also touch package.json / go.mod
(a path rule would miss those, since not all files match).

Verify a rule works:  timbers pending --explain   (shows each commit's
keep/skip reason: author/message/infra/documented/ack/...).

Requires the timbers binary that runs the gate to be >= v0.22.0 (author:) /
>= v0.22.4 (msg:). An older binary parses these as path patterns and silently
exempts nothing.`

// newTimbersignoreHelpCmd registers a help topic for .timbersignore so
// `timbers help timbersignore` (and `timbers timbersignore`) explain the
// exemption rules — the lever pending/doctor output points users at.
func newTimbersignoreHelpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "timbersignore",
		Short: "Explain .timbersignore exemption rules (path / author: / msg:)",
		Long:  timbersignoreGuide,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println(timbersignoreGuide)
			return nil
		},
	}
}
