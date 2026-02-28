+++
title = 'Pr Description'
date = '2026-02-28'
tags = ['example', 'pr-description']
+++

Generated with `timbers draft pr-description --last 20 | claude -p --model opus`

---

## Why

This batch covers v0.9.0 through v0.11.0-dev: terminal color visibility fixes, hook reliability rework, auto-commit for `timbers log`, template improvements, and a marketing landing page. The thread connecting most changes is reducing friction in the agent workflow — invisible colors, broken hooks, and manual commit steps were all silent failures that eroded trust in the tool.

## Design Decisions

- **Actionable warnings over anchor-reset command** for stale anchors after squash merges. The anchor self-heals on next `timbers log`, so messaging alone is sufficient — an explicit reset command would add surface area for a transient problem.
- **Removed `PostToolUse` hook entirely** rather than fixing it. Hook fires correctly and reads stdin, but Claude Code doesn't surface stdout from PostToolUse. The existing `Stop` hook already covers the same case (`timbers pending` at session end) with visible output.
- **Pathspec-scoped auto-commit** (`git commit -- <path>`) in `timbers log` to close the staged-but-uncommitted gap. Considered a separate branch (like beads/entire.io) but entries must be filesystem-visible for `timbers prime`/`draft` without worktree indirection.
- **`exec-summary` renamed to `standup`** with no backward-compat alias — pre-GA project with low usage, alias complexity isn't worth it.
- **PR description template rewritten around intent/decisions** rather than diff summaries, since agents already review diffs. Reviewers need the "why" that code doesn't carry.
- **`AdaptiveColor` + `--color` flag** over full theme config (env vars, config files). Covers 95% of terminal compatibility cases without maintenance burden. Lipgloss `HasDarkBackground()` deferred to a future pass.
- **Check-before-generate** for empty devblog entries rather than teaching the LLM to refuse — cheaper and deterministic vs. probabilistic refusal.
- **`grep` on raw stdin JSON** for hook commit detection rather than `jq`-based parsing. Simpler, no dependencies, negligible false-positive risk for matching `git commit`.
- **Pipe-first generation defaults** (`timbers draft <template> | claude -p --model opus`) match subscription users. Doctor generation check shows env var names, not provider labels, so users know exactly what to set.

## Risk & Reviewer Attention

- **`GitCommitFunc` injection in `FileStorage`** (`internal/ledger/storage.go`) — the auto-commit scopes to the entry file via pathspec (`--`), but verify it can't sweep other staged files into the timbers commit.
- **Retired event cleanup** in hook setup — `retiredEvents` list removes `PostToolUse` on upgrade. Confirm the removal logic (`removeTimbersHooksFromEvent`) doesn't affect other plugins sharing the same hook event.
- **`hasExactHookCommand` upgrade detection** — replaces the old skip-if-any-exists behavior that left broken hooks in place. Edge case: users who manually edited hook commands may get unexpected replacements.
- **Landing page uses Tailwind Play CDN and GSAP ScrollTrigger** — fine for a project site but these are runtime dependencies loaded from CDN on every page view.

## Scope

Broad cross-cutting change touching CLI commands (`--color` flag plumbed through all `NewPrinter` call sites), hook setup/teardown, ledger storage (auto-commit), draft templates, prime workflow coaching, CI workflows, and the Hugo site (landing page, examples, terminal block fixes). Core query/export/show paths are untouched.

## Test Plan

- `just check` — lint and test suite covers `GitCommitFunc` injection, hook detection/upgrade logic, and color mode resolution.
- Verify `timbers log` auto-commits entry file without sweeping staged changes.
- Test `--color never/auto/always` on both light and dark terminals.
- Run `timbers doctor` to confirm generation check detects CLI vs API key availability.
- `timbers prime` in a repo with a stale anchor (post-squash-merge) should warn, not error.
