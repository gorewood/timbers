# Timbers

[![Release](https://img.shields.io/github/v/release/gorewood/timbers)](https://github.com/gorewood/timbers/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/gorewood/timbers)](https://goreportcard.com/report/github.com/gorewood/timbers)

**Git knows what changed. Timbers captures why.**

AI agents write more code every day. Git tracks what they changed. Commit messages hint at how. But the *why* — the reasoning, trade-offs, and decisions — lives in session logs that get compacted and PR comments nobody reads twice. Six months later, you're staring at agent-written code with no idea why it was done that way.

Timbers is a development ledger that captures what/why/how as structured JSON files in `.timbers/` — portable, queryable, and durable. Agents document their reasoning at the moment it exists. Humans harvest insights whenever they need them.

```bash
# Record work (agents or humans)
timbers log "Switched to cursor-based pagination" \
  --why "Offset pagination skips rows when items are inserted between pages" \
  --how "Opaque cursor tokens encoding created_at + id" \
  --notes "Offset was simpler but users reported duplicate items in feeds. Cursors are stable under concurrent writes."

# Generate artifacts from your ledger
timbers report decision-digest --model opus
timbers report project-update --model opus
```

**[Website](https://gorewood.github.io/timbers/)** · **[Tutorial](docs/tutorial.md)** · **[Examples](https://gorewood.github.io/timbers/examples/)** · **[Dev Blog](https://gorewood.github.io/timbers/posts/)**

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
# Or omit "what" to snapshot the selected commit subject(s)
timbers log --why "why" --how "how"
timbers query --last 10   # Query your ledger
```

## Core Commands

| Command | Purpose |
|---------|---------|
| `log` | Record work with what/why/how (optional `--notes` for deliberation) |
| `ack` | Record why a commit intentionally needs no content entry |
| `amend` | Correct an existing ledger entry |
| `pending` | Show commits awaiting documentation |
| `query` | Retrieve entries by time, tags, or Git range |
| `show` | Display a single entry |
| `export` | Export as JSON or Markdown |
| `draft` | Generate documents from your ledger (changelogs, reports, blogs) |
| `report` | Run a report profile with configured scope and compact input |
| `prime` | Session context injection for agents |
| `status` | Repository and ledger state |
| `doctor` | Diagnose storage, configuration, hooks, and ledger integrity |

All commands support `--json`. Write operations support `--dry-run`.

## Document Generation

Use `report` for repeatable report profiles with a useful default scope. Use
`draft` when you want to choose the template and entry range explicitly. Both
print the resolved prompt without `--model`, so either can be piped to an LLM
CLI; both can execute directly with `--model`.

```bash
# Profile supplies its default scope and compact decision-oriented input
timbers report decision-digest
timbers report decision-digest --model opus

# Recurring user and stakeholder update (defaults to the last 7 days)
timbers report project-update --model opus

# An explicit scope replaces the profile default
timbers report decision-digest --since 30d --model opus
```

The `draft` command renders templates with your ledger entries, producing changelogs, reports, decision digests, and more — either by piping to an LLM CLI or with built-in LLM execution via `--model`.

```bash
# Pipe to any LLM CLI (uses your subscription, not API tokens)
timbers draft changelog --since 7d | claude -p --model opus
timbers draft standup --since 1d | gemini
timbers draft pr-description --range main..HEAD | codex exec -m gpt-5-codex-mini -

# Built-in LLM execution (for CI/CD or when no CLI is available)
timbers draft standup --since 1d --model opus

# List available templates
timbers draft --list
```

**Built-in templates:** `changelog`, `decision-digest`, `devblog`, `pr-description`, `project-update`, `release-notes`, `sprint-report`, `standup`

The decision-digest template extracts explicit choices and trade-offs from `--why` and `--notes` into a retrospective report. It deliberately does not create or replace a project's authoritative ADRs.

**Model guidance:** Use `opus` for best output quality. For local generation, pipe to your LLM CLI of choice (`claude -p`, `gemini`, `codex exec`) — this uses your subscription instead of API tokens. For CI/CD, use `--model opus` with an API key.

## Configuration

Timbers uses `~/.config/timbers/` as its global configuration directory (`%AppData%\timbers` on Windows, or `$XDG_CONFIG_HOME/timbers` if set).

```
~/.config/timbers/
├── env              # API keys (loaded as fallback when not in environment)
├── templates/       # Global custom templates (available in all repos)
```

### API Keys

For LLM-powered commands (`report --model`, `draft --model`, `generate`), set API keys in `~/.config/timbers/env`:

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
{{entries_json}}
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
timbers log "Added write-through caching to product queries" \
  --why "Read-aside cache served stale data after writes — write-through keeps it consistent" \
  --how "Cache update in same transaction as DB write, TTL fallback" \
  --notes "Write-behind was faster but risks data loss on crash. Consistency wins over throughput here."
```

Agent-friendly features: `--json` everywhere, `prime` for context injection, `pending` for clear signals, `--notes` for capturing deliberation, structured errors with recovery hints.

**Agent environment support:** Timbers is built and tested with [Claude Code](https://claude.ai/claude-code), which has the deepest integration via hooks that auto-inject `timbers prime` at session start. The CLI itself is agent-agnostic — any agent that can run shell commands can use timbers.

For **non-Claude agents** (Gemini CLI, Cursor, Windsurf, Codex, Kilo Code, Continue, Aider, etc.), add this to your agent's instruction file (`AGENTS.md`, `GEMINI.md`, `.cursor/rules/`, `.windsurfrules`, etc.):

```
At the start of every session, run `timbers prime` and follow the workflow it describes.
After completing work, run `timbers pending` to check for undocumented commits,
then `timbers log "what" --why "why" --how "how"` to document your work.
The positional `what` is optional: when omitted, Timbers snapshots the selected
commit subject(s). Supply it when those subjects do not clearly describe the work.
Use `--notes` when you explored alternatives or made a real choice.
```

Native hooks and setup commands for additional agent environments are planned for a future release.

## How It Works

Entries are JSON files in `.timbers/`, committed to your repo alongside your code. Each `timbers log` creates its own git commit — you'll see `timbers: document ...` commits interleaved with your code commits. The optional pre-commit hook enforces documentation before each new commit, so with hooks enabled you'll see roughly one entry commit per code commit. If commits slip through without documentation (hook bypassed, batch workflow, or hooks not installed), `timbers log` gracefully falls back to batch mode — one entry covers all pending commits.

Timbers snapshots contributor identities at log time so attribution survives
rebases, squashes, and pruned clones. See
[Contributor Attribution](docs/contributor-attribution.md) for `--who`,
provenance, compatibility, and the downstream contract.

This is intentional: separate commits enable reliable tracking of what has been
documented and ensure entries travel with every clone without special
configuration. The entry's `what`, `why`, and `how` are capture-time snapshots,
so later SHA rewrites do not erase the explanation. Timbers relinks known
one-to-one local rewrites when possible, and range queries can discover entry
files across squash merges. A destructive or many-to-one rewrite can still
leave stored SHAs stale; reports continue from stored text and omit unavailable
Git enrichment rather than dropping the entry.

The trade-off is a noisier `git log` (roughly 2x the commit count with hooks). Agents handle this automatically (timbers filters entry commits internally). For humans who want a clean view:

```bash
git log --invert-grep --grep="^timbers: document"
```

See [docs/design-decisions.md](docs/design-decisions.md) for the full rationale, including alternatives that were evaluated and why they were rejected.

## Documentation

- [Tutorial](docs/tutorial.md) — Setup, capture workflow, agent integration
- [Contributor Attribution](docs/contributor-attribution.md) — Persisted identities and downstream contract
- [Publishing Artifacts](docs/publishing-artifacts.md) — CI/CD for changelogs, reports, blogs
- [Agent Reference](docs/agent-reference.md) — Command reference for agent integration
- [LLM Commands](docs/llm-commands.md) — Export, draft, report, and generate commands
- [Original Spec](docs/spec.md) — Historical v1 design and schema background
- [Agent DX Guide](docs/agent-dx-guide.md) — CLI design patterns for agents

## Example Artifacts

Timbers' own development ledger is used to generate these examples via `timbers draft` and `timbers report`. Each link is a live artifact produced from real data:

| Artifact | Description |
|----------|-------------|
| [Changelog](https://gorewood.github.io/timbers/examples/changelog/) | Keep a Changelog format, grouped by type |
| Decision Digest | Retrospective summary of explicit decisions from the *why* and *notes* fields |
| [Project Update](https://gorewood.github.io/timbers/examples/project-update/) | Weekly progress and implications for users and stakeholders |
| [Standup](https://gorewood.github.io/timbers/examples/standup/) | Daily standup from recent work |
| [Release Notes](https://gorewood.github.io/timbers/examples/release-notes/) | User-facing release notes |
| [Sprint Report](https://gorewood.github.io/timbers/examples/sprint-report/) | Categorized sprint summary with scope and highlights |
| [Dev Blog](https://gorewood.github.io/timbers/posts/) | Weekly dev blog posts (Carmack .plan style) |

> **A note on quality:** Most of these entries were historically backfilled using the now-retired `timbers catchup` command, which inferred what/why/how from commit messages and diffs. Projects that use `timbers log` from day one produce significantly richer output — especially in the decision digest, where explicit rationale matters most.

## Development

```bash
just setup    # First-time setup
just check    # Lint + test (required before commit)
just fix      # Auto-fix lint issues
just run      # Run the CLI
just site-test   # Test Timbermill collection materialization
just site-build  # Build the static demo into site/_site/
just site-serve  # Preview the demo locally
```

## Acknowledgments

Timbers was built and tracked with [Beads](https://github.com/steveyegge/beads), a lightweight issue tracker stored in Git refs. Beads' CLI design — structured output, agent-friendly ergonomics, Git-native storage — directly inspired the patterns in timbers' [Agent DX Guide](docs/agent-dx-guide.md). If you're building tools that agents need to use, studying beads is a great place to start.

## License

[MIT](LICENSE)
