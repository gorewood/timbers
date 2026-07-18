---
name: project-update
description: Recurring user-facing update from recent project work
version: 2
report:
  scope:
    since: 7d
  projection: narrative
  format: markdown
  quiet_output: _No user-facing changes in this range._
---
Generate a concise project update from these development log entries.

**Audience**: Users, operators, and interested stakeholders who do not follow
the repository day to day. They want to know what changed in their experience,
why it matters, and whether they need to do anything.

**Editorial test**: Include an item only when the entries support at least one
of these claims:
- A capability became available or materially changed.
- A user-visible defect or reliability problem was fixed.
- Performance, compatibility, security, or operability changed in a way users
  can observe.
- A migration, deprecation, or known limitation affects current use.

Exclude internal refactors, tests, documentation maintenance, dependency bumps,
CI changes, planning, and development-site work unless the entry explicitly
states a user-visible effect. Do not translate internal work into vague claims
such as "improved reliability" or "better performance."

**Consolidation**: Several entries may describe one outcome. Produce one item
for the final user-visible result, not a chronology of implementation steps.

**Format** (omit empty sections):

```markdown
# Project Update

Two or three sentences describing the shape of the period and the most useful
change for readers. Do not invent a theme when the work is mixed.

## New
- Outcome and why a reader would use it.

## Improved
- Existing behavior that materially changed.

## Fixed
- User-visible problem and its resolved behavior.

## Action required
- Breaking change, migration, deprecation, or configuration action.

## Known limitations
- Only an unresolved constraint explicitly present in the entries.
```

**Style**:
- Plain, factual, and confident. Prefer concrete behavior over marketing copy.
- Use second person only for actions or capabilities the reader can actually
  take.
- Use `backticks` for commands, flags, configuration keys, and file paths.
- Keep bullets short. A small update should remain small.

**Attribution**: Contributor snapshots are publication metadata, not material
for the update body. Do not add credits, biographies, roles, or productivity
claims. A publishing layer may render a byline separately.

**Constraints**:
- Use only facts in the entries. Do not infer benefits, impact, roadmap, or
  urgency.
- Do not mention commit counts, file counts, diff statistics, or test counts.
- Do not expose private deliberation that does not affect the reader.
- If nothing passes the editorial test, output exactly
  `_No user-facing changes in this range._` and stop.

**Output discipline**:
- Perform selection, filtering, and consolidation silently. Never output candidate lists, skipped entries, drafting notes, or statements about what you are about to write.
- Output the project update only. No preamble, acknowledgment, or sign-off.
- The first line must be `# Project Update` or the exact quiet line.

## Entries ({{entry_count}}) | {{date_range}}

{{entries_json}}
