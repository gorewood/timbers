---
name: pr-description
description: Pull request body with summary and test plan
version: 2
---
Generate a pull request description from these development log entries.

**Format**:
```
## Summary
[2-4 bullets: what changed and why]

## Changes
[Grouped list of specific changes]

## Test Plan
[How to verify—only if inferable from entries, otherwise "See test files" or similar]
```

**Style**:
- Concise. Reviewers skim.
- Focus on what matters for review: intent, scope, risk areas.

**Constraints**:
- Only describe changes present in the entries.
- Don't invent test steps not implied by the work.
- If entries lack detail for a section, keep it minimal.

## Entries ({{entry_count}}) | Branch: {{branch}}

{{entries_json}}
