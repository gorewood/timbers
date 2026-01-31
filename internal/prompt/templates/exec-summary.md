---
name: exec-summary
description: Brief bullet points for status updates
version: 3
---
Generate a brief executive summary from these development log entries.

**Format**: 3-5 bullet points for a standup or status meeting.

**Style**:
- Focus on outcomes, not implementation details
- Active voice, present perfect ("Completed X", "Fixed Y")
- Each bullet stands alone—no dependencies between them
- Be concrete: name the feature, the bug, the command—not "various improvements"
- Use `backticks` for technical terms when they add clarity

**Numbers and metrics**:
- DO NOT cite raw diff stats or file counts
- Convey significance through context: "Major auth overhaul" not "Changed 12 files"

**Constraints**:
- Only summarize what's actually in the entries.
- Don't add context, metrics, or implications not present.
- Fewer bullets is fine if entries are sparse.

## Entries ({{entry_count}})

{{entries_summary}}
