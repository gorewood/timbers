---
name: standup
description: Daily standup from recent work
version: 4
---
Generate a standup update from these development log entries.

**Format**: Bullet points, ordered by importance. Blockers and friction first.

**Priority ordering**:
1. Blockers — anything preventing progress, waiting on others, stuck
2. Friction — things that took longer than expected, surprising obstacles
3. Completed work — what shipped or landed
4. In-progress — what's actively being worked on
5. Next steps — what's planned, if entries hint at it

**Style**:
- Active voice, outcome-focused: "Shipped X" not "X was completed"
- Be concrete: name the feature, the bug, the command — not "various improvements"
- Use `backticks` for technical terms when they add clarity
- Each bullet stands alone — no dependencies between them
- Let content dictate length. 3 bullets for a quiet day, 7 for a busy one.
  Don't pad sparse days or crush dense ones.

**Surfacing PM/manager-relevant signals**:
- If something took much longer than the description implies, say so
- If work is concentrated in one area, note why (technical debt, complex domain, etc.)
- If entries mention blocked, slow, or frustrating work, surface it prominently
- "What's next" or "blocked by" context when entries hint at it

**Numbers and metrics**:
- DO NOT cite raw diff stats or file counts
- Convey significance through context: "Major auth overhaul" not "Changed 12 files"

**Constraints**:
- Only summarize what's actually in the entries.
- Don't add context, metrics, or implications not present.
- Fewer bullets is fine if entries are sparse.

**Output discipline**:
- Output the standup ONLY. No preamble, commentary, acknowledgment, or meta-discussion.
- Do not begin with "Here is..." or "I'll generate..." or any thinking-out-loud.
- Do not end with "Let me know..." or any sign-off.
- The first line of your response must be part of the document itself.

## Entries ({{entry_count}})

{{entries_summary}}
