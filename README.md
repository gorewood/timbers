# Timbers

[![Release](https://img.shields.io/github/v/release/gorewood/timbers)](https://github.com/gorewood/timbers/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/gorewood/timbers)](https://goreportcard.com/report/github.com/gorewood/timbers)

**Git knows what changed. Timbers captures why.**

A development ledger that pairs Git commits with structured rationale — what you did, why you did it, how you approached it — stored as portable Git notes.

## Installation

```bash
# One-liner (Linux/macOS)
curl -fsSL https://raw.githubusercontent.com/gorewood/timbers/main/install.sh | bash

# Or with Go
go install github.com/gorewood/timbers/cmd/timbers@latest
```

## Quick Start

```bash
# Initialize (one-time)
timbers init

# Record work
timbers log "Fixed auth bypass" \
  --why "User input wasn't sanitized before JWT validation" \
  --how "Added validation middleware before auth handler"

# Query your ledger
timbers query --last 10
```

## Why Timbers?

As AI agents take on more development work, human oversight becomes harder and more critical. Git history shows *what* changed. Commit messages hint at *how*. But the *why* — the reasoning, constraints, decisions, context — lives in agent session logs that get compacted, Slack threads that scroll away, and PR comments that nobody reads twice.

Six months later, you're staring at agent-written code wondering "why did it do it this way?" The answer is gone.

Timbers captures that context at the moment it exists and stores it durably alongside your code. The ledger grows as agents document their work. Humans harvest insights when they need them — executive summaries, decision logs, queryable history.

## Core Commands

| Command | Purpose |
|---------|---------|
| `log` | Record work with what/why/how |
| `pending` | Show commits awaiting documentation |
| `query` | Search entries by time, tags, or content |
| `show` | Display a single entry |
| `export` | Export as JSON or Markdown |
| `draft` | Generate documents from your ledger (changelogs, reports, blogs) |
| `prime` | Session context injection for agents |
| `notes` | Manage Git notes sync (`init`, `push`, `fetch`, `status`) |
| `status` | Repository and notes state |

All commands support `--json`. Write operations support `--dry-run`.

## Document Generation

The `draft` command renders templates with your ledger entries, producing changelogs, reports, decision logs, and more — either as text for piping to an LLM or with built-in LLM execution via `--model`.

```bash
# Pipe to external LLM
timbers draft changelog --since 7d | claude -p

# Built-in LLM execution
timbers draft exec-summary --last 10 --model opus
timbers draft decision-log --last 20 --model haiku

# List available templates
timbers draft --list
```

**Built-in templates:** `changelog`, `decision-log`, `devblog`, `exec-summary`, `pr-description`, `release-notes`, `sprint-report`

The decision-log template is particularly valuable — it extracts the *why* behind each change into an architectural decision record. No other tool produces this from structured commit data.

**Model guidance:** For best output quality, use `opus`. For routine generation at lower cost, `haiku` or `local` models work well. Experiment to find the model that gives you the best output at your preferred price point.

## Configuration

Timbers uses `~/.config/timbers/` as its global configuration directory (`%AppData%\timbers` on Windows, or `$XDG_CONFIG_HOME/timbers` if set).

```
~/.config/timbers/
├── env              # API keys (loaded as fallback when not in environment)
├── templates/       # Global custom templates (available in all repos)
```

### API Keys

For LLM-powered commands (`draft --model`, `generate`, `catchup`), set API keys in `~/.config/timbers/env`:

```bash
mkdir -p ~/.config/timbers
cat > ~/.config/timbers/env << 'EOF'
ANTHROPIC_API_KEY=sk-ant-...
# OPENAI_API_KEY=sk-...
# GOOGLE_API_KEY=...
EOF
```

Env file resolution (first match wins, environment variables always take precedence):

1. `$CWD/.env.local` — per-repo override
2. `$CWD/.env` — per-repo
3. `~/.config/timbers/env` — global fallback

### Custom Templates

Create custom templates for project-specific or personal reporting needs:

```bash
# Global (available in all repos)
mkdir -p ~/.config/timbers/templates
cat > ~/.config/timbers/templates/weekly-standup.md << 'EOF'
Summarize this week's work for a standup meeting.
Format as Completed / In Progress / Blockers (3-5 bullets each).
## Entries
EOF

# Per-repo (takes precedence over global)
mkdir -p .timbers/templates
# Same format, placed in .timbers/templates/<name>.md
```

Template resolution: `.timbers/templates/` → `~/.config/timbers/templates/` → built-in.

## Agent Integration

Timbers is designed for agents to use directly:

```bash
# Session start: agent gets context
timbers prime

# After work: agent documents it
timbers pending
timbers log "Implemented rate limiting" \
  --why "API abuse detected in logs" \
  --how "Token bucket with Redis backend"
timbers notes push
```

Agent-friendly features: `--json` everywhere, `prime` for context injection, `pending` for clear signals, structured errors with recovery hints.

**Agent environment support:** Timbers is currently tested with [Claude Code](https://claude.ai/claude-code). The CLI itself is agent-agnostic (any tool that can run shell commands works), but the `setup` and `doctor` commands currently target Claude Code hooks. Support for Gemini CLI, Codex CLI, and other agent environments will be added as demand warrants.

## Documentation

- [Tutorial](docs/tutorial.md) — Setup, catching up history, agent integration
- [Publishing Artifacts](docs/publishing-artifacts.md) — CI/CD for changelogs, reports, blogs
- [Agent Reference](docs/agent-reference.md) — Command reference for agent integration
- [LLM Commands](docs/llm-commands.md) — Draft, generate, catchup commands
- [Spec](docs/spec.md) — Full specification
- [Agent DX Guide](docs/agent-dx-guide.md) — CLI design patterns for agents

## Dogfood Artifacts

Timbers' own development ledger is used to generate showcase artifacts via `timbers draft`. These demonstrate what the tool produces from real data.

> **A note on quality:** Timbers was bootstrapped late in its own development, so most ledger entries were backfilled using `timbers catchup` — which infers what/why/how from commit messages and diffs. Catchup can only make inferences from what Git already records, producing weaker entries than what you get from documenting work in the moment. This is exactly why timbers exists: the reasoning that *isn't* in the commit message is the part that matters most. The dogfood artifacts are illustrative of the format and pipeline, but a project that uses `timbers log` from day one will produce significantly richer output.

## Development

```bash
just setup    # First-time setup
just check    # Lint + test (required before commit)
just fix      # Auto-fix lint issues
just run      # Run the CLI
```

## License

[MIT](LICENSE)
