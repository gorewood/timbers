# LLM Commands

Timbers provides four commands for LLM integration, forming a pipeline from raw data extraction to automated documentation.

---

## Command Overview

| Command | Purpose | Output |
|---------|---------|--------|
| `export` | Raw data extraction | JSON/Markdown |
| `draft` | Template rendering with entries | Text for piping OR LLM response (with --model) |
| `generate` | Ad-hoc LLM completion primitive | LLM response text |
| `catchup` | Auto-generate entries from undocumented commits | Ledger entries |

---

## 1. Export — Raw Data Extraction

Extract ledger entries as structured data for external pipelines.

```bash
# JSON to stdout (default)
timbers export --last 5

# Markdown to stdout
timbers export --since 7d --format md

# By commit range
timbers export --range v1.0.0..v1.1.0

# To files in directory
timbers export --last 10 --out ./exports/
```

**Flags:**
- `--last N` — Export last N entries
- `--since <duration|date>` — Export entries since duration (24h, 7d) or date (2026-01-17)
- `--until <duration|date>` — Export entries until duration (24h, 7d) or date (2026-01-17)
- `--range A..B` — Export entries in commit range
- `--format json|md` — Output format (default: json for stdout, md for --out)
- `--out <dir>` — Write to directory instead of stdout
- `--json` — Structured JSON output for scripting

**Use Cases:**
- Feed entries to external LLMs or analysis tools
- Archive entries for backup or sharing
- Integration with CI/CD pipelines

---

## 2. Draft — Template Rendering

Render templates with ledger entries for LLM consumption. By default, outputs text for piping to external LLMs. Use `--model` for built-in LLM execution.

```bash
# Pipe to external LLM
timbers draft changelog --since 7d | claude -p

# Append custom instructions
timbers draft devblog --last 10 --append "Focus on physics engine changes" | claude -p

# By commit range
timbers draft pr-description --range main..HEAD | claude -p

# List available templates
timbers draft --list

# Show template content
timbers draft changelog --show

# Built-in LLM execution (no piping needed)
timbers draft changelog --since 7d --model local
timbers draft standup --last 10 --model haiku
timbers draft devblog --last 20 --model flash --append "Focus on physics"
```

**Flags:**
- `--last N` — Use last N entries
- `--since <duration|date>` — Use entries since duration or date
- `--until <duration|date>` — Use entries until duration or date
- `--range A..B` — Use entries in commit range
- `--append <text>` — Append extra instructions to the prompt
- `--list` — List available templates
- `--show` — Show template content without rendering
- `-m, --model <name>` — Execute with built-in LLM instead of outputting text
- `-p, --provider <name>` — Provider override (anthropic, openai, google, local)
- `--json` — Structured JSON output (includes rendered prompt and entries)

### Available Templates

Built-in templates (use `timbers draft --list` for current list):

| Template | Purpose |
|----------|---------|
| `changelog` | Generate release changelogs |
| `devblog` | Developer blog post (Carmack .plan style) |
| `standup` | Daily standup from recent work |
| `pr-description` | Pull request descriptions |
| `release-notes` | User-facing release notes |
| `sprint-report` | Sprint/iteration summaries |

### Template Resolution Order

1. `.timbers/templates/<name>.md` — Project-local
2. `~/.config/timbers/templates/<name>.md` — User global
3. Built-in templates

### Custom Templates

Create project-specific templates:

```bash
mkdir -p .timbers/templates
cat > .timbers/templates/my-template.md << 'EOF'
# My Custom Template

Repository: {{.RepoName}}
Branch: {{.Branch}}

## Entries

{{range .Entries}}
### {{.Summary.What}}

**Why:** {{.Summary.Why}}
**How:** {{.Summary.How}}

{{end}}

{{if .AppendText}}
---
Additional Instructions: {{.AppendText}}
{{end}}
EOF
```

---

## 3. Generate — LLM Completion Primitive

A composable primitive for piping any text through an LLM. Defaults to local LLM server.

```bash
# Use local LLM (default)
timbers generate "Explain recursion"

# Use cloud providers
timbers generate "Explain recursion" --model haiku
timbers generate "Explain recursion" --model sonnet
timbers generate "Explain recursion" --model gemini-flash

# Pipe input through stdin
echo "Summarize this code" | timbers generate

# With system prompt
timbers generate "Write tests" --model sonnet --system "You are a Go expert"

# Read from file
timbers generate "Summarize" --input ./notes.txt
```

**Flags:**
- `-m, --model <name>` — Model name (default: local)
- `-p, --provider <name>` — Provider override (anthropic, openai, google, local)
- `-s, --system <prompt>` — System prompt
- `-i, --input <file>` — Input file
- `--temperature <float>` — Temperature (0.0-2.0, 0 uses model default)
- `--max-tokens <int>` — Max tokens to generate
- `--timeout <seconds>` — Request timeout (default: 120)
- `--json` — Structured JSON output

### Model Shortcuts

| Provider | Shortcuts |
|----------|-----------|
| Anthropic | `haiku`, `sonnet`, `opus` (or `claude-haiku`, `claude-sonnet`, `claude-opus`) |
| OpenAI | `nano`, `mini`, `gpt-5` (or `openai-nano`, `openai-mini`) |
| Google | `flash`, `flash-lite`, `pro` (or `gemini-flash`, `gemini-pro`) |
| Local | `local` (default — uses loaded model in LM Studio/Ollama) |

---

## 4. Catchup — Auto-Generate Entries

Generate ledger entries for undocumented commits using an LLM. Groups pending commits by work-item or day and generates what/why/how summaries.

