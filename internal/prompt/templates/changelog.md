---
name: changelog
description: Conventional changelog grouped by type
version: 2
---
Generate a changelog from these development log entries.

**Format**: Conventional changelog with sections (include only those with entries):
- **Added** - New features
- **Changed** - Changes to existing functionality
- **Fixed** - Bug fixes
- **Removed** - Removed features

**Style**:
- Past tense, one line per item
- Group by date (most recent first)
- Derive categories from entry content—don't force entries into categories that don't fit

**Constraints**:
- Only include what's in the entries. Don't infer additional changes.
- If an entry doesn't clearly fit a category, use your best judgment or skip it.
- Don't add version numbers unless present in the entries.

## Entries ({{entry_count}})

{{entries_json}}
