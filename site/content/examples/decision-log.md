+++
title = 'Decision Log'
date = '2026-06-11'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

## ADR-1: Decouple release tagging from ancillary site-content generation

**Status:** Accepted
**Date:** 2026-05-27

**Context:** `just release` regenerated site examples by fanning out parallel `claude -p` calls as part of the release recipe. One of those calls (the decision-log example) flaked on a transient LLM hiccup and aborted an entire version release — a tag that was otherwise ready to ship was blocked by a non-essential content step.

**Decision:** Wrap the `just examples` step in the release recipe with `|| echo WARNING` so a failure warns and the release proceeds to commit/tag/push. Site examples regenerate on a separate path. The reasoning: ancillary, LLM-driven content generation is inherently flaky and must never sit on the critical path of shipping a tag.

**Consequences:**
- A transient LLM failure can no longer block a release.
- Example content can drift from the shipped tag until regenerated separately — the release no longer guarantees examples are fresh.
- Establishes a boundary: LLM-dependent steps are best-effort, not release gates.

## ADR-2: Honor `--anchor` at zero pending and name off-first-parent anchors

**Status:** Accepted
**Date:** 2026-05-28

**Context:** A user (osprey) hit a gap where the anchor sat off the first-parent line: `timbers log --anchor` still refused at zero detected pending even though the flag's name promises "use this anchor," and `timbers pending` reported a bare "No pending commits" that conflated a genuinely clean state with a count computed from an off-first-parent anchor.

**Decision:** Make `--anchor` honor its promise — at zero detected pending with `--anchor` set, `getLogCommits` falls back to documenting a single-commit range (`LogRange(anchor^, anchor)`). Name `--range` in the refusal message as the explicit escape hatch. Separately, `timbers pending` at `count:0` now prints a conditional note (keyed on `AnchorOffFirstParentLine`) that names the off-first-parent situation and points at `--explain`/`--range`.

**Consequences:**
- `--anchor` becomes usable in the off-first-parent / zero-pending case it previously dead-ended on.
- Commit-resolution helpers were extracted to `log_resolve.go` to stay under the file-length limit.
- "No pending commits" is no longer ambiguous about whether it reflects a clean tree or an off-line anchor computation.
- Does not change the default anchor-detection heuristic — only what happens when the user overrides it.

## ADR-3: Refuse `timbers log` on a dirty tree, with no `--allow-dirty` escape

**Status:** Accepted
**Date:** 2026-06-01

**Context:** Through v0.22.7 `timbers log` warned-and-proceeded on a dirty tree, which let phantom entries reach upstream: an aborted gated commit leaves staged changes in the index, then `timbers log` auto-commits an entry pathspec-scoped to the entry file only — the entry rides the old HEAD while the feature work stays unstaged. A field report from an agent in Constructured/osprey-strike observed two such phantoms in a single session. The existing warning already told users to "commit first to avoid phantom entries"; the tool wasn't enforcing what it asked for.

**Decision:** Replace the `printer.Warn` with a `UserError` return guarded by `!flags.dryRun` (so `--dry-run` stays usable for inspecting an entry mid-debug). Deliberately do **not** add `--allow-dirty`: the protocol has no case for logging uncommitted work, and the flag would reopen the exact footgun the refusal closes. Option B from the field report (create the entry but skip the auto-commit) was rejected because the trigger is the gate *aborting* commits — if the next commit also aborts, the entry sits dirty indefinitely and dirty entries compound. A follow-on refinement adds a gate-side hint (`hasStagedChanges`, conditional on a non-empty index) that explains why the caller's commit vanished and points at `git diff --cached`; it was kept conditional rather than unconditional to avoid misleading a `git commit --amend --no-edit` against an undocumented HEAD.

