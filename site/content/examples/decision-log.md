+++
title = 'Decision Log'
date = '2026-05-21'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

## ADR-1: Compact prime output by default, full guide behind opt-in flag

**Status:** Accepted
**Date:** 2026-05-07

**Context:** Default `timbers prime` injection at session start was spending agent context budget on repeated coaching text — the same protocol guidance reloaded every session. With multiple agents priming per session, the cumulative token cost was substantial, and most of the payload was operational boilerplate the agent had already seen.

**Decision:** Ship a compact v2 renderer as the default `prime` output, preserving only operational ledger safeguards (pending state, stale-anchor warning, recent entries). Move the full coaching guide behind a `--full`/`guide` mode. Claude hooks switched to the new hook mode.

**Consequences:**
- Smaller per-session context payload across all agents using prime injection.
- Compact mode initially regressed three affordances — truncated IDs were unresolvable, custom PRIME.md customizations became invisible, and JSON `Mode` field was hardcoded before the flag was interpreted. Fixed in a follow-up: full IDs restored, a hint surfaces when PRIME.md is overridden, JSON honestly reports the requested mode.
- Agents that need the full coaching guide must now opt in explicitly via `--full`.

---

## ADR-2: Single source of truth for protocol text via `internal/protocol` package

**Status:** Accepted
**Date:** 2026-05-20

**Context:** Protocol guidance (commit ordering, stale-anchor recovery) needed to appear in both the full PRIME doc and the MCP subset, with different consumers needing different subsets of the same content. A push-before-log race had also stranded an entry locally in osprey-strike — the protocol said "commit, log" but didn't bold the no-push-between-them rule.

**Decision:** Extract shared protocol/stale-anchor sections into an `internal/protocol` package. Both `cmd/timbers` (full PRIME doc) and `internal/mcp` (subset) compose from these constants. Rewrote the ordering checklist with an explicit "never push between commit and log" callout. Added `IsPushedToUpstream` + `printer.Warn` so `timbers log` warns when a documented commit is already on upstream but the entry isn't.

**Consequences:**
- Const-plus-const concatenation gives compile-time composition without runtime overhead.
- Different consumers can keep their own subsets in sync as the canonical text evolves.
- Reviewer noted prose-only protocol tests would miss a reordered checklist; added an explicit ordering-position assertion as part of the sanity test.
- A single-file approach was considered (inspired by folding skip-authors into `.timbersignore`) but rejected because workflow text is fundamentally compositional.

---

## ADR-3: Gate pending detection on first-parent line, not full DAG

**Status:** Accepted
**Date:** 2026-05-18

**Context:** Parallel agents share `.timbers/` via merges. The pre-existing full-DAG gate was blocking agent A's commits on agent B's undocumented commits — wrong actor for the gate to fire on. Author-based attribution was considered first, but all agents in the target setup commit as the same git identity, making author filtering a no-op.

**Decision:** Add `LogFirstParent` to the git layer; refactor `GetPendingCommits` behind a `firstParent bool`; route `HasPendingCommits` through the gate path. Add a `dropEmptyFileChanges` gate-only filter so clean merges and `--allow-empty` commits don't block (display path keeps the conservative empty=unknown rule). Add a `TIMBERS_SKIP_CROSS_AGENT_DEBT` env-var bypass for the narrower case where the merge commit itself touched source during conflict resolution.

**Consequences:**
- Git-native solution works regardless of how agents configure their commit identity.
- Display path and gate path now have different semantics — gate is lenient, display still surfaces merges for awareness. The divergence is intentional but adds cognitive load.
- Cross-agent debt env-var escape hatch ships as a stopgap; the merge-commit-touched-source case isn't fully solved.
- Author-based attribution path explicitly rejected for this user's setup.

---

## ADR-4: Hooks share one `hasActionablePending()` definition of "actionable"

**Status:** Accepted
**Date:** 2026-05-10

