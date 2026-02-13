+++
title = 'Release Notes'
date = '2026-02-13'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---

# Release Notes

## New Features

- You can now capture your reasoning process with the `--notes` flag on `timbers log` — record what alternatives you considered and why you chose one approach over another. The `--why` flag captures the verdict; `--notes` captures the journey.
- Timbers now includes an MCP server (`timbers serve`) that works with any MCP-compatible editor — Claude Code, Cursor, Windsurf, Gemini CLI, and more — providing 6 tools (`pending`, `prime`, `query`, `show`, `status`, `log`) over stdio transport.
- Timbers is now agent-environment neutral. The `init`, `doctor`, `setup`, and `uninstall` commands work across multiple AI coding environments. Adding support for a new environment requires a single file implementing the `AgentEnv` interface.
- Entry storage now uses a `YYYY/MM/DD` directory layout under `.timbers/`, providing better filesystem performance at scale while keeping entries merge-safe across concurrent worktrees.

## Improvements

- `timbers pending` no longer reports entry commits as undocumented work — the chicken-and-egg problem from file-based storage has been resolved.
- MCP read tools advertise `idempotentHint`, allowing editors to optimize with caching and retry logic.
- The `--no-claude` flag has been replaced with the more generic `--no-agent` (the old flag still works as a deprecated alias).
- Documentation across README, tutorial, agent reference, and spec now covers the `--notes` flag with clear guidance on when to use it.
- The `onboard` snippet includes a `command -v` check for graceful degradation when timbers isn't installed — team members see an install URL instead of an error.

## Breaking Changes

- Storage has pivoted from git notes to `.timbers/` flat files. If you have existing entries in git notes, run the migration script or use `timbers catchup` to recreate them. The `timbers notes push/fetch` commands have been removed.
- The `--no-claude` flag on `timbers init` is deprecated in favor of `--no-agent`. The old flag continues to work but will be removed in a future release.
