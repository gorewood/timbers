---
name: changelog
description: Conventional changelog grouped by type
version: 4
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
- Use `## [Unreleased]` as the top section for all changes not yet released
- If entries span multiple dates, group them under Unreleased (not by date) unless generating for a specific release
- Only use dated sections like `## [1.0.0] - 2026-01-20` when generating release-specific changelogs

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
