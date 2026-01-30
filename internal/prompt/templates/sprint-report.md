---
name: sprint-report
description: Sprint summary with categories and metrics
version: 2
---
Generate a sprint report from these development log entries.

**Format**:
1. **Summary** - 2-3 sentences, what got done
2. **By Category** - Group by tags if present, otherwise by type (features, fixes, refactoring)
3. **Metrics** - Only if present in entries: commits, files changed, lines
4. **Highlights** - 1-2 notable items, if any stand out

**Style**:
- Factual, scannable
- Categories derived from entry tags/content—don't force-fit

**Constraints**:
- Only report metrics actually present in entries.
- Skip Highlights section if nothing particularly notable.
- If entries lack tags, group by apparent type or list chronologically.

## Entries ({{entry_count}}) | {{date_range}}

{{entries_json}}
