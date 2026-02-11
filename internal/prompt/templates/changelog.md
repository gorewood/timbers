---
name: changelog
description: Conventional changelog grouped by type
version: 5
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
- **Added** - New features, commands, flags, capabilities
- **Changed** - Changes to existing functionality
- **Fixed** - Bug fixes
- **Removed** - Removed features
- **Technical** - Architecture, internals, infrastructure (optional, for dev-facing changes)

**Grouping**:
- Use `## [Unreleased]` as the top section by default
- If the user appends release version info (e.g., "This is release v0.3.0"), use a versioned heading instead: `## [0.3.0] - 2026-02-10` (with today's date)
- If entries span multiple dates, group them under a single version section (not by date)

**Style**:
- Past tense, one line per item
- Start each item with a verb: Added, Fixed, Changed, Removed, Improved
- Use `backticks` for commands, flags, function names, file paths
- Be specific: "Fixed crash in `parseConfig()` when path contains spaces" not "Fixed config bug"
- Group related items together within each section

**Numbers and metrics**:
- DO NOT cite raw diff stats like "10 insertions, 3 deletions"
- If scope matters, convey it naturally: "Major refactor of auth system" not "Changed 15 files"

**Constraints**:
- Only include what's in the entries. Don't infer additional changes.
- If an entry doesn't clearly fit a category, use your best judgment or skip it.
- Always include the Keep a Changelog header—this is non-negotiable.

## Entries ({{entry_count}})

{{entries_json}}
