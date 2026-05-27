+++
title = 'Decision Log'
date = '2026-05-27'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

## ADR-1: Gate the post-commit hook on actionable pending work via a shared helper

**Status:** Accepted
**Date:** 2026-05-10

**Context:** A real bug surfaced in gorewood/vellum where commits touching only `.beads/issues.jsonl` triggered the post-commit hook's "undocumented work" nudge — even though `timbers log` would refuse to log them. `pending`, `log`, and the post-commit hook each carried their own notion of what counts as undocumented, and the divergence produced contradictory signals: the hook nudged the agent to log a commit the logger itself wouldn't accept. Root cause: `runPostCommitHook` never checked `HasPendingCommits` before printing.

**Decision:** Extract a `hasActionablePending()` helper combining `IsRepo`, `IsInteractiveGitOp`, `.timbers/` existence, storage construction, and `HasPendingCommits`, and route both pre-commit and post-commit hooks through it. The narrower fix (inline the `HasPendingCommits` gate in `runPostCommitHook` only) was rejected because the shared helper eliminates the divergence at its root — any future hook (e.g. post-rewrite) inherits the same gate for free, and the parity test reads as a single assertion.

**Consequences:**
- All hooks share one definition of "actionable"; contradictory signals are eliminated and new hooks get correct gating by default.
- The test harness needed a `seedFile` escape hatch so `.timbersignore` can be baked into the initial commit — otherwise adding it as a separate commit makes that commit itself actionable.
- Any future change to what counts as actionable now propagates to every hook through one helper.

## ADR-2: Scope the commit gate to the first-parent line over author attribution

**Status:** Accepted
**Date:** 2026-05-18

**Context:** Parallel agents share a `.timbers/` ledger through merges. The existing gate walked the full commit DAG, so agent A's commit was blocked by agent B's undocumented commits — the gate fired on the wrong actor. The feature request proposed author-based attribution as its first option.

**Decision:** Scope the gate to the first-parent line (`LogFirstParent`) rather than author attribution. Author filtering was rejected because all agents in the target setup commit under the same git identity, so an author filter would no-op; first-parent is git-native and works regardless of identity config. A regression test then exposed a residual case — `git merge --no-ff` puts a merge commit M on the first-parent line, and M still blocked — so a gate-only `dropEmptyFileChanges` filter was added: clean merges and `--allow-empty` commits have empty file lists and add no work to this branch's first-parent line. `TIMBERS_SKIP_CROSS_AGENT_DEBT` remains as an escape hatch for the narrow case where a merge commit itself touched source (conflict resolution).

**Consequences:**
- The gate fires on the committing branch's own work rather than merged-in debt, and works without per-agent identity configuration.
- The display path (`timbers pending`) deliberately keeps the conservative "empty = unknown" rule so merge SHAs still surface for awareness — gate and display now treat empty merges asymmetrically by design.
- Reviewer flagged misleading env-var docs and a contradictory test name; both corrected before commit.
- The env-var escape hatch is a blunt instrument for conflict-resolution merges that touch source — no finer-grained handling of that case.

## ADR-3: Fold author-skip globs into .timbersignore over a dedicated skip-authors file

**Status:** Accepted
**Date:** 2026-05-20

**Context:** The impending q-redshifted autofix pipeline needed a way to skip bot-authored commits in pending detection. The first implementation added a dedicated `.timbers/skip-authors` file.

**Decision:** After the user pushed back on file sprawl, the skip-author config was folded into `.timbersignore` via an `author:<glob>` line prefix, parsed by `classifyTimbersIgnoreLine` (the same parser yields both path and author rule shapes). This mirrors the `.gitignore` family convention and gives a single source of truth for repo skip config.

**Consequences:**
- One file holds all repo skip config, idiomatic to anyone familiar with `.gitignore`.
- Edge case: the `author:` prefix collides with literal paths beginning `author:` — extremely rare, since `:` is forbidden in Windows filenames.
- `filepath.Match` character-class semantics mean GitHub-bot matching needs a prefix-wildcard workaround (documented in the DX guide).

## ADR-4: timbers ack — record an honest skip-with-reason over fabrication or --no-verify

**Status:** Accepted
**Date:** 2026-05-20

**Context:** Laura's osprey-strike friction report showed merge SHAs sitting in `pending` with no obvious next action. The only existing ways to clear them were to fabricate a ledger entry or bypass the hook with `--no-verify` — both dishonest about what actually happened.

**Decision:** Add a `timbers ack` command that records a structured skip-with-reason: an `ack_*.json` record under `.timbers/YYYY/MM/DD/` with `kind="ack"` under the `timbers.devlog/v1` schema. `AckedSet` is threaded through `filterCommits` parallel to `docSet`, in the same single scan per pending check.

