---
name: pr-description
description: Pull request body with summary and test plan
version: 1
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
