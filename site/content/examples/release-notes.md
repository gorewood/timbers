+++
title = 'Release Notes'
date = '2026-03-02'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---

## New Features

- You can now get a health check right at session start — `timbers prime` flags missing hooks and integrations with a fix hint so you catch setup issues before you start working
- After each commit, a gentle reminder nudges you to run `timbers log` — set up automatically with `timbers init --hooks` or `timbers doctor --fix`
- `timbers draft --models` shows which LLM providers and API keys are available, so you can pick the right CLI for piping without guessing
- You can now pipe drafts through Codex and Gemini CLIs in addition to Claude — see `timbers draft --models` for verified syntax

## Improvements

- Adopting timbers in an existing repo no longer floods you with "pending" commits — pre-timbers history is recognized and skipped
- The `exec-summary` template is now called `standup` — easier to find and remember
- `timbers doctor` checks whether you have a generation CLI or API key configured and tells you exactly what to set
- Terminal output is now readable on dark backgrounds (Solarized Dark, etc.)
- Stale anchor warnings now explain what happened and what to do, instead of leaving you guessing after a squash merge

## Bug Fixes

- `timbers log` no longer hard-errors after a squash merge — stale anchors are handled gracefully and self-heal on the next entry
- `timbers prime` no longer crashes when encountering a stale anchor
- Example recipes now work on macOS default bash (3.x compatibility)
- Fixed a security vulnerability in a dependency (CVE-2026-27896)
- Removed the PostToolUse hook that fired but never displayed output — the Stop hook now reliably checks for pending work at session end
