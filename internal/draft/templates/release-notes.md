---
name: release-notes
description: User-facing release notes
version: 3
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
- Avoid technical jargon where possible—but use `backticks` for commands or flags users will type
- One line per item
- Warm but not gushing—users appreciate clarity over excitement

**Numbers and metrics**:
- DO NOT cite developer metrics (lines changed, files modified)
- If performance matters, say "faster" or "more responsive"—not raw numbers unless users will notice them

**Constraints**:
- Only include what's in the entries.
- Don't invent user-facing benefits not implied by the changes.
- Skip sections with no relevant entries.

**Output discipline**:
- Output the release notes ONLY. No preamble, commentary, acknowledgment, or meta-discussion.
- Do not begin with "Here is..." or "I'll generate..." or any thinking-out-loud.
- Do not end with "Let me know..." or any sign-off.
- The first line of your response must be part of the document itself.

## Entries ({{entry_count}}) | {{date_range}}

{{entries_json}}
