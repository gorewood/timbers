+++
title = 'Decision Log'
date = '2026-06-01'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

## ADR-1: Poll the GitHub release then install the published artifact as a normal consumer

**Status:** Accepted
**Date:** 2026-05-27

**Context:** The `just release` recipe tagged and pushed, but the dev's local binary always came from an identical *local* build — so the actual release pipeline (CI build, GH release, `install.sh`) went unverified on every cut. The user framed the fix as wanting the recipe to install "as if a normal consumer."

**Decision:** Append a bounded poll loop (`gh release view <tag> --jq assets length`, ~10m deadline, 15s interval) after `git push`; once assets appear, run `just install-release` and assert the installed `--version` contains the tag. A cron follow-up (the user's initial framing) was rejected — CI can't install on the dev's laptop so it has to be local anyway, and a poll-and-install at the tail of the recipe has no moving parts. Relies on `.goreleaser draft:false` so `latest` equals the just-pushed tag.

**Consequences:**
- Every release sanity-checks the full publish pipeline and leaves the local binary current with no manual follow-up.
- Adds up to ~10m of polling to `just release`; on timeout it prints a recovery hint since the release is already pushed.
- Verification is local-only — a consumer install on a *different* platform remains unverified.

## ADR-2: Migrate beads from server-mode Dolt + JSONL-in-git to embedded Dolt + canonical refs/dolt/data

**Status:** Accepted
**Date:** 2026-05-27

**Context:** The repo sat in a broken hybrid: server-mode Dolt couldn't persist under the sandbox (it re-imported JSONL on every call), and a stale CLI remote plus a 3-week-old `refs/dolt/data` existed while the steering docs claimed "no remote." JSONL was the only current source of truth (server Dolt last modified Mar 22; `refs/dolt/data` topped out May 7), so the state was incoherent and partly silently broken.

**Decision:** Align to the canonical embedded model already validated in osprey-strike. Stop/kill the server, set aside the stale DB and backup dir, flip `metadata.json` to embedded, `bd bootstrap` to rebuild `embeddeddolt` from the trusted 60-issue JSONL, drop and re-seed `refs/dolt/data` fresh, flip `export.auto`/`git-add` to false, and untrack + gitignore `issues.jsonl`. Rebuild-from-JSONL was judged safe precisely because JSONL was the freshest state.

**Consequences:**
- Sync works under the sandbox (no localhost TCP); clones inherit the mode via committed `metadata.json`/`config.yaml` and onboard via `bd bootstrap`.
- Dolt audit history was lost in the rebuild — accepted, since the prior Dolt state was already stale.
- Bootstrap detection order is load-bearing: `refs/dolt/data` had to be dropped *and* `.beads/backup` set aside so bootstrap fell through to current JSONL instead of cloning stale state.
- Hand-edits to the Sync/Landing sections landed inside bd's managed markers (tracked separately for relocation).

## ADR-3: Make .timbersignore discoverable and lint the `author:dependabot[bot]` glob footgun

**Status:** Accepted
**Date:** 2026-05-27

**Context:** An agent had to research whether author-matching even existed, and could silently write `author:dependabot[bot]` — a no-op glob, because `[..]` is a character class, not a literal. The exemption lever was invisible and the footgun failed silently.

**Decision:** Surface the lever across four surfaces (a `pending` hint when `.timbersignore` is non-empty, `pending --explain`, a `help timbersignore` topic, and an onboard blurb), and add a doctor lint that flags literal-looking `[..]` classes in `author:`/`msg:` globs with the canonical bot recipe. Extracted a shared `pendingRange()` so the filtering gate and the new `--explain` classifier resolve the range through one path instead of duplicating the side-branch short-circuits.

**Consequences:**
- Turns a research-and-trap into a one-liner; doctor catches the no-op glob before it silently fails to filter.
- Four beads landed in one commit (a cohesive `.timbersignore`-UX theme; `pending.go` carried two of them and a clean split would have needed `git add -p`).
- Shared `pendingRange()` means the gate and `--explain` cannot drift on range resolution; the 40K-line gate test suite is the regression net.

## ADR-4: Make the just-release site-example regeneration non-fatal

**Status:** Accepted
**Date:** 2026-05-27

**Context:** Site-content generation runs parallel `claude -p` calls during release. A transient LLM flake on the decision-log example aborted an entire version release — a publish step being held hostage by an ancillary content step.

**Decision:** Wrap `just examples` in the release recipe with `|| echo WARNING` so a failure warns and the release proceeds to commit/tag/push; examples regenerate separately.

**Consequences:**
- A transient LLM hiccup can no longer block shipping a tag.
- A released version may briefly ship with stale site examples until the separate regeneration runs.

## ADR-5: Honor `--anchor` at zero pending and name the off-first-parent state

**Status:** Accepted
**Date:** 2026-05-28

**Context:** osprey hit a gap when the workset anchor sat off the first-parent line: `timbers pending` reported a bare "No pending commits" (conflating *clean* with *computed from an off-line anchor*), and `timbers log --anchor` still refused at zero detected pending — despite the flag's name promising "use this anchor."

**Decision:** `getLogCommits` falls back to `LogRange(anchor^, anchor)` (a single-commit range) when pending is empty and `--anchor` is set; the refusal now names `--range` as the explicit escape hatch; and `pending` at count:0 prints a note keyed on `AnchorOffFirstParentLine`. Commit-resolution helpers were extracted to `log_resolve.go` for the file-length limit.

**Consequences:**
- `--anchor` now does what its name promises even at zero pending, and the off-first-parent situation is named rather than left ambiguous.
- A ref-aware mock and `TestLogAnchorBypassesZeroPending` lock the behavior in.

## ADR-6: Converge `countAutoSkipped` onto the shared identity classifier

**Status:** Accepted
**Date:** 2026-05-28

**Context:** `timbers status` undercounted housekeeping-skipped commits when a `msg:` glob matched — `countAutoSkipped` ran a parallel inline identity chain that was never updated for the new `msg:` rule, so status silently misreported whether a freshly-added `msg:` rule was filtering anything. Both arch and code reviewers caught it independently in the deep review pass.

**Decision:** Delegate identity classification to `classifyByIdentity` (the same chain `filterByRules` already uses) so visibility and the gate share one source of truth. Keeping a separate loop for symmetry with the gate's `filterByRules`/`ExplainPending` split was considered and rejected — the bug *was* the loops drifting, so re-converging is the fix, not a separation to preserve.

**Consequences:**
- Status counts and gate decisions can no longer diverge on identity rules.
- One fewer parallel classification path to keep in sync as new rule types are added.

## ADR-7: Close the gate-abort phantom-entry footgun — refuse dirty-tree log, then hint staged changes

**Status:** Accepted
**Date:** 2026-06-01

**Context:** A field report from an agent in osprey-strike observed two phantom ledger entries in one v0.22.7 session. The mechanism: an aborted gated commit leaves staged changes in the index, then `timbers log` auto-commits an entry pathspec-scoped to the entry file only — the entry rides the old HEAD while the feature work stays unstaged. v0.22.7 only warned-and-proceeded, and the gate refusal itself never explained why the caller's commit had vanished.

**Decision:** Two moves on the same footgun. First (v0.22.8), replace the dirty-tree `printer.Warn` with a `UserError` return guarded by `!flags.dryRun`; deliberately *not* add `--allow-dirty`, since the protocol has no case for logging uncommitted work and the flag would reopen the exact hole. The report's Option B (create entry but skip auto-commit) was rejected — the trigger is the gate *aborting*, so a dirty entry would sit indefinitely and compound. Second, add a `HasStagedChanges()`-gated hint after gate abort that names the trigger and points at `git diff --cached`, and reframe "Run timbers log first" → "Document the PRIOR commit(s) first." An unconditional hint was rejected because "staged changes remain" would mislead a `git commit --amend --no-edit` against an undocumented HEAD.

**Consequences:**
- Phantom entries can no longer reach upstream, and the confusing post-abort state is explained before the caller reaches the now-refusing `timbers log`.
- `--dry-run` remains usable for inspecting an entry mid-debugging.
- One extra `git diff --cached` call per hook invocation (within budget); the hint stays silent on a clean index.
- No path exists to log against a deliberately dirty tree — accepted, since the protocol has no such case.

## ADR-8: Reframe the gate as in-session reasoning capture, lenient about foreign and stale work

**Status:** Accepted
**Date:** 2026-06-01

**Context:** The user reframed the gate's purpose — it exists to capture *live, in-session* reasoning, accepting misses outside the session — but the existing gate didn't distinguish, so it fired on foreign-author and stale commits that can't be productively documented. timbers-vlh was bumped P3→P1 to make the gate provenance-aware.

**Decision:** Author a plan reviewed by three independent Opus subagents with focused lenses. The pragmatist cut four items (status breakdown, default footer, `--include-foreign`, the session-author directive); the correctness-skeptic surfaced the load-bearing AuthorDate-vs-CommitDate issue; the agent-UX advocate identified the `pending.count` semantic flip as the single most important UX decision, pushed the default window 4h→24h, and flagged the silent same-author stale-skip as the unrecoverable failure mode. Explicit tensions were resolved on the record: kept the `.timbersignore` session-window directive (the rebase finding makes window correctness load-bearing); moved the stale-self warning *off* the gate to a non-blocking post-commit note (a gate warning would defeat the leniency reframe); cut `--include-foreign` entirely (a backfill workflow with no real user yet).

**Consequences:**
- Establishes the contract every downstream v0.23.0 phase implements; the rationale is captured in a Decided-in-Review section so it's durable.
- Accepts by design that the gate will *miss* out-of-session work rather than nag about it.
- Defers `--include-foreign` — an open question if a backfill consumer ever materializes.

## ADR-9: Use mailmap-resolved author email (`%aE`) and CommitDate, grounded by a real-repo tally

**Status:** Accepted
**Date:** 2026-06-01

**Context:** Before locking heuristic defaults, the plan required an empirical tally; the user pointed at osprey-strike (multi-author, bot-heavy, the origin of most timbers feature work) as the best signal. The provenance classifier needed both an author identity and a staleness clock, and the naive choices were both wrong.

**Decision:** Use `%aE` (mailmap-resolved), not `%ae` — osprey-strike's `.mailmap` coalesces the operator's two emails, and `%ae` would have produced a **61% silent false-negative rate** on the operator's own work; `.mailmap` is git's canonical multi-email mechanism and gets the right answer for free (this also killed the planned session-author directive as redundant). Use `CommitDate` (`%ct`), not `AuthorDate`, for staleness — AuthorDate stays put across rebase/amend, so it would silently auto-skip work the user just touched. Implemented by keeping the existing `Date` field as `AuthorDate` (backward compat) and adding `CommitDate` as a sibling; diffstat code was extracted to `diffstat.go` for the 350-line limit. The tally also confirmed the 24h window, OR composition, and strict mailmap-equality.

**Consequences:**
- A multi-email operator is no longer misread as a foreign author, and recently-rebased in-session work is no longer mis-skipped as stale.
- `%aE` requires the operator to maintain a `.mailmap` for the multi-email case — repos without one fall back to raw author email.
- The session-author directive was dropped as superseded by mailmap.

## ADR-10: Land the provenance classifier as the last step in the skip chain, configured on Storage

**Status:** Accepted
**Date:** 2026-06-01

**Context:** Phase 3 of v0.23.0 — the classifier had to become part of `classifyCommit` *without* yet being wired into the gate decision (that came in phase 5), so precedence and safe-degradation could be locked in by tests independent of the gate-path wiring.

**Decision:** Place `classifyByProvenance` at the END of the chain (after infra → identity → content) so a documented or acked commit keeps its decision-relevant reason rather than relabeling as foreign-author just because the email differs. Passing `ProvenanceConfig` as a parameter to `classifyCommit` was rejected — it bloats the signature across `filterByRules`, `classifyCommit`, `traceFilterDecisions`, and `ExplainPending`, and the config shares the Storage's per-repo lifecycle, so it lives as a `provenance` field (zero value = disabled). Deliberately no clock-injection pattern: `ProvenanceConfig.Now` is a plain `time.Time` callers populate (`time.Now()` in production, a fixed instant in tests).

**Consequences:**
- Earlier-wins precedence is verified by tests against a foreign+stale commit before any gate wiring exists; the change is observable only via `--explain` until phase 5.
- Provenance config is per-Storage, not global — matches repo lifecycle but means every Storage must be configured to enable it.
- Tests get reproducibility from a fixed `Now` with no clock plumbing.

## ADR-11: Split `NewStorage` (raw) from `NewDefaultStorage` (production-wired) to keep host config out of tests

**Status:** Accepted
**Date:** 2026-06-01

**Context:** Wiring provenance into `NewStorage` made it load the host's `user.email` at construction, so every test that built mock commits without a matching `AuthorEmail` was falsely classified as foreign-author — three test suites failed immediately on `just check` (the host leaked `robert.bergman@gmail.com`).

**Decision:** Two iterations. The first attempt zeroed out `newTestStorage`'s provenance after construction — but that only covered the ledger package's tests. The final shape: `NewStorage` no longer auto-loads provenance (returns a zero-config Storage), `NewDefaultStorage` is the production entry point that calls `LoadProvenanceConfig`, and `Storage.SetProvenance` is the public way for tests and external callers to pin config. A `NewStorageForTest` variant was rejected because Storage already has multiple test entry points (mcp, cmd/timbers, ledger) and a new variant would force changes in 10+ places; the raw/production split mirrors an existing pattern and minimizes churn.

**Consequences:**
- Tests no longer inherit host git config; provenance is explicit per Storage (matching v0.22.7's skipMessages pattern).
- Production callers must use `NewDefaultStorage` to get provenance — a raw `NewStorage` silently disables it.
- A doctor check warns when `user.email` is unset (the safe-degradation contract: `ConfigUserEmail()` returns empty on any error).

## ADR-12: Add a per-repo `.timbersignore` session-window directive, coercing invalid values to the default

**Status:** Accepted
**Date:** 2026-06-01

**Context:** The 24h staleness default is safe per the tally, but the agent-UX review flagged long-running sessions (orchestrator + subagent fanouts running 4–10h, all-day refactors) that would exceed it and silently stale-skip their own work. The plan review carried a standing tension — the pragmatist wanted to cut the directive; the agent-UX advocate wanted per-repo tuning.

**Decision:** Keep the directive as opt-in (repos that don't set it get 24h). Parse it via a separate `LoadSessionWindow(root)` pass over `.timbersignore`, isolated from `loadSkipConfig` to avoid threading a 4-tuple→5-tuple change through 10+ call sites that destructure it (the file is tiny, so dual-parse cost is negligible). A unified struct return from `readTimbersIgnore` was rejected because the existing parser is well-tested and the dual-pass is the smallest viable change. Coerce negative or zero durations to the default rather than treating them as parse errors, because Go's `time.ParseDuration` accepts negatives as valid `Duration` values and diverging would break the documented grammar. A `checkSessionWindow` doctor check warns on malformed values.

**Consequences:**
- Long-session repos can extend the window; default repos are unaffected.
- The well-tested 4-tuple parser stays untouched, at the cost of a second parse pass over the file.
- Malformed/negative/zero values degrade safely to 24h (with a doctor warning) rather than erroring.

## ADR-13: Redefine prime's `pending.count` as in-session blocking work only, surfacing foreign work as counts without subjects

**Status:** Accepted
**Date:** 2026-06-01

**Context:** The agent-UX review identified `pending.count` as the field every agent prompt anchors on. Under the new gate semantics, if it stayed the raw total, agents would overcount and burn tokens trying to document foreign work they shouldn't touch — flagged as the single most important agent-UX change in v0.23.0.

**Decision:** `gatherPrimeContext` now calls both `GetPendingCommits` (the in-session set that drives `Count`) and `ExplainPending` (for bucketing into `OutOfSession`/`Stale`); `buildPrimePending` takes both and was extracted to `prime_build.go` for the 350-line limit. Foreign-author commits are deliberately NOT placed in the `Commits` slice — only their count surfaces — because surfacing subject lines leads agents to fabricate what/why/how rather than respect "not yours to document." `ExplainPending` errors are non-fatal: it's visibility-only, so a classify failure still ships the correct in-session `Count` from the existing filtered path.

**Consequences:**
- In-session-only cases see an unchanged `Count` (the flip is invisible to them) while the contract is reframed for everyone; agents target zero on in-session work only.
- Agents cannot fabricate documentation for foreign commits because they never see the subjects — only the count.
- The compact human-output "n out-of-session commits skipped" diagnostic was deferred to the post-commit-note phase; the JSON schema change is the load-bearing part here.

## ADR-14: Surface auto-skipped stale-self commits via a non-blocking post-commit note

**Status:** Accepted
**Date:** 2026-06-01

**Context:** The agent-UX review named one failure mode as unrecoverable in-session — a marathon session running past the 24h window silently stale-skips its own work, and the agent never learns it happened. The reframe (ADR-8) forbids putting a warning on the gate itself, since that would defeat the leniency.

**Decision:** Restructure `runPostCommitHook` around a `classifyPostCommitState` helper that walks `ExplainPending` and buckets reasons into actionable (in-session — the existing "document this" nudge) and stale-self (a new note). The note doesn't block — it preserves the signal that something was auto-skipped, references `DefaultSessionWindow` so the operator sees what threshold tripped, and points at `--explain`. Foreign-author skips deliberately stay silent (the operator already chose not to be that author). `TIMBERS_SKIP_CROSS_AGENT_DEBT=1` short-circuits both surfaces. Listing the stale-self SHAs was rejected as noisy in long sessions (count + an `--explain` pointer is the balance); once-per-session rate-limiting was deferred, since a count-based note doesn't grow with repetition.

**Consequences:**
- The one unrecoverable failure mode now leaves a visible signal without reintroducing a blocking gate.
- Foreign-author skips remain silent by design — no signal if foreign work is unexpectedly large.
- The note fires per-commit; the operator's commit cadence bounds its frequency rather than an explicit limiter.
