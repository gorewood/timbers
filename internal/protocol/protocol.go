// Package protocol holds the canonical agent-facing protocol text fragments
// that timbers injects into agent contexts. Sections live here so they
// stay in sync across the multiple call sites that compose workflow
// content (timbers prime, the MCP server, future integrations).
//
// Why a separate package: cmd/timbers (main package) and internal/mcp both
// need the same protocol section. internal/mcp can't import cmd/timbers
// (cycle), so the shared text lives below both.
package protocol

// SessionProtocol is the canonical session-protocol text injected by
// `timbers prime` and the MCP server. Defines the commit → log → push
// ordering rule that prevents the push-before-log race. When edited,
// both consumers pick up the change automatically.
const SessionProtocol = `<protocol>
# Session Protocol

After each git commit, run timbers log to document what you committed.
Entries reference commit SHAs, so the commit must exist before the entry.
Document each commit individually — batching loses commit-level granularity.

**Order matters: commit → timbers log → push. Never push between the two.**
timbers log writes an entry file and auto-commits it as a separate commit.
If you push after commit but before log, the content commit ships but the
entry commit stays stranded on your local branch — when teammates branch
off main, the content commit appears as pending with no entry visible.

Session checklist (in order):
- [ ] git add && git commit (commit code first — entry references this SHA)
- [ ] timbers log "what" --why "why" --how "how" (auto-commits the entry)
- [ ] git push (sends both content commit AND entry commit together)
- [ ] timbers pending (should be zero before session end)
</protocol>`

// StaleAnchorGuidance is the canonical stale-anchor text injected by
// both timbers prime and the MCP server. Tells agents how to react when
// the latest entry's anchor commit has been rewritten (squash merge or
// rebase) so they don't re-document already-covered work.
const StaleAnchorGuidance = `<stale-anchor>
# Stale Anchor After Squash Merge

If timbers warns that the anchor commit is missing from history, this typically
means a branch was squash-merged or rebased. The pending list may show commits
that are already documented by entries from the original branch.

What to do:
- Do NOT try to catch up or re-document these commits
- If the squash-merged branch had timbers entries, the work is already covered
- Just proceed with your normal work — the anchor self-heals the next time you
  run timbers log after a real commit
</stale-anchor>`
