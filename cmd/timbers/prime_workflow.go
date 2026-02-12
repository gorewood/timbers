// Package main provides the entry point for the timbers CLI.
package main

// defaultWorkflowContent is the default workflow instructions for agent onboarding.
// This can be overridden by placing a .timbers/PRIME.md file in the repo root.
const defaultWorkflowContent = `# CRITICAL: Session Protocol
- [ ] git add && git commit (commit code FIRST)
- [ ] timbers log "what" --why "why" --how "how" (document committed work)
- [ ] timbers pending (MUST be zero before session end)
- [ ] git push (entries are committed files, push to sync)

IMPORTANT: Always commit code before running timbers log. Entries must
describe committed work, not work-in-progress. Timbers will warn if
the working tree is dirty.

# Post-Commit Rule (MANDATORY)
After EVERY git commit, you MUST run:
  timbers log "what you did" --why "design decision" --how "approach"
Do not batch commits. Document each commit immediately.

# Writing Good Why Fields
The --why flag captures *design decisions*, not feature descriptions.

BAD (feature description):
  --why "Users needed tag filtering for queries"
  --why "Added amend command for modifying entries"

GOOD (design decision):
  --why "OR semantics chosen over AND because users filter by any-of, not all-of"
  --why "Partial updates via amend avoid re-entering unchanged fields"
  --why "Chose warning over hard error for dirty-tree check to avoid blocking CI"

Ask yourself: why THIS approach over alternatives? What trade-off did you make?

# Writing Good Notes (optional)
The --notes flag captures the *journey* to a decision, not just the verdict.
Skip it for routine work. Use it when you explored alternatives or made a real choice.

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

What would help someone revisiting this decision in 6 months understand HOW you got there?

# Core Rules (MANDATORY)
- You MUST commit code first, then document with timbers log
- You MUST capture design decisions in --why, not feature summaries
- You MUST run ` + "`timbers pending`" + ` before session end (MUST be zero)
- You MUST run ` + "`git push`" + ` to sync ledger to remote (entries are committed files)

# Essential Commands
### Recording Work
- ` + "`timbers log \"what\" --why \"why\" --how \"how\"`" + ` - Run after EVERY commit
- ` + "`timbers pending`" + ` - MUST be zero before session end

### Querying
- ` + "`timbers query --last 5`" + ` - Recent entries
- ` + "`timbers show <id>`" + ` - Single entry details

### Generating Documents
- ` + "`timbers draft --list`" + ` - List available templates
- ` + "`timbers draft release-notes --last 10`" + ` - Render for piping to LLM
- ` + "`timbers draft devblog --since 7d --model opus`" + ` - Generate directly

### Sync
- Entries are committed files in .timbers/ — use standard git push/pull
`
