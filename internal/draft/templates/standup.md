---
name: standup
description: Daily standup from recent work
version: 5
---
Generate a standup update from these development log entries.

**Audience**: Teammates and a manager. They want signal in 30 seconds: what's blocked, what slipped, what shipped, what's next. They do NOT want a full activity log.

**Format**: Bullet points, ordered by importance. Blockers and asks first.

**Priority ordering**:
1. **Blockers** — anything preventing progress, waiting on someone, stuck
2. **Asks for help** — explicit "I need someone with X context to look at Y"; pull from entries that mention being stuck on a domain the operator doesn't own
3. **Friction** — things that took longer than expected, surprising obstacles, time burned ("Spent 3 hours chasing a flaky test before finding the actual issue in test setup")
4. **Completed work** — what shipped or landed
5. **In-progress** — what's actively being worked on
6. **Next steps** — what's planned, if entries hint at it

**Style**:
- Active voice, outcome-focused: "Shipped X" not "X was completed"
- Be concrete: name the feature, the bug, the command — not "various improvements"
- Use `backticks` for technical terms when they add clarity
- Each bullet stands alone — no dependencies between them
- Let content dictate length. 3 bullets for a quiet day, 7 for a busy one. Don't pad sparse days or crush dense ones.

**Surfacing PM/manager-relevant signals**:
- If something took much longer than the description implies, say so — with a number when entries provide one ("Burned a day on flaky CI before the real fix"), without if they don't
- If work is concentrated in one area, note why (technical debt, complex domain, blocked by spec)
- If entries mention blocked, slow, or frustrating work, surface it prominently
- "What's next" or "blocked by" context when entries hint at it

**Asks for help — examples**:
- "Could use eyes from someone who knows the auth middleware on the new token-refresh path"
- "Stuck on the Postgres deadlock; anyone seen this pattern before?"
- Only emit when entries actually indicate the operator was looking for input. Don't fabricate asks.

**Collaboration texture** (light, optional):
- It's fine to surface "the agent handled X while I focused on Y" when entries explicitly capture that division — it helps standup attendees calibrate. Skip when entries don't mention it.

**Numbers and metrics**:
- DO NOT cite raw diff stats or file counts
- Convey significance through context: "Major auth overhaul" not "Changed 12 files"
- Time-burn numbers when entries supply them ("Lost a morning to env config drift") are GOOD standup texture — they're the kind of detail that warns the team off the same trap

**Constraints**:
- Only summarize what's actually in the entries
- Don't add context, metrics, or implications not present
- Don't fabricate asks for help — only emit when entries clearly indicate the operator wants input
- Fewer bullets is fine if entries are sparse

**Output discipline**:
- Output the standup ONLY. No preamble, commentary, acknowledgment, or meta-discussion.
- Do not begin with "Here is..." or "I'll generate..." or any thinking-out-loud.
- Do not end with "Let me know..." or any sign-off.
- The first line of your response must be part of the document itself.

## Entries ({{entry_count}})

{{entries_summary}}
