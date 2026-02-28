+++
title = 'Pr Description'
date = '2026-02-28'
tags = ['example', 'pr-description']
+++

Generated with `timbers draft pr-description --since 2026-02-10 --until 2026-02-11 | claude -p --model opus`

---

## Why

Timbers needed to move from git notes storage to file-per-entry storage (`.timbers/`) to eliminate merge conflicts in concurrent worktrees and simplify the git interface. This also required rewriting the Claude Code hook system (from shell scripts to JSON settings, then from a single hook to multi-event lifecycle coverage), and reshaping the CLI surface (`prompt` → `draft`, config centralization, graceful degradation) to support real adoption by agents and teams.

## Design Decisions

- **File-per-entry over git notes**: Individual JSON files enable atomic writes (temp+rename), eliminate merge conflicts across worktrees, and narrowed `GitOps` from 8 methods to 4. Trade-off: more files in the repo, but git handles sparse directories well.
- **YYYY/MM/DD directory layout over flat**: Flat directory won't scale at 100+ commits/day. Daily buckets stay sparse while providing natural partitioning. `WalkDir` replaces `ReadDir`.
- **JSON settings hooks over shell scripts**: Claude Code reads hooks from `settings.json`/`settings.local.json`, not `.claude/hooks/*.sh`. The shell script approach caused 0% hook adoption — `timbers prime` never ran at session start.
- **4 reinforcement points over single SessionStart hook**: Single hook had 0% compliance for undocumented commits. SessionStart + PreCompact + PostToolUse(post-commit) + Stop covers the full session lifecycle.
- **Project-level hooks over global**: Global hook ran `timbers prime` in every repo, including uninitiated ones. Project-level is the natural scope since timbers requires per-repo init.
- **Git hooks opt-in (`--hooks`) over opt-out (`--no-hooks`)**: Git pre-commit hooks conflicted with tools like beads that need pre-commit for critical operations.
- **`draft` over `prompt`**: "Prompt" is developer jargon that undersells document generation. "Draft" reads as the action it performs (draft a changelog, draft release notes).
- **ADR decision-log template**: The `--why` field data is uniquely suited to architectural decision record extraction — no existing template exploited it.
- **`.env.local` for API keys**: Claude Code conflicts with `ANTHROPIC_API_KEY` in the environment (confuses it with OAuth). File-based fallback avoids the clash.
- **`command -v` guard in hooks**: Team members cloning a timbers-enabled repo without timbers installed would get hook errors at every session start, blocking adoption. Warn with install URL instead.
- **Markdown changelog output over structured JSON**: Changelogs are human-readable documents. Group-by-tag with duplication over primary-tag assignment because discoverability in context matters more than deduplication.

## Risk & Reviewer Attention

- **Storage migration path**: 37 entries migrated from git notes to `.timbers/` files. The git notes code is fully removed — there's no backward compatibility. Repos using the old format need the migration script.
- **`internal/ledger/storage.go`**: `atomicWrite` extracted for the temp+rename pattern. Verify the rename-across-filesystems edge case.
- **`internal/setup/claude.go`**: Complete rewrite from shell script manipulation to JSON settings manipulation (`readSettings`/`writeSettings`/`addTimbersPrime`/`removeTimbersPrime`). Complex parsing logic with backward-compat detection via `isTimbersPrimeCommand()`.
- **Hook upgrade path**: `addTimbersHooks`/`removeTimbersHooks` must preserve existing non-timbers hooks in settings. Verify the `filterGroups`/`filterHooks` helpers don't drop user hooks.
- **`internal/config`**: New cross-platform config dir resolution (`TIMBERS_CONFIG_HOME` → `XDG_CONFIG_HOME` → Windows AppData → `~/.config/timbers`). Verify Windows path handling if that's a target platform.

## Scope

Major cross-cutting change: touches storage (`internal/ledger`), CLI commands (`cmd/timbers/`), hook integration (`internal/setup`), config resolution (`internal/config`), environment loading (`internal/envfile`), output formatting (`internal/output`), draft templates, documentation, CI, and the Hugo site. The `internal/draft` package (renamed from `internal/prompt`) and `internal/export` are structurally intact but updated for the storage interface change.

## Test Plan

- 14 integration tests cover multi-branch merge scenarios for file-per-entry storage.
- `internal/setup/` has 27 tests across 3 files covering JSON hook manipulation, backward-compat detection, and upgrade paths.
- Table-driven tests for changelog grouping, filtering, tag duplication, JSON output, and empty states.
- `doctor --fix` auto-install path and prime silent-exit guard path have dedicated tests.
- Run `just check` (lint + full test suite).
