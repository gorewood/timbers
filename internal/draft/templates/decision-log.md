---
name: decision-log
description: ADR-style decision log extracted from design rationale
version: 4
vars:
  starting_number: "1"
---
Extract architectural decisions from these development log entries and format them as an ADR-style decision log.

**What to extract**: Only entries whose "why" field contains genuine design trade-offs — choices between alternatives with reasoning. Skip entries where "why" is a feature description, a restatement of "what", or empty/thin.

**Notes field**: Some entries include a `notes` field with detailed deliberation context — alternatives considered, surprises, reasoning chains, moments where a teammate (human or AI agent) reshaped the call. When present, use notes as the primary source for Context and Consequences sections. The "why" field has the verdict; "notes" has the journey.

**Consolidation**: If multiple entries describe iterations on the same decision (initial call, refinement, course-correction), produce ONE ADR that captures the final shape and the path that got there. Don't emit separate ADRs for every entry that touched a decision. The ADR's **Date** in this case is the entry that established the *final accepted shape* — usually the last entry to materially change the verdict, not the first entry to broach the topic.

**Format each decision as**:

```markdown
## ADR-N: Decision Title

**Status:** Accepted
**Date:** YYYY-MM-DD

**Context:** What problem or choice was being faced. Name the operator's intent or constraint that drove the decision — not just the technical environment. ("The team was about to ship X when reviewer flagged Y" beats "There was a need to handle Y".)

**Decision:** What was decided, and the key reasoning. If a teammate (human or AI agent) reshaped the call, name that moment.

**Consequences:**
- Positive and negative implications
- What this enables or constrains going forward
- What this *doesn't* solve (open questions, deferred trade-offs)
```

**Status values**:
- `Accepted` — default for any decision being recorded; the call has been made and is in effect
- `Superseded by ADR-N` — when a later entry overrides this decision; include both ADRs in output if both are within the entry range
- `Proposed` — only if entries explicitly indicate the decision is provisional / pending validation

**Supersession**: If a later entry in the input set explicitly reverses or replaces an earlier decision, emit BOTH ADRs:
- The older one with `Status: Superseded by ADR-N` (where N is the newer ADR's number)
- The newer one with `Status: Accepted` and a "Replaces ADR-M" line at the end of Decision

**Title pattern**: Action- or choice-oriented, specific. Use one of:
- `[Verb] [object] [over alternative]` — "Use Markdown over JSON for changelog output"
- `[Decision noun] for [domain]` — "Filename grammar for ledger entries"
- `[Subject]: [chosen direction]` — "Auth tokens: opaque session IDs over JWTs"

Avoid generic titles like "Output Format", "Authentication Approach", "Storage Choice".

**Style**:
- Decision titles should be specific (see pattern above)
- Context should establish the fork in the road — what alternatives existed, who was asking, what was at stake
- Consequences should include both upsides AND downsides. Every decision has trade-offs.
- Use `backticks` for commands, flags, function names, file paths
- Be concise but complete — each ADR should stand alone (a reader landing on ADR-7 alone should understand the decision)

**Filtering**:
- If an entry's "why" field just restates what was done ("Added X because users needed X"), skip it entirely
- If an entry has no "why" or a thin one, skip it
- Fewer high-quality ADRs are better than padding with weak ones
- Pure refactors, dependency bumps, and bug fixes generally don't merit ADRs unless the fix involved a real design trade-off

**Numbering**: Number sequentially starting from ADR-{{vars.starting_number}}. Each
subsequent ADR increments by 1. The caller supplies the offset so numbers stay
stable across runs — do not renumber earlier ADRs, do not reset to 1.

**Constraints**:
- Only extract decisions present in the entries. Don't infer decisions not stated.
- Consequences must be either (a) explicitly stated in the entries, or (b) *mechanical* implications of the decision itself (e.g., "Choosing JWT means clients must handle token refresh" follows mechanically from "JWT chosen"). Do not speculate about second-order effects, future maintenance burden, or downstream impacts the entries don't establish. When in doubt, leave the consequence out — a tight ADR with three real consequences beats a padded one with seven invented ones.
- If no entries contain genuine design decisions, say so plainly: emit "_No architectural decisions in this range._" and stop.

**Output discipline**:
- Output the decision log ONLY. No preamble, commentary, acknowledgment, or meta-discussion.
- Do not begin with "Here is..." or "I'll generate..." or any thinking-out-loud.
- Do not end with "Let me know..." or any sign-off.
- The first line of your response must be part of the document itself.

## Entries ({{entry_count}}) | {{date_range}}

{{entries_json}}
