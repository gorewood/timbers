# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Added goreleaser configuration for automated GitHub releases with `install.sh` script and checksum verification
- Added `--limit/-l` and `--group-by/-g` flags to `catchup` command for batch size control and grouping strategy
- Added `--model/-m` and `--provider/-p` flags to `prompt` command for direct LLM execution
- Added `ListEntriesWithStats()` for `status --verbose` with non-timbers note diagnostics
- Added `ErrNotTimbersNote` and schema validation in entry parsing for graceful handling of non-timbers git notes
- Added installation instructions to README with curl|bash and go install methods
- Added lipgloss styling helpers (`Table`, `Box`, `Section`, `KeyValue`) with TTY-aware rendering
- Added multi-provider LLM client with `generate` and `catchup` commands
- Added `HTTPDoer` interface for testable HTTP mocks
- Added Just recipes for LLM report generation with `just prompt` command
- Added comprehensive tutorial covering installation, batch catchup, daily workflow, agent integration, querying, LLM reports, and troubleshooting
- Added documentation for human oversight and publishing artifacts with CI/CD strategies
- Added workflow instructions to `prime` command with override support
- Added `uninstall` command to remove timbers from repository
- Added auto mode that parses commit messages for automatic entry generation
- Added batch grouping by beads trailers or dates
- Added `prompt` command with 8 built-in LLM templates and resolution chain
- Added `query` command with tag and time filters
- Added JSON and Markdown export capabilities
- Added `prime` command for session context injection
- Added `skill` command for agent skills documentation
- Added `notes` subcommands for git notes synchronization
- Added git exec wrapper with notes layer abstraction
- Added ledger entry schema with validation
- Added core CLI commands: `status`, `pending`, `log`, and `show`
- Added Cobra CLI scaffold with Charmbracelet styling
- Added comprehensive agent DX patterns guide

### Changed
- Rewrote README with problem/solution framing
- Set up goreleaser infrastructure with GitHub Actions workflow

### Fixed
- Fixed empty content handling in LLM responses
- Added input validation and truncated error bodies for LLM client reliability

### Technical
- Git notes storage under `refs/notes/timbers` for portable, syncable ledger
- Structured entry schema `timbers.devlog/v1` with ISO8601 timestamps
- Entry ID format: `tb_<timestamp>_<anchor-short-sha>`
- Workset tracking with anchor commit, commit range, and diffstat
- TTY detection for automatic color/format adjustment
- Structured error handling with exit codes (0=success, 1=user, 2=system, 3=conflict)
- Spec-first design with dialectical refinement approach
