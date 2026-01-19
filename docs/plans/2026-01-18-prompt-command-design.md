# Design: `timbers prompt` Command

**Status:** Draft
**Date:** 2026-01-18

## Overview

`timbers prompt` renders a template with ledger entries injected, outputting a ready-to-pipe LLM prompt. This enables composable narrative generation without coupling timbers to specific LLM tooling.

```bash
timbers prompt changelog --since 7d | claude
timbers prompt exec-summary --last 5 | llm
timbers prompt devblog-gamedev --since 2w | claude "write in first person"
```

## Design Principles

1. **Prompt-only output** - timbers outputs text, user pipes to their LLM of choice
2. **Template-driven** - all report shapes defined as templates, not code
3. **User-extensible** - custom templates in `.timbers/templates/`
4. **Good defaults** - built-in templates for common use cases

## Command Specification

```
timbers prompt <template> [flags]

Arguments:
  <template>    Template name (required)

Flags:
  --last <n>              Use last N entries
  --since <duration|date> Use entries since duration (24h, 7d) or date
  --range <A..B>          Use entries in commit range
  --append <text>         Append extra instructions to the prompt
  --list                  List available templates
  --show                  Show template content without rendering
  --json                  Output as JSON (template + entries, for debugging)
```

### Examples

```bash
# Generate changelog for last week
timbers prompt changelog --since 7d | claude

# Executive summary for standup
timbers prompt exec-summary --last 3 | llm

# PR description for current branch
timbers prompt pr-description --range main..HEAD | claude

# List available templates
timbers prompt --list

# Preview template before using
timbers prompt changelog --show

# Custom template
timbers prompt my-weekly-digest --since 7d | claude

# Add extra instructions
timbers prompt changelog --since 7d --append "Use emoji section headers" | claude
timbers prompt devblog-gamedev --last 10 --append "Focus on the physics system work" | claude
```

## Template Resolution

Templates are resolved in order:

1. `.timbers/templates/<name>.md` (project-local)
2. `~/.config/timbers/templates/<name>.md` (user global)
3. Built-in templates (embedded in binary)

This allows users to:
- Override built-in templates per-project
- Define personal templates that work across all repos
- Share project templates via version control
- Fall back to sensible built-in defaults

## Template Format

Templates use markdown with YAML frontmatter and simple `{{variable}}` placeholders.

```markdown
---
name: changelog
description: Conventional changelog grouped by date
version: 1
---
Generate a changelog from these development log entries.

Format as a conventional changelog with sections: Added, Changed, Fixed, Removed.
Group entries by date. Use past tense. Be concise.

## Entries

{{entries_json}}
```

### Available Variables

| Variable | Description |
|----------|-------------|
| `{{entries_json}}` | Full entries as JSON array |
| `{{entries_summary}}` | Compact summary (id, date, what, why) |
| `{{entry_count}}` | Number of entries |
| `{{date_range}}` | Human-readable date range |
| `{{repo_name}}` | Repository name |
| `{{branch}}` | Current branch |

### Frontmatter Fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Template identifier |
| `description` | Yes | One-line description for --list |
| `version` | No | Template version (for compatibility) |
| `default_flags` | No | Default flag values (e.g., `{since: "7d"}`) |

## Built-in Templates

### 1. changelog

Conventional changelog format for release documentation.

```markdown
---
name: changelog
description: Conventional changelog grouped by date
---
Generate a changelog from these development log entries.

Use conventional changelog format with sections:
- **Added** - New features
- **Changed** - Changes to existing functionality
- **Fixed** - Bug fixes
- **Removed** - Removed features

Group by date (most recent first). Use past tense. One line per entry.
Link to commits where relevant.

## Entries ({{entry_count}})

{{entries_json}}
```

### 2. exec-summary

Executive summary for standups and status updates.

