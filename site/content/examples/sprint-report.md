+++
title = 'Sprint Report'
date = '2026-02-10'
tags = ['example', 'sprint-report']
+++

Generated with `timbers draft sprint-report --last 30 --model opus`

---

# Sprint Report: 2026-01-19 → 2026-02-10

## Summary

This sprint took timbers from project foundation through two tagged releases (v0.1.0, v0.2.0). The core recording workflow, query/export pipeline, and LLM-powered document generation all shipped, alongside substantial agent integration work (prime, hooks, onboarding) and release infrastructure. The latter half of the sprint focused heavily on polish, review fixes, and developer experience hardening.

## By Category

### Core & Foundation
- Project scaffold: spec, Cobra CLI with Charmbracelet styling, agent DX guide
- Internal packages: git exec wrapper with notes layer, ledger entry schema with validation
- Core commands: `status`, `pending`, `log`, `show` for the recording workflow
- Robustness for non-timbers git notes — graceful skip via `ErrNotTimbersNote` and schema validation

### Features
- `query` command with `--tag` filtering (OR semantics), time/range filters, JSON/Markdown output
- `export` command with `--tag` filtering extracted to shared `entry_filter.go`
- `amend` command for modifying existing entries (`--what`/`--why`/`--how`/`--tag`, `--dry-run`)
- `changelog` command generating grouped markdown from ledger entries
- `prompt` → `draft` rename with built-in LLM execution (`--model`, `--provider`), 8 built-in templates, and template resolution chain
- `catchup` command with `--limit` and `--group-by` flags for batch processing
- ADR `decision-log` template for extracting why-field data into architectural decision records
- Multi-provider LLM client with `HTTPDoer` interface for testability

### Agent DX
- `prime` command for session context injection with workflow instructions and override support
- `skill` command for agent self-documentation
- Claude hooks switched from global to project-level scope; git hooks made opt-in (`--hooks`)
- `init` now creates empty notes ref so `prime` works immediately after setup
- Prime output updated to surface `draft` command for document generation

### CLI & Output
- `lipgloss` styling helpers (`Table`, `Box`, `Section`, `KeyValue`) applied across commands with TTY-aware rendering
- Errors/warnings routed to stderr when piped; draft status hint emitted to stderr
- `doctor` enhanced with CONFIG section (config dir, env files, API keys, templates) and GitHub releases version check

### Configuration & Environment
- Centralized config dir with cross-platform support (`TIMBERS_CONFIG_HOME`, `XDG_CONFIG_HOME`, Windows AppData fallback) in `internal/config`
- `.env.local` / `.env` file loading for API keys (avoids Claude Code OAuth conflicts)

### Release & CI
- goreleaser setup with GitHub Actions workflow, `install.sh` with checksum verification
- v0.1.0 release readiness: MIT LICENSE, README rewrite, `prompt`→`draft` rename propagation, CHANGELOG, CONTRIBUTING
- v0.2.0 pre-release review: 3-agent review team, 5 parallel fix subagents for naming, output patterns, and global state issues
- CI workflow for test + lint

### Docs
- Comprehensive 8-part tutorial (install → catchup → daily workflow → agent integration → troubleshooting)
- Human oversight documentation with publishing-artifacts guide and CI/CD strategies
- README rewritten multiple times: OSS audience with badges, problem/solution framing, configuration section
- Agent DX guide updated for v0.2 learnings, opt-in hooks, and project-level defaults

## Scope

30 entries spanning the full lifecycle — from spec-first design through two releases. The early sprint was broad foundational work (core packages, CLI scaffold, query/export/LLM pipeline). The latter half narrowed to review-driven polish: cross-platform config, pipe ergonomics, hook scoping, and closing gaps surfaced by three independent reviewers. Touches ranged across the entire codebase including internals (`git`, `config`, `envfile`), all major commands, templates, docs, and CI.

## Highlights

- **`prompt` → `draft` rename with built-in LLM execution** — a significant UX pivot: the command now reads as the action it performs and can execute against LLM providers directly, eliminating the pipe-to-external-tool workflow.
- **Post-v0.2.0 review cycle** — three independent reviewers converged on the prime-after-init gap as highest priority; the fix (`git.InitNotesRef()` creating an empty notes namespace via git plumbing) unblocked the entire new-user onboarding path.