```bash
# Preview what would be created
timbers catchup --model haiku --dry-run

# Create entries
timbers catchup --model haiku

# With custom tags
timbers catchup --model haiku --tag "historical" --tag "backfill"

# Specific commit range
timbers catchup --model haiku --range abc123..def456

# Parallel processing
timbers catchup --model sonnet --parallel 10

# Push notes after creating
timbers catchup --model haiku --push
```

**Flags:**
- `-m, --model <name>` — Model name (default: local)
- `-p, --provider <name>` — Provider override
- `--dry-run` — Preview entries without writing
- `--range A..B` — Specific commit range
- `--batch-size <int>` — Max commits per LLM call (default: 20)
- `--parallel <int>` — Concurrent LLM calls (default: 5)
- `--tag <name>` — Tags to add to all entries (repeatable)
- `--push` — Push notes after creating entries
- `--json` — Structured JSON output

### Catchup Workflow

```bash
# 1. Check what's undocumented
timbers pending

# 2. Preview what catchup would create
timbers catchup --model haiku --dry-run

# 3. Create the entries
timbers catchup --model haiku

# 4. Verify
timbers pending  # Should show fewer/no pending commits
```

---

## Environment Variables

| Variable | Purpose |
|----------|---------|
| `ANTHROPIC_API_KEY` | Required for Anthropic models (haiku, sonnet, opus) |
| `OPENAI_API_KEY` | Required for OpenAI models (nano, mini, gpt-5) |
| `GOOGLE_API_KEY` | Required for Google models (flash, pro) |
| `LOCAL_LLM_URL` | Local server URL (default: `http://localhost:1234/v1`) |

---

## Model Recommendations

**Local or cheap models are adequate for most Timbers tasks.** The prompts involve straightforward summarization and extraction—no complex reasoning required.

### Recommended Defaults

| Tier | Models | Cost | Best For |
|------|--------|------|----------|
| Free | `local` | $0 | Daily use, privacy-sensitive, offline |
| Cheap | `haiku`, `flash`, `nano` | ~$0.25/M tokens | Batch operations, CI/CD |
| Premium | `sonnet`, `pro`, `mini` | ~$3-15/M tokens | When quality matters |

### Practical Guidance

- **Start with `local`**: If you have LM Studio or Ollama running, local models handle Timbers tasks well
- **Use cheap cloud for batch**: `catchup` with 100+ commits? Use `haiku` or `flash` for speed and low cost
- **Reserve premium for polish**: Only escalate to sonnet/opus if output quality isn't meeting expectations

**Cost example:** Generating 50 changelog entries with haiku costs ~$0.02. Local is free.

---

## Flag Consistency

These flags work consistently across `draft`, `generate`, and `catchup`:

| Flag | Short | Description |
|------|-------|-------------|
| `--model` | `-m` | Model name (haiku, sonnet, local, etc.) |
| `--provider` | `-p` | Provider override (anthropic, openai, google, local) |

---

## Composition Patterns

### Pattern 1: External LLM Piping

Use `draft` to render templates, pipe to your preferred LLM CLI.
This uses your subscription — no API key needed.

```bash
# Claude Code (claude -p reads stdin, --model selects the model)
timbers draft changelog --since 7d | claude -p --model opus

# Gemini CLI (auto-detects piped input, -m selects model)
timbers draft standup --since 1d | gemini
timbers draft standup --since 1d | gemini -m gemini-2.5-pro

# Codex CLI (exec - reads prompt from stdin, -m selects model)
timbers draft pr-description --range main..HEAD | codex exec -
timbers draft pr-description --range main..HEAD | codex exec -m gpt-5-codex-mini -
```

### Pattern 2: Built-in LLM Execution

Use `--model` for simpler one-liner execution:

```bash
# Direct execution (recommended for most use cases)
timbers draft changelog --since 7d --model local
timbers draft standup --last 5 --model haiku
timbers draft pr-description --range main..HEAD --model flash
```

### Pattern 3: Built-in LLM via Generate

Chain export or prompt output through generate:

```bash
# Render prompt, pipe through built-in LLM
timbers draft changelog --since 7d | timbers generate --model haiku

# Custom prompts with exported data
timbers export --last 3 --format md | timbers generate "Summarize these changes" --model sonnet
```

### Pattern 4: Automated Backfill

Catch up on undocumented history:

```bash
# Full backfill workflow
timbers pending                           # See what's missing
timbers catchup --model haiku --dry-run   # Preview
timbers catchup --model haiku             # Execute
git push                                  # Sync to remote
```

### Pattern 5: CI/CD Integration

```bash
# In release workflow
timbers draft release-notes --range $PREV_TAG..$NEW_TAG | \
  timbers generate --model haiku > RELEASE_NOTES.md

# Generate PR description
timbers draft pr-description --range main..HEAD | \
  timbers generate --model haiku
```

---

## JSON Output

All commands support `--json` for structured output:

```bash
# Export with JSON
timbers export --last 5 --json

# Prompt rendering with metadata
timbers draft changelog --since 7d --json

# Generate with response metadata
timbers generate "Hello" --model haiku --json

# Catchup results
timbers catchup --model haiku --dry-run --json
```

---

## Error Handling

LLM commands return structured errors:

```json
{
  "error": "ANTHROPIC_API_KEY not set",
  "code": 1,
  "hint": "Set ANTHROPIC_API_KEY environment variable or use --model local"
}
```

Exit codes follow timbers conventions:
- `0` — Success
- `1` — User error (missing API key, invalid model, bad flags)
- `2` — System error (network failure, LLM timeout)
