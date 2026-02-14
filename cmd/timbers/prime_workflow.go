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

Ask yourself: what's the one-sentence trade-off?
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

What would help someone revisiting this decision in 6 months?
</notes-coaching>

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
- ` + "`timbers draft release-notes --last 10`" + ` - render for piping to LLM
- ` + "`timbers draft devblog --since 7d --model opus`" + ` - generate directly

Sync:
- Entries are committed files in .timbers/ — use standard git push/pull
</commands>
`
