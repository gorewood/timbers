---
name: release-notes
description: User-facing release notes
version: 2
---
Generate user-facing release notes from these development log entries.

**Audience**: End users, not developers.

**Format** (include only sections with content):
- **New Features**
- **Improvements**
- **Bug Fixes**
- **Breaking Changes**

**Style**:
- Benefit-oriented language ("You can now..." not "Added support for...")
- Avoid technical jargon where possible
- One line per item

**Constraints**:
- Only include what's in the entries.
- Don't invent user-facing benefits not implied by the changes.
- Skip sections with no relevant entries.

## Entries ({{entry_count}}) | {{date_range}}

{{entries_json}}
