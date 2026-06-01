+++
title = 'Decision Log'
date = '2026-06-01'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

## ADR-1: Resolve off-first-parent anchor friction with diagnostics and targeted filtering, not an algorithm rewrite

**Status:** Accepted
**Date:** 2026-05-21

**Context:** A user (Laura) hit what read as "pending scrambling" on v0.22.0 when her work spanned a merge topology — `--batch` grouped a Work-item trailer's local and cross-branch commits, and the resulting entry anchored to a side-branch SHA, breaking the linear-anchor mental model downstream. The initial plan was to change the pending-detection algorithm to make latest-entry selection first-parent-aware. An independent reviewer (separate context) caught that plan as a regression: it would lose the anchor entirely for the common "entry on a feature branch, then merge" flow already exercised by `TestBranchMerge_EntryOnBranch_NoneOnMain`, with the exact counterexample cited as "Phase 1 is wrong; drop it."

**Decision:** Ship diagnostics first and prove the algorithm was already correct before touching it. Phase 0 added `IsOnFirstParentLine` / `LatestAnchorOffFirstParent` to surface the topology in `pending` and `doctor`, plus a contract test for Laura's pathology — which *passed on existing code*, the empirical signal that the friction was opacity, not incorrectness. The actual fixes were then narrow: `pickBatchAnchor` prefers the first commit on HEAD's first-parent line over `commits[0]` (falling back to `commits[0]` only for pure cross-agent debt), and `filterByRules` gained a direct `docSet` membership check alongside an `anchorShortCircuit` gate fallback for off-first-parent anchors. The reviewer reshaped the entire approach away from an algorithm change; a second reviewer traced the `--batch` line as the true culprit. Discovered while writing the test: the gate fallback alone returned unfiltered commits because `docSet` had only ever been consulted for revert detection, never for direct "is this documented" membership — both fixes were required together.

**Consequences:**
- Laura's topology class reaches zero pending without disturbing the validated feature-branch-then-merge workflow.
- `documented` becomes a distinct classify-reason in the `TIMBERS_DEBUG` trace and the auto-skip count.
- The planned anchor-selection rewrite was abandoned as unnecessary; the deferred question of anchor-*set* termination (vs. selection) was explicitly left open for real-volume evidence.
- Adds diagnostic surface area (`IsOnFirstParentLine`, doctor checks) that must be maintained even though no algorithm changed.

## ADR-2: Document the ack-for-rebase pattern over building content-matching relink

**Status:** Accepted
**Date:** 2026-05-27

**Context:** A 5-proposal feature request asked for automatic content-match relinking so a rebased commit (new SHA, same content) would stay "documented." The requester's preferred option (A) was auto content-matching via byte-identical diffs or reflog. Timbers has no content-matching today — `docSet` is literal SHAs — so option A would be built from scratch, and its naive form is unsound: rebases shift context lines, and reflog is absent on fresh clones and CI. A sound version requires storing `patch-id` at log time, i.e. a schema change.

**Decision:** Document the existing `ack` path rather than build new machinery — `ack` already produces a structured record that counts as documented, so it satisfies the rebase-relink requirement as-is. Added a `RebaseRelinkGuidance` protocol section (wired into both `prime` and MCP compose sites), a copy-pasteable ack reason form (`rebased; content in <entry-id>`), a DX-guide subsection, and a protocol test. Following Gall's Law, the requester's preference order was inverted to D→B→maybe-A: ship the doc now, graduate to a typed `ack --to <entry>` only if the free-text link proves fragile, and build `patch-id` gating only with real volume evidence.

**Consequences:**
- Turns a from-scratch feature into documentation of an existing capability — cheapest correct fix.
- Distinct from the existing stale-anchor guidance: there the anchor is GC'd and self-heals; here it stays reachable and does not, so the cases needed separate sections.
- Leaves the structured `ack --to` form and `patch-id`-based content matching explicitly deferred — the free-text link is unenforced and could drift if heavily used.

## ADR-3: Use a `msg:` subject-glob skip rule over path-based skipping in `.timbersignore`

