+++
title = 'Standup'
date = '2026-02-28'
tags = ['example', 'standup']
+++

Generated with `timbers draft standup --last 20 | claude -p --model opus`

---

**Standup — Feb 13–28**

- **Stale anchor after squash merges confused real users** — shipped v0.10.2 patch with actionable warnings and coaching. Chose self-healing behavior over an explicit reset command.
- **`PostToolUse` hook was silently broken since creation** — `$TOOL_INPUT` was always empty because Claude Code passes JSON on stdin, not env vars. Ultimately removed the hook entirely since the `Stop` hook covers the same case. Shipped in v0.10.1.
- **Dark terminal color visibility was a recurring issue** — Color 8 (bright black) invisible on Solarized Dark. Fixed twice: first with a `--color` flag (v0.10.0), then replaced hardcoded `lipgloss.Color` with `AdaptiveColor` for automatic light/dark switching.
- **Agent `claude -p` piping broken inside Claude Code sessions** — `CLAUDECODE` env var must be unset or the stateless pipe is treated as a nested session. Added workaround to prime coaching.
- **Cut three releases in two weeks**: v0.9.0 (coaching rewrite informed by Opus 4.6 prompt guide), v0.10.0 (color flag, auto-commit entries, content safety), v0.10.1 + v0.10.2 (bug fixes).
- **`timbers log` now auto-commits the entry file** — eliminates the staged-but-uncommitted gap that confused users.
- **Revised draft templates**: renamed standup for discoverability, shifted PR template to intent/decisions (agents already review diffs), defaulted to pipe-first generation for subscription users.
- **Added `govulncheck` to CI and auto-version updates to the landing page** — version badge had drifted two releases behind.
- **Shipped marketing landing page** — competitive positioning against Entire.io. Polished terminal blocks and quick-start layout.
- **Devblog now skips generation when no entries exist** — previously invoked the LLM anyway, which produced apology posts.
