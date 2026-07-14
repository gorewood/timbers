---
name: decision-digest
description: Retrospective digest of explicit design decisions
version: 2
report:
  scope:
    last: 20
  projection: decision
  format: markdown
  quiet_output: _No explicit design decisions in this range._
---
Extract explicit design decisions from these development log entries and format them as a retrospective decision digest.

This digest is a report, not an authoritative architecture decision record. Project ADRs and design documents remain the source of truth.

**What to extract**: Only entries whose `why` or `notes` field records a genuine choice between alternatives and the reason for that choice. Skip feature descriptions, implementation summaries, routine fixes, and thin rationale.

**Notes field**: Use `notes` as the primary source for alternatives and trade-offs when present. The `why` field usually contains the verdict; `notes` may contain the deliberation.

**Consolidation**: If several entries describe the same decision evolving, produce one digest item for the final shape. Cite every entry that materially contributed to it.

**Format when decisions exist**:

```markdown
# Decision Digest

_Retrospective summary from development-ledger entries. Project ADRs and design documents remain authoritative._

## Specific decision title

**Observed:** YYYY-MM-DD
**Sources:** `entry-id`, `entry-id`

**Context:** The explicit problem or choice recorded in the entries.

**Decision:** What was chosen and why.

**Trade-offs:** Only benefits, costs, constraints, or open questions stated in the entries. Omit this section when none were recorded.
```

**Source citations**:
- Cite Timbers entry IDs from the input, not commit hashes inferred from elsewhere.
- Use the date of the entry that records the final decision as `Observed`.
- Source citations identify the evidence for the digest; they do not turn it into an ADR.

**Title pattern**: Make titles action- or choice-oriented and specific. Prefer "Use Markdown over JSON for changelog output" over "Output Format."

**Filtering**:
- Do not infer a decision from code changes, commit metadata, or implementation details.
- Do not promote a feature choice into an architectural decision unless the rationale records a real alternative or constraint.
- Pure refactors, dependency bumps, and routine bug fixes usually do not belong.
- Fewer well-supported decisions are better than a comprehensive activity log.

**Constraints**:
- Do not assign ADR numbers, lifecycle statuses, or supersession relationships.
- Do not claim that a decision is accepted, proposed, or authoritative.
- Do not invent consequences. Include only trade-offs explicitly present in `why` or `notes`.
- If no entries contain an explicit design decision, output exactly `_No explicit design decisions in this range._` and stop.

**Output discipline**:
- Output the decision digest only. No preamble, acknowledgment, or sign-off.
- When decisions exist, the first line must be `# Decision Digest`.

## Entries ({{entry_count}}) | {{date_range}}

{{entries_json}}
