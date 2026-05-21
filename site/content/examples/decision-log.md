+++
title = 'Decision Log'
date = '2026-05-21'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

## ADR-1: Restore commit-before-log ordering with a push-detection warning

**Status:** Accepted
**Date:** 2026-05-20

**Context:** A parallel-agent run in osprey-strike stranded a timbers entry locally — the operator pushed the documented commit before running `timbers log`, so the entry never reached upstream. The session-start protocol said "commit, log" but didn't bold the no-push-between-them rule, and `timbers log` itself gave no signal even though the upstream-tracking data was already available.

**Decision:** Rewrite the protocol text to explicitly order commit → log → push with a "never push between them" callout, and add `IsPushedToUpstream` to the git layer so `timbers log` can fire `printer.Warn` when the documented commit is already on `@{u}` but the entry isn't. Extracted shared protocol/stale-anchor text into `internal/protocol` so both `cmd/timbers` (full PRIME doc) and `internal/mcp` (subset) compose from one source via const+const concatenation.

**Consequences:**
- Protocol text and warning share one definition of the failure mode — operators get the rule, agents get the runtime nudge.
- Adding a third PRIME consumer later inherits the shared protocol text automatically.
- Compile-time composition keeps prose subsetting cheap, but consumers can no longer mutate the protocol text at runtime.
- Does not prevent the push race — it only detects it after the fact, so a sufficiently fast `git push` between commit and `timbers log` still strands the entry until the next `timbers log` invocation.

---

## ADR-2: Scope the timbers gate to the first-parent line for parallel agents

**Status:** Accepted
**Date:** 2026-05-18

**Context:** Parallel agents share `.timbers/` via merges, but the full-DAG pending gate blocked agent A's commit on agent B's undocumented commits — the wrong actor for the gate to fire on. Author-based attribution was considered first (option 1) and dropped: in this user's setup, all agents commit as the same git identity, so the filter would no-op.

