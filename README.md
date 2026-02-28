# Timbers

[![Release](https://img.shields.io/github/v/release/gorewood/timbers)](https://github.com/gorewood/timbers/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/gorewood/timbers)](https://goreportcard.com/report/github.com/gorewood/timbers)

**Git knows what changed. Timbers captures why.**

AI agents write more code every day. Git tracks what they changed. Commit messages hint at how. But the *why* — the reasoning, trade-offs, and decisions — lives in session logs that get compacted and PR comments nobody reads twice. Six months later, you're staring at agent-written code with no idea why it was done that way.

Timbers is a development ledger that captures what/why/how as structured JSON files in `.timbers/` — portable, queryable, and durable. Agents document their reasoning at the moment it exists. Humans harvest insights whenever they need them.

```bash
# Record work (agents or humans)
timbers log "Fixed auth bypass" \
  --why "Chose validation over rate limiting — catches root cause" \
  --how "Added validation middleware before auth handler" \
  --notes "Debated rate limiting vs input validation. Rate limiting masks the problem."

# Generate artifacts from your ledger
timbers draft decision-log --last 20 --model opus
```

**[Website](https://gorewood.github.io/timbers/)** · **[Dev Blog](https://gorewood.github.io/timbers/posts/)**

## Installation

```bash
# One-liner (Linux/macOS)
curl -fsSL https://raw.githubusercontent.com/gorewood/timbers/main/install.sh | bash

# Or with Go
go install github.com/gorewood/timbers/cmd/timbers@latest
```

## Quick Start

```bash
timbers init              # One-time setup
timbers log "what" \      # Record work
  --why "why" --how "how"
timbers query --last 10   # Query your ledger
```

## Core Commands

| Command | Purpose |
|---------|---------|
| `log` | Record work with what/why/how (optional `--notes` for deliberation) |
| `pending` | Show commits awaiting documentation |
| `query` | Search entries by time, tags, or content |
| `show` | Display a single entry |
| `export` | Export as JSON or Markdown |
| `draft` | Generate documents from your ledger (changelogs, reports, blogs) |
| `prime` | Session context injection for agents |
| `status` | Repository and ledger state |

All commands support `--json`. Write operations support `--dry-run`.

## Document Generation

The `draft` command renders templates with your ledger entries, producing changelogs, reports, decision logs, and more — either by piping to an LLM CLI or with built-in LLM execution via `--model`.

```bash
# Pipe to any LLM CLI (uses your subscription, not API tokens)
timbers draft changelog --since 7d | claude -p --model opus
timbers draft standup --since 1d | gemini
timbers draft pr-description --range main..HEAD | codex exec -m gpt-5-codex-mini -

# Built-in LLM execution (for CI/CD or when no CLI is available)
timbers draft standup --last 10 --model opus

# List available templates
timbers draft --list
```

**Built-in templates:** `changelog`, `decision-log`, `devblog`, `pr-description`, `release-notes`, `sprint-report`, `standup`

The decision-log template is particularly valuable — it extracts the *why* behind each change into an architectural decision record, enriched by `--notes` when agents capture their deliberation process. No other tool produces this from structured commit data.

**Model guidance:** Use `opus` for best output quality. For local generation, pipe to your LLM CLI of choice (`claude -p`, `gemini`, `codex exec`) — this uses your subscription instead of API tokens. For CI/CD, use `--model opus` with an API key. For high-volume batch operations (e.g., `catchup` over hundreds of commits), `haiku` or `local` models offer a lower-cost alternative.

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
  --why "Token bucket chosen over sliding window for simplicity" \
  --how "Token bucket with Redis backend" \
  --notes "Considered sliding window but token bucket covers our load patterns"
```

Agent-friendly features: `--json` everywhere, `prime` for context injection, `pending` for clear signals, `--notes` for capturing deliberation, structured errors with recovery hints.

**Agent environment support:** Timbers is built and tested with [Claude Code](https://claude.ai/claude-code), which has the deepest integration via hooks that auto-inject `timbers prime` at session start. The CLI itself is agent-agnostic — any agent that can run shell commands can use timbers.

For **non-Claude agents** (Gemini CLI, Cursor, Windsurf, Codex, Kilo Code, Continue, Aider, etc.), add this to your agent's instruction file (`AGENTS.md`, `GEMINI.md`, `.cursor/rules/`, `.windsurfrules`, etc.):

```
At the start of every session, run `timbers prime` and follow the workflow it describes.
After completing work, run `timbers pending` to check for undocumented commits,
then `timbers log "what" --why "why" --how "how"` to document your work.
Use `--notes` when you explored alternatives or made a real choice.
```

Native hooks and setup commands for additional agent environments are planned for a future release.

## Documentation

- [Tutorial](docs/tutorial.md) — Setup, catching up history, agent integration
- [Publishing Artifacts](docs/publishing-artifacts.md) — CI/CD for changelogs, reports, blogs
- [Agent Reference](docs/agent-reference.md) — Command reference for agent integration
- [LLM Commands](docs/llm-commands.md) — Draft, generate, catchup commands
- [Spec](docs/spec.md) — Full specification
- [Agent DX Guide](docs/agent-dx-guide.md) — CLI design patterns for agents

## Example Artifacts

Timbers' own development ledger is used to generate these examples via `timbers draft`. Each link is a live artifact produced from real data:

| Artifact | Description |
|----------|-------------|
| [Changelog](https://gorewood.github.io/timbers/examples/changelog/) | Keep a Changelog format, grouped by type |
| [Decision Log](https://gorewood.github.io/timbers/examples/decision-log/) | ADR-style architectural decisions extracted from the *why* field |
| [Standup](https://gorewood.github.io/timbers/examples/standup/) | Daily standup from recent work |
| [Release Notes](https://gorewood.github.io/timbers/examples/release-notes/) | User-facing release notes |
| [Sprint Report](https://gorewood.github.io/timbers/examples/sprint-report/) | Categorized sprint summary with scope and highlights |
| [Dev Blog](https://gorewood.github.io/timbers/posts/) | Weekly dev blog posts (Carmack .plan style) |

> **A note on quality:** Most of these entries were backfilled using `timbers catchup`, which infers what/why/how from commit messages and diffs. Projects that use `timbers log` from day one will produce significantly richer output — especially in the decision-log, where the *why* field matters most.

## Development

```bash
just setup    # First-time setup
just check    # Lint + test (required before commit)
just fix      # Auto-fix lint issues
just run      # Run the CLI
```

## Acknowledgments

Timbers was built and tracked with [Beads](https://github.com/steveyegge/beads), a lightweight issue tracker stored in Git refs. Beads' CLI design — structured output, agent-friendly ergonomics, Git-native storage — directly inspired the patterns in timbers' [Agent DX Guide](docs/agent-dx-guide.md). If you're building tools that agents need to use, studying beads is a great place to start.

## License

[MIT](LICENSE)