**Consequences:**
- Phantom entries from the aborted-gate path can no longer be created.
- `--dry-run` remains the inspection escape; there is intentionally no way to log against a dirty tree otherwise.
- The error message and gate hint are self-contained (likely trigger, diagnostic command, escape) so the confusion loop is short-circuited before the caller reaches the refusing `timbers log`.
- One extra `git diff --cached` call per hook invocation (accepted as within budget).

## ADR-4: Reframe the pre-commit gate around in-session provenance

**Status:** Accepted
**Date:** 2026-06-01

**Context:** The operator reframed the gate's purpose: it exists to capture *live in-session reasoning*, accepting misses outside the session. The existing gate didn't distinguish provenance, so it fired on foreign-author and stale commits that can't be productively documented — wasting agent tokens trying to document work that isn't theirs. `timbers-vlh` was bumped P3→P1 and a plan doc authored. The draft was reviewed by three independent Opus subagents (pragmatist, correctness-skeptic, agent-UX advocate), each reshaping the design.

**Decision:** Make the gate provenance-aware via a pure `classifyByProvenance` step threaded into the skip chain **last** — after infra → identity → content — so a documented or acked commit keeps its existing decision rather than being relabeled foreign-author. The correctness-skeptic surfaced the load-bearing `AuthorDate`-vs-`CommitDate` issue (see ADR-5); the agent-UX advocate identified the `pending.count` semantic flip as the single most important UX decision (see ADR-7), pushed the staleness default 4h→24h, and flagged the silent same-author stale-skip as the unrecoverable failure mode (see ADR-8). The pragmatist cut `--include-foreign` (no real user yet for the backfill workflow it served), a status breakdown, a default footer, and the session-author directive. A pre-implementation tally against osprey-strike (multi-author, bot-heavy real-world usage) grounded the heuristic defaults in real data and confirmed 24h / OR-composition / strict-mailmap-equality / `CommitDate`.

**Consequences:**
- The gate stops firing on foreign/stale work, aligning it with its reframed purpose.
- Landing the classifier as a standalone tested unit before wiring it into the gate path locked in precedence ordering and safe-degradation independent of the gate wiring.
- Provenance is held on `Storage` (per-repo lifecycle, loaded at construction) rather than passed as a parameter, avoiding signature bloat across `filterByRules`/`classifyCommit`/`traceFilterDecisions`/`ExplainPending`.
- Misses *outside* the session are accepted by design — the gate no longer attempts to capture all reasoning, only live reasoning. Backfilling foreign work has no supported path in this version.

## ADR-5: Use `CommitDate` over `AuthorDate` and `%aE` mailmap resolution for provenance

**Status:** Accepted
**Date:** 2026-06-01

**Context:** Implementing the provenance classifier (ADR-4) required choosing which timestamp drives staleness and which email identifies authorship. The osprey-strike tally proved both were load-bearing: an existing `.mailmap` coalesces the operator's two emails (`bob@redshifted.io` ↔ `robert.bergman@gmail.com`), and using `%ae` instead of `%aE` would have produced a 61% silent false-negative rate on the operator's own work.

**Decision:** Switch `commitFormat` to emit `%aE` (mailmap-resolved author email) so timbers honors `.mailmap` — git's canonical mechanism for the multi-email case, which gets the right answer for free and let the session-author directive be killed entirely. Add `CommitDate` (`%ct`) as a sibling to the existing `Date` field (kept as `AuthorDate` for backward compatibility) and drive the staleness check from it: `AuthorDate` stays fixed across rebase/amend, so using it would silently auto-skip work the user just touched, whereas `CommitDate` advances and reflects the real clock.

**Consequences:**
- A multi-email operator with a `.mailmap` is no longer misread as a foreign author.
- Rebased/amended in-session work is correctly treated as recent rather than stale-skipped.
- `git.Commit` carries both dates; existing callers reading `Date`/`AuthorDate` are unaffected.
- Diffstat code was extracted to `diffstat.go` (~90 lines) to keep the file under the 350-line limit.
- Repos without a `.mailmap` get no benefit from `%aE` — the strict-equality match still depends on the author email matching `user.email` exactly.

