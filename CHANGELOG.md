# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `timbers log` command for recording development ledger entries with what/why/how
- `timbers pending` command for showing undocumented commits since last entry
- `timbers status` command for displaying repository and Git notes state
- `timbers show` command for displaying individual ledger entries
- `timbers query` command for searching entries by tags and content
- `timbers export` command for pipeline exports (JSON and Markdown formats)
- `timbers prime` command for session context injection in agent workflows
- `timbers skill` command for emitting agent skill documentation
- `timbers notes init` subcommand for initializing Git notes remote
- `timbers notes push` subcommand for syncing notes to remote
- `timbers notes fetch` subcommand for pulling notes from remote
- `timbers notes status` subcommand for checking notes sync status
- `--auto` flag for auto-extracting what/why/how from commit messages
- `--batch` flag for creating entries grouped by work item or day
- `--json` flag on all commands for structured output
- `--dry-run` flag on write operations for safe previewing
- Integration tests for full workflow validation
- Command grouping in help output (Core, Query, Sync, Agent)

### Technical
- Git notes storage under `refs/notes/timbers` for portable, syncable ledger
- Structured entry schema: `timbers.devlog/v1` with ISO8601 timestamps and commit tracking
- Entry ID format: `tb_<timestamp>_<anchor-short-sha>` for unique identification
- Workset tracking: captures anchor commit, commit range, and diffstat metadata
- TTY detection for automatic color/format adjustment based on output piping
- Structured error handling with exit codes: 0 (success), 1 (user error), 2 (system error), 3 (conflict)
- Recovery hints in error messages for agent troubleshooting
- No global state; all dependencies injected for testability