```markdown
---
name: exec-summary
description: Executive summary bullet points for status updates
---
Generate a brief executive summary from these development log entries.

Format as 3-5 bullet points suitable for a standup or status meeting.
Focus on outcomes and impact, not implementation details.
Use active voice, present perfect tense ("Completed X", "Fixed Y").

## Entries ({{entry_count}})

{{entries_summary}}
```

### 3. sprint-report

Sprint/iteration summary with metrics.

```markdown
---
name: sprint-report
description: Sprint summary grouped by tag/category with stats
---
Generate a sprint report from these development log entries.

Structure:
1. **Summary** - 2-3 sentence overview
2. **By Category** - Group entries by tags (features, bugs, refactoring, etc.)
3. **Metrics** - Total commits, files changed, lines added/removed
4. **Highlights** - Notable achievements or decisions

## Entries ({{entry_count}}) | {{date_range}}

{{entries_json}}
```

### 4. pr-description

Pull request description with summary and test plan.

```markdown
---
name: pr-description
description: Pull request body with summary and test plan
---
Generate a pull request description from these development log entries.

Format:
## Summary
[2-4 bullet points of what changed and why]

## Changes
[Grouped list of specific changes]

## Test Plan
[How to verify these changes work]

Keep it concise. Focus on reviewer needs.

## Entries ({{entry_count}}) | Branch: {{branch}}

{{entries_json}}
```

### 5. release-notes

User-facing release notes.

```markdown
---
name: release-notes
description: User-facing release notes for version bumps
---
Generate user-facing release notes from these development log entries.

Write for end users, not developers. Focus on:
- New capabilities they can use
- Problems that are now fixed
- Breaking changes they need to know about

Avoid technical jargon. Use benefit-oriented language.
Group into: New Features, Improvements, Bug Fixes, Breaking Changes.

## Entries ({{entry_count}}) | {{date_range}}

{{entries_json}}
```

### 6. devblog-gamedev

Example dev blog for game development audience.

```markdown
---
name: devblog-gamedev
description: "Dev blog post for game development audience"
---
Write a dev blog post from these development log entries.

Audience: Game developers and enthusiasts following the project.

Voice and style:
- Conversational but technical
- Share the "why" behind decisions
- Include war stories and lessons learned
- Reference game dev concepts (ECS, render loops, input handling, etc.)
- Okay to geek out about interesting problems
- Use headers to break up sections
- End with "what's next" teaser

Length: 800-1200 words

## Development Log Entries

{{entries_json}}
```

### 7. devblog-opensource

Example dev blog for open source maintainer voice.

```markdown
---
name: devblog-opensource
description: "Dev blog post for open source community"
---
Write a dev blog post from these development log entries.

Audience: Open source contributors and users of the project.

Voice and style:
- Transparent about challenges and tradeoffs
- Acknowledge community contributions
- Explain architectural decisions
- Include "how to help" callouts
- Link to relevant issues/PRs conceptually
- Welcoming to newcomers
- Technical but accessible

Length: 600-1000 words

## Development Log Entries

{{entries_json}}
```

### 8. devblog-startup

Example dev blog for startup/founder voice.

```markdown
---
name: devblog-startup
description: "Dev blog post for startup/founder audience"
---
Write a dev blog post from these development log entries.

Audience: Startup founders, indie hackers, and tech entrepreneurs.

Voice and style:
- Building in public energy
- Honest about struggles and wins
- Connect technical work to business outcomes
- Share metrics where relevant
- Lessons that transfer to other projects
- Personal but professional
- Forward momentum focus

Length: 500-800 words

## Development Log Entries

{{entries_json}}
```

## Output Format

### Default (stdout)

Plain text prompt ready for piping:

```
Generate a changelog from these development log entries.
...

## Entries (5)

[{"id": "tb_2026-01-18...", ...}, ...]
```

### JSON (--json)

Structured output for debugging or alternative workflows:

```json
{
  "template": "changelog",
  "template_path": "built-in",
  "prompt": "Generate a changelog from...",
  "entry_count": 5,
  "entries": [...]
}
```

### List (--list)

Shows available templates with source location:

