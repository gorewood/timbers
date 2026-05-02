// Package main provides the entry point for the timbers CLI.
package main

// defaultWorkflowContent is the default workflow instructions for agent onboarding.
// This can be overridden by placing a .timbers/PRIME.md file in the repo root.
const defaultWorkflowContent = `<protocol>
# Session Protocol

After each git commit, run timbers log to document what you committed.
Entries reference commit SHAs, so the commit must exist before the entry.
Document each commit individually — batching loses commit-level granularity.

Session checklist:
- [ ] git add && git commit (commit code first)
- [ ] timbers log "what" --why "why" --how "how" (document committed work)
- [ ] timbers pending (should be zero before session end)
- [ ] git push (timbers log auto-commits entries, push to sync)
</protocol>

<why-coaching>
# Writing Good Why Fields

The --why flag captures the *verdict* — the design decision in one sentence.
This matters because draft templates extract architectural decisions from why
fields. Feature descriptions produce shallow ADRs; verdicts produce useful ones.

BAD (feature description):
  --why "Users needed tag filtering for queries"
  --why "Added amend command for modifying entries"

BAD (journey belongs in --notes):
  --why "Debated AND vs OR semantics, talked to users, decided OR because..."

GOOD (verdict):
  --why "OR semantics chosen over AND because users filter by any-of, not all-of"
  --why "Partial updates via amend avoid re-entering unchanged fields"
  --why "Chose warning over hard error for dirty-tree check to avoid blocking CI"

When work is blocked, slow, or frustrating — say so in --why or --notes.
"Spent 2h debugging flaky CI before finding the actual issue" is valuable
standup context that --why "Fixed auth test" alone doesn't capture.

When the operator (human or agent driving the call) made a judgment that
wasn't obvious — overriding a default, going against a teammate's first
suggestion, picking the harder path for a reason — name it. The verdict
isn't just "what was decided" but "what the deciding party chose to
prioritize." That framing is what makes downstream artifacts (ADRs,
devblogs, PR descriptions) sound human instead of mechanical.

Ask yourself: what's the one-sentence trade-off, and whose call was it?
</why-coaching>

<notes-coaching>
# Writing Good Notes

The --notes flag captures the *journey* to a decision. The why field has the
verdict; notes have the path you took to get there.

Include --notes when any of these apply:
- You chose between 2+ viable approaches
- You rejected an obvious approach for a non-obvious reason
- Something surprised you during implementation
- The decision constrains future options or creates lock-in
- A teammate unfamiliar with context would find the choice non-obvious

Skip --notes for: version bumps, typo fixes, dependency updates,
mechanical refactors, straightforward bug fixes with obvious causes.

BAD (restates what/why):
  --notes "Added notes field to support richer template output"

BAD (form-filling):
  --notes "## Alternatives: A, B, C. ## Decision: Picked B."

GOOD (thinking out loud):
  --notes "Debated exec wrapping vs HTTP API vs calling internal packages
  directly. Exec was simplest but meant double-parsing. HTTP felt over-
  engineered for a local tool. Treating MCP as a thin shell over the library
  layer meant zero business logic duplication — handlers are 20-40 lines each.
  Surprise: go-sdk auto-generates JSON schemas from struct tags."

GOOD (short):
  --notes "Considered AND vs OR for multi-tag queries. AND felt correct
  formally but real usage is 'show me anything tagged security or auth.'"

GOOD (collaboration texture):
  --notes "Agent's first pass added a config flag for AND-vs-OR. I pushed
  back — flag-driven defaults usually mean we didn't pick. We talked through
  who actually filters by all-tags-required (no one we could name) and
  dropped the flag. The single-mode default is the verdict; the flag would
  have been the cop-out."

What would help someone revisiting this decision in 6 months? And — when
a teammate (human, agent, or reviewer) reshaped the call — what did they
catch that the first pass missed?
</notes-coaching>

<operator-voice>
# Operator Voice and Collaboration

Timbers entries become source material for changelogs, ADRs, devblogs, and
PR descriptions. The drier the entry, the drier the downstream artifact.

Two lightweight habits inject real signal:

**1. Operator intent over technical motivation.** "Why" answers the
trade-off, but it can also answer "why this work landed on the table at
all." A reader who isn't in the repo wants to know the human angle: what
the operator was trying to accomplish, what frustration prompted it, what
bet they're making. If the work was reactive (a bug a user hit, a metric
that drifted, a complaint), say so. If it was proactive (a hunch, an
architectural bet), say that too.

**2. Surface collaboration when it changed the outcome.** Modern repos
are co-authored — humans, AI agents, reviewers — often in the same
session. When a teammate (any flavor) pushed back, surfaced a missed
case, or proposed something better than the first plan, capture it. The
moment of correction is high-value narrative; without it, the entry
reads as "I was right all along," which is rarely the truth.

Examples:
  --why "Renamed across the pricing module after the agent flagged drift
        between API and UI naming — caught a stale assumption I was about
        to ship."
  --why "Spent the morning fighting flaky CI before the actual fix landed
        — the real bug was in test setup, not the feature."

Skip this when the work was genuinely solo and uneventful. Don't fabricate
collaboration that didn't happen — that's worse than dry.
</operator-voice>

<content-safety>
# Content Safety

Entries are committed to git and may be visible in public repositories.
Never include in timbers entries:
- API keys, tokens, passwords, or secrets
- Personal names, emails, or identifying information
- Internal URLs, IP addresses, or infrastructure details
- Customer data or business-sensitive information

Focus on technical decisions, not people or credentials.
</content-safety>

<commands>
# Essential Commands

Recording work:
- ` + "`timbers log \"what\" --why \"why\" --how \"how\" [--notes \"deliberation\"]`" + ` - after each commit
- ` + "`timbers pending`" + ` - check for undocumented commits

Querying:
- ` + "`timbers query --last 5`" + ` - recent entries
- ` + "`timbers show <id>`" + ` - single entry details

Generating documents:
- ` + "`timbers draft --list`" + ` - list available templates
- ` + "`timbers draft release-notes --last 10 | claude -p --model opus`" + ` - pipe to CLI (uses subscription)
- ` + "`timbers draft devblog --since 7d --model opus`" + ` - direct API call (uses API key)
- Prefer piping to CLI when available — it uses your subscription instead of API tokens.

Creating pull request descriptions:
- ` + "`timbers draft pr-description --range $(git merge-base main HEAD)..HEAD`" + `
  ` + "`  | claude -p --model opus`" + `
  Generates a PR body from entries on the current branch.

Daily standup:
- ` + "`timbers draft standup --since 1d | claude -p --model opus`" + `
  (use --since 2d or 3d after weekends/gaps)

Sync:
- Entries are committed files in .timbers/ — use standard git push/pull
</commands>

<pr-authoring>
# Authoring Pull Requests with Ledger Context

When you (the agent) are about to open a PR and the operator has NOT given
you specific instructions for the PR body, default to drafting it from the
timbers ledger via the pr-description template. The entries you wrote
through the session ARE the PR description's source material — using them
keeps intent, design decisions, and risk areas consistent between what the
operator told you and what reviewers see.

When this applies:
- Operator says "open a PR" / "ship the PR" / "create the PR" without
  dictating body contents
- The branch has timbers entries committed in its range
- You're not in plan/dry-run mode

When this does NOT apply:
- Operator dictates the PR body directly ("PR title X, description Y")
- Operator pastes a template they want filled in verbatim
- The branch has zero timbers entries in range (no source material)
- Documentation-only or trivial single-line PRs (the template's tiny-PR
  path is fine, but operator may prefer something even shorter)

Recommended flow:

  timbers draft pr-description --range $(git merge-base main HEAD)..HEAD \
    | claude -p --model opus

Pipe the output through your standard PR-creation tool (gh pr create,
your harness, etc.). The pr-description template adapts to PR size and
omits sections that lack source material — it will not pad an empty body.

If a draft section comes back empty (e.g., no Design Decisions because
entries were thin on --why content), that's signal: either the entries
need beefing up before the PR opens, or the section genuinely doesn't
apply to this change. Don't manufacture content to fill it.
</pr-authoring>

<git-log>
# Git Log & Separate Commits

Each timbers log creates its own git commit. This is by design — it enables
reliable pending detection and clean filtering. To view git log without
ledger entries: git log --invert-grep --grep="^timbers: document"

See docs/design-decisions.md in the timbers repo for the full rationale.
</git-log>

<stale-anchor>
# Stale Anchor After Squash Merge

If timbers warns that the anchor commit is missing from history, this typically
means a branch was squash-merged or rebased. The pending list may show commits
that are already documented by entries from the original branch.

What to do:
- Do NOT try to catch up or re-document these commits
- If the squash-merged branch had timbers entries, the work is already covered
- Just proceed with your normal work — the anchor self-heals the next time you
  run timbers log after a real commit
</stale-anchor>
`
