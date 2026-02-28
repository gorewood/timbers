+++
title = 'Sprint Report'
date = '2026-02-28'
tags = ['example', 'sprint-report']
+++

Generated with `timbers draft sprint-report --since 2026-02-10 --until 2026-02-14 | claude -p --model opus`

---

## Sprint Report: Feb 10–14, 2026

### Summary

Timbers shipped six releases (v0.5.0 through v0.10.0) across five days, anchored by two major architectural changes: a complete storage pivot from git notes to file-per-entry storage under `.timbers/`, and a new MCP server exposing 6 tools over stdio. The sprint also overhauled hook integration, added a `notes` field for capturing design deliberation, and launched a marketing landing page.

### By Category

**Storage Pivot**
- Replaced git notes with `.timbers/<id>.json` flat files — atomic writes, no merge conflicts, simpler `GitOps` interface (8 methods → 4)
- Migrated 37 existing entries from git notes to file storage
- Added `YYYY/MM/DD` directory layout for scale, with 14 integration tests proving merge safety
- Filtered ledger-only commits from `GetPendingCommits` to break the chicken-and-egg loop where entry commits appeared as undocumented
- Removed all git notes code and references

**MCP Server**
- `timbers serve` subcommand using `go-sdk` v1.3.0 with 6 tools (`log`, `pending`, `query`, `show`, `status`, `prime`)
- Handlers call `internal/` packages directly — no CLI wrapper overhead
- Added `idempotentHint` to read-only tools for client caching

**Hooks & Agent Integration**
- Rewrote Claude Code hooks from shell scripts to JSON `settings.json` format (the old approach was a complete no-op)
- Expanded from single `SessionStart` hook to 4 events: `SessionStart`, `PreCompact`, `Stop`, `PostToolUse`
- Fixed `PostToolUse` hook to read stdin instead of empty `$TOOL_INPUT` env var
- Switched hooks from global to project-level scope
- Added graceful degradation — warns with install URL when `timbers` isn't on PATH

**Coaching & Entry Quality**
- Added `--notes` field for capturing deliberation (journey to a decision, distinct from `--why` verdict)
- Rewrote coaching: motivated rules (explain *why* each rule exists), concrete 5-point notes trigger checklist, XML section tags, reduced imperative density
- Tightened `--why` coaching to differentiate from `--notes`
- Added PII/content safety guardrails to prime output

**CLI & Developer Experience**
- `--color` flag (`never/auto/always`) for terminal color scheme compatibility (Solarized Dark fix)
- Auto-commit entry files in `timbers log` — eliminates the staged-but-uncommitted gap
- Renamed `prompt` → `draft` command; added ADR `decision-log` template
- Added `changelog` command with tag-based grouping
- Centralized config directory with cross-platform support (`XDG_CONFIG_HOME`, Windows `AppData`)
- `.env.local` support for API keys (avoids Claude Code OAuth conflicts)
- Enhanced `doctor` with config checks and GitHub release version check
- Routed errors/warnings to stderr when piped

**Architecture**
- `AgentEnv` interface with registry pattern — each agent environment (Claude, future Gemini/Codex) is self-contained via `init()` registration
- Refactored `init`, `doctor`, `setup`, and `uninstall` to iterate `AllAgentEnvs()`

**Site & Marketing**
- Built marketing landing page: dark theme, GSAP animations, terminal-styled code blocks
- Published Hugo site to GitHub Pages with 5 example artifacts generated from real ledger data
- Regenerated examples twice as ledger grew (55 → 69 entries)
- Fixed Hugo `baseURL` after org migration; chained devblog → pages deploy

**Releases & CI**
- Tagged v0.5.0 (multi-event hooks), v0.6.0 (storage pivot), v0.7.0 (MCP server), v0.8.0 (notes field), v0.9.0 (coaching rewrite), v0.10.0 (color flag + auto-commit + content safety)
- Switched devblog CI from weekly to daily
- Fixed stale git-notes fetch in CI

### Highlights

- **Storage pivot** was the highest-risk change — replacing the entire persistence layer across 26 files while maintaining backward compatibility. The file-per-entry approach proved its merit immediately: merge conflicts disappeared, and the simpler `GitOps` interface (4 methods vs 8) made the MCP server implementation straightforward.

- **The coaching rewrite** validated a council decision: three perspectives independently concluded that quality problems were clarity problems, not model-specific problems. Adding *motivation* to rules (explaining why each exists) and concrete trigger checklists produced measurably better agent output — entries after the rewrite contain genuine design trade-offs instead of feature descriptions.
