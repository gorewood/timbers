+++
title = 'Sprint Report'
date = '2026-02-14'
tags = ['example', 'sprint-report']
+++

Generated with `timbers draft sprint-report --since 7d | claude -p --model opus`

---

# Sprint Report: Feb 9–14, 2026

## Summary

A dense 6-day sprint that shipped 4 releases (v0.5.0 through v0.10.0), pivoted the entire storage layer from git notes to file-based storage, added an MCP server, rewrote the coaching system, and stood up a marketing site. 54 entries across core infrastructure, agent integration, and developer experience.

## By Category

### Storage Pivot
- Replaced git notes with `.timbers/<id>.json` file-per-entry storage — atomic writes, no merge conflicts, simplified `GitOps` interface from 8 to 4 methods
- Added YYYY/MM/DD directory layout for scalability at high commit volumes
- Migrated 37 existing entries from git notes to new format
- Removed all git notes code and references
- Filtered ledger-only commits from `GetPendingCommits` to break the chicken-and-egg loop where entry commits appeared as undocumented
- Auto-commit entry files in `timbers log` — closes the staged-but-uncommitted gap that confused users

### MCP Server
- Built `timbers serve` with 6 tools over stdio transport using `go-sdk` v1.3.0
- Handlers call `internal/` packages directly — zero business logic duplication
- Added `idempotentHint` to read tools for client caching/retry optimization
- Released as v0.7.0

### Agent Integration & Hooks
- Rewrote Claude Code hook integration from shell scripts to JSON `settings.json` format — the old shell script approach was a complete no-op (0% adoption)
- Expanded to 4-event lifecycle hooks: `SessionStart`, `PreCompact`, `Stop`, `PostToolUse`
- Fixed `PostToolUse` hook to read stdin instead of empty `$TOOL_INPUT` env var (broken since creation)
- Added graceful degradation — hooks warn with install URL instead of erroring when `timbers` is missing
- Switched from global to project-level hooks

### Architecture & Extensibility
- Introduced `AgentEnv` interface with registry pattern — adding Gemini/Codex support is now a single-file task
- Refactored `init`, `doctor`, `setup`, and `uninstall` to use the new interface
- `--no-claude` flag replaced with generic `--no-agent` (deprecated alias kept)

### Coaching & Content Quality
- Rewrote coaching: motivated rules (explaining WHY), concrete 5-point notes trigger checklist, XML section tags, reduced imperative density
- Added `--notes` field for deliberation capture — distinct from `--why` (verdict vs journey)
- Tightened why/notes differentiation in coaching
- Added PII/content safety coaching as a guardrail for public repos

### Developer Experience
- Added `--color` flag (`never`/`auto`/`always`) for terminal color scheme compatibility (Solarized Dark fix)
- Renamed `prompt` command to `draft`
- Added ADR `decision-log` template
- Centralized config dir with cross-platform support (`XDG_CONFIG_HOME`, Windows `AppData`)
- Added `.env.local` support for API keys
- Routed errors/warnings to stderr when piped
- Enhanced `doctor` with CONFIG section and version check against GitHub releases

### Site & Docs
- Built marketing landing page with Tailwind + GSAP animations
- Published Hugo site to GitHub Pages with 5 example artifacts
- Regenerated examples from richer ledger data (real entries with notes >> backfilled entries)
- Documented `--notes` across all docs (README, tutorial, spec, agent reference)
- Updated onboard snippet and 7 doc files for storage pivot

### CI & Release
- Fixed Hugo `baseURL` after org migration (`rbergman` → `gorewood`)
- Chained devblog workflow to pages deploy via `workflow_dispatch`
- Switched devblog CI from weekly to daily
- Removed stale git-notes fetch from CI
- Shipped v0.5.0, v0.6.0, v0.7.0, v0.8.0, v0.9.0, v0.10.0

### Testing & Fixes
- Fixed `TestCommitFiles` to use isolated temp repo instead of live HEAD
- Fixed `TestRepoRoot` for worktree compatibility
- 14 integration tests for multi-branch merge scenarios
- Added `--tag` filtering to both `query` and `export` commands with OR semantics

## Highlights

- **Storage pivot**: The move from git notes to `.timbers/` files touched 54+ files and fundamentally changed how the tool works — atomic writes, merge-safe concurrent worktrees, and a dramatically simpler Git interface. This was the prerequisite for everything else.
- **Coaching rewrite informed by Opus 4.6 prompt guide**: A council debate converged on "good coaching IS Opus-optimized coaching" — no model-specific variants needed. The concrete notes trigger checklist and motivated rules represent a qualitative shift in how the tool shapes agent behavior.