**Consequences:**
- A commit can be honestly marked "intentionally not documented, here's why" without fabricating work or disabling the gate.
- Ack records are first-class ledger artifacts (same schema and versioning), so they sync and audit like entries.
- Reviewer caught an `AckedSet` double-scan and a dead `pushedMsg` parameter on round 1, then a checklist-ordering test gap and an author-trace error on round 2.
- This mechanism later became the foundation for the rebase-relink workflow (see ADR-7).

## ADR-5: Compose protocol text from const fragments over a single-file blob

**Status:** Accepted
**Date:** 2026-05-20

**Context:** A push-before-log race in osprey-strike stranded an entry locally. Fixing it required rewriting the workflow protocol text — and that text is consumed by two different surfaces: the full PRIME doc (`cmd/timbers`) and a subset shown via MCP (`internal/mcp`).

**Decision:** Extract the shared protocol and stale-anchor sections into an `internal/protocol` package as Go string consts, composed by each consumer from the fragments it needs. A single-file approach was considered (inspired by the `.timbersignore` single-source decision in ADR-3) but rejected: the workflow text is fundamentally compositional — different consumers need different subsets. Const+const concatenation gives compile-time composition with no runtime concat overhead.

**Consequences:**
- One source of truth for protocol prose, with each surface assembling its own view; no duplicated text drifting between PRIME and MCP.
- Enables later additions (e.g. the rebase-relink section in ADR-7) to be written once and wired into both compose sites.
- A prose-only protocol test would miss a reordered checklist, so an explicit ordering-position assertion was added after the reviewer flagged the gap.

## ADR-6: Pending detection for merge topology — surgical fixes over an algorithm rewrite

**Status:** Accepted
**Date:** 2026-05-21

**Context:** Laura's v0.22.0 friction read as "pending scrambling" against side-branch merges, and the initial plan was to rewrite the pending-detection algorithm to be first-parent-aware (the "Phase 1" change).

**Decision:** An independent reviewer (separate context) caught that plan as a regression — a first-parent-aware "latest entry" would lose the anchor entirely for the common "entry on a feature branch, then merge" workflow already exercised by `TestBranchMerge_EntryOnBranch_NoneOnMain` — and supplied the exact counterexample. The course-correction shipped a diagnostic-only Phase 0 instead: an `IsOnFirstParentLine` walk plus a `LatestAnchorOffFirstParent` signal wired into `pending` and `doctor`, with a contract test capturing Laura's pathology. That test passed on the *unchanged* algorithm — the empirical signal that the friction was opacity, not incorrectness. The real root causes then surfaced: (1) `--batch` grouped a Work-item trailer's local + cross-branch commits and naively picked `commits[0]`, sometimes a side-branch SHA — fixed by `pickBatchAnchor`, which returns the first commit on HEAD's first-parent line and falls back to `commits[0]` only for pure cross-agent debt; (2) `filterByRules` used `docSet` only for revert detection, never for direct "is this commit documented" membership — adding that membership check, plus an off-first-parent gate fallback routing through `CommitsReachableFrom` + `filterCommits`, drove Laura's class to zero pending. Writing the membership test was what exposed the missing filter — the reviewer's "route through all-reachable + docSet" recommendation had assumed a filter that didn't yet exist. The planned algorithm change was abandoned as unnecessary.

**Consequences:**
- The merge-topology friction class is resolved with small targeted fixes and no algorithm-level change; the docSet algorithm was confirmed correct.
- `pickBatchAnchor` resolves HEAD best-effort via `git.HEAD()` and degrades to legacy `commits[0]` behavior on git error rather than failing the batch run.
- The Phase 0 diagnostic still ships value: it flags the residual pure-cross-agent-debt case where `pickBatchAnchor` falls back to `commits[0]`.
- "documented" is now a distinct classify-reason in the `TIMBERS_DEBUG` trace and `countAutoSkipped` count.
- Both fixes were required together — the gate fallback alone returned the unfiltered reachable list because direct docSet membership wasn't yet a thing.

## ADR-7: Document ack-for-rebase over building content-matching relink

**Status:** Accepted
**Date:** 2026-05-27

**Context:** A five-proposal feature request asked for a way to relink ledger entries after a rebase rewrites commit SHAs — entries reference literal SHAs in their docSet, which a rebase invalidates. The requester's preferred option (A) was automatic content-matching relink.

**Decision:** Ship documentation of the existing `ack` path instead of building anything: `ack` already produces a structured record that counts as "documented," so it satisfies the rebase-relink requirement at zero new code. The requester's preference order was inverted to D→B→maybe-A on Gall's Law grounds. Timbers has no content-matching today (docSet is literal SHAs), and the naive byte-identical-diff/reflog form of (A) is unsound — rebases shift context lines, and reflog is absent on fresh clones and CI; a sound content-match needs a patch-id stored at log time, which is a schema change. So: document the `ack` workflow now (a new `RebaseRelinkGuidance` const wired into both PRIME and MCP, copy-pasteable `rebased; content in <entry-id>` ack reasons, locked by a protocol test); graduate to a typed `ack --to <entry>` only if the free-text link proves fragile; build patch-id gating only with real volume evidence.