## ADR-6: Per-repo `session-window` directive, opt-in over a hardcoded 24h default

**Status:** Accepted
**Date:** 2026-06-01

**Context:** The tally confirmed 24h as a safe default staleness window, but the agent-UX review flagged that long-running sessions (orchestrator + subagent fanouts running 4–10h, all-day refactors) could exceed it and silently stale-skip their own work. The pragmatist wanted to cut per-repo tuning entirely; the agent-UX advocate wanted it. The correctness-skeptic's rebase/amend finding made window correctness load-bearing enough to tip the call.

**Decision:** Add a `session-window:` directive to `.timbersignore`, parsed by a separate `LoadSessionWindow(root)` pass rather than threading a new return value through `loadSkipConfig`'s 10+ call sites (a 4-tuple→5-tuple change would break every destructuring test). The directive is opt-in: repos that set nothing get 24h. Malformed values coerce to the default (negative/zero too — Go's `time.ParseDuration` accepts negatives as valid, so treating them as errors would diverge from the documented grammar), and a `checkSessionWindow` doctor check warns on bad values with the grammar in the hint.

**Consequences:**
- Long-running-session repos can widen the window instead of silently stale-skipping their own commits.
- The dual-pass parse incurs a negligible re-read of a tiny file rather than a refactor of the well-tested existing parser.
- `SessionWindowResult` exposes `Window`/`Raw`/`ParseErr` for doctor diagnostics; safe-degradation means a malformed config never breaks the gate, only warns.
- Repos that never set the directive keep the 24h default — no behavior change for the common case.

## ADR-7: `pending.count` counts in-session blocking work only; foreign work surfaces as counts, not subjects

**Status:** Accepted
**Date:** 2026-06-01

**Context:** The agent-UX review identified `pending.count` as the field every agent prompt anchors on. Under the new provenance gate (ADR-4), if `count` remained the raw total, agents would silently overcount and waste tokens trying to document foreign work. This was called the single most important agent-UX change in v0.23.0.

**Decision:** Flip `pending.count` to mean in-session blocking work only. `gatherPrimeContext` calls `GetPendingCommits` (the in-session set driving `Count`) and `ExplainPending` (for bucketing into `OutOfSession`/`Stale`). Foreign-author commits are deliberately **not** included in the `Commits` slice — only their count surfaces — because the review flagged that agents reading subject lines will fabricate what/why/how rather than respect "not yours to document." `ExplainPending` errors are non-fatal: if the classify-everything pass fails, the correct in-session `Count` from the existing filtered path still ships.

**Consequences:**
- For in-session-only cases the change is invisible (`Count` unchanged); for mixed cases it reframes the contract so agents target the right zero.
- Agents can no longer hallucinate documentation for foreign commits because they never see those subjects.
- `buildPrimePending` was extracted to `prime_build.go` to keep `prime.go` under the 350-line limit; the `SessionProtocol` checklist text was updated to clarify the in-session count is what targets zero.
- A compact human-output diagnostic ("n out-of-session commits skipped") was deferred to the post-commit-note work — the JSON schema change is the load-bearing part here.

## ADR-8: Surface stale-self auto-skips as a non-blocking post-commit note

**Status:** Accepted
**Date:** 2026-06-01

**Context:** The agent-UX review named one failure mode as unrecoverable in-session: a marathon session running past the staleness window silently stale-skips its *own* work, and the agent never learns it happened. A stale-self warning could not go in the gate itself — that would defeat the lenience reframe of ADR-4.

**Decision:** Add a post-commit note (not a gate block) via a new `classifyPostCommitState` helper that walks `ExplainPending` and buckets reasons into actionable (the existing in-session "document this commit" nudge) and stale-self (the new note). The note references `DefaultSessionWindow` so the operator sees which threshold tripped, and both surfaces fire silently when their count is zero. Foreign-author skips stay silent (the operator already chose not to be that author); `TIMBERS_SKIP_CROSS_AGENT_DEBT=1` short-circuits both.

