+++
title = 'Decision Log'
date = '2026-03-22'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

# Timbers Decision Log

## ADR-1: File-Based Fallback for Entry Discovery After Squash Merge

**Context:** `query --range` matched entries by checking if their anchor commit SHA appeared in the target range's git history. After a squash merge, feature branch SHAs are replaced by a single squash commit, so anchor-based matching returned zero results — entries created on the feature branch became invisible from `main`.

**Decision:** Added a `git diff --name-only` fallback in `entry_filter.go` that finds entries by checking which `.timbers/` files were modified in the commit range. When anchor matching returns zero results, the file-based path kicks in via `EntryPathsInRange` in `storage.go`.

**Consequences:**
- Entries are now discoverable regardless of merge strategy (merge commit, squash, rebase)
- Two code paths for range matching — anchor-first with file fallback — adds maintenance surface
- File-based matching is less semantically precise (it finds entries whose files changed, not entries whose anchors are in range), but this is acceptable since the fallback only fires when anchor matching fails entirely
- Enables `timbers draft pr-description --range` to work correctly after squash merges, which is the common GitHub flow

---

## ADR-2: Three-Voice Essay Structure Over Stream-of-Consciousness for Devblog Template

**Context:** The devblog template (used by `timbers draft devblog`) originally produced Carmack `.plan`-style stream-of-consciousness output. This generated flat recaps that read like status updates rather than engaging technical essays.

**Decision:** Replaced with a structured three-voice approach — Storyteller, Conceptualist, and Practitioner — scaffolded with Hook → Work → Insight → Landing structure. Each voice contributes a different dimension: narrative drive, conceptual depth, and practical grounding.

**Consequences:**
- Richer, more engaging essays with clear takeaways instead of flat recaps
- Template is more prescriptive, which constrains the LLM's creative freedom but produces more consistent quality
- Added a no-headers constraint after test generation revealed the LLM was over-structuring output
- Three named voices may feel artificial if the source entries are thin — the template quality is bounded by entry quality

---

## ADR-3: Append-Section Hook Strategy Over Backup-and-Chain

**Context:** Timbers needed to install a `pre-commit` hook, but in multi-tool environments (especially with beads), the previous backup-and-chain strategy took ownership of hook files. This conflicted with beads' `core.hooksPath`, and silent skips left agents unaware of their options. The tool was "stompy" — it assumed it was the only hook manager.

**Decision:** Implemented a four-tier environment classification (uncontested / existing / known-override / unknown-override) with an append-section strategy. Instead of renaming the existing hook and chaining to it, timbers appends a clearly delimited section to the existing hook file. Symlinks and binaries are refused with guidance rather than maintaining a second code path.

**Consequences:**
- Eliminates `.original` / `.backup` file proliferation and the rename-and-chain code path
- Plays nicely with beads and other hook managers — no longer takes exclusive ownership
- Tier-aware messaging in `init` and `doctor` gives agents clear guidance instead of silent failures
- Decided against prompting for hooks in Tier 1 (uncontested) `init` — Claude Code Stop hook is the primary enforcement mechanism, git hooks are a bonus. Adding an interactive prompt was a UX regression.
- New `hooks status` command provides visibility into the hook state

---

## ADR-4: Respect `core.hooksPath` Instead of Hardcoding `.git/hooks/`

**Context:** Beads sets `core.hooksPath=.beads/hooks/` to manage its own hooks. Timbers was hardcoding `.git/hooks/` for hook installation, so hooks were installed where git never reads them — completely invisible.

**Decision:** `GetHooksDir` now reads `git config core.hooksPath` and falls back to `.git/hooks/` only if unset. `GeneratePreCommitHook` takes a `hooksDir` param for dynamic chain backup path.

**Consequences:**
- Hooks now install to wherever git actually reads them, fixing the beads coexistence bug
- This is explicitly a band-aid: both beads and timbers want to own the pre-commit hook, and `core.hooksPath` is winner-take-all. The real fix is a plugin/chain mechanism in beads' pre-commit so timbers can register without owning the hook file.
- Fallback to `.git/hooks/` preserves behavior for repos without `core.hooksPath` set
- Created a dependency on beads' hook architecture — future beads changes to hook management could break timbers again

---

## ADR-5: Capture Exit Code Before Chaining Pre-Commit Hooks

**Context:** Timbers' generated pre-commit hook was designed to block commits when entries were pending, then chain to a backup hook. But it unconditionally `exec`'d the backup hook, so timbers' non-zero exit was silently overridden — commits always succeeded regardless of pending state.

**Decision:** Capture the exit code after `timbers hook run`, and exit early with that code if non-zero, before `exec`'ing the backup hook.

**Consequences:**
- Timbers can now actually block commits when pending entries exist, which was the entire point of the hook
- If timbers fails for any reason (binary missing, crash), it blocks the chain — acceptable since `timbers hook run` is designed to be fast and gracefully degrade
- Simple two-line fix (capture + early exit) rather than restructuring the hook generation

---

## ADR-6: Unique Dolt Ports Over Shared Default to Prevent Cross-Repo Collisions

**Context:** Beads uses a local Dolt SQL server per repository. Multiple repos using the default port 3307 caused collisions with misleading error messages. Separately, Claude Code's sandbox blocking localhost TCP was being misdiagnosed as stale PIDs rather than a sandbox issue.

**Decision:** Moved from hardcoded port 3307 to unique ports (initially explicit 3308, then beads 0.59 introduced hash-derived ports, then 0.60 moved to OS-assigned ephemeral ports). Documented sandbox workarounds and recovery procedures in `CLAUDE.md`.

**Consequences:**
- Eliminates cross-repo port collisions entirely (ephemeral ports are guaranteed unique)
- Hardcoded overrides were explicitly removed because they defeat the collision-prevention mechanism
- No port configuration needed — simplifies setup and documentation
- Documented the sandbox/localhost TCP issue prevents future misdiagnosis of connection failures
- Port is no longer predictable, which means external tooling can't assume a known port — must discover it from beads

---

## ADR-7: Gitignore Beads Backup State as Machine-Local Runtime Data

**Context:** The `.beads/backup/` directory contained timestamps and dolt commit hashes that changed on every `bd` operation, creating noisy diffs in git status and commits.

**Decision:** Added `backup/` to `.beads/.gitignore` and untracked existing files with `git rm -r --cached`.

**Consequences:**
- Eliminates noisy diffs from machine-local runtime state
- Backup state is no longer shared across clones — each machine manages its own backup timestamps
- Clean `git status` makes it easier to spot actual changes
