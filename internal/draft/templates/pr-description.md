---
name: pr-description
description: PR body focused on intent, decisions, and risk areas
version: 4
---
Generate a pull request description from these development log entries.

In an age of agent-assisted code review, reviewers can read diffs themselves.
A good PR description answers the questions that diffs don't: why does this
change exist, what trade-offs were made, and where should reviewers focus?

**Format**:
```
## Why

[1-3 sentences: the problem or need this PR addresses, and the intent behind
the approach. Not a restatement of the title — explain *why now* and *why
this way*.]

## Design Decisions

[Bullet list of key trade-offs, alternatives considered, and choices made.
Pull from --why and --notes fields. Only include decisions that a reviewer
would find non-obvious or worth knowing. Omit if entries lack design context.]

## Risk & Reviewer Attention

[Where should reviewers look closely? What's subtle, what could break,
what has edge cases? Be specific: name files, functions, or behaviors.
Omit if the change is straightforward.]

## Scope

[1-2 sentences conveying the shape of the change: what areas of the codebase
are touched, whether it's a focused fix or a cross-cutting change. Texture,
not stats — "Touches the CLI layer and storage, leaves templates alone" not
"12 files changed".]

## Test Plan

[How to verify this works. Only include steps inferable from entries.
If entries don't hint at testing, use "See test files" or similar.]
```

**Style**:
- Concise. Reviewers skim.
- Lead with intent, not implementation.
- Use `backticks` for file names, function names, flags, commands.
- Call out breaking changes or behavioral shifts explicitly.
- Active voice.

**Constraints**:
- Only describe changes present in the entries.
- Don't invent test steps not implied by the work.
- If entries lack detail for a section, omit that section entirely rather
  than filling it with generic content.
- The "Design Decisions" section is the highest-value section — prioritize it.

## Entries ({{entry_count}}) | Branch: {{branch}}

{{entries_json}}
