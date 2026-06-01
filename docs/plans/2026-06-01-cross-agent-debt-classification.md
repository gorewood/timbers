# Plan: Cross-Agent Debt Classification (v0.23.0)

**Status:** Reviewed (3-perspective: pragmatist + correctness-skeptic + agent-UX)
**Date:** 2026-06-01
**Target version:** v0.23.0 (minor — gate behavior change, not strictly backward-compatible)
**Owner bead:** timbers-vlh
**Related (shipped):** timbers-88k (v0.22.8 refuse-on-dirty), timbers-0or (v0.22.9 staged-changes hint)

## Reframing

The gate's purpose is **to capture live reasoning while it exists** — agent in-session deliberation, subagent debate, operator decisions made in the moment, anything that wouldn't otherwise land in a git commit message. Strict gating pays off when the signal is alive in the session; it generates pure friction once the signal is gone.

Currently the gate cannot tell the difference. It treats every undocumented commit on first-parent line as recoverable debt and blocks the next commit until it's documented. In multi-author / multi-session / cross-agent flows, most pending commits are **write-offs** — their reasoning context has expired — not debt.

The fix is to make the gate's decision **provenance-aware**: strict for in-session work, lenient for foreign or stale work, with the rest of the system surfacing the foreign set so the operator can still see it.

## Principles

1. **Strict where the signal lives, lenient where it doesn't.** Be willing to skip a commit silently when capturing reasoning is no longer possible.
2. **Generous skip heuristics, with safe defaults.** False-positive blocking (real pain, agent gets stuck) costs more than false-negative miss (lost signal that was already lost). When a degraded environment (missing config, parse failure) could cause silent feature-disable, fail SAFE — treat everything as in-session and emit a loud diagnostic.
3. **Skipped ≠ invisible.** Foreign-session commits must still show up in `pending --explain` so the operator can ack or backfill if desired.
4. **Agent-facing semantics anchor on the in-session number.** `pending.count` (the number agents read) means "in-session blocking" only — never the raw total. The session-end contract "pending should be 0" then targets the right set.
5. **Layered with deliberate policy.** `.timbersignore` (intentional skips like dependabot) layers on top of automatic provenance skips (lost-signal skips). Two different purposes; both compose.

## In Scope (v0.23.0)

### Gate behavior

- **Auto-skip foreign-session commits** in the pre-commit gate decision (`runPreCommitHook` → `hasActionablePending`).
- **Auto-skip stale commits** older than a configurable window (default **24h**; see "Heuristics" for why not 4h).
- **Composition rule:** a commit is gate-blocking iff it is BOTH recent AND attributed to the current committer. Either condition failing → silent skip.

### Heuristics

Two signals, OR-combined (either one demoting the commit to skipped):

1. **Author-email mismatch (mailmap-resolved).** `commit.AuthorEmail != git config user.email` → skip. Strict equality on the **mailmap-resolved** email. Use `%aE` in `commitFormat()` (not `%ae`) so a repo's `.mailmap` coalesces alternate emails for the same operator (e.g. `Bob Bergman <bob@redshifted.io> <robert.bergman@gmail.com>`). No domain wildcards, no name fallback — `.mailmap` is the canonical mechanism for the multi-email case and git already honors it everywhere.

2. **CommitDate staleness.** `now - commit.CommitDate > window` → skip. **Use `%ct` (CommitDate), not `%at` (AuthorDate).**
   - AuthorDate is preserved across rebases and `--amend` operations. Using it would cause the heuristic to silently skip work the user is *actively* touching in-session (rebase moves the commit, but AuthorDate stays old → stale → skip).
   - CommitDate advances every time the commit is recorded on the current DAG. Stale-by-CommitDate genuinely means "this commit hasn't moved in $window."
   - This requires extending `internal/git/commit.go` `commitFormat()` to include `%ct` and adding a `CommitDate time.Time` field to `git.Commit`. Non-optional — without it, the heuristic is wrong.

3. **Session ID match (future, deferred).** A precise in-session signal via embedded session ID in commit trailers. Defer until the email + window heuristic shows real gaps in tally data.

