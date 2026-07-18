---
title: 'Project Update'
date: '2026-07-18'
tags: ['example', 'project-update']
authors: ["Bob Bergman"]
---

Generated with `timbers report project-update --since 7d | claude -p --model opus`

---

This period reworks how Timbers records and reports who did the work, and replaces the Hugo demo site with the new Timbermill publishing harness. The most useful change: entries now capture durable contributor attribution at commit time, so credit survives rebases, squashes, and pruning.

## New
- Contributor attribution is now persisted on each entry. Git authors and `Co-authored-by` trailers are captured (mailmap-normalized) when you log, so person-level credit survives history rewrites. Attribution is automatic; use `--who "Name <email>"` only for pairing, shared work, bots, or corrections, where it replaces the entire automatic set. You can also amend attribution on existing entries.
- `timbers report` runs first-class report profiles — a low-friction path from captured rationale to a finished report (including a decision digest and a user-focused project update) without rebuilding selection and prompt conventions by hand. Generated reports fail closed rather than emit partial artifacts.
- The published site now runs on Timbermill (Eleventy-based) with a new "Working Mill" theme, dark mode, and reduced-motion-safe styling, replacing the previous Hugo demo while preserving existing routes.

## Fixed
- Corrupt or malformed ledger files are no longer silently dropped from queries and generated artifacts. They now surface through `timbers doctor` and query output, and `draft`/`report` generation stops before emitting incomplete results.
- Standup reports now include contributor attribution, which the previous text input omitted.
- Generated reports no longer leak model selection or drafting analysis; every built-in template now presents a finished, direct artifact.

## Action required
- The `catchup` workflow has been removed. It generated low-confidence historical rationale stored indistinguishably from authored reasoning; first-log baselines, batch logging, and acknowledgements cover the legitimate adoption cases. Existing historical entries are preserved.
- Built-in generated ADRs are replaced by non-authoritative decision digests that cite source entries. Your native project ADRs remain the authoritative record.

## Known limitations
- Multi-repository rollups and native ADR/design-document ingestion are not yet available; these are deferred to the broader Timbermill publishing work.
