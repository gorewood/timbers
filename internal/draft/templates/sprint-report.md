---
name: sprint-report
description: Sprint summary with categories, carry-overs, and signals
version: 4
---
Generate a sprint report from these development log entries.

**Audience**: PMs, managers, and the team itself reviewing the cycle. Sometimes shared with stakeholders. They want signal — what shipped, what slipped, what to discuss in retro — not a comprehensive log.

**Format** (omit any section with no content):

1. **Summary** — 2-3 sentences. The shape of the sprint, not just "what got done." If the sprint had a theme (a single feature, a debt paydown, a launch ramp), name it. If it didn't, say so plainly.
2. **By Category** — Group by tags if present, otherwise by type (features, fixes, refactoring). Specific outcomes, not categories of effort.
3. **Highlights** — 1-2 items the reader will remember. Optional. See criteria below.
4. **Friction & Carry-overs** — What slowed work, what didn't land, what's blocked into next cycle. Optional but high-value when entries surface it.
5. **Worth discussing** — One thing for retro: a surprise, a tension, a decision worth ratifying out loud. Optional.

**Highlights criteria** — pick items that match at least one:
- Surprised the team (a fix that turned out to be deeper, a feature that landed faster than expected)
- Most user-visible win
- Hardest fight (worth recognizing the slog)
- Unblocked future work in a way the title doesn't reveal

Skip Highlights if nothing meets the bar. Padding with the most-recent-feature is worse than omitting the section.

**Friction & Carry-overs**:
- Pull from entries that explicitly mention being blocked, slow, frustrating, or incomplete
- Pull from entries with tags like `blocked`, `flaky`, `incident`
- One line each. Be plain about what's not done: "X spec drafted but implementation deferred to next sprint" is more useful than "Continued work on X."
- Do NOT fabricate carry-overs. If entries are silent on what didn't land, omit this section.

**Worth discussing** (retro fodder):
- The single most interesting thing that happened — usually a surprise or a decision that took real deliberation
- Pull from entry `notes` fields when they capture deliberation or course-correction
- One sentence, framed as a discussion prompt: "Worth discussing: whether the new auth pattern should be the default for other modules" — not a summary

**Operator-voice texture** (light touch):
- If entries make it clear that AI agents drove substantial portions of the work, a single context line in the Summary is appropriate ("Heavy agent involvement on the X subsystem; operator review caught Y") — but only when entries say so. Don't fabricate partnership.
- This is calibration for the reader, not credit allocation.

**Style**:
- Factual, scannable
- Categories derived from entry tags/content — don't force-fit
- Use `backticks` for technical terms, commands, file names
- Be specific about what shipped, not vague ("auth improvements" → "JWT refresh token support")

**Numbers and metrics**:
- DO NOT cite raw diff stats ("362 insertions, 45 deletions")
- Convey scope through texture: "a focused fix", "a substantial refactor", "scattered changes across the CLI"
- Entry counts are OK as a rough volume signal ("12 entries this sprint") but file/line counts are robotic
- Velocity stats invite manipulation — skip them unless explicitly requested

**Constraints**:
- Only report what's actually in the entries
- Skip optional sections when nothing meets their bar
- If entries lack tags, group by apparent type or list chronologically
- A short, honest report beats a padded "complete" one

**Output discipline**:
- Output the sprint report ONLY. No preamble, commentary, acknowledgment, or meta-discussion.
- Do not begin with "Here is..." or "I'll generate..." or any thinking-out-loud.
- Do not end with "Let me know..." or any sign-off.
- The first line of your response must be part of the document itself.

## Entries ({{entry_count}}) | {{date_range}}

{{entries_json}}