```
Built-in:
  changelog        Conventional changelog grouped by date
  exec-summary     Executive summary bullet points for status updates
  sprint-report    Sprint summary grouped by tag/category with stats
  pr-description   Pull request body with summary and test plan
  release-notes    User-facing release notes for version bumps
  devblog-gamedev  Dev blog post for game development audience
  devblog-opensource  Dev blog post for open source community
  devblog-startup  Dev blog post for startup/founder audience

Global (~/.config/timbers/templates/):
  my-weekly        Custom weekly digest template

Project (.timbers/templates/):
  changelog        [overrides built-in] Project-specific changelog format
  team-standup     Team standup format
```

## Implementation Notes

### Template Storage

Built-in templates embedded via `//go:embed templates/*.md` directive.

```go
//go:embed templates/*.md
var builtinTemplates embed.FS
```

### Template Rendering

Simple string replacement - no complex template engine needed:

```go
func renderTemplate(tmpl string, vars map[string]string) string {
    result := tmpl
    for k, v := range vars {
        result = strings.ReplaceAll(result, "{{"+k+"}}", v)
    }
    return result
}
```

### Package Structure

```
cmd/timbers/
    prompt.go           # Command implementation

internal/
    prompt/
        template.go     # Template loading and resolution
        render.go       # Variable substitution
        builtin.go      # Built-in template registry
    templates/
        changelog.md
        exec-summary.md
        sprint-report.md
        pr-description.md
        release-notes.md
        devblog-gamedev.md
        devblog-opensource.md
        devblog-startup.md
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Template not found, invalid flags |
| 2 | Git/system error |

## Error Messages

```bash
$ timbers prompt nonexistent
Error: template "nonexistent" not found
Hint: Run 'timbers prompt --list' to see available templates

$ timbers prompt changelog
Error: specify --last N, --since <duration|date>, or --range A..B
```

## Future Considerations

### Not in scope for v1

- **Template inheritance** - Override parts of a template
- **Conditional sections** - Include/exclude based on entry content
- **Multiple entry formats** - Only JSON and summary for now
- **Template validation** - Just fail on missing variables

### Potential v2 additions

- `timbers prompt init` - Scaffold custom template
- `timbers prompt validate` - Check template syntax
- Template variables for filtering (e.g., `{{entries_tagged:security}}`)

## Acceptance Criteria

```bash
# AC1: Basic prompt generation
timbers prompt changelog --last 5
# Outputs prompt text to stdout

# AC2: Pipes to LLM
timbers prompt changelog --last 5 | claude
# LLM receives prompt and generates changelog

# AC3: List templates
timbers prompt --list
# Shows built-in and custom templates

# AC4: Custom template
echo "..." > .timbers/templates/custom.md
timbers prompt custom --last 3
# Uses custom template

# AC5: Template override
cp builtin-changelog.md .timbers/templates/changelog.md
timbers prompt changelog --last 3
# Uses user's changelog, not built-in

# AC6: Show template
timbers prompt changelog --show
# Displays template content

# AC7: JSON output
timbers prompt changelog --last 3 --json
# Outputs JSON with template and entries

# AC8: All entry filters work
timbers prompt changelog --since 7d
timbers prompt changelog --range v1.0..v1.1
# Both work correctly

# AC9: Append extra instructions
timbers prompt changelog --last 5 --append "Use emoji headers"
# Prompt ends with appended text
```

## Resolved Questions

1. **Template discovery** - Yes, `--list` scans project-local, user global, and built-in. Resolution order: `.timbers/templates/` → `~/.config/timbers/templates/` → built-in.

2. **Entry format preference** - `{{entries_json}}` and `{{entries_summary}}` are sufficient. LLMs handle JSON well; template authors instruct the LLM how to format output.

3. **Prompt suffixes** - Yes, `--append` flag adds extra instructions. e.g., `timbers prompt changelog --last 5 --append "Use emoji headers"`

---

## Appendix: Full Template Examples

See `internal/templates/*.md` for complete built-in templates once implemented.
