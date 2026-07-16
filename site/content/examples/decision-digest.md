---
title: 'Decision Digest'
date: '2026-07-16'
tags: ['example', 'decision-digest']
---

Generated with `timbers draft decision-digest --last 20 | claude -p --model opus`

---

The user wants me to extract design decisions and format a digest. Let me work through the entries.

Let me identify entries with genuine choices between alternatives:

1. **tb_...a7e399** — Steer agents toward contributor attribution. Notes: kept progressive, explicit steering only for specific cases. Related to attribution contract. Some rationale but more of a refinement of dca822.

2. **tb_...dca822** — Persist contributor attribution. Why: workset SHAs disappear after rebase/squash → need capture-time snapshot rather than later Git/LLM join. Notes: explicit --who replaces complete set to avoid ambiguous merge semantics. **Genuine decision** with alternatives (capture-time snapshot vs later join; full-replace vs merge).

3. **9320fc, 1695a4, 616b50** — site styling. No real alternatives. Skip.

4. **fd16c4** — ignore worktrees. Routine. Skip.

5. **bbe3b3** — docs refresh. Skip.

6. **292a0d** — Replace Hugo with Timbermill/Eleventy. Why: Hugo tightly coupled to one renderer, no reusable path. **Decision**: config-driven Eleventy over Hugo. Has rationale/alternative.

7. **9ccccf** — docs reconcile. Skip.

8. **545516** — feat(report) decision digest profiles. Feature, but see plan dd9638.

9. **d5b0e0** — inventory doc drift. Skip.

10. **aa972e** — plan Eleventy migration. Why: needed proven in-repo shape before independently versioned companion repo. This is a decision: in-repo before separate repo. Contributes to 292a0d migration. Could fold.

11. **dd9638** — define report profiles. Why: repeatable reports need small profile contract without creating a second workflow engine or hiding prompt behavior. Notes: kept draft as lower-level primitive, stored text as durable source when Git enrichment unavailable. **Decision**: profiles as thin contract over draft, not new engine.

12. **e3210f** — surface corrupt entries. Why: malformed files must never disappear silently. Decision: fail closed. Some choice (fail-closed vs partial). Borderline — it's a "must never" invariant. There's a genuine design stance: retain corrupt paths, fail closed before partial artifacts. I'd consider it.

13. **4f92ec** — Replace ADRs with decision digests. Why: heuristic numbered ADRs compete with authoritative ADRs, ambiguity re status/numbering/ownership. Notes: native ADRs remain authoritative; future publishing ingest directly rather than heuristic reconciliation. **Strong decision**.

14. **9eff03** — derive missing what from commits. Why: repeating commit subjects adds friction while reports need durable text surviving rewrites. Notes: report-time Git lookup optional enrichment; stored what is fallback; patch IDs deferred because they don't solve many-to-one squash merges. **Decision** with explicit alternatives + deferral rationale.

15. **16496f** — Retire inferred catchup workflow. Why: LLM-generated historical rationale was low-confidence data stored indistinguishably from authored reasoning; existing features cover legitimate cases. Notes: preserves historical entries. **Decision**: remove catchup.

16. **09f482** — Unify cross-agent-debt gate + resolve --anchor. Why: detailed root cause. Notes: Rejected report's ask to make hook and pending share one computation — gate (first-parent) and display (full-DAG) deliberately different. Also considered splitting into three commits, dropped .timbers-absent guard. **Decision**: keep gate and display computations separate. Genuine choice between alternatives explicitly. Also --anchor resolution. This has real deliberation. Include.

17. **abf4b1** — doctor detection for outdated hooks. Notes: Direct content comparison over stamped version number — keeps single source of truth. Scoped to post-rewrite only. **Decision**: content comparison vs version stamp. Genuine alternative considered.

18. **cafcf0** — POSIX hook for dash. Bug fix, but notes discuss considered alternatives (always run test under dash — left it). This is a routine bug fix. The decision content is thin re design. Skip — it's a bugfix; the "considered making the test always run under dash" is minor. Skip.

Now let me consolidate. Related clusters:

- **Contributor attribution**: dca822 (main) + a7e399 (steering refinement). Consolidate into one, final shape a7e399 date 2026-07-16. Both materially contribute.

- **Report/digest**: dd9638 (plan) + 545516 (impl) — report profiles as thin contract. Consolidate.

- **ADR → digest**: 4f92ec standalone. Related to report profiles but distinct decision (non-authoritative digest vs numbered ADRs). Keep separate.

- **Eleventy migration**: aa972e (plan: in-repo before companion repo) + 292a0d (impl: Eleventy over Hugo). Consolidate. Final date 2026-07-14.

- **Derive what from commits**: 9eff03 standalone.

- **Retire catchup**: 16496f standalone.

- **Corrupt entries fail-closed**: e3210f. Include — genuine invariant/stance.

- **Cross-agent-debt gate separation**: 09f482. Include.

- **Doctor content comparison vs version stamp**: abf4b1. Include.

Let me reconsider which are truly "design decisions with a real alternative + reason" vs feature descriptions.