**Context:** Pending, log, and the post-commit hook were drifting on what counts as undocumented work. The hook nudged agents to log commits that `timbers log` would refuse, creating contradictory signals. A real bug surfaced in gorewood/vellum where `.beads/issues.jsonl`-only commits triggered the nudge because `runPostCommitHook` never checked `HasPendingCommits`.

**Decision:** Extract `hasActionablePending()` helper combining `IsRepo`, `IsInteractiveGitOp`, `.timbers/` existence, storage construction, and `HasPendingCommits` checks. Route both pre-commit and post-commit hooks through it. A narrower inline fix in `runPostCommitHook` was considered but rejected — the helper eliminates the divergence root cause and any future hook (post-rewrite, etc.) inherits the same gate for free.

**Consequences:**
- Adding a third hook later requires no special-casing to stay consistent.
- Parity is now a single one-line assertion in tests.
- Test harness needed a `seedFile` escape hatch so `.timbersignore` can be baked into the initial commit; otherwise the ignore file itself becomes actionable.

---

## ADR-5: Drop Dolt remote; embedded JSONL is the sole sync channel

**Status:** Accepted
**Date:** 2026-05-08

**Context:** A previous agent had skipped committing auto-staged JSONL after misreading a one-time bd 1.0.x schema flip (records gained `_type` discriminator and reordered). The Dolt remote had been unused since 2026-04-29, and `bd dolt push/pull` no-op without one anyway. Two sync channels (Dolt + JSONL) created room for misinterpretation.

**Decision:** Remove the bd SQL remote. Document embedded-only mode in AGENTS.md. Tell agents to commit bd 1.0.x's `_type`-prefixed JSONL rewrites without reverting. Add a `bd export | diff` drift-recovery recipe. Drop `bd dolt push` from session-close workflow. Keeping the remote "just in case" was considered and rejected: the gitignored `.beads/dolt/` working DB plus the committed JSONL is sufficient for full reconstruction.

**Consequences:**
- Single sync channel — no more "which one is authoritative" ambiguity.
- A future Dolt remote can be re-added with one command if multi-machine federation is ever needed.
- Agents now have an explicit drift-recovery procedure when the JSONL looks "wrong."

---

## ADR-6: `.timbersignore` carries author globs via `author:` prefix

**Status:** Accepted
**Date:** 2026-05-20

**Context:** The impending q-redshifted autofix pipeline needed a bot-author skip path. An initial implementation used a dedicated `.timbers/skip-authors` file. The user pushed back on file sprawl.

**Decision:** Extend `.timbersignore` with `author:<glob>` lines via `classifyTimbersIgnoreLine`. Single parser yields both rule shapes (path globs and author globs). Mirrors the `.gitignore` family idiom and gives a single source of truth for repo skip config.

**Consequences:**
- One file to teach agents about, not two.
- Edge case: the `author:` prefix collides with literal paths starting with `author:` — extremely rare since `:` is forbidden in Windows filenames, but documented.
- Author glob examples (exact-name, email-domain, GitHub-bot prefix-wildcard) needed explicit doc coverage because `filepath.Match`'s character-class semantics aren't obvious.

---

## ADR-7: `timbers ack` for honest skip-with-reason instead of fabrication

**Status:** Accepted
**Date:** 2026-05-20

**Context:** Agents hitting commits they shouldn't document (merges, vendored changes, automated work) had two bad options: fabricate an entry, or bypass the hook with `--no-verify`. Neither leaves an audit trail.

**Decision:** Ship `timbers ack` storing acknowledgements under `.timbers/YYYY/MM/DD/ack_*.json` with `kind="ack"` under the existing `timbers.devlog/v1` schema. Thread `AckedSet` through `filterCommits` parallel to `docSet` (single scan per pending check).

**Consequences:**
- Honest "skip with reason" is now a first-class operation with a persistent record.
- Reuses the existing devlog schema rather than introducing a new file format.
- Reviewer caught an `AckedSet` double-scan on round one — corrected before commit.

---

## ADR-8: Phase 0 diagnostic-only response to side-branch latest-anchor friction

**Status:** Superseded by ADR-9
**Date:** 2026-05-21

