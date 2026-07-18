---
title: 'Decision Digest'
date: '2026-07-16'
summary: 'Source-cited choices and trade-offs captured retrospectively without claiming ADR authority.'
tags: ['example', 'decision-digest']
authors: ['Bob Bergman']
---

Generated with `timbers report decision-digest --last 20 | claude -p --model opus`.

---

_Retrospective summary from development-ledger entries. Project ADRs and design documents remain authoritative._

## Snapshot contributor identity at capture time instead of joining from Git later

**Observed:** 2026-07-16
**Sources:** `tb_2026-07-16T18:36:00Z_dca822`, `tb_2026-07-16T20:00:52Z_a7e399`

**Context:** Downstream person-level credit needs a source of contributor identity. Workset SHAs can disappear after rebase, squash, shallow cloning, or pruning, which would break a later Git or LLM join.

**Decision:** Capture an optional `contributors` snapshot at write time rather than reconstructing identity later. Automatic attribution uses mailmap-normalized Git authors and `Co-authored-by` trailers; `--who` intentionally replaces that set for shared work, bots, or corrections.

**Trade-offs:** Full replacement avoids ambiguous merge semantics. Bot identities remain valid contributors, so person-only views need an explicit human roster rather than heuristics.

## Replace Hugo with a config-driven in-repo Eleventy harness

**Observed:** 2026-07-14
**Sources:** `tb_2026-07-14T18:56:47Z_aa972e`, `tb_2026-07-14T18:56:49Z_292a0d`

**Context:** The Hugo demo was tightly coupled to one renderer and offered no reusable path for native project artifacts or a sufficiently clear reading experience. The publishing harness needed a proven in-repo shape before considering an independently versioned companion repository.

**Decision:** Prove the harness in-repo first (deferring a separate companion repo), then replace Hugo with a config-driven Eleventy materialization built on a platform-neutral Markdown collection contract, preserved public routes, and privacy/containment checks.

## Model report profiles as a thin contract over `draft`, not a second workflow engine

**Observed:** 2026-07-14
**Sources:** `tb_2026-07-14T18:56:46Z_dd9638`, `tb_2026-07-14T18:56:48Z_545516`

**Context:** Users need a low-friction path from captured rationale to a repeatable report without manually rebuilding selection and prompt conventions.

**Decision:** Add a small YAML-frontmatter profile contract (scope, projection, format, quiet output) rather than building a second workflow engine or hiding prompt behavior. `draft` remains the lower-level primitive, and stored text remains the durable source when Git enrichment is unavailable.

## Replace generated numbered ADRs with non-authoritative decision digests

**Observed:** 2026-07-14
**Sources:** `tb_2026-07-14T16:01:48Z_4f92ec`

**Context:** Heuristically generated numbered ADRs competed with authoritative project ADRs and created ambiguity about status, numbering, and ownership.

**Decision:** Replace the built-in with a non-authoritative digest that cites source entries and drops numbering and lifecycle claims.

**Trade-offs:** Native ADRs and design documents remain authoritative; the digest cannot assign their status, numbering, or lifecycle.

## Snapshot commit subjects when `what` is omitted; defer patch IDs

**Observed:** 2026-07-14
**Sources:** `tb_2026-07-14T16:01:41Z_9eff03`

**Context:** Repeating commit subjects in ledger entries adds capture friction, but reports need durable text that survives history rewrites.

**Decision:** When `what` is omitted, snapshot the resolved commit subjects at capture time; preserve explicit positional summaries and fail when Git has no usable subject.

**Trade-offs:** Report-time Git lookup remains optional enrichment; stored `what` is the fallback after rebase or squash merge. Patch IDs were deferred because they do not solve many-to-one squash merges.

## Fail closed on corrupt ledger entries rather than emitting partial artifacts

**Observed:** 2026-07-14
**Sources:** `tb_2026-07-14T18:56:30Z_e3210f`

**Context:** Malformed ledger files must never disappear silently from queries or generated artifacts.

**Decision:** Retain corrupt-file paths during reads, surface them through `doctor` and human query output, and make `draft`/`report` generation fail closed before emitting partial artifacts.

## Retire the inferred catchup workflow

**Observed:** 2026-07-14
**Sources:** `tb_2026-07-14T15:55:04Z_16496f`

**Context:** LLM-generated historical rationale was low-confidence data stored indistinguishably from authored reasoning, and the first-log baseline, batch logging, acknowledgements, and ignore rules already covered legitimate adoption cases.

**Decision:** Remove the catchup command and implementation rather than keep low-confidence inferred reasoning in the ledger.

**Trade-offs:** The removal deliberately preserves historical ledger entries, generated changelog examples, dated posts, and dated plans as records of prior behavior.

## Keep the cross-agent-debt gate and the pending display as separate computations

**Observed:** 2026-07-10
**Sources:** `tb_2026-07-10T08:51:38Z_09f482`

**Context:** A bug report asked for the Stop hook's block and `timbers pending`'s output to share one computation. The gate uses first-parent history; the display uses the full DAG.

**Decision:** Keep the computations separate. The first-parent gate prevents one agent's merged debt from blocking another agent, while the full-DAG pending view preserves repository-wide visibility. Align the hooks on the `TIMBERS_SKIP_CROSS_AGENT_DEBT` opt-out without collapsing those scopes.

**Trade-offs:** Collapsing gate and display into one computation would regress the isolation that keeps parallel-agent worktrees from blocking each other.

## Detect stale hooks by content comparison, not a stamped version number

**Observed:** 2026-06-26
**Sources:** `tb_2026-06-26T19:42:50Z_abf4b1`

**Context:** Installed hooks never picked up generator improvements because hook installation skips when a section is already present, and nothing surfaced the staleness.

**Decision:** Compare the installed post-rewrite section directly against the current generator output rather than tracking a stamped version/hash marker, so there is a single source of truth with nothing to bump.

**Trade-offs:** Scoped to the post-rewrite hook only — pre/post-commit hooks are thin shims to the binary, so only post-rewrite carries self-contained logic that can rot.