e3210f: "malformed ledger files must never disappear silently" — the alternative is silent skip vs surface+fail-closed. That's a genuine stance. Include but concise.

Let me write. Ordering: reverse chronological is natural. I'll order by Observed date descending.

Dates:
- Contributor attribution: 2026-07-16 (a7e399/dca822)
- Eleventy migration: 2026-07-14
- Report digest profiles: 2026-07-14
- ADR → digest: 2026-07-14
- Derive what from commits: 2026-07-14
- Corrupt entries: 2026-07-14
- Retire catchup: 2026-07-14
- Cross-agent gate: 2026-07-10
- Doctor content comparison: 2026-06-26

Let me write the digest.

For sources I cite the timbers entry IDs (full IDs as given in input).

Let me draft each carefully, only including trade-offs explicitly stated.# Decision Digest

_Retrospective summary from development-ledger entries. Project ADRs and design documents remain authoritative._

## Snapshot contributor identity at capture time instead of joining from Git later

**Observed:** 2026-07-16
**Sources:** `tb_2026-07-16T18:36:00Z_dca822`, `tb_2026-07-16T20:00:52Z_a7e399`

**Context:** Downstream person-level credit needs a source of contributor identity. Workset SHAs can disappear after rebase, squash, shallow cloning, or pruning, which would break a later Git or LLM join.

**Decision:** Capture an optional v1 `contributors` snapshot on the entry at write time (mailmap-normalized Git authors plus `Co-authored-by` trailers) rather than reconstructing identity afterward. Explicit `--who` replaces the entire automatic set. Agent-facing guidance keeps this progressive: automatic attribution is the default, and `--who` is reserved for intentional full-set replacement (pairing, shared work, bots, correction).

**Trade-offs:** Explicit `--who` replaces the complete automatic set to avoid ambiguous merge semantics. Valid bot identities remain identities; downstream person-only views must match a known-human roster rather than guessing. Any `--who` value warns that it replaces the full automatic set.

## Replace Hugo with a config-driven in-repo Eleventy harness

**Observed:** 2026-07-14
**Sources:** `tb_2026-07-14T18:56:47Z_aa972e`, `tb_2026-07-14T18:56:49Z_292a0d`

**Context:** The Hugo demo was tightly coupled to one renderer and offered no reusable path for native project artifacts or a sufficiently clear reading experience. The publishing harness needed a proven in-repo shape before considering an independently versioned companion repository.

**Decision:** Prove the harness in-repo first (deferring a separate companion repo), then replace Hugo with a config-driven Eleventy materialization built on a platform-neutral Markdown collection contract, preserved public routes, and privacy/containment checks.

## Model report profiles as a thin contract over `draft`, not a second workflow engine

**Observed:** 2026-07-14
**Sources:** `tb_2026-07-14T18:56:46Z_dd9638`, `tb_2026-07-14T18:56:48Z_545516`

**Context:** Users need a low-friction path from captured rationale to a repeatable report without manually rebuilding selection and prompt conventions.

**Decision:** Add a small YAML-frontmatter profile contract (scope, projection, format, quiet output) rather than building a second workflow engine or hiding prompt behavior. `draft` remains the lower-level primitive.

**Trade-offs:** Stored text is the durable source when Git enrichment is unavailable.

## Replace generated numbered ADRs with non-authoritative decision digests

**Observed:** 2026-07-14
**Sources:** `tb_2026-07-14T16:01:48Z_4f92ec`

**Context:** Heuristically generated numbered ADRs competed with authoritative project ADRs and created ambiguity about status, numbering, and ownership.

**Decision:** Replace the built-in with a non-authoritative digest that cites source entries and drops numbering and lifecycle claims.

**Trade-offs:** Native ADRs and design documents remain authoritative. Future publishing should ingest those Markdown artifacts directly alongside Timbers-generated reports rather than attempting heuristic reconciliation.

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

**Decision:** Rejected collapsing them into one computation — the divergence is deliberate so agent A isn't blocked by agent B's already-merged debt. Instead, the Stop hook was made to honor the same `TIMBERS_SKIP_CROSS_AGENT_DEBT` opt-out as the sibling hooks, `--anchor` is resolved to a concrete SHA before storage, and the entry commit self-exempts via the env var (not `--no-verify`, so foreign hooks still run).

**Trade-offs:** Collapsing gate and display into one computation would regress the isolation that keeps parallel-agent worktrees from blocking each other.

## Detect stale hooks by content comparison, not a stamped version number

**Observed:** 2026-06-26
**Sources:** `tb_2026-06-26T19:42:50Z_abf4b1`

**Context:** Installed hooks never picked up generator improvements because hook installation skips when a section is already present, and nothing surfaced the staleness.

**Decision:** Compare the installed post-rewrite section directly against the current generator output rather than tracking a stamped version/hash marker, so there is a single source of truth with nothing to bump.

**Trade-offs:** Scoped to the post-rewrite hook only — pre/post-commit hooks are thin shims to the binary, so only post-rewrite carries self-contained logic that can rot.
