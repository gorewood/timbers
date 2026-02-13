+++
title = 'Sprint Report'
date = '2026-02-13'
tags = ['example', 'sprint-report']
+++

Generated with `timbers draft sprint-report --since 7d | claude -p --model opus`

---

# Sprint Report: 2026-02-09 → 2026-02-13

## Summary

This sprint shipped three releases (v0.6.0, v0.7.0, v0.8.0) in five days, pivoting the storage layer, adding MCP server support, and making timbers agent-environment neutral. The sprint also introduced the `--notes` field for deliberation capture — the first structural addition to the entry schema since v0.1.0.

## By Category

### Storage Pivot (v0.6.0)
- Pivoted from git notes to flat directory files (`.timbers/<id>.json`) with atomic writes via temp+rename
- Adopted `YYYY/MM/DD` directory layout for filesystem scalability at high commit volumes
- Migrated all 37 existing entries from git notes to the new format
- Removed all git notes code, simplifying the `GitOps` interface from 8 to 4 methods
- Built 14 integration tests proving file-per-entry storage is inherently merge-safe across branches
- Resolved the pending chicken-and-egg problem: entry commits no longer appear as undocumented work

### MCP Server (v0.7.0)
- Added `timbers serve` subcommand with 6 tools over stdio transport using go-sdk v1.3.0
- Moved filter functions to `internal/ledger/` for sharing between CLI and MCP handlers
- Added `idempotentHint` to read tools for client-side caching optimization
- Added `validateLogInput` to catch empty strings the MCP SDK's required-field check misses
- Updated onboard snippet and all documentation for the storage pivot

### Agent-Environment Neutral (v0.8.0)
- Introduced `AgentEnv` interface with registry pattern — each environment self-registers via `init()`
- Built `ClaudeEnv` as reference implementation wrapping existing setup functions
- Refactored `init`, `doctor`, `setup`, and `uninstall` to iterate `AllAgentEnvs()`
- Replaced `--no-claude` with generic `--no-agent` (deprecated alias preserved)
- Added `--notes` flag for capturing deliberation context alongside design decisions
- Tightened why coaching to differentiate from notes: why is the verdict, notes is the journey
- Updated all documentation (README, tutorial, agent reference, spec) for `--notes`

### Testing & Fixes
- Fixed `TestCommitFiles` — used isolated temp repo instead of live HEAD (merge commits return empty from `diff-tree`)
- Fixed `TestRepoRoot` — replaced hardcoded directory name with absolute path check for worktree compatibility
- Dogfood round 3 validated notes coaching with Opus subagents: selective usage (1/3 commits), genuine thinking-out-loud quality

## Highlights

- **Storage pivot was the highest-risk change in timbers' history** — touching all 26 source files, migrating 37 entries, and removing the entire git notes subsystem. The decision to use individual files was validated by 14 merge integration tests proving conflict-free concurrent worktree operation.
- **The `AgentEnv` registry pattern** transforms multi-environment support from "change N files per environment" to "add one file." The deliberation notes on this entry — debating registry vs switch, analyzing the goal of single-file additions — are exactly the kind of content `--notes` was designed to capture.
