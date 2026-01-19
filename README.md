# Timbers

**Git knows what changed. Timbers captures why.**

A development ledger that pairs Git commits with structured rationale—what you did, why you did it, how you approached it—stored as portable Git notes.

## The Problem

Git history shows *what* changed. Commit messages hint at *how*. But the *why*—the reasoning, constraints, decisions, context—lives in Slack threads, PR comments, and session memory that evaporates when the conversation ends.

Six months later, you're staring at a commit wondering "why did we do it this way?" The answer is gone.

## The Solution

Timbers creates a durable development ledger:

```bash
# After completing work, capture the context
timbers log "Fixed auth bypass" \
  --why "User input wasn't sanitized before JWT validation" \
  --how "Added validation middleware before auth handler"

# See what needs documenting
timbers pending

# Generate reports from your ledger
timbers prompt changelog --since 7d | claude
```

Each entry captures:
- **What**: The work summary
- **Why**: The reasoning and constraints
- **How**: The approach taken
- **Context**: Commits, diffstat, timestamps (harvested from Git)

Entries are stored as Git notes—they travel with your repo, sync to remotes, and survive rebases.

## Quick Start

```bash
# Install
go install github.com/rbergman/timbers/cmd/timbers@latest

# Initialize notes sync (one-time)
timbers notes init origin

# Record work
timbers log "Added rate limiting" --why "Prevent API abuse" --how "Token bucket algorithm"

# Check for undocumented commits
timbers pending

# Query your ledger
timbers query --last 10
timbers query --since 7d --tags security

# Generate LLM-ready reports
timbers prompt changelog --since 7d | claude
timbers prompt pr-description --range main..HEAD | llm
```

## Commands

**Recording**
- `log` - Record work with what/why/how
- `pending` - Show commits awaiting documentation

**Querying**
- `query` - Search entries by time, tags, or content
- `show` - Display a single entry
- `export` - Export as JSON or Markdown

**Reporting**
- `prompt` - Render templates for LLM piping (changelog, pr-description, exec-summary, etc.)

**Sync**
- `notes init|push|fetch|status` - Manage Git notes sync

**Agent Integration**
- `prime` - Session context injection with workflow instructions
- `skill` - Self-documentation for building agent skills

**Admin**
- `status` - Repository and notes state
- `uninstall` - Remove timbers from a repo (or `--binary` to also remove the CLI)

All commands support `--json`. Write operations support `--dry-run`.

## Why Structured Rationale?

Commit messages are too small. PRs are too scattered. Docs go stale. Timbers gives you:

1. **Queryable history**: Find all security-related decisions from Q4
2. **LLM-ready context**: Pipe entries to Claude for changelogs, summaries, blog posts
3. **Onboarding gold**: New team members can understand not just what the code does, but why it's shaped that way
4. **Audit trail**: Track decisions for compliance, post-mortems, or your future self

## Agent-First Design

Timbers is built for AI agent workflows:
- `--json` everywhere for structured consumption
- `prime` injects workflow context at session start
- `pending` provides clear "what needs attention" signal
- `prompt` generates LLM-ready output with built-in templates
- Structured errors with recovery hints
- Exit codes follow conventions (0=success, 1=user error, 2=system error)

```bash
# Agent session start
timbers prime

# Agent session end
timbers pending && timbers log "..." --why "..." --how "..."
timbers notes push
```

## Documentation

- [Spec](docs/spec.md) - Full specification
- [Agent DX Guide](docs/agent-dx-guide.md) - CLI design patterns for agents

## Development

```bash
just setup    # First-time setup
just check    # Lint + test (required before commit)
just fix      # Auto-fix lint issues
just run      # Run the CLI
```

## License

MIT