**Status:** Accepted
**Date:** 2026-05-27

**Context:** Release changelog commits should not appear as pending, but they can't be cleanly path-skipped: `filterByRules` only skips a commit when *every* file matches a rule, and the release commit also touches `site/layouts/index.html` (the version badge). A path rule broad enough to skip the changelog would also hide legitimate landing-page edits to that file.

**Decision:** Add a third `.timbersignore` rule kind — `msg:` — that matches the commit subject, mirroring the existing `author:` glob design, and skip releases via `msg:chore: changelog for v*`. Subject-matching is the precise tool (it targets the commit, not its files) and generalizes to other housekeeping subjects like version bumps. The rule threads through `loadSkipConfig` → `Storage.skipMessages` → `classifyByIdentity` (reason `message`); `filterByRules` was refactored to reuse `classifyByIdentity`, which both fit the complexity budget and single-sourced skip semantics with the debug trace. Discovered mid-change: `.timbersignore` was being silently gitignored by the repo's allowlist `.gitignore`, which would have made any skip rule clone-local — fixed by allowlisting it.

**Consequences:**
- Housekeeping commits skip precisely without globally hiding files they happen to touch.
- The visibility path and the gate now share one classification chain — but that sharing must be kept intact: a later bug (`countAutoSkipped` left on a parallel inline identity chain) silently undercounted `msg:`-matched commits in `timbers status` precisely because the loops had drifted. The fix was re-converging them on `classifyByIdentity`, confirming single-sourcing is the invariant to defend, not a coincidence to preserve.
- `.timbersignore` is now committed, so every clone and CI reads the same config.

## ADR-4: Detect binary shadowing in `doctor` via version-token comparison

**Status:** Accepted
**Date:** 2026-05-27

**Context:** The git hook runs whatever `timbers` is first on `PATH`. During the v0.22.3 release, a `dev` `go install` binary in mise's GOBIN shadowed the current `~/.local/bin` build; the stale hook didn't recognize acks, so `timbers ack` wrote its record but the auto-commit was gate-blocked with a confusing `failed to commit ack file`. Surfacing the divergence proactively beats debugging that error after the fact.

**Decision:** Add a `doctor` "Binary Shadowing" check that enumerates `timbers` binaries on `PATH` (deduped by resolved path) and warns when the first reports a different version *token* than a shadowed one. Comparison is on the version token, not the full `--version` string, so two builds of the same version with differing commit/date suffixes don't false-warn; `?` tokens are skipped to avoid false positives on broken shims. `WriteAck`'s commit-failure path now explains the staged-but-uncommitted state and points at upgrade + `doctor` + `git commit` recovery.

**Consequences:**
- A silent, hard-to-diagnose failure mode (stale shadowing binary blocking commits) becomes a proactive warning.
- Token-only comparison trades exactness for precision — it won't flag two genuinely different builds that share a version token.
- Adds a `PATH`-walk cost to `doctor` and a maintenance dependency on the `--version` token format.

## ADR-5: `just release` installs the published artifact as a normal consumer

**Status:** Accepted
**Date:** 2026-05-27

**Context:** After tagging, the only thing keeping the local binary current was a manual follow-up, and nothing exercised the real consumer path (CI build → GitHub release → `install.sh`) — the identical local build had been installed instead, so a broken release pipeline could ship undetected.

