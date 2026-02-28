# Timbers Tutorial

A step-by-step guide to setting up Timbers for your project, catching up existing history, and using it effectively with AI agents and as a human developer.

---

## Part 1: Installation & Setup

### Install Timbers

```bash
# macOS / Linux
curl -fsSL https://raw.githubusercontent.com/gorewood/timbers/main/install.sh | bash

# Or from source (note: go install puts a versionless binary in GOBIN)
go install github.com/gorewood/timbers/cmd/timbers@latest
```

Verify the installation:

```bash
timbers --version
timbers --help
```

### Initialize a Repository

Navigate to your Git repository and initialize Timbers:

```bash
cd /path/to/your/repo

# Initialize Timbers (creates .timbers/ directory, installs hooks)
timbers init
```

This creates a `.timbers/` directory in your repository root where ledger entries are stored as JSON files, organized by date (`YYYY/MM/DD/`). Entries are regular tracked files that sync with `git push` and `git pull`.

**What just happened?**
- `.timbers/` directory created for ledger entry storage
- Git hooks installed for workflow integration
- Your repo is now ready to store and sync development ledger entries

### Verify Setup

```bash
timbers status
```

You should see:
```
Repository Status
─────────────────
  Repo:    your-repo
  Branch:  main
  HEAD:    abc1234

Timbers Ledger
──────────────
  Storage:  .timbers/
  Entries:  0
```

---

## Part 2: Catching Up Existing History

If your repository has existing commits without Timbers entries, you have several options for catching up.

### Option A: Document Recent Work Only

If you only care about documenting work going forward, simply start using `timbers log` after your next piece of work. Past commits will remain undocumented, but that's often fine.

### Option B: Batch by Logical Phases

For a more complete history, group related commits into logical entries. Use `--anchor` to specify which commit the entry attaches to, and `--range` to specify which commits it covers.

**Step 1: See what's pending**

```bash
timbers pending
```

This shows all commits without entries.

**Step 2: Identify logical groupings**

Look at your commit history and identify natural phases:

```bash
git log --oneline | head -30
```

Group commits that represent coherent pieces of work—a feature, a refactor, a bug fix campaign.

**Step 3: Create batch entries**

```bash
# Example: Document a feature that spans commits abc123 to def456
timbers log "Switched to cursor-based pagination" \
  --why "Offset pagination skips rows when items are inserted between pages" \
  --how "Opaque cursor tokens encoding created_at + id" \
  --anchor def456 \
  --range abc123..def456 \
  --tag api --tag performance
```

**Step 4: Repeat for each logical phase**

Work backwards from most recent to oldest, or focus on the most important phases.

### Option C: Auto-Extract from Commit Messages

If your commit messages already contain good information, try auto mode:

```bash
timbers log --auto
```

This attempts to parse what/why/how from your commit messages. Review with `--dry-run` first:

```bash
timbers log --auto --dry-run
```

### Option D: Batch by Work Items

If your commits include work-item trailers (like `Beads: bd-abc123` or `Fixes: #42`), batch mode groups them automatically:

```bash
timbers log --batch
```

This creates one entry per work-item group.

---

## Part 3: Daily Workflow

Once set up, here's the typical workflow.

### Recording Work

After completing a piece of work, document it:

```bash
timbers log "Fixed race condition in cache invalidation" \
  --why "Users seeing stale data after updates" \
  --how "Added mutex around cache write, extended TTL check"
```

**Required fields:**
- The first argument is **what** you did
- `--why` explains the verdict — why *this* approach over alternatives
- `--how` describes the approach or implementation

**Optional fields:**
- `--notes` for deliberation context — the journey to the decision (alternatives explored, trade-offs weighed). Skip for routine work; use when you made a real choice.
- `--tag` for categorization (repeatable)
- `--work-item` for linking to issue trackers (e.g., `--work-item jira:PROJ-123`)
- `--minor` for trivial changes (skips why/how requirement)

### Checking for Undocumented Work

Before ending a session:

```bash
timbers pending
```

If there are pending commits, document them before you forget the context.

### Syncing Entries

Entries are regular tracked files in `.timbers/`, so they sync with standard Git commands:

```bash
# Push your entries to the remote
git push

# Fetch entries from collaborators
git pull
```

---

## Part 4: Agent Integration

Timbers is designed for AI agent workflows. Here's how to set up your agent.

### Session Start: Prime the Agent

At the beginning of each agent session, inject context:

```bash
timbers prime
```