**Decision:** Add `LogFirstParent` to the git layer and refactor `GetPendingCommits` behind a `firstParent` bool. The gate path (`GetGatePendingCommits`) walks first-parent only, applies a gate-only `dropEmptyFileChanges` filter (clean merges and `--allow-empty` commits add no work to the branch's first-parent line), and routes `HasPendingCommits` through it. A `TIMBERS_SKIP_CROSS_AGENT_DEBT` env var short-circuits `hasActionablePending` for the narrower case where a merge commit itself touched source during conflict resolution.

**Consequences:**
- Git-native attribution works regardless of how agents configure their identities.
- The display path keeps the conservative "empty file list = unknown" rule, so `timbers pending` still surfaces clean merges for awareness — gate and display deliberately disagree.
- Conflict-resolution merges that touch source still block the gate; the env var is the documented escape hatch.
- A regression-test gap forced adding the `dropEmptyFileChanges` filter mid-implementation when `git merge --no-ff` was found to still block on the first-parent line.

---

## ADR-3: Phase 0 pending-detection diagnostic instead of changing the algorithm

**Status:** Accepted
**Date:** 2026-05-21

**Context:** Laura's v0.22.0 friction in osprey-strike read as "pending scrambling," and the original plan was a first-parent-aware latest-entry selection (Phase 1). An independent reviewer in a separate context caught that as a regression for the common "entry on feature branch then merge" workflow — `TestBranchMerge_EntryOnBranch_NoneOnMain` already encoded the contract that would break. Their critique was specific: "Phase 1 is wrong; drop it" with the exact counterexample.

**Decision:** Ship Phase 0 as pure diagnostic — `IsOnFirstParentLine` does a bounded first-parent walk, `LatestAnchorOffFirstParent` disambiguates from stale-anchor, and `pending` and `doctor` wire the diagnostics into JSON, human, and check surfaces. A new integration test codifies the regression contract for the Laura pathology; current code already passes it, which is the empirical signal that Phase 1's algorithm change may not be needed at all — the friction was opacity, not incorrectness. Phase 1 is reconsidered as anchor-set termination rather than anchor selection.

**Consequences:**
- Soak time on real diagnostic data before any algorithm change — avoids a regression to a working solo-dev-feature-branch flow.
- The contract test passing on unchanged code becomes the gating signal for whether Phase 1 ships at all.
- Operators get topology visibility and pointers to existing escape hatches in the meantime, without a behavior change.
- Defers the actual fix if the friction turns out to require an algorithm change; the diagnostic alone won't resolve cases where the current behavior is genuinely wrong.

---

## ADR-4: Author-skip rules live inside `.timbersignore`, not a dedicated file

**Status:** Accepted
**Date:** 2026-05-20

**Context:** The impending q-redshifted autofix pipeline needed a bot-author skip path. Initial implementation added a dedicated `.timbers/skip-authors` file. The user pushed back on file sprawl during review.

**Decision:** Fold author globs into `.timbersignore` via an `author:<glob>` line prefix parsed by `classifyTimbersIgnoreLine`, yielding both path-rule and author-rule shapes from one parser. Mirrors the `.gitignore` family idiom and gives a single source of truth for repo skip config.

**Consequences:**
- One file to inspect and document for all skip configuration.
- Collides with literal paths starting with `author:` — accepted because `:` is forbidden in Windows filenames, making the collision vanishingly rare.
- Malformed globs silently drop (consistent with existing path-rule behavior), which a reviewer initially mistook for a missing edge case.
- `filepath.Match`'s character-class semantics force a prefix-wildcard workaround for the common GitHub-bot pattern, documented in `docs/agent-dx-guide.md`.

---

## ADR-5: Add `timbers ack` for honest skip-with-reason

**Status:** Accepted
**Date:** 2026-05-20

**Context:** Operators hitting pending commits they shouldn't document had only two paths: fabricate an entry or bypass the gate with `--no-verify`. Neither produces an honest record.

**Decision:** Add `timbers ack` storing entries under `.timbers/YYYY/MM/DD/ack_*.json` with `kind=ack` under `timbers.devlog/v1`. `AckedSet` threads through `filterCommits` parallel to `docSet`, sharing a single scan per pending check.

**Consequences:**
- Operators get a third path: explicit skip with a reason that's preserved in the ledger.
- Acks share the same date-partitioned storage shape as entries — no new directory schema.
- The single-scan threading avoids a double-walk over commits that an earlier draft introduced (reviewer caught it).
- Adds a second `kind` value the storage layer must handle; future ledger consumers need to filter accordingly.

---

## ADR-6: Gate the post-commit hook through `hasActionablePending`

**Status:** Accepted
**Date:** 2026-05-10

**Context:** Pending, log, and the post-commit hook were diverging on what counts as undocumented work — `runPostCommitHook` never checked `HasPendingCommits` before nudging. Real bug surfaced in gorewood/vellum, where `.beads/issues.jsonl`-only commits triggered the nudge for a commit `timbers log` would have refused. A narrower fix (inline the gate in `runPostCommitHook` only) was considered.

**Decision:** Extract `hasActionablePending()` combining `IsRepo`, `IsInteractiveGitOp`, `.timbers/` existence, storage construction, and `HasPendingCommits`. Both pre-commit and post-commit hooks route through it. A parity test asserts both hooks agree.

**Consequences:**
- Eliminates the divergence root cause — any future hook (post-rewrite, etc.) inherits the same gate definition.
- The parity test reads as one assertion instead of duplicating cases per hook.
- The test harness needed a `seedFile` escape hatch so `.timbersignore` can be baked into the initial commit; otherwise it becomes actionable itself and reorders the skipped-vendor case.
- One more function on the critical path of every commit; the helper's cost has to stay cheap.

---

## ADR-7: Drop the Dolt remote in favor of embedded JSONL transport only

**Status:** Accepted
**Date:** 2026-05-08

**Context:** A prior agent had skipped committing auto-staged `.beads/issues.jsonl` after misreading bd 1.0.x's one-time JSONL schema flip (records gained `_type` discriminator and reordering) as corruption. The Dolt remote had been unused since 2026-04-29 and `bd dolt push/pull` no-op without one anyway.

**Decision:** Remove the Dolt SQL remote (`origin → git+ssh://...gorewood/timbers.git`). Patch AGENTS.md sync-model bullet to call out embedded-only mode, instruct agents to commit bd 1.0.x's `_type`-prefixed JSONL rewrites without reverting, add a `bd export | diff` drift-recovery recipe, and drop `bd dolt push` from the session-close workflow.

**Consequences:**
- Single sync channel — JSONL committed via `git push` is the only path; eliminates the misread that triggered the previous agent confusion.
- Full reconstruction still works from the gitignored `.beads/dolt/` working DB plus the committed JSONL.
- A future Dolt remote can be re-added with one command if multi-machine federation is ever needed — the deletion is reversible.
- Loses the option of out-of-band Dolt-level sync; any future federation plan has to re-introduce and re-test the remote.

---

## ADR-8: Compact prime output by default, full guide behind `--full`

**Status:** Accepted
**Date:** 2026-05-08

**Context:** Default `timbers prime` injection was spending session-context tokens on repeated coaching that the agent had already internalized after the first few sessions.

**Decision:** Add a compact v2 renderer as the default, move the full guide behind `--full`/`guide` modes, update Claude hooks to use hook mode, and align stale-anchor prime output with pending.

**Consequences:**
- Smaller session-context payload at session start; coaching is still reachable on demand.
- Subsequent follow-ups exposed that compact had over-trimmed: full IDs (not ellipsis), a PRIME.md customization hint, honest JSON mode reporting, and health-width parity all had to be added back in a follow-up commit.
- Operators who customize PRIME.md see a one-line hint instead of having their content auto-merged into compact output — keeps compact tight at the cost of one indirection step.
- New consumers (MCP, JSON) inherit the `Mode` field; an earlier hardcoded value leaked into them and had to be fixed.

---

## ADR-9: ADR template requires Status/Date/supersession and operator-intent Context

**Status:** Accepted
**Date:** 2026-05-02

**Context:** The decision-log template (v1-v5) produced ADRs without scaffolding for reversals, so decision logs accumulated without any way to express "this was superseded by X." A structured `agent_involvement: high|medium|low` field on entries was also considered.

**Decision:** Add explicit Status field, Date semantics, and supersession handling to the decision-log template. Tighten Context to require operator intent or the constraint that drove the decision. Reject the `agent_involvement` field — too much taxonomy, encourages fabrication; the existing `notes` field handles it fine when used well.

**Consequences:**
- Decision logs now have first-class support for reversals and replacements.
- Context sections push past technical environment into operator intent, making cold-readers calibrate faster.
- The taxonomy stays minimal — `notes` remains the catchall for collaboration texture rather than a new schema field.
- Templates relying on the old shape need version bumps; downstream consumers must handle the new fields.

---

## ADR-10: Anti-fabrication clauses target soft content, not just hard claims

**Status:** Accepted
**Date:** 2026-05-02

**Context:** A codex second-opinion review surfaced a unifying failure mode across six templates — soft fabrication (emotions, themes, consequences, vague benefits) was the same risk as hard fabrication (invented test plans, invented metrics) just at lower visibility. Fabrication risk had been treated as binary.

**Decision:** Per-template anti-fabrication clauses for soft-content targets: devblog emotion anti-fabrication, ADR consequences tightening, release-notes soft-benefit tightening, sprint-report theme threshold, standup stale-next-steps. Reject the suggestion to weaken devblog's named-voice literary scaffolding (Atwood, Fowler, Orosz) — those voices do directional work the LLM reads as guidance, not mandatory mood; the right fix for thin-log overreach is the new "don't fabricate emotion when entries lack it" rule, not blanket softening.

**Consequences:**
- Every accepted patch is a hardened anti-fabrication clause for a specific soft-content target — consistent framing across templates.
- Operators get more honest output when entries are thin: empty sections instead of invented content.
- The named-voice scaffolding stays, which keeps the templates' aesthetic intent intact at the cost of being more opinionated than a neutral default.
- Versions bumped on templates with material changes; consumers pinning old versions don't get the hardening automatically.

---

## ADR-11: Coach PR-authoring as a prime workflow nudge, not a template change

**Status:** Accepted
**Date:** 2026-05-02

**Context:** Agents opening PRs without explicit body instructions defaulted to ad-hoc summaries that drift from session-documented intent. A more aggressive "always draft from ledger unless told otherwise" rule was considered.

**Decision:** Add a `<pr-authoring>` section to `defaultWorkflowContent` in `prime_workflow.go` with explicit when-this-applies and when-this-does-not lists, the recommended pipe-through-claude flow, and an empty-section signal — missing Design Decisions means thin entries, not a license to fabricate. Guidance lives in prime workflow (WHEN to invoke) rather than the pr-description template itself (HOW the template works).

**Consequences:**
- Operators who want one-line PR bodies for trivial work aren't overridden.
- The empty-section signal converts a fabrication risk into a feedback loop: thin entries surface in PR review instead of being papered over.
- Separation between "when to use a template" and "how a template works" stays clean; both can evolve independently.
- Agents that ignore prime workflow text still produce ad-hoc summaries — this is coaching, not enforcement.