**Context:** Laura's friction on v0.22.0 read as "pending scrambling." An initial plan proposed an algorithm change (first-parent-aware latest entry). An independent reviewer caught that this would regress the common "entry on feature branch then merge" workflow already covered by `TestBranchMerge_EntryOnBranch_NoneOnMain` — losing the anchor entirely for solo-dev-feature-branch flows. Their critique was specific: "Phase 1 is wrong; drop it."

**Decision:** Ship pure diagnostic in v0.22.1 — no algorithm change. Add `IsOnFirstParentLine` (bounded first-parent walk) and `LatestAnchorOffFirstParent` (disambiguates from stale-anchor). Wire diagnostics into pending and doctor (JSON + human + check surfaces). Codify the regression contract for the Laura pathology in a new integration test. The contract test passing on current code is the empirical signal that the algorithm change may not be needed at all — the friction may be opacity, not incorrectness.

**Decision was reshaped by the independent reviewer's regression counterexample**, which converted a planned algorithm change into a diagnostic-soak phase.

**Consequences:**
- No regression risk for solo-dev-feature-branch users.
- Existing escape hatches now have visibility — users see the topology.
- Doesn't actually fix Laura's case; defers the fix until the diagnostic data clarifies whether an algorithm change is warranted.
- Two pre-existing integration test rot issues surfaced and were repaired separately (`commitEntry` helper predating v0.17 auto-commit; `SameDayEntries` predating v0.18 filename-safe encoding).

---

## ADR-9: Fix `--batch` entry anchor by preferring first-parent-line commit

**Status:** Accepted
**Date:** 2026-05-21

**Context:** After v0.22.1 shipped diagnostics-only, a second reviewer dug into Laura's transcript and identified the actual culprit: `--batch` mode grouped a Work-item trailer's local + cross-branch commits, then naively picked `commits[0]`, which sometimes landed on a side-branch SHA. The resulting entry was structurally valid but its anchor pointed to a side branch, breaking linear-anchor assumptions downstream.

**Decision:** Add `pickBatchAnchor` helper that iterates the group's commits and returns the first one on HEAD's first-parent line; falls back to `commits[0]` when no commit qualifies (pure cross-agent debt case, where the off-first-parent-line diagnostic from v0.22.1 surfaces the residual situation). HEAD resolved best-effort via `git.HEAD()` — fails gracefully back to legacy behavior on git error rather than failing the batch run.

A companion fix in pending detection: extract `anchorShortCircuit` helper handling both stale-anchor and off-first-parent gate triggers, routing both through `CommitsReachableFrom` + `filterCommits`. Add direct `docSet[commit.SHA]` membership check in `filterByRules` before ack/revert checks (it was previously only used for revert detection). Both fixes together drop Laura's class to zero pending; either alone is insufficient.

Replaces ADR-8.

**Consequences:**
- Much smaller fix than the originally-planned algorithm change.
- Diagnostic surface from v0.22.1 still ships value — surfaces residual cross-agent debt where `pickBatchAnchor` falls back to `commits[0]`.
- New "documented" classify-reason in the `TIMBERS_DEBUG` trace and `countAutoSkipped` count.
- Two converging independent reviewers (one caught the original algorithm plan as a regression; the other localized the bug to `--batch`) materially reshaped the call.

---

## ADR-10: `TIMBERS_DEBUG` env-truthy trace knob over a permanent flag

**Status:** Accepted
**Date:** 2026-05-20

**Context:** Pending-detection trace output is high-value for debugging but high-noise in normal operation. A persistent `--debug` flag would either need plumbing through every entry point or get ignored.

**Decision:** Gate trace output in `skipcount.go` on `TIMBERS_DEBUG` env-truthy. Same env-truthy helper covers the `TIMBERS_SKIP_CROSS_AGENT_DEBT` bypass (ADR-3).

**Consequences:**
- Zero plumbing — any caller (including external scripts) can flip it.
- Standard pattern for users who already understand `DEBUG=1`.
- No discoverability from `--help`; relies on docs.
