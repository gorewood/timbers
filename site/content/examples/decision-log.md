+++
title = 'Decision Log'
date = '2026-03-23'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---



# Architectural Decision Log — Timbers (2026-03-05 to 2026-03-24)

## ADR-1: Union Strategy Over Fallback for `--range` Entry Discovery

**Context:** The `--range` flag used two methods to find entries in a commit range: anchor-based matching (checking if an entry's `anchor_commit` is an ancestor of commits in the range) and file-based diff discovery (using `git diff --name-only` to find `.timbers/` files). Originally, diff discovery only triggered as a fallback when anchor matching returned zero results.

**Decision:** Both discovery paths now run unconditionally and their results are unioned and deduplicated. The fallback approach silently dropped entries in partial-stale scenarios — when *some* anchors matched but others didn't (e.g., entries from a squash-merged feature branch mixed with entries from direct commits). Union ensures all discoverable entries surface regardless of which method finds them.

**Consequences:**
- Entries are no longer silently lost when a range contains a mix of valid and stale anchors
- Slight increase in git operations per `--range` query (both paths always execute)
- Eliminates a class of bugs where the fix for one merge strategy breaks another
- Makes `--range` behavior independent of merge strategy, which is the correct invariant

## ADR-2: Graceful Degradation Over Hard Errors for Stale Anchors

**Context:** After squash-merge or rebase, entry `anchor_commit` SHAs disappear from `main`'s history. This caused `pending` to dump hundreds of false-positive commits (every reachable commit since the anchor was "missing") and `HasPendingCommits` to return errors that blocked post-commit hooks. Agents would then attempt to re-document all these commits, creating duplicate entries.

**Decision:** Chose graceful degradation: `HasPendingCommits` returns `false` (not an error) on stale anchors, and `pending` reports 0 actionable commits with a guidance message. Hooks continue working, agents aren't confused, and the anchor self-heals on the next real `timbers log`.

**Consequences:**
- Hooks never block on stale anchors — agents can keep committing
- Risk of genuinely pending commits being hidden if the stale-anchor heuristic misfires, but this is preferable to the alternative of hundreds of false positives
- `doctor` gained merge-strategy checks (`pull.rebase`, `merge.ff`) and stale-anchor detection, giving users a way to proactively identify risky configurations
- Constrains future design: any feature relying on `HasPendingCommits` must handle the "anchor might be stale" case without assuming the answer is authoritative

## ADR-3: File-Based Diff Fallback for Post-Squash Entry Discovery

**Context:** `query --range` returned empty results after squash merges because entries store feature-branch SHAs as `anchor_commit`, and those SHAs don't exist in `main`'s history after squash. The entries *are* present as files in the repository, but the commit-ancestry matching algorithm couldn't find them.

**Decision:** Added a fallback path using `git diff --name-only A..B -- .timbers/` to discover entries by their file presence in the diff, triggered when anchor-based matching returns zero results. (Later evolved into the union strategy in ADR-1.)

**Consequences:**
- Entries become discoverable regardless of merge strategy — squash, rebase, and merge commits all work
- Required exposing `EntryPathsInRange` in `storage.go` and adding `DiffNameOnly` to the `GitOps` interface, expanding the interface surface
- File-based discovery can find entries that weren't logically "created" in that range (e.g., if an entry file was modified for unrelated reasons), though this is unlikely in practice

## ADR-4: Three-Voice Essay Structure Over Stream-of-Consciousness for Devblog Template

**Context:** The devblog template used a Carmack `.plan` style — stream-of-consciousness narrative. Generated output was flat recaps that read more like changelogs than essays. The template needed to produce engaging technical writing from structured ledger entries.

**Decision:** Replaced with a three-voice structure (Storyteller/Conceptualist/Practitioner) with Hook→Work→Insight→Landing scaffolding. Each voice brings a different lens: narrative arc, conceptual framing, and practical detail.

**Consequences:**
- Generated posts have clearer takeaways and more varied prose rhythm
- Higher prompt complexity — the template is harder to maintain and tune
- Added a no-headers constraint after test generation revealed that headers broke the essay feel
- The voice structure is opinionated — it may not suit all content types, but devblog posts benefit from narrative structure over flat reporting

## ADR-5: Gitignore Beads Backup State as Machine-Local Runtime Data

**Context:** The `.beads/backup/` directory contains timestamps and Dolt commit hashes that change on every `bd` operation. These were being tracked in git, creating noisy diffs on every commit.

**Decision:** Treat backup state as machine-local runtime data: added `backup/` to `.beads/.gitignore` and untracked existing files with `git rm -r --cached`.

**Consequences:**
- Eliminates noise from diffs and git status
- Backup state is no longer shared across clones — each machine maintains its own
- If backup state were ever needed for debugging sync issues across machines, it would need to be shared out-of-band

## ADR-6: Hash-Derived Ports Over Hardcoded Ports for Dolt Server

**Context:** Beads uses a local Dolt SQL server for issue tracking. The default port (3307) caused cross-repo collisions when multiple projects ran simultaneously. An intermediate fix (hardcoding port 3308) worked but defeated the collision-prevention mechanism introduced in beads 0.59.

**Decision:** Adopted beads 0.59's hash-derived port scheme — ports are computed from a hash of the repo path, making collisions unlikely without manual configuration. Cleared all hardcoded port overrides from `metadata.json` and `config.yaml`.

**Consequences:**
- Multiple repos can run Dolt servers simultaneously without port conflicts
- Port numbers are no longer predictable or stable across machines (a repo gets different ports on different paths), which complicates manual debugging
- Later superseded by beads 0.60's OS-assigned ephemeral ports, making hash-derived ports themselves a transitional step
