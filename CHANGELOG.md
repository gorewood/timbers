# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-02-10

Initial public release.

### Added

- **Core CLI** with `log`, `pending`, `show`, `status`, `query`, and `export` commands
- **`log` command** for recording work with structured what/why/how fields, `--auto` mode for extracting from commit messages, `--batch` mode for grouping by work-item trailers
- **`pending` command** showing undocumented commits since last ledger entry
- **`query` command** with `--last`, `--since`, `--until`, `--tags`, and `--oneline` filtering
- **`export` command** for JSON and Markdown output, with `--tag` filtering and directory output via `--out`
- **`draft` command** (document generation) rendering templates with ledger entries for LLM consumption or direct execution via `--model`
- **Built-in templates**: `changelog`, `decision-log`, `devblog`, `exec-summary`, `pr-description`, `release-notes`, `sprint-report`
- **`generate` command** as a composable LLM completion primitive with multi-provider support (Anthropic, OpenAI, Google, local)
- **`catchup` command** for auto-generating entries from undocumented commits using LLMs, with `--batch-size`, `--parallel`, and `--dry-run` support
- **`amend` command** for updating existing ledger entries (what/why/how/tags)
- **`prime` command** for agent session context injection with `--verbose` flag for recent entry details
- **`notes` subcommands** (`init`, `push`, `fetch`, `status`) for Git notes sync management
- **`init` command** for full setup including notes, hooks, and Claude integration
- **`onboard` command** for generating CLAUDE.md integration snippets
- **`doctor` command** for health checks and diagnostics
- **`uninstall` command** for clean removal from repos
- Multi-provider LLM client supporting Anthropic, OpenAI, Google, and local (LM Studio/Ollama) models
- `--json` flag on all commands for structured output
- `--dry-run` flag on all write operations
- Structured error JSON with recovery hints and consistent exit codes (0/1/2/3)
- Lipgloss terminal styling with TTY-aware rendering
- Pipe-safe output: errors and warnings routed to stderr when stdout is piped
- Goreleaser configuration for cross-platform binary releases (Linux, macOS, Windows)
- Install script (`install.sh`) with checksum verification
- GitHub Actions workflows for releases, dev blog generation, and CI (test + lint)
- Comprehensive documentation: tutorial, agent reference, LLM commands guide, publishing artifacts guide, agent DX guide, and spec

[0.1.0]: https://github.com/gorewood/timbers/releases/tag/v0.1.0