This outputs:
- Repository and branch info
- Ledger status (entry count, pending commits)
- Recent entries
- Workflow instructions (session close protocol, essential commands)

**For Claude Code users**, add to your `CLAUDE.md`:

```markdown
## Development Workflow

At session start:
  timbers prime

After completing work:
  timbers log "what" --why "why" --how "how" [--notes "deliberation"]

At session end:
  timbers pending  # Verify all work documented
  git push         # Sync to remote
```

### Session End: Document and Sync

Agents should follow this checklist before completing:

1. `timbers pending` — Check for undocumented commits
2. `timbers log "..." --why "..." --how "..."` — Document the work
3. `git push` — Sync to remote

### Customizing Agent Onboarding

Create `.timbers/PRIME.md` in your repo root to override the default workflow instructions:

```bash
# See default content
timbers prime --export > .timbers/PRIME.md

# Edit to customize for your project
vim .timbers/PRIME.md
```

### Agent Reference

For building agent skills that integrate with Timbers, see [docs/agent-reference.md](agent-reference.md) for the complete command reference and contract.

---

## Part 5: Querying Your Ledger

### View Recent Entries

```bash
timbers query --last 5
timbers query --last 10 --oneline  # Compact view
```

### Filter by Time

```bash
timbers query --since 7d      # Last 7 days
timbers query --since 2w      # Last 2 weeks
timbers query --since 24h     # Last 24 hours
timbers query --since 2026-01-01  # Since a specific date
timbers query --since 2026-01-01 --until 2026-01-15  # Date range
```

### Filter by Tags

```bash
timbers query --tags security
timbers query --tags "auth,security"  # Multiple tags (OR)
```

### View a Single Entry

```bash
timbers show tb_2026-01-15T10:30:00Z_abc123
```

### Export for Processing

```bash
# JSON for programmatic use
timbers export --last 10 --json

# Markdown for documentation
timbers export --since 7d --format markdown

# Pipe to files
timbers export --last 20 --json > entries.json
```

---

## Part 6: LLM-Powered Reports

The `draft` command renders templates with your entries for piping to LLMs.

### Available Templates

```bash
timbers draft --list
```

Built-in templates:
- `changelog` — Conventional changelog format
- `decision-log` — ADR-style architectural decisions extracted from the *why* and *notes* fields
- `devblog` — Developer blog post (Carmack .plan style)
- `standup` — Daily standup from recent work
- `pr-description` — Pull request description
- `release-notes` — User-facing release notes
- `sprint-report` — Sprint/iteration report

### Built-in LLM Execution

Instead of piping to external tools, use `--model` for direct LLM execution:

```bash
# Built-in execution (no piping needed)
timbers draft changelog --since 7d --model local         # Local LLM
timbers draft standup --since 1d --model haiku       # Cloud (Anthropic)
timbers draft pr-description --range main..HEAD --model flash  # Cloud (Google)
```

This is equivalent to piping but simpler for quick use.

### Generate Reports

Pipe the rendered prompt to any LLM CLI — this uses your subscription, not API tokens:

```bash
# Claude Code (-p reads stdin and exits)
timbers draft changelog --since 7d | claude -p
timbers draft devblog --last 20 --append "Focus on the new plugin system" | claude -p

# Gemini CLI (auto-detects piped input)
timbers draft standup --since 1d | gemini

# Codex CLI (exec - reads prompt from stdin)
timbers draft pr-description --range main..HEAD | codex exec -m gpt-5-codex-mini -

# Or use built-in LLM execution (requires API key, no CLI needed)
timbers draft changelog --since 7d --model local
timbers draft standup --since 1d --model haiku
```

**Shortcut with just:** If you're developing timbers itself, use the just recipes:

```bash
just draft changelog --since 7d           # Uses haiku model by default
just draft-model sonnet devblog --last 20 # Specify a different model
```

### Preview Templates

```bash
# See template content without rendering
timbers draft changelog --show

# See what data would be sent
timbers draft changelog --last 5
```

### Custom Templates

Create project-specific templates in `.timbers/templates/`:

```bash
mkdir -p .timbers/templates
cat > .timbers/templates/weekly-standup.md << 'EOF'
Summarize this week's development work for a standup meeting.

Format as:
- **Completed**: What got done
- **In Progress**: What's ongoing
- **Blockers**: Any issues

Keep it concise (3-5 bullets per section).

## Entries
EOF
```

Then use it:

```bash
timbers draft weekly-standup --since 7d | claude -p
```