**Consequences:**
- The rebase-relink need is met immediately with no schema change and no unsound heuristic.
- This case is deliberately distinct from the existing stale-anchor guidance: there the anchor is GC'd and self-heals; here the rewritten commit stays reachable and does not self-heal, so explicit acknowledgment is required.
- Typed `ack --to <entry>` and patch-id-based content matching are explicitly postponed until evidence justifies them — open questions, not committed work.
- Relies on the free-text ack reason as the human-readable link; if that proves error-prone, the typed form is the next step.

## ADR-8: Skip release housekeeping commits via msg: subject globs over path-based rules

**Status:** Accepted
**Date:** 2026-05-27

**Context:** Release changelog commits were cluttering pending detection and should be auto-skipped. The obvious approach was a path-based `.timbersignore` rule matching `CHANGELOG.md` plus `site/`.

**Decision:** Add a third `.timbersignore` line kind, `msg:<glob>`, matching against the commit subject, and skip release commits via `msg:chore: changelog for v*`. Path-based skip was rejected because `filterByRules` only skips a commit when *every* file matches a rule — and the release commit also touches `site/layouts/index.html` (the version badge), so a path rule broad enough to skip the release would also hide legitimate landing-page edits. A subject glob skips the commit precisely and generalizes to other housekeeping subjects (version bumps, release commits). The `msg:` parser mirrors the existing `author:` glob design (ADR-3), threading through `loadSkipConfig`/`Storage.skipMessages` into `classifyByIdentity` with reason `"message"`.

**Consequences:**
- Release commits skip precisely without suppressing real edits to files they happen to touch, and the rule generalizes to other housekeeping subjects.
- Refactoring `filterByRules` to reuse `classifyByIdentity` both flattened the loop under the complexity budget and single-sourced skip semantics with the debug trace.
- `.timbersignore` was discovered to be gitignored by the repo's allowlist `.gitignore`, which would have silently made the skip clone-local — fixed by allowlisting it so every clone and CI reads the same config.

## ADR-9: Detect binary shadowing in doctor via version-token comparison

**Status:** Accepted
**Date:** 2026-05-27

**Context:** During the v0.22.3 release, a stale `dev` go-install binary in mise's GOBIN shadowed the current `~/.local/bin` build on PATH. The git hook runs whichever `timbers` is first on PATH, so the stale binary silently blocked commits by not recognizing `ack` records — surfacing only as a confusing "failed to commit ack file."

**Decision:** Add a `doctor` "Binary Shadowing" check that enumerates `timbers` binaries on PATH (deduped by resolved path) and warns when the first reports a different version than a shadowed one. Comparison is on the version *token* rather than the full `--version` string, so two builds of the same version with different commit/date suffixes don't false-warn; `?` tokens are skipped to avoid false positives on broken shims. `WriteAck`'s commit-failure path was also rewritten to explain the staged-but-uncommitted state and point at upgrade + doctor + `git commit` recovery.

**Consequences:**
- PATH divergence is surfaced proactively instead of being debugged from a cryptic ack-commit failure.
- Token-level comparison trades some sensitivity (it won't flag same-version-but-different-build shadowing) for zero false positives on legitimate dual builds.
- Both the pass path (same version) and the warn path (a fake older binary prepended to PATH) were verified.

## ADR-10: Poll-and-install the published release artifact over a cron follow-up

**Status:** Accepted
**Date:** 2026-05-27

**Context:** After `just release` tags and pushes, the official binary is published asynchronously by GitHub Actions. The dev's local binary stays stale until a manual follow-up, and nothing exercises the published artifact end to end. The user's initial framing was a cron follow-up.

**Decision:** Append a bounded poll loop to the tail of `just release` (`gh release view <tag>`, ~10m deadline, 15s interval); once assets appear, run `just install-release` and verify the installed `--version` contains the tag. The cron follow-up was rejected as having more moving parts — CI can't install on the dev's laptop, so the install has to be local anyway, and a poll-and-install at the tail is simpler. `install-release` is used rather than pinning VERSION, deliberately, so the release is exercised "as a normal consumer" per the user's intent — sanity-checking the whole pipeline (CI build, GH release, `install.sh`). Relies on `.goreleaser draft:false` so `latest` equals the just-pushed tag once assets appear.

**Consequences:**
- Every release self-tests its own publish and install path as a real consumer would, and the local binary stays current without a manual step.
- On poll timeout, a recovery hint is printed since the tag-and-push already succeeded — only the local install is incomplete.
- `just release` now blocks for up to ~10 minutes waiting on GitHub Actions rather than returning immediately after push.
