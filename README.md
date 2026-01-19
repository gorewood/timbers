# Timbers

**Git knows what changed. Timbers captures why.**

A development ledger that pairs Git commits with structured rationale—what you did, why you did it, how you approached it—stored as portable Git notes.

## The Problem

As AI agents take on more development work—from assisted coding to nearly autonomous feature delivery—human oversight becomes both harder and more critical. Git history shows *what* changed. Commit messages hint at *how*. But the *why*—the reasoning, constraints, decisions, context—lives in agent session logs that get compacted, Slack threads that scroll away, and PR comments that nobody reads twice.

Six months later, you're staring at agent-written code wondering "why did it do it this way?" The answer is gone.

**In the era of vibe engineering, high-level understanding of large volumes of agent-assisted work is increasingly critical.** Timbers exists to maintain human oversight and comprehension as development becomes more automated.

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
timbers prompt changelog --since 7d | claude -p
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
timbers prompt changelog --since 7d | claude -p          # Pipe to external LLM
timbers prompt changelog --since 7d --model local        # Built-in (local LLM)
timbers prompt pr-description --last 5 --model haiku     # Built-in (cloud)
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
- `prompt` - Render templates for LLM piping, or execute directly with `--model`

**Sync**
- `notes init|push|fetch|status` - Manage Git notes sync

**Agent Integration**
- `prime` - Session context injection with workflow instructions
- `skill` - Self-documentation for building agent skills

**Admin**
- `status` - Repository and notes state
- `uninstall` - Remove timbers from a repo (or `--binary` to also remove the CLI)

All commands support `--json`. Write operations support `--dry-run`.

## Human Oversight at Scale

When agents write code, humans need to stay in the loop without reading every line. Timbers enables:

1. **Executive summaries**: `timbers prompt exec-summary --since 7d --model local` — understand a week of agent work in 5 bullets
2. **Queryable decisions**: Find all security-related changes, all work on a feature, all decisions by a particular agent session
3. **Onboarding acceleration**: New team members understand not just *what* the code does, but *why* it's shaped that way
4. **Audit trail**: Track the reasoning behind changes for compliance, post-mortems, or when "the agent did something weird"

The ledger grows automatically as agents document their work. Humans harvest insights when they need them.

## Agent-Native Workflow

Timbers is designed for agents to use directly. The typical session:

```bash
# Session start: agent gets context
timbers prime

# Session end: agent documents work
timbers pending  # Check for undocumented commits
timbers log "Implemented rate limiting" \
  --why "API abuse detected in logs" \
  --how "Token bucket with Redis backend"
timbers notes push
```

Humans rarely need to run `timbers log` directly—agents keep the ledger current. Humans use:
- `timbers query` — to review what happened
- `timbers prompt` — to generate summaries and reports

**Agent-friendly features:**
- `--json` on every command
- `prime` for session context injection
- `pending` for clear "what needs attention" signal
- Structured errors with recovery hints
- Exit codes: 0=success, 1=user error, 2=system error

## Model Recommendations

For most Timbers tasks—changelog generation, summaries, catchup—**local or inexpensive models work well**. The prompts are straightforward extraction and summarization tasks that don't require frontier reasoning.

**Recommended defaults:**
- `local` — Free, private, fast. Use LM Studio or Ollama with any capable model (Llama, Qwen, etc.)
- `haiku` / `flash` / `nano` — Cheap cloud options (~$0.25/M tokens). Excellent for batch operations.

Reserve expensive models (sonnet, opus, gpt-5) for complex analysis or when quality isn't meeting expectations.

## Documentation

- [Tutorial](docs/tutorial.md) - Step-by-step setup, catching up history, agent integration
- [Publishing Artifacts](docs/publishing-artifacts.md) - Generating changelogs, reports, blogs via CI/CD
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