**Consequences:**
- Self-stale-skipped work now leaves a recoverable signal without blocking the commit.
- Listing the stale-self SHAs was rejected (could grow large/noisy in long sessions) in favor of a count plus a pointer to `--explain`; rate-limiting was deferred since a count-based note doesn't grow with repetition and commit cadence naturally bounds its frequency.
- The note walks the display range (full-DAG, includes side-branch work) rather than the gate's first-parent-strict range — the wider scope is correct for visibility.
- The note is advisory only; it does not prevent a marathon session from stale-skipping work, only informs after the fact.

## ADR-9: Terse-default log output with an on-demand entry panel

**Status:** Accepted
**Date:** 2026-06-08

**Context:** Laura, a timbers user, couldn't scan log output: `output.KeyValue` rendered fields with no key alignment and no value wrapping, so long Why/How/Notes dumped as one soft-wrapped line and fields bled together. The original handoff proposed a `--preview` flag and considered making a readable panel the default log output.

**Decision:** Adopt a terse-default + on-demand-panel design rather than making the panel the default for `timbers log`. The reasoning: agents call `timbers log` without `--json`, so a default box would cost context tokens on every commit for no agent benefit. Drop the proposed `--preview` flag entirely once it became clear that `timbers show --latest` already renders the same panel for free — making a "preview that also writes" redundant and semantically odd. Implement a shared `output.FieldsBox` renderer (finally wiring up the unused `output.Box` helper): rounded border at a TTY, borderless when piped, keys aligned to a common column, values wrapped with a hanging indent.

**Consequences:**
- Human readers get a scannable panel via `show`/`--dry-run`; agents pay no per-commit token cost for box chrome.
- The flag surface stays small — no `--preview`, no "preview writes" weirdness.
- Folded in two latent bug fixes: `--dry-run` dropped Notes, and diffstat was formatted differently in `show` vs `dry-run`.
- Wide-rune wrapping (byte-length vs display-width) is a known gap, deferred to `timbers-qbm`.

## ADR-10: Render non-TTY timbers errors as a single plain line

**Status:** Accepted
**Date:** 2026-06-11

**Context:** An agent misread a blocked commit as success: the pre-commit gate correctly rejected the commit, but `git commit 2>&1 | tail -1` plus rtk compression showed a blank line, so the failure looked like nothing. Root cause was `fang`: `fang.Execute` wraps stderr in a `colorprofile.Writer` before calling the error handler, so `DefaultErrorHandler`'s `w.(term.File)` non-TTY guard never fires — the padded ERROR box (which ends in a blank line) renders even into a pipe, and `tail`/compressors then crop the real failure to nothing.

**Decision:** Install a custom `fang` error handler (`newErrorHandler(stderrIsTTY)`) that checks the real `os.Stderr`: a TTY delegates to `fang.DefaultErrorHandler` (keeping the styled box), a non-TTY prints `err.Error()` as one plain line. Chosen over the narrower alternative of `os.Exit`-in-the-gate because the handler is testable (no `os.Exit` killing the test process) and it fixes **every** piped timbers error, not just the gate. Additionally route the gate's `[timbers]` advice block to stderr and make the returned gate error self-contained (names tool, cause, fix, bypass) so the one line that survives `tail -1` is actionable.

**Consequences:**
- Piped/compressed consumers now always see a non-empty, actionable error line — agents can no longer misread a blocked commit as success.
- Interactive TTY users keep the styled `fang` ERROR box.
- A regression test asserts the non-TTY single-line contract.
- A pre-existing nit (`useColor` checks stdout while the gate printer now writes stderr) was left out of scope — the block applies no styling, so it's moot.
