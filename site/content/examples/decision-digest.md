---
title: 'Decision Digest'
date: '2026-07-18'
tags: ['example', 'decision-digest']
authors: ["Bob Bergman"]
---

Generated with `timbers report decision-digest --last 20 | claude -p --model opus`

---

_Retrospective summary from development-ledger entries. Project ADRs and design documents remain authoritative._

## Replace Hugo with an in-repo Timbermill/Eleventy publishing harness

**Observed:** 2026-07-14
**Sources:** `tb_2026-07-14T18:56:47Z_aa972e`, `tb_2026-07-14T18:56:49Z_292a0d`

**Context:** The Hugo demo was tightly coupled to a single renderer and offered no reusable path for native project artifacts or a clear reading experience.

**Decision:** Build Timbermill as an Eleventy-based publishing harness that replaces Hugo, and prove its shape in-repo before considering an independently versioned companion repository.

## Ship recurring reports as a minimal profile contract, not a workflow engine

**Observed:** 2026-07-16
**Sources:** `tb_2026-07-14T18:56:46Z_dd9638`, `tb_2026-07-16T21:34:46Z_952783`

**Context:** Repeatable reports needed a low-friction path from captured rationale to output without manually rebuilding selection and prompt conventions.

**Decision:** Define a small report-profile contract with a default scope, rather than a second workflow engine, and keep prompt behavior visible instead of hidden.

**Trade-offs:** Multi-repository rollups, native ADR ingestion, renderer extensions, audience navigation, and theme contracts were deliberately deferred to the broader Timbermill design pass.

## Replace generated numbered ADRs with non-authoritative decision digests

**Observed:** 2026-07-14
**Sources:** `tb_2026-07-14T16:01:48Z_4f92ec`, `tb_2026-07-14T18:56:48Z_545516`

**Context:** Heuristically generated, numbered ADRs competed with authoritative project ADRs and created ambiguity about status, numbering, and ownership.

**Decision:** Emit decision digests as retrospective reports and leave native ADRs and design documents as the authoritative source.

## Snapshot contributor identity at capture time, with `--who` replacing the automatic set

**Observed:** 2026-07-16
**Sources:** `tb_2026-07-16T18:36:00Z_dca822`, `tb_2026-07-16T20:00:52Z_a7e399`

**Context:** Workset SHAs can disappear after rebase, squash, shallow clone, or pruning, so person-level credit cannot rely on a later Git or LLM join.

**Decision:** Persist a capture-time contributor snapshot on each entry, keeping the contract domain-neutral and additive; bot identities count as valid identities.

**Trade-offs:** An explicit `--who` value replaces the entire automatic contributor set to avoid ambiguous merge semantics.

## Add one rich Working Mill theme without building a theme engine

**Observed:** 2026-07-16
**Sources:** `tb_2026-07-16T17:29:54Z_ea25f6`

**Context:** The neutral baseline was readable but did not give Timbers a distinctive brand or demonstrate the capture-to-report transformation visually.

**Decision:** Ship a single rich theme while preserving the static host-neutral publishing contract, rather than building a general theme engine.

**Trade-offs:** The neutral theme remains recoverable only in Git history; runtime theme selection is deferred until a second real consumer exists.

## Keep generated reports free of meta-process, retaining narrative only in the devblog

**Observed:** 2026-07-18
**Sources:** `tb_2026-07-18T12:22:21Z_518536`

**Context:** Some generated reports exposed model selection and drafting analysis instead of finished artifacts.

**Decision:** Forbid meta-process across every template so reports present finished, evidence-backed artifacts, while keeping narrative in the devblog because that is the artifact's intent.

## Surface corrupt ledger entries on read instead of dropping them silently

**Observed:** 2026-07-14
**Sources:** `tb_2026-07-14T18:56:30Z_e3210f`

**Context:** Malformed ledger files could vanish silently from queries and generated artifacts.

**Decision:** Surface corrupt entries on reads so they fail loudly rather than disappearing without notice.