**Decision:** Append a bounded poll loop to `just release` — `gh release view <tag>` on a ~10-minute deadline at 15s intervals — and on publish run `just install-release`, verifying the installed `--version` contains the tag. A cron follow-up was considered (the user's initial framing) but rejected: a poll-and-install at the tail of `just release` has no moving parts, and since CI can't install on the developer's laptop the step has to run locally anyway. Relies on `.goreleaser draft:false` so `latest` equals the just-pushed tag once assets appear; `install-release` stays the genuine consumer path rather than pinning `VERSION`.

**Consequences:**
- Every release sanity-checks the full publish-and-install pipeline as a side effect of cutting it.
- The local binary stays current automatically.
- On poll timeout the release is already pushed, so the step prints a recovery hint rather than failing — the verification is best-effort, not a gate.

## ADR-6: Make `just release` site-example regeneration non-fatal

**Status:** Accepted
**Date:** 2026-05-27

**Context:** Ancillary site-content generation runs parallel `claude -p` calls during a release. One of them (decision-log) flaked on a transient LLM hiccup and aborted an entire version release — a best-effort content step blocked shipping a tag.

**Decision:** Wrap `just examples` in the release recipe with `|| echo WARNING` so a failure warns and the release proceeds to commit/tag/push; examples regenerate separately. A transient LLM hiccup must not sit on the critical path of shipping a tag.

**Consequences:**
- Releases no longer hostage to flaky LLM content generation.
- Site examples can lag a release until regenerated separately — accepted as the cost of decoupling best-effort work from the release critical path.

## ADR-7: Honor `--anchor` at zero pending and name `--range` as the escape hatch

**Status:** Accepted
**Date:** 2026-05-28

**Context:** The `--anchor` flag's name promises "use this anchor," but at zero detected pending `timbers log` still refused — leaving no documented path to log when the anchor sat off the first-parent line (the residual gap from the off-first-parent work in ADR-1). A bare `No pending commits` from `timbers pending` likewise conflated "clean" with "computed from an off-line anchor."

**Decision:** Make `getLogCommits` fall back to `LogRange(anchor^, anchor)` — documenting a single commit — when pending is empty and `--anchor` is set, honoring the flag's named promise. The refusal message now names `--range` as the explicit escape hatch, and `timbers pending` at count 0 prints a note keyed on `AnchorOffFirstParentLine` so the off-line situation is named rather than masked as "clean." Commit-resolution helpers were extracted to `log_resolve.go` to stay under the file-length limit.

**Consequences:**
- Closes the off-first-parent / zero-pending gap with a predictable flag contract — `--anchor` means what it says.
- Users get a named situation plus two pointers (`--explain`, `--range`) instead of an ambiguous "clean" report.
- Adds a single-commit-range code path and a ref-aware mock that must be kept aligned with the gate.

## ADR-8: Refuse `timbers log` on a dirty tree and explain vanished commits at the gate

**Status:** Accepted
**Date:** 2026-06-01

**Context:** A field report from an agent in Constructured/osprey-strike observed two phantom ledger entries in one v0.22.7 session. The failure path: an aborted gated commit leaves staged changes in the index; the agent then runs `timbers log`, which auto-commits an entry scoped to the entry file only — so the entry rides the old HEAD while the feature work stays unstaged. v0.22.7 merely warned-and-proceeded, and the pre-commit hook's refusal never explained *why* the caller's commit had vanished.

**Decision:** Replace the dirty-tree `printer.Warn` in `log.go` with a `UserError` guarded by `!flags.dryRun` (so `--dry-run` stays usable for mid-debug inspection); the error names the likely trigger (aborted gated commit), the diagnostic (`git diff --cached`), and the `--dry-run` escape. Option B from the report (create the entry but skip auto-commit) was rejected: since the trigger is the gate *aborting* commits, the entry would sit dirty indefinitely and dirty entries compound. A `--allow-dirty` flag was deliberately *not* added — the protocol has no case for logging uncommitted work, and the flag would reopen the exact footgun. As a complementary surface, the pre-commit hook now prints a "staged changes remain in the index" hint via `HasStagedChanges()` (`git diff --cached --name-only`), gated so it stays silent on a clean index — an unconditional hint was rejected because it would mislead a `git commit --amend --no-edit` against an undocumented HEAD.

**Consequences:**
- The tool now enforces the commit-first ordering its warning had only requested, and no entry files are created on the refusal path.
- `--dry-run` remains the sanctioned way to inspect an entry on a dirty tree; there is no way to log uncommitted work, by design.
- The hook hint costs one extra `git` call per invocation (judged within budget) and short-circuits the "where did my commit go?" confusion before the caller reaches the now-refusing `timbers log`.
- Companion beads (`timbers-cwa`, `timbers-cs0`) capture deferred work on auto-detecting cross-agent debt in the gate.
