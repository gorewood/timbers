---
title: 'Release Notes'
date: '2026-07-16'
tags: ['example', 'release-notes']
---

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---

Looking at the entries, I need to filter for user-observable changes and write release notes.

Let me work through what's user-facing versus internal:

**User-facing:**
- Contributor attribution on entries (`--who`, automatic capture) — new capability
- `report` command with decision digest profiles — new command
- Derive missing `what` from commit subjects — capability change
- Corrupt entries surfaced on reads / fail-closed — bug fix + reliability
- Retired `catchup` workflow — breaking removal
- Replaced generated ADRs with decision digests — breaking/behavior change to `draft`
- Cross-agent-debt gate + `--anchor` SHA resolution bug fixes — user-observable (hooks blocking session end, phantom entries)
- doctor detection + `--fix` for outdated post-rewrite hooks — new capability
- POSIX hook fix (dash) — bug fix users on Debian/Ubuntu/CI hit
- Timbermill site replaces Hugo — this is the *demo site*, not the product; site polish entries are not the shipped CLI

**Excluded:** site theme/polish (4sh entries), docs reconciliation, plan docs, worktree gitignore, release skip config.

Here are the release notes.

---

## New Features

- You can now capture **contributor attribution** on entries — Git authors and `Co-authored-by` trailers are recorded automatically, so person-level credit survives rebase, squash, shallow clone, or pruning. Use `--who` to set the full contributor set explicitly for pairing, bots, or corrections.
- New `timbers report` command turns captured rationale into a decision digest without hand-rebuilding selection and prompt conventions.
- `timbers log` now derives `what` from your commit subjects when you omit it, so you don't have to retype what the commit already says.
- `timbers doctor` now detects outdated post-rewrite hooks and repairs them with `timbers doctor --fix`, so existing repos pick up hook improvements instead of silently running stale ones.

## Improvements

- Corrupt or malformed ledger files are now surfaced through `timbers doctor` and query output instead of silently vanishing; `draft` and `report` refuse to emit partial artifacts when the ledger can't be fully read.

## Bug Fixes

- The session-end hook no longer blocks on another agent's already-paid work in parallel-agent worktrees, and it now honors `TIMBERS_SKIP_CROSS_AGENT_DEBT` like the other hooks.
- `timbers log --anchor HEAD` now resolves to a real commit SHA at write time instead of storing the literal `HEAD`, and errors on unresolvable refs rather than writing a phantom entry.
- `timbers log` no longer fails with `failed to commit entry file` when sibling agent work is pending — logging your entry no longer trips the very debt gate it's clearing.
- The post-rewrite hook now runs under `dash` (Debian/Ubuntu and CI `/bin/sh`); previously it errored and silently skipped relinking entries after a history rewrite.

## Breaking Changes

- **Removed the `timbers catchup` workflow.** Use the first-log baseline, `timbers log --batch`, and ignore rules to onboard existing history instead — these already cover legitimate adoption cases. Existing catchup-generated entries are preserved.
- **The built-in ADR template now produces a non-authoritative decision digest**, not numbered ADRs — generated output no longer claims ADR numbers, status, or lifecycle. If you relied on the old numbered-ADR output, treat your native project ADRs as the authoritative source and publish them directly.

---

skipped: all site/Timbermill theme work (that's the demo site, not the CLI), docs-drift reconciliation, plan docs, and the agent-steering-prose entry (a7e399) — add back only if the demo site *is* the product you're shipping notes for.
