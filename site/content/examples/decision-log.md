+++
title = 'Decision Log'
date = '2026-03-23'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

# Timbers Decision Log

## ADR-1: Graceful Degradation Over Hard Errors for Stale Anchors

**Context:** After a squash-merge, the anchor commit referenced in timbers entries disappears from `main`'s history. This caused `pending` to dump all reachable commits and `HasPendingCommits` to return an error, which blocked git hooks and confused agents relying on timbers for workflow state.

**Decision:** Return zero actionable commits with guidance instead of erroring. `HasPendingCommits` returns `false` (not an error) when the anchor is stale. The anchor self-heals on the next `timbers log` after a real commit.

**Consequences:**
- Hooks never block on stale state — agents and CI continue working after squash-merges
- `pending` may undercount (showing 0 when there are genuinely undocumented commits) until the anchor heals
- `doctor` gains `merge-strategy` and `stale-anchor` checks to surface the condition explicitly rather than silently swallowing it
- Shifts the failure mode from "loud and blocking" to "quiet and potentially missed" — acceptable because the anchor heals automatically

## ADR-2: File-Based Fallback Over Anchor-Only Matching for Commit-Range Queries

**Context:** `query --range`, `export --range`, and `draft --range` used anchor-commit ancestry to find entries within a commit range. After squash-merge, feature branch SHAs are not in `main`'s history, so anchor matching returns zero results even though the entry files exist in the diff.

**Decision:** When anchor-based matching returns zero results, fall back to `git diff --name-only A..B -- .timbers/` to discover entries by their file paths in the diff. Added `DiffNameOnly` to the `GitOps` interface and `EntryPathsInRange` to `storage.go`.

**Consequences:**
- Entries are discoverable regardless of merge strategy (rebase, squash, merge commit)
- Fallback is only triggered on zero results, so the common case (no squash) pays no extra cost
- Creates a secondary discovery mechanism that doesn't depend on commit graph topology — entries become durable artifacts tied to file presence, not just commit ancestry
- If `.timbers/` files are moved or renamed, the fallback would also break — but this is unlikely given the stable directory convention

## ADR-3: Three-Voice Essay Structure Over Stream-of-Consciousness for Devblog Template

**Context:** The devblog template used a Carmack `.plan` style — stream-of-consciousness daily logs. This produced flat recaps that read like commit summaries rather than engaging technical essays.

**Decision:** Replaced with a structured three-voice approach (Storyteller/Conceptualist/Practitioner) with Hook→Work→Insight→Landing scaffolding. Added tone calibration examples and entry mining guidance to the template.

**Consequences:**
- Generated posts have clearer narrative arc and distinct analytical perspectives
- Template is more opinionated — less flexibility for entries that don't fit the three-voice pattern
- Added a no-headers constraint after test generation revealed formatting issues
- Higher quality output at the cost of a more complex template that's harder to iterate on

## ADR-4: Append-Section Strategy Over Backup-and-Chain for Hook Installation

**Context:** Timbers was "stompy" in multi-tool environments. The backup-and-chain strategy took ownership of hook files, conflicted with beads' `core.hooksPath`, and silent skips left agents unaware of their options. Multiple tools competing for hook ownership created a fragile winner-take-all dynamic.

**Decision:** Implemented four-tier environment classification (uncontested/existing/known-override/unknown-override) with an append-section strategy replacing backup-and-chain. Symlinks and binaries are refused with guidance instead of maintaining a second code path. Steering via Claude Code Stop hook is the primary enforcement mechanism; git hooks are a bonus.

**Consequences:**
- Timbers coexists with other hook-owning tools instead of fighting for control
- No interactive prompt added to `init` for Tier 1 environments — avoids UX regression for the common case
- Eliminates `.original`/`.backup` file management complexity entirely
- `hooks status` command gives agents and users visibility into hook state
- Trade-off: append-section can't handle symlinked or binary hook files — these are explicitly refused rather than silently mishandled
- Old-format migration path needed for existing installations

## ADR-5: Unique Dolt Ports Over Shared Default for Cross-Repo Collision Prevention

**Context:** Multiple repos using beads defaulted to Dolt port 3307, causing cross-repo collisions with misleading error messages. Separately, Claude Code's sandbox blocking localhost TCP was misdiagnosed as stale PIDs, compounding the confusion.

**Decision:** Moved from hardcoded port 3307 to unique ports (initially 3308, then beads 0.59 introduced hash-derived ports, and 0.60 moved to OS-assigned ephemeral ports). Documented sandbox workarounds and recovery procedures in `CLAUDE.md`.

**Consequences:**
- Cross-repo Dolt collisions eliminated — each repo gets a unique port automatically
- Hardcoded port overrides were explicitly cleared because they defeat the collision-prevention mechanism
- Recovery is documented for the sandbox-blocks-localhost failure mode, which is a Claude Code platform constraint rather than a timbers bug
- Port assignment evolved through three strategies (hardcoded → hash-derived → ephemeral) in rapid succession, suggesting the design space wasn't fully explored upfront

## ADR-6: Respect `core.hooksPath` Over Hardcoded `.git/hooks/`

**Context:** Beads sets `core.hooksPath=.beads/hooks/` to manage its own hooks. Timbers was hardcoding `.git/hooks/` for hook installation, so hooks were installed where git never reads them — silently broken.

**Decision:** `GetHooksDir` now reads `git config core.hooksPath` and falls back to `.git/hooks/`. `GeneratePreCommitHook` takes a `hooksDir` parameter for dynamic chain backup paths.

**Consequences:**
- Hooks install to the correct location regardless of `core.hooksPath` configuration
- This is explicitly a band-aid: both beads and timbers want to own the pre-commit hook, and `core.hooksPath` is winner-take-all
- The real fix requires beads to expose a plugin/chain mechanism so timbers can register without owning the hook file
- Creates a dependency on beads' hook directory structure — if beads changes its `core.hooksPath` layout, timbers follows automatically but may need its append logic updated
