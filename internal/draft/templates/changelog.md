---
name: changelog
description: Conventional changelog grouped by type
version: 7
---
Generate a changelog from these development log entries following the [Keep a Changelog](https://keepachangelog.com/) format.

**Output structure** (use this exact format):

```markdown
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- ...

### Changed
- ...
```

**Sections** (include only those with entries):
- **Added** — Brand-new capabilities the user can now invoke (commands, flags, features)
- **Changed** — Modifications to existing capabilities' behavior, shape, or output
- **Fixed** — Bug fixes
- **Removed** — Removed features (must include migration hint if non-trivial)
- **Deprecated** — Features still working but slated for removal
- **Security** — Vulnerabilities fixed (call out CVE if known)
- **Internal** — Architecture, refactors, dev-tooling (optional, omit unless meaningful to extension authors or downstream packagers)

**Added vs Changed** — common confusion:
- A new flag on an existing command → **Changed** (the command itself isn't new)
- A new command → **Added**
- A new optional behavior toggled by config → **Changed** (existing surface gained an option)
- A new top-level subsystem the user can opt into → **Added**

**Consolidation** — entries are not bullets:
- If multiple entries iterate on the same feature/fix, merge them into ONE bullet at the outcome level. "Added flag, fixed flag bug, lifted test coverage on flag" is one bullet ("Added `--watch` flag"), not three.
- If entries describe a multi-step rollout, the changelog records the *destination*, not the journey.
- Aim for one bullet per shipped *outcome*, not one bullet per entry.

**What to exclude** (do NOT include in any section):
- Internal refactors with no observable behavior change
- Test-only changes (new tests, refactored tests, coverage lifts)
- Documentation-only changes (unless the documentation IS the deliverable)
- Dependency bumps unless they materially change behavior or fix a security issue
- Tooling changes (CI tweaks, linter config, formatter rules)
- Anything where every entry's "what" is clearly invisible to consumers of this project

If applying these exclusions leaves a section empty, omit the section. If applying them leaves the entire changelog empty, output the header and "## [Unreleased]\n\n_No user-facing changes._" rather than padding with internal noise.

**Breaking changes**:
- Prepend `**BREAKING:**` to any bullet whose effect would force a consumer to change their code, config, or workflow
- Surface them at the top of the relevant section (Added/Changed/Removed)
- Include a one-line "what to do" hint when the migration isn't obvious

**Grouping**:
- Use `## [Unreleased]` as the version-section heading in default mode
- In versioned-release mode use `## [0.3.0] - YYYY-MM-DD` (today's date)
- If entries span multiple dates, group them under a single version section (not by date)

**Output format**:
- Output raw markdown only — never wrap in code fences (no ``` blocks)

**Style**:
- Past tense, one line per item
- Start each item with a verb: Added, Fixed, Changed, Removed, Improved
- Use `backticks` for commands, flags, function names, file paths
- Be specific *at the user-visible level*: "Fixed crash in `parseConfig()` when path contains spaces" not "Fixed config bug" — but "Refactored internal config loader" should not appear at all (it's invisible)
- Group related items together within each section

**Numbers and metrics**:
- DO NOT cite raw diff stats like "10 insertions, 3 deletions"
- If scope matters, convey it naturally: "Major refactor of auth system" not "Changed 15 files"
- Performance improvements: cite numbers ONLY when entries provide them ("3× faster startup"); never invent.

**Constraints**:
- Only include what's in the entries. Don't infer additional changes.
- If an entry doesn't clearly fit a category, use your best judgment or skip it (preferring skip over force-fit).

**Header rule** (resolves the "always include header" / "omit for versioned" tension):
- Default mode (Unreleased): emit the full top-level `# Changelog` header AND the preamble AND the `## [Unreleased]` section.
- Versioned-release mode (caller appends version info, e.g. "This is release v0.3.0"): emit ONLY the version section starting at `## [0.3.0] - YYYY-MM-DD`. The caller is appending to an existing CHANGELOG.md that already has the top-level header.

**Output discipline**:
- Output the document ONLY. No preamble, commentary, acknowledgment, or meta-discussion.
- Do not begin with "Here is..." or "I'll generate..." or any thinking-out-loud.
- Do not end with "Let me know..." or any sign-off.
- The first line of your response must be part of the document itself.

## Entries ({{entry_count}})

{{entries_json}}