Template resolution order:
1. `.timbers/templates/<name>.md` (project-local)
2. `~/.config/timbers/templates/<name>.md` (user global)
3. Built-in templates

### Model Recommendations

| Use Case | Recommended Model |
|----------|-------------------|
| Quality content (changelogs, blog posts, decision logs) | `opus` (best reasoning and writing quality) |
| Daily use | `local` (free, private, fast) |
| Batch backfill (100+ entries) | `haiku`, `flash`, `nano` (cheap cloud) |

**Quality matters for published artifacts.** Decision logs, blog posts, and changelogs benefit from frontier-model reasoning — the difference between shallow summaries and genuine insight is significant. Use `opus` for anything you'd share externally.

**Cost perspective:** Processing 100 entries with haiku costs ~$0.01-0.05. Local is free. When piping through `claude -p --model opus`, you use your subscription rather than API tokens.

---

## Part 7: Common Human Uses

Beyond agent workflows, Timbers is useful for human developers.

### Onboarding New Team Members

```bash
# Show project evolution
timbers query --last 50

# Find security-related decisions
timbers query --tags security

# Understand recent changes
timbers query --since 30d
```

New team members can understand not just *what* the code does, but *why* it's shaped that way.

### Post-Mortems

```bash
# Find entries around an incident date
timbers query --since 2026-01-10 --until 2026-01-15

# Look for related tags
timbers query --tags "auth,security"
```

### Generating Documentation

```bash
# Monthly changelog (built-in LLM)
timbers draft changelog --since 30d --model local > CHANGELOG-january.md

# Architecture decision records
timbers query --tags architecture --json | \
  jq '.[] | "## \(.summary.what)\n\n**Why:** \(.summary.why)\n\n**How:** \(.summary.how)\n"' -r
```

### Code Review Context

When reviewing a PR, check if there's a Timbers entry:

```bash
timbers query --range main..feature-branch
```

This shows the documented rationale for the changes.

### Personal Development Log

Use Timbers as a work journal:

```bash
# End of day ritual
timbers pending
timbers log "Debugged flaky test in CI" \
  --why "Race condition in setup was the root cause, not timing" \
  --how "Added synchronization barrier before assertion" \
  --notes "Initially suspected timing issue, but adding sleep didn't help. Traced to shared state in setup."

# Weekly review
timbers query --since 7d
timbers draft standup --since 7d | claude -p
```

---

## Part 8: Removing Timbers

If you need to remove Timbers from a repository:

```bash
# Remove from current repo only (keeps binary installed)
timbers uninstall

# Preview what would be removed
timbers uninstall --dry-run

# Also remove the binary
timbers uninstall --binary
```

This removes:
- `.timbers/` directory and all ledger entries from the repository
- Git hooks installed by Timbers
- Optionally, the Timbers binary

---

## Quick Reference

### Essential Commands

| Command | Purpose |
|---------|---------|
| `timbers log "what" --why "..." --how "..."` | Record work |
| `timbers pending` | Show undocumented commits |
| `timbers query --last N` | View recent entries |
| `timbers prime` | Session context for agents |
| `git push` | Sync entries to remote |
| `timbers draft <template> --model local` | Generate report with built-in LLM |

### Flags Available Everywhere

| Flag | Purpose |
|------|---------|
| `--json` | Structured JSON output |
| `--help` | Command help |

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | User error (bad args, not found) |
| 2 | System error (git failed, I/O error) |
| 3 | Conflict (entry exists, state mismatch) |

---

## Troubleshooting

### "Not in a git repository"

Timbers requires a Git repository. Initialize one or navigate to an existing repo.

### "Timbers not initialized"

Run `timbers init` to create the `.timbers/` directory and install hooks.

### Entries not syncing

Entries are tracked files in `.timbers/`. Sync them like any other file:

```bash
# Pull entries from collaborators
git pull

# Push your entries
git push
```

If entries were committed but not pushed, `git status` will show the branch is ahead of the remote.

### Entry ID format

Entry IDs follow the pattern: `tb_<ISO8601-timestamp>_<anchor-short-sha>`

Example: `tb_2026-01-15T10:30:00Z_abc123`

---

## Next Steps

1. **Start documenting** — Begin with `timbers log` after your next piece of work. Use `--notes` when you explored alternatives.
2. **Set up agent workflow** — Add `timbers prime` to your agent's session start
3. **Explore templates** — Try `timbers draft changelog --since 7d | claude -p`
4. **Customize** — Create `.timbers/PRIME.md` or custom templates for your project

The goal is simple: capture *why* before it disappears.
