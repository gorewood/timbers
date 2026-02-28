+++
title = 'Pr Description'
date = '2026-02-27'
tags = ['example', 'pr-description']
+++

Generated with `timbers draft pr-description --last 20 | claude -p --model opus`

---

## Why

Polish pass from v0.9.0 through v0.10.2+: fix dark terminal visibility, eliminate broken hooks, close workflow gaps (auto-commit entries, stale anchor handling), and harden the generation pipeline. The common thread is removing friction — invisible text, manual steps, silent failures, and confusing edge cases that accumulated as features shipped.

## Design Decisions

- **PostToolUse hook removed rather than fixed.** Diagnosed that `$TOOL_INPUT` is always empty (stdin, not env vars). Fixed stdin reading first, then removed PostToolUse entirely — Claude Code doesn't surface hook stdout, and the existing `Stop` hook covers the same case via `timbers pending` at session end.
- **Actionable warnings over anchor-reset command for stale anchors.** After squash merges, the anchor self-heals on next `timbers log`. Messaging alone is sufficient; an explicit reset command would add surface area for a transient problem.
- **`exec-summary` renamed to `standup` with no alias.** Pre-GA project with low usage — backward compat adds complexity for no real benefit. Clean break.
- **PR description template rewritten for agent-era review.** Agents already read diffs; the template now focuses on intent, trade-offs, and reviewer attention — questions diffs don't answer.
- **Auto-commit scoped via pathspec (`-- <path>`).** The staged-but-uncommitted gap confused users, but sweeping other staged files into a timbers commit would be dangerous. Pathspec commit eliminates the manual step safely.
- **`AdaptiveColor` + `--color` flag over full theme config.** Color 8 (bright black) invisible on Solarized Dark. Considered env vars and config files but deferred — `AdaptiveColor` + explicit flag covers 95% of cases without maintenance burden.
- **Check-before-generate for devblog CI.** Empty entry sets invoked the LLM, which generated apology posts. Gating on entry count is cheaper and cleaner than teaching the LLM to refuse.
- **`govulncheck` kept separate from `just check`.** Not in golangci-lint, needs its own tool dep. Separate recipe avoids blocking CI on stdlib patches.
- **Doctor generation check shows env var names, not provider labels.** Users need to know exactly what to set. CI hint omits CLI suggestion since CI uses API keys.
- **Pipe-first generation defaults** (`timbers draft ... | claude -p --model opus`) match subscription users and avoid API token costs.

## Risk & Reviewer Attention

- **Hook upgrade logic** in `claude_parse.go`: `hasExactHookCommand` detects stale vs current hooks, `removeTimbersHooksFromEvent` cleans before re-adding. Old skip-if-any-exists behavior left broken hooks in place forever — verify the replacement path handles partial states.
- **`GitCommitFunc` injection** in `FileStorage` for auto-commit: confirm pathspec scoping prevents accidental inclusion of unrelated staged files.
- **`retiredEvents` cleanup**: retired hooks are removed on upgrade. Verify this doesn't break users who haven't upgraded yet or who have custom hooks on the same events.
- **`ResolveColorMode`** plumbed through all `NewPrinter` call sites — any missed call site would ignore `--color`.

## Scope

Touches CLI commands (`log`, `prime`, `doctor`), output/color layer, hook management (`setup/`), draft templates, CI workflows (`devblog.yml`), the Hugo landing page, and site content. Core ledger storage and entry schema unchanged. Template engine unchanged — only template content revised.

## Test Plan

- `just check` (lint + test) must pass — covers auto-commit injection, hook detection, color resolution, and entry validation.
- Integration tests in `internal/integration/` exercise the log-pending cycle with auto-commit.
- Verify `timbers doctor` output in both terminal and `--json` modes for generation check.
- Manual: run `timbers prime` after a squash merge to confirm stale anchor warning renders correctly.
- Manual: `timbers log` on a dark terminal with `--color=auto` to confirm `AdaptiveColor` picks visible colors.
