---
name: decision-log
description: ADR-style decision log extracted from design rationale
version: 1
---
Extract architectural decisions from these development log entries and format them as an ADR-style decision log.

**What to extract**: Only entries whose "why" field contains genuine design trade-offs — choices between alternatives with reasoning. Skip entries where "why" is a feature description, a restatement of "what", or empty/thin.

**Notes field**: Some entries include a `notes` field with detailed deliberation context — alternatives considered, surprises, reasoning chains. When present, use notes as the primary source for Context and Consequences sections. The "why" field has the verdict; "notes" has the journey.

**Format each decision as**:

```markdown
## ADR-N: Decision Title

**Context:** What problem or choice was being faced.

**Decision:** What was decided, and the key reasoning.

**Consequences:**
- Positive and negative implications
- What this enables or constrains going forward
```

**Style**:
- Decision titles should be specific: "Markdown Output Over JSON for Changelogs" not "Output Format"
- Context should establish the fork in the road — what alternatives existed
- Consequences should include both upsides and downsides. Every decision has trade-offs.
- Use `backticks` for commands, flags, function names, file paths
- Be concise but complete — each ADR should stand alone

**Numbering**: Number sequentially starting from ADR-1.

**Filtering**:
- If an entry's "why" field just restates what was done ("Added X because users needed X"), skip it entirely
- If an entry has no "why" or a thin one, skip it
- Fewer high-quality ADRs are better than padding with weak ones

**Constraints**:
- Only extract decisions present in the entries. Don't infer decisions not stated.
- Consequences may go slightly beyond what's stated if they're logical implications, but don't speculate wildly.
- If no entries contain genuine design decisions, say so plainly.

**Output discipline**:
- Output the decision log ONLY. No preamble, commentary, acknowledgment, or meta-discussion.
- Do not begin with "Here is..." or "I'll generate..." or any thinking-out-loud.
- Do not end with "Let me know..." or any sign-off.
- The first line of your response must be part of the document itself.

## Entries ({{entry_count}}) | {{date_range}}

{{entries_json}}
