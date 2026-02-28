+++
title = 'Sprint Report'
date = '2026-02-27'
tags = ['example', 'sprint-report']
+++

Generated with `timbers draft sprint-report --last 20 | claude -p --model opus`

---

## Sprint Report: Feb 13–28, 2026

### Summary

Three releases shipped (v0.10.0, v0.10.1, v0.10.2) on top of the v0.9.0 coaching rewrite, delivering terminal color compatibility, auto-commit for `timbers log`, and a marketing landing page. The hook system saw a fix-then-remove cycle after discovering Claude Code doesn't surface `PostToolUse` stdout. Late-sprint work revised draft templates and added a `doctor` generation check.

### By Category

**CLI Features**
- `--color` flag (`never/auto/always`) plumbed through all `NewPrinter` call sites for terminals that don't report color schemes
- `timbers log` now auto-commits the entry file via pathspec-scoped `git commit`, eliminating the staged-but-uncommitted gap
- `doctor` gained a generation readiness check — detects `claude` CLI and LLM API keys, shows env var names (not provider labels)
- Renamed `exec-summary` template to `standup`; rewrote `pr-description` to focus on intent/decisions over diff rehash
- Added pipe-first generation defaults and `CLAUDECODE=` workaround for nested session guards

**Terminal Compatibility**
- Replaced all hardcoded `lipgloss.Color(8)` with `AdaptiveColor` — bright black was invisible on Solarized Dark
- `--color` flag gives users explicit override since terminals don't reliably report dark/light

**Hook System**
- Fixed `PostToolUse` hook to read stdin instead of empty `$TOOL_INPUT` env var (broken since creation)
- Then removed `PostToolUse` entirely — Claude Code doesn't surface hook stdout, so `Stop` hook covers the same check
- Added retired event cleanup so upgrades remove stale hooks automatically

**Stale Anchor Handling**
- Fixed `ErrStaleAnchor` handling in `prime` that produced confusing output after squash merges
- Added actionable warnings and coaching section — self-healing behavior makes an explicit reset command unnecessary

**Site & Marketing**
- Built marketing landing page: dark theme, GSAP animations, terminal-styled code blocks (positioned against Entire.io)
- Polished terminal blocks — left-aligned bodies, continuation indent, wider quick start container
- Updated landing page to v0.10.0; added auto-version-bump in `just release`
- Regenerated all four site examples from a 69-entry ledger

**CI & Tooling**
- `govulncheck` added as separate recipe (not gating `just check` to avoid blocking on stdlib patches)
- Devblog workflow now skips generation when no entries exist — previously invoked the LLM which produced apology posts
- Deleted 10 blank posts from that bug

**Releases**
- **v0.9.0**: Coaching rewrite (motivated rules, concrete notes triggers, XML structure)
- **v0.10.0**: `--color` flag, auto-commit, content safety coaching, hook fix, landing page
- **v0.10.1**: `PostToolUse` removal, site refresh
- **v0.10.2**: Stale anchor fix

### Highlights

- **The `PostToolUse` arc** is a good case study: the hook was silently broken from day one (`$TOOL_INPUT` always empty), got properly fixed to read stdin, then got removed entirely two days later when it turned out Claude Code swallows hook stdout anyway. The `Stop` hook already covered the use case.
- **Landing page launch** — first public-facing marketing surface beyond the README, deliberately positioned opposite Entire.io's automatic capture approach.