**Default rule for v0.23.0:** skip if `(author-email differs) OR (CommitDate older than window)`. Window default = **24h**. (Rationale: 4h is too short — silent same-author skip is unrecoverable, and the worst failure mode is the agent's own marathon-session work disappearing from the gate. 24h covers all realistic agent sessions including long autonomous loops and orchestrator + subagent fanouts. Cross-agent work is caught by email-mismatch regardless of age.)

### Safe degradation (REQUIRED — these are correctness, not polish)

- **Empty `git config user.email`** → fall back to "treat all commits as in-session." Otherwise every commit mismatches and the gate is silently disabled. `timbers doctor` surfaces a high-severity warning when user.email is unset.
- **Malformed `session-window`** → fall back to default + emit one-line stderr warning on the next gate invocation, surfaced via `timbers doctor`. Never silently use a different window than the user asked for.
- **Clock skew / future-dated commits (`CommitDate > now`)** → not "stale" (negative duration). Test locks this in.

### `.timbersignore` directive (one only)

- `session-window: <duration>` — override the staleness window per repo. Parse via Go `time.ParseDuration`. Document the supported grammar explicitly: accepts `4h`, `2h30m`, `15m`, `90m`; rejects `1d`, `4 hours`, `4H`, `4hr`. Malformed → default + warning (per safe-degradation above).
- `session-author:` (multi-email same-operator) — **deferred indefinitely.** `.mailmap` is git's canonical mechanism for this case and timbers gets it for free by reading `%aE` (per Heuristics above). Phase 1 tally confirmed `.mailmap` handles real multi-email same-operator workflows (`Bob Bergman <bob@redshifted.io> <robert.bergman@gmail.com>` in osprey-strike). A timbers-specific directive would duplicate that without adding capability.

### Visibility surface — ONE place, not four

- **`timbers pending --explain` gains a `provenance` column.** Values: `in-session` (default for kept commits), `foreign-author`, `stale`, `foreign-author+stale`. Provenance applies only to commits that would otherwise be kept by the existing skip chain (infra → identity → content). Precedence is locked: existing reasons win first; provenance is the last classifier.
- **`timbers prime` JSON: `pending.count` changes semantics.** Becomes the in-session blocking count (the number agents anchor on). Add sibling fields `pending.out_of_session` and `pending.stale` for visibility. This is the load-bearing UX change: every agent prompt that says "drive pending to zero" continues to work correctly without rewording.
- **`timbers prime` human output gets ONE extra line** when there's out-of-session work: `n out-of-session commits skipped (--explain to inspect)`. No full breakdown; the line is a diagnostic pointer, not a workflow surface.

Cut from v0.23.0 (ship when there's a real consumer):
- ~~`timbers pending` default-output footer~~ — `--explain` is the right place.
- ~~`timbers status` provenance breakdown~~ — duplicates prime; ship if/when an operator asks.
- ~~`timbers pending --include-foreign` flag~~ — backfill ergonomic for a workflow that doesn't exist yet. If an operator wants to see foreign commits, `pending --explain` shows them. Add when the catchup flow has a real user.
- ~~`timbers log --batch --include-foreign` flag~~ — same reasoning.

### Protocol / agent-facing copy updates

- **`internal/protocol/protocol.go SessionProtocol`** — update the session-end checklist line so it targets the right number. Current: `timbers pending (should be zero before session end)`. Under the new semantics, raw pending may include out-of-session items that the agent should NOT try to drive to zero. Reword to make clear it's the in-session count that matters. Candidate: `timbers pending — your in-session commits should all be documented (out-of-session entries auto-skip; --explain to inspect)`.
- **Hook copy** — pre-commit refusal and post-commit reminder stay as-is. With auto-skip on they fire less, and when they do fire the work IS the user's, so existing copy is correct.

### Stale-self visibility

When the gate auto-skips a commit on staleness-only (NOT email-mismatch) — i.e., the user's OWN stale work — emit a single one-line note to stderr from the **post-commit** hook saying so. The note must not block, but must surface the silent same-author skip so a marathon-session agent doesn't lose its own signal completely without any record. Wording TBD; something like `[timbers] auto-skipped stale commit <short> (>24h old); use 'timbers log --range' to backfill if needed`. Foreign-author skips stay silent (the operator already chose to not be that author).

### Env var lifecycle

- `TIMBERS_SKIP_CROSS_AGENT_DEBT=1` — **kept for backward compat**. Now redundant in normal flows; documented as "manual override for cases the heuristic misses." No deprecation in v0.23.0 — re-evaluate in v0.24+.

## Out of Scope (deferred to v0.23.x or later)

- **Session ID propagation.** Cleanest in-session signal but requires agent-runtime + trailer convention work. Defer.
- **Co-author trailer awareness.** Defer until we see a real case.
- **Domain-wildcard author matching.** Defer until tally shows the strict heuristic missing real cases.
- **`session-author` directive** for multi-email same-operator. Deferred per above.
- **`--include-foreign` / catchup flow.** Defer.
- **`timbers status` provenance breakdown.** Defer.
- **`timbers pending` default footer.** Defer.
- **Auto-promoting foreign work into `.timbersignore`.** No — the operator decides what to skip permanently.

## Heuristic Tuning: Tally Before Tune (Shrunk)

Run a tally before locking defaults:

1. Walk the last ~30 commits on `main` in **this repo only** (one repo, recent first-parent history — covers the high-signal sample; multi-repo + 200 commits is procrastination dressed as rigor).
2. For each commit, label: would I want the gate to block here under the reframe? (yes/no).
3. Compute what `(email-mismatch OR CommitDate > 24h)` would have decided.
4. Tabulate: false-positive blocks (the pain we're fixing — target zero), false-negative skips (tolerable, since signal was already lost).

A 10-minute shell script reproducing the heuristic against `git log --first-parent --format='%H %ce %ct %s'` is sufficient. If email-mismatch alone gets us to zero false-positives in the sample, the time-window collapses to safety-net status (still ship, since rebase/amend edge cases need it). If both contribute, ship both. Decide from data.

This phase precedes any implementation work.

## Risks

1. **Silent same-author stale-skip in marathon sessions.** Mitigation: 24h default (not 4h), CommitDate (not AuthorDate), post-commit stderr note on stale-only skip.
2. **Misconfigured environment silently disabling the gate.** Mitigation: empty user.email → all-in-session fallback, with doctor warning.
3. **Existing operators losing the strict-blocking contract.** Mitigation: explain provenance in CHANGELOG; the data is still visible via `--explain`.
4. **Time-window default wrong for some workflows.** Mitigation: `session-window` directive for per-repo tuning, plus malformed-value safety.
5. **Rebase / amend silently misclassified.** Mitigation: CommitDate is the load-bearing fix here. Without it the entire heuristic is unsafe — this is the single highest-impact technical decision in the plan.

## Implementation Phases

1. **Tally.** ~30-commit shell-script tally on this repo's main. Confirm or revise window default. Document results inline in this plan as an addendum.
2. **Extend `git.Commit` with CommitDate.** Add `%ct` to `commitFormat()` and `CommitDate time.Time` field. Tests cover the new field.
3. **Add provenance classifier.** New reason values (`foreign-author`, `stale`, `foreign-author+stale`) in `classifyCommit`. Provenance is last in the precedence chain. Extend `ClassifiedCommit` / `ExplainPending`.
4. **Safe degradation.** Empty user.email → all-in-session. Malformed session-window → default + warning. Doctor surfaces both.
5. **Wire auto-skip into the gate.** `hasActionablePending` consults the new classification. Pre-commit gate fires less.
6. **Add `.timbersignore session-window` directive.** Parsing + plumbing.
7. **Update prime semantics.** `pending.count` → in-session blocking only. New sibling fields. One-line human-output diagnostic when foreign exists. Update `SessionProtocol` checklist text.
8. **Stale-self post-commit note.** One-line stderr emission when own stale commit auto-skips.
9. **Docs.** CHANGELOG with explicit "what changed" callout; `agent-dx-guide.md` mention; protocol copy update.

Phase 1 is gating — no phase 2+ until tally confirms defaults.

## Acceptance Criteria (v0.23.0)

- [ ] Tally completed; results inlined in this plan (addendum at bottom).
- [ ] `git.Commit` gains `CommitDate` from `%ct`; existing tests pass with the new field.
- [ ] Pre-commit gate skips commits matching the heuristic; tests cover email-mismatch and CommitDate-staleness paths.
- [ ] **Test:** rebased commit retains in-session status (rebase shifts CommitDate; AuthorDate-based logic would have failed — this is the regression test).
- [ ] **Test:** `git commit --amend` of a stale commit stays in-session.
- [ ] **Test:** empty `git config user.email` → all-in-session fallback; doctor surfaces warning.
- [ ] **Test:** malformed `session-window: 1d` → default + stderr warning + doctor warning.
- [ ] **Test:** future-dated commit (`CommitDate > now`) does NOT skip as stale.
- [ ] **Test:** precedence — foreign-author + documented + acked → reason is `documented` (or `ack`), not `foreign-author`. Existing skip chain wins first.
- [ ] **Test:** foreign-author revert of documented commit → reason `revert`, not `foreign-author`.
- [ ] **Test:** first-parent vs full-DAG walks produce consistent provenance reasons for the same commit.
- [ ] `pending --explain` displays a provenance reason per commit (human + JSON).
- [ ] `prime` JSON: `pending.count` is in-session blocking; `pending.out_of_session` and `pending.stale` are sibling fields. One-line human-output diagnostic when foreign exists.
- [ ] `internal/protocol/protocol.go SessionProtocol` checklist line updated; tests cover the new wording.
- [ ] Post-commit hook emits stale-self note on own-stale auto-skip.
- [ ] `.timbersignore session-window: <duration>` directive parses and overrides default; grammar documented.
- [ ] `TIMBERS_SKIP_CROSS_AGENT_DEBT=1` continues to work (regression).
- [ ] All existing gate tests pass.
- [ ] CHANGELOG entry explicitly calls out the gate behavior change and `pending.count` semantics change.
- [ ] One real cross-session integration test (foreign author commits + current author commits, gate decisions verified end-to-end).

## Decided in Review

The plan was reviewed by three independent perspectives (pragmatist, correctness-skeptic, agent-UX advocate). Key adjustments:

- **CommitDate, not AuthorDate** (correctness-skeptic): non-negotiable. Required parse-format extension. Without it, rebase/amend silently misclassify in-session work.
- **24h default, not 4h** (agent-UX): the worst failure mode is silent same-author stale-skip in marathon sessions; 4h made that likely. 24h covers realistic agent sessions; cross-agent is caught by email regardless.
- **`pending.count` semantics flip to in-session only** (agent-UX): every agent prompt anchors on this field. Without the change, agents overcount and waste tokens on foreign work.
- **Empty user.email degrades to all-in-session** (correctness-skeptic + agent-UX): a misconfigured env would otherwise silently disable the entire gate.
- **Cut: status breakdown, default-output footer, `--include-foreign` flags** (pragmatist): no concrete consumer; defer until the catchup workflow has a real user.
- **Cut: multi-repo + 200-commit tally** (pragmatist): shrunk to 30 commits in this repo, ~10 min shell script. Multi-repo over-rigor for a heuristic that's cheap to tune post-hoc.
- **Kept: `session-window` directive** despite pragmatist's cut suggestion. Correctness-skeptic's rebase/amend finding makes the window's correctness load-bearing, and agent-UX's "4h too short / 24h might still be wrong for some workflows" makes per-repo tuning worth shipping rather than baking into a constant.
- **Added: stale-self post-commit stderr note** (agent-UX): own-stale silent skip is the one failure mode an agent can't recover from in-session; surfacing it as a non-blocking note preserves the reframe while avoiding signal-loss.

## Open Questions to Resolve in Phase 1 (Tally)

1. **Time-window default — 24h, or something else?** Confirm via tally.
2. **Composite reason format in `--explain`.** Single column with composite values (`foreign-author+stale`) or two separate columns? Pick during phase 3.
3. **JSON output schema for `out_of_session` / `stale`** — top-level sibling fields, or nested under `pending`? Pick during phase 7.
4. **Whether the stale-self post-commit note should be repeatable** (every commit) or rate-limited (once per session). Defer until phase 8.

---

## Phase 1 Tally Results (2026-06-01)

Ran the heuristic against two repos: `osprey-strike` (the real signal — multi-author, bot-heavy, multi-email same-operator) and `timbers` (single-author baseline).

### Key discovery: `.mailmap` IS the right email resolver

`osprey-strike/.mailmap` already coalesces Bob's two emails:

```
Bob Bergman <bob@redshifted.io> <robert.bergman@gmail.com>
```

Raw `%ae` would have caused **strict-equality to silently skip 11 of Bob's 18 first-parent commits** in the recent 100-commit window (61% false-negative on his own work). `%aE` (mailmap-resolved) coalesces them correctly. Git already supports this; timbers should use `%aE` in `commitFormat()`, not `%ae`.

This makes the deferred `session-author:` directive even less urgent — `.mailmap` is git's canonical mechanism for the multi-email same-operator case, exists in industry-convention form already (`git shortlog`, `git blame`, `git log`, `gitsh` all honor it), and the osprey-strike repo already uses it for Conduit's per-person activity views. timbers gets the right answer for free by using `%aE`.

### Sample 1: osprey-strike, 100 first-parent commits (mailmap-resolved)

Per-author breakdown:

| Author (mailmap-resolved) | Commits |
|---|---|
| `noreply@argoproj.io` (argocd-image-updater bot) | 51 |
| `bob@redshifted.io` (Bob, with `robert.bergman@gmail.com` coalesced) | 18 |
| `conduit-publish-bot@constructured.com` | 17 |
| `gary@88keys.net` (human teammate) | 7 |
| `laura.kolker@gmail.com` (human teammate) | 4 |
| `noam@redshifted.io` (human teammate) | 3 |

Two bots + three human teammates + Bob in a 100-commit window. This is the multi-author/bot reality the reframe is targeting.

### Sample 2: osprey-strike, gate-decision simulation at a Bob-active moment

Anchored "now" at Bob's most recent activity (`now = 1780087674`, 2026-05-30 17:27 UTC, 60s after his last commit). 87 prior commits in scope:

| Bucket | Count |
|---|---|
| IN-SESSION (gate blocks) | 3 |
| foreign-author (within 24h) | 4 |
| stale (Bob's, > 24h) | 15 |
| foreign+stale | 65 |

**Gate would block on 3 in-session Bob commits.** All 3 are real, recent Bob work — no false-positives. Today's gate would have demanded documentation for all 87.

Bob's active session in that cluster spanned 7.4h (Unix gap from 8acb53ce to e0d7dae5). With the proposed 24h window, the entire session is correctly classified in-session. **With the original 4h default, the 7.4h-old commit would have been stale-skipped — silent same-author miss.** Validates the agent-UX push to 24h.

### Sample 3: timbers repo, 50 first-parent commits

| Bucket | Count |
|---|---|
| IN-SESSION (gate blocks) | 8 |
| stale (Bob's, > 24h) | 39 |
| foreign+stale (github-actions[bot] daily devblog) | 3 |
| foreign-author (within 24h) | 0 |

Single-author repo modulo one scheduled GitHub Actions bot. All 8 in-session blocks are real and were documented this session. Heuristic produces correct results.

### Decisions confirmed

- ✅ **Use `%aE` (mailmap-resolved), not `%ae`.** Adds zero new timbers config; uses git's canonical convention. This is a load-bearing change — without it, multi-email same-operator repos silently misclassify 60%+ of in-session work.
- ✅ **24h default window** (not 4h). Bob's recent session ran 7.4h; 4h would have caused silent stale-skip mid-session.
- ✅ **OR composition** of email + staleness. Each catches a different real case (bots/teammates → email; long-ago-but-same-author → staleness).
- ✅ **Strict equality on mailmap-resolved email.** No name-match fallback needed; mailmap handles the multi-email case.
- ✅ **`session-author:` directive stays deferred.** `.mailmap` covers the same need with a standard tool.

### Plan revision

One new acceptance criterion lands from this phase:

- [ ] **`git.Commit` populated from `%aE` (mailmap-resolved author email), not `%ae`.** Test: a repo with `.mailmap` coalescing two emails produces a single author identity in `git.Commit.AuthorEmail`. Without this, the strict-email heuristic silently misclassifies multi-email same-operator work as foreign — the exact case osprey-strike hit.

No other plan changes from the tally. Phase 2 (extend `git.Commit` with `CommitDate` + `%aE`) is unblocked.
