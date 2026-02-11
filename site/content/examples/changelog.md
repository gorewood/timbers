+++
title = 'Changelog'
date = '2026-02-10'
tags = ['example', 'changelog']
+++

Generated with `timbers draft changelog --last 30 --model opus`

---

# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Added `timbers amend` command to modify existing ledger entries with `--what`/`--why`/`--how`/`--tag` flags, `--dry-run` preview, and JSON support
- Added `--tag` filtering to `timbers query` command with OR semantics for broad discovery
- Added `--tag` filtering to `timbers export` command for feature parity with `query`
- Added `timbers changelog` command that generates markdown changelogs from ledger entries, grouped by tag
- Added `timbers draft` command (renamed from `prompt`) for generating documents from ledger data
- Added ADR decision-log template for extracting architectural decision records from entries
- Added `--limit` and `--group-by` flags to `catchup` command for batch size and grouping control
- Added built-in LLM execution to draft command via `--model`/`--provider` flags
- Added multi-provider LLM client with `generate` and `catchup` commands
- Added `.env.local` support for API keys as environment variable fallback
- Added CONFIG section and version check to `doctor` command, querying GitHub releases API
- Added cross-platform config directory support via `internal/config` package with `TIMBERS_CONFIG_HOME`, `XDG_CONFIG_HOME`, and Windows AppData resolution
- Added `init` step that creates empty notes ref so `prime` works immediately after initialization
- Added lipgloss styling helpers (`Table`, `Box`, `Section`, `KeyValue`) with TTY-aware rendering across CLI commands
- Added lipgloss style sets to `doctor` and `init` commands
- Added query, export, and agent integration commands (`prime`, `skill`, `notes` subcommands)
- Added auto mode for parsing commit messages and batch grouping by work-item or date
- Added 8 built-in LLM prompt templates with resolution chain
- Added `ErrNotTimbersNote` and schema validation for graceful handling of non-timbers git notes
- Added `ListEntriesWithStats()` for `status --verbose`
- Added goreleaser infrastructure with GitHub Actions workflow, `install.sh` with checksum verification, and `just install-local`
- Added CI workflow for test and lint
- Added MIT LICENSE, CHANGELOG, and CONTRIBUTING.md
- Added comprehensive tutorial covering installation, batch catchup, daily workflow, agent integration, querying, and troubleshooting
- Added publishing-artifacts documentation with GitHub Actions examples and model recommendations
- Added `prime` workflow instructions output with override support
- Added `uninstall` command for clean removal from repo (optionally binary)
- Added tests for `prime` guard (silent-exit path) and `doctor --fix` (auto-install path)
- Added just recipes for LLM report generation

### Changed
- Renamed `prompt` command to `draft` to better communicate document generation intent
- Renamed `gpt-5` alias to `gpt` for clarity
- Renamed `warn` style to `skip` in `init` command styles
- Switched Claude hooks from global to project-level scope (global available via `--global` flag)
- Changed git hooks from default-on to opt-in via `--hooks` flag (previously `--no-hooks` to opt out)
- Routed errors and warnings to stderr when stdout is piped to prevent corrupting piped output
- Added draft status hint to stderr when output is piped
- Surfaced `draft` command in `prime` output so agents discover document generation
- Rewrote README with problem/solution framing, badges, OSS audience focus, and configuration documentation
- Updated agent DX guide for opt-in hooks, project-level defaults, and v0.2 learnings
- Propagated `prompt` → `draft` rename across documentation, justfile, and CI

### Fixed
- Fixed `ListTemplates` override bug in template resolution
- Fixed organization owner references (`rbergman` → `gorewood`) in goreleaser and install script
- Fixed global state issue in `status.go` closure pattern
- Fixed inconsistent output patterns in `amend.go` and icon usage in `uninstall.go`

### Technical
- Established project foundation with spec-first design, Cobra CLI scaffold with Charmbracelet styling, and agent DX patterns guide
- Implemented git exec wrapper with notes layer, ledger entry schema with validation, and core workflow commands (`status`, `pending`, `log`, `show`)
- Created `internal/config` package with `Dir()` function for centralized config directory resolution
- Created `internal/envfile` package for `.env.local` and `.env` file loading
- Added `git.InitNotesRef()` using git plumbing (`mktree` + `commit-tree` + `update-ref`) to create empty notes namespace during init
- Implemented `HTTPDoer` interface for LLM client testability and added input validation with truncated error bodies
- Extracted `filterEntriesByTags` and `entryHasAnyTag` to `entry_filter.go` for reuse across commands
- Extracted `canUseOptimizedPath()` and `applyQueryFilters()` to reduce cyclomatic complexity in query command
- Added `output.Printer` `WithStderr()` for routing `Warn()` to stderr
- Renamed `prompt.go` to `draft.go` for file naming consistency
