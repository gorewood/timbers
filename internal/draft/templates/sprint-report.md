---
name: sprint-report
description: Sprint summary with categories and metrics
version: 3
---
Generate a sprint report from these development log entries.

**Format**:
1. **Summary** - 2-3 sentences, what got done
2. **By Category** - Group by tags if present, otherwise by type (features, fixes, refactoring)
3. **Scope** - Convey the breadth/depth of work (optional, only if meaningful)
4. **Highlights** - 1-2 notable items, if any stand out

**Style**:
- Factual, scannable
- Categories derived from entry tags/content—don't force-fit
- Use `backticks` for technical terms, commands, file names
- Be specific about what shipped, not vague ("auth improvements" → "JWT refresh token support")

**Numbers and metrics**:
- DO NOT cite raw diff stats ("362 insertions, 45 deletions")
- Convey scope through texture: "a focused fix", "a substantial refactor", "scattered changes across the CLI"
- Entry counts are OK ("5 entries this sprint") but file/line counts are robotic

**Constraints**:
- Only report what's actually in the entries.
- Skip Highlights section if nothing particularly notable.
- If entries lack tags, group by apparent type or list chronologically.

**Output discipline**:
- Output the sprint report ONLY. No preamble, commentary, acknowledgment, or meta-discussion.
- Do not begin with "Here is..." or "I'll generate..." or any thinking-out-loud.
- Do not end with "Let me know..." or any sign-off.
- The first line of your response must be part of the document itself.

## Entries ({{entry_count}}) | {{date_range}}

{{entries_json}}
