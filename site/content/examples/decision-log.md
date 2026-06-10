+++
title = 'Decision Log'
date = '2026-06-10'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

## ADR-1: Migrate beads sync from server-mode Dolt to embedded Dolt + `refs/dolt/data`

**Status:** Accepted
**Date:** 2026-05-27

**Context:** The repo was in a broken hybrid state. Server-mode Dolt couldn't persist under the sandbox (it re-imported JSONL on every call), and a stale CLI remote plus a 3-week-old `refs/dolt/data` existed while the steering docs claimed "no remote." A coherent sync model was needed, and osprey-strike's already-embedded setup was the reference.

**Decision:** Adopt the canonical embedded-Dolt + `refs/dolt/data` model. Rebuild `embeddeddolt` from the trusted 60-issue `issues.jsonl` — the only current source of truth (server Dolt last modified Mar 22, `refs/dolt/data` topped out May 7) — then drop the stale ref, re-seed fresh, flip `export.auto`/`export.git-add` to false, and untrack/gitignore the JSONL. Followed osprey-strike's migration doc as the playbook, adapted for the server→embedded delta osprey didn't have.

**Consequences:**
- Sync works under the sandbox (no TCP); clones inherit via `bd bootstrap`; verified by a fresh-clone round-trip.
- Lost Dolt audit history (rebuild-from-JSONL discards it) — accepted because JSONL was the only current truth.
- Bootstrap detection order is load-bearing: had to drop `refs/dolt/data` *and* set aside `.beads/backup` so bootstrap fell through to the current JSONL instead of cloning stale state.
- Hand-edits landed inside `bd`'s managed markers (timbers-069 tracks relocating them).

## ADR-2: Honor `--anchor` at zero pending instead of refusing

**Status:** Accepted
**Date:** 2026-05-28

**Context:** osprey hit a gap where an anchor off the first-parent line yielded 0 detected pending, and `timbers log --anchor` refused — despite the flag's name promising "use this anchor." Separately, a bare "No pending commits" conflated "clean" with "computed from an off-line anchor."

**Decision:** At 0 pending with `--anchor` set, `getLogCommits` falls back to `LogRange(anchor^, anchor)` to document the single commit, and the refusal now names `--range` as the explicit escape hatch. In pending's `count:0` branch, print a note keyed on `AnchorOffFirstParentLine` so the off-line-anchor case is named rather than hidden.

**Consequences:**
- `--anchor` honors its name; off-first-parent work is documentable without guessing.
- `pending` no longer silently presents an off-line-anchor computation as "clean."
- Commit-resolution helpers extracted to `log_resolve.go` for the file-length limit; required a ref-aware mock to lock in via `TestLogAnchorBypassesZeroPending`.

## ADR-3: Refuse `timbers log` on a dirty tree

**Status:** Accepted
**Date:** 2026-06-01

**Context:** v0.22.7 warned-and-proceeded on a dirty tree, which let phantom entries reach upstream — an aborted gated commit leaves staged work in the index, then `timbers log` auto-commits an entry pathspec-scoped to the entry file only, riding the old HEAD while the feature work stays unstaged. A field report from an agent in osprey-strike observed two phantoms in one session against v0.22.7.

**Decision:** Replace the `printer.Warn` with a `UserError`, guarded by `!flags.dryRun` so `--dry-run` stays usable for inspecting an entry mid-debug. Deliberately did **not** add `--allow-dirty` — the protocol has no case for logging uncommitted work, and the flag would reopen the exact footgun. Rejected the report's Option B (create the entry, skip the auto-commit): since the trigger is the gate *aborting* commits, a deferred entry would sit dirty indefinitely and dirty entries compound. A follow-up added a `HasStagedChanges()`-gated hint pointing at `git diff --cached`, so the caller learns *why* their commit vanished before reaching the now-refusing `log` call.

**Consequences:**
- Phantom entries closed; the tool now enforces what the warning previously only asked.
- One extra `git` call per hook invocation for the conditional hint — the unconditional version was rejected because it would mislead `git commit --amend --no-edit` against an undocumented HEAD.
- No path to log genuinely uncommitted work — accepted as outside the protocol.

## ADR-4: Reframe the commit gate to capture in-session reasoning, accepting misses outside the session

**Status:** Accepted
**Date:** 2026-06-01

**Context:** The user reframed the gate's purpose — capture *live in-session* reasoning, accept misses outside the session. The existing gate didn't distinguish, so it fired on foreign/stale work that can't be productively documented. timbers-vlh was bumped P3→P1 and a plan doc was authored, then reviewed by three independent Opus subagents (pragmatist, correctness-skeptic, agent-UX advocate).

**Decision:** Make the gate provenance-aware via OR-composed heuristics (foreign-author OR stale) with a 24h default window. The reviewers materially reshaped the spec: the pragmatist cut four items (status breakdown, default footer, `--include-foreign`, `session-author`) and shrank the tally; the correctness-skeptic surfaced the `AuthorDate`-vs-`CommitDate` hazard (see ADR-5); the agent-UX advocate named `pending.count` as the load-bearing UX field (see ADR-9), pushed the window 4h→24h, and flagged the silent same-author stale-skip as the unrecoverable failure mode (see ADR-10). The rationale was synthesized into a durable Decided-in-Review section.

**Consequences:**
- The gate stops firing on un-documentable foreign/stale work; rationale is durable for implementers across the phased rollout.
- Kept the `.timbersignore` `session-window` directive over the pragmatist's objection, because the rebase/amend finding made window correctness load-bearing (see ADR-8).
- Did **not** add a stale-self warning to the gate itself — it would defeat the lenience reframe; moved to a non-blocking post-commit note (ADR-10).
- `--include-foreign` cut entirely from v0.23.0 — it serves a backfill workflow with no real user yet.

## ADR-5: Use `CommitDate` and `%aE` (mailmap-resolved) for the classifier's staleness and identity signals

**Status:** Accepted
**Date:** 2026-06-01

**Context:** Before locking heuristic defaults, the plan called for a tally against `osprey-strike` (multi-author, bot-heavy, real-world multi-agent workflows — the origin of most recent timbers feature work). Two signal choices were at stake: which date drives staleness, and which email format drives authorship.

**Decision:** Use `CommitDate`, not `AuthorDate` — `AuthorDate` stays put across rebase/amend, so a staleness check keyed on it would silently auto-skip work the user just touched. Use `%aE`, not `%ae` — osprey-strike's existing `.mailmap` coalesces Bob's two emails (`bob@redshifted.io` ↔ `robert.bergman@gmail.com`), and `%ae` would have produced a 61% false-negative rate on Bob's own work. Mailmap is git's canonical multi-email mechanism and gets the right answer for free; this single tally finding also killed the proposed `session-author` directive. Kept the existing `Date` field as `AuthorDate` for backward compat and added `CommitDate` as a sibling.

**Consequences:**
- Rebased/amended in-session commits retain in-session status; multi-email operators aren't misread as foreign.
- Mailmap reuse means no new config surface for the multi-email case.
- `commitFormat` now emits `%aE`/`%at`/`%ct` in fixed positions; diffstat code (~90 lines) extracted to `diffstat.go` for the 350-line limit.
- The tally ran ~30 min (two repos, three "now" anchors) vs the ~10 min projected — judged worth it for the mailmap finding.

## ADR-6: Run the provenance classifier last in the skip chain

**Status:** Accepted
**Date:** 2026-06-01

**Context:** Phase 3 landed the classifier as a standalone, fully tested unit before wiring it into the gate decision. Where it sits in `classifyCommit`'s chain determines whether provenance can override an existing classification.

**Decision:** Thread `classifyByProvenance` at the **end** of the chain (after infra → identity → content) so earlier reasons keep their decision-relevance — a documented or acked commit doesn't relabel as foreign-author just because the email differs. Held `ProvenanceConfig` on `Storage` (zero-valued = disabled) rather than passing it as a parameter; the parameter form was rejected because it would bloat the signature across `filterByRules`, `classifyCommit`, `traceFilterDecisions`, and `ExplainPending`, and the config shares `Storage`'s per-repo lifecycle. Landed with empty config (not yet wired into the gate) so chain order is verified independently of the phase-5 gate wiring.

**Consequences:**
- Precedence (earlier-wins) is locked by 4 subtests against a foreign+stale commit before any gate-path wiring exists.
- `ProvenanceConfig.Now` is a plain `time.Time` field — tests pin a fixed instant (`Now=2026-06-01` noon) with no clock-injection plumbing.
- Provenance must be set explicitly per `Storage`; a caller that forgets gets disabled classification from the zero value (the failure mode ADR-7 addresses).

## ADR-7: Split construction into `NewStorage` (zero-config) and `NewDefaultStorage` (production-wired)

**Status:** Accepted
**Date:** 2026-06-01

**Context:** Wiring provenance loading into `NewStorage` made it read the host's `user.email` at construction, so every test that built mock commits without a matching `AuthorEmail` was falsely classified foreign-author — three test suites failed on the first `just check`. The fix went through two iterations.

**Decision:** The first attempt zeroed `s.provenance` inside `newTestStorage`, but that only covered the ledger package's tests. The accepted second attempt moved `LoadProvenanceConfig` out of `NewStorage` into a new `NewDefaultStorage` production entry point: `NewStorage` returns zero-config `Storage`, and `Storage.SetProvenance` is the public pin for tests and external callers. This mirrors the existing raw-vs-production constructor pattern and avoids a `NewStorageForTest` variant that would have forced changes across 10+ call sites in `mcp`, `cmd/timbers`, and `ledger` tests.

**Consequences:**
- Test and production paths cleanly separated with minimal test churn; surfaces the load-bearing semantic that provenance must be explicit per `Storage`.
- Two construction entry points to keep straight — production code must call `NewDefaultStorage` or silently get disabled provenance.
- Doctor tests need `GIT_CONFIG_GLOBAL=/dev/null` to defeat host-config leakage (`robert.bergman@gmail.com` leaked during initial runs).

## ADR-8: Add a per-repo `session-window` directive to `.timbersignore`

**Status:** Accepted
**Date:** 2026-06-01

**Context:** The tally confirmed 24h as a safe default, but the agent-UX advocate flagged that long-running sessions (orchestrator + subagent fanouts running 4–10h, all-day refactors) could exceed it and silently stale-skip their own work. The pragmatist had wanted to cut the directive; the correctness-skeptic's rebase/amend finding made window correctness load-bearing, so it was kept as cheap insurance.

**Decision:** Parse `session-window` via a separate `LoadSessionWindow(root)` pass over `.timbersignore` rather than threading a new return value through `loadSkipConfig`'s 10+ call sites (a 4-tuple→5-tuple break for every test that destructures it); the file is tiny, so the dual-parse cost is negligible. Coerce zero/negative durations to the default rather than erroring — Go's `time.ParseDuration` accepts negatives as valid, and treating them as malformed would diverge from the documented grammar. Doctor's `checkSessionWindow` warns on malformed values with the grammar in the hint.

**Consequences:**
- Long-running sessions can opt into a wider window; repos that set nothing get the safe 24h default.
- The isolated pass keeps the well-tested existing parser untouched at the cost of reading the file twice (accepted: negligible).
- Negative/zero windows silently coerce to default rather than erroring at parse time — surfaced only via doctor.

## ADR-9: Redefine `pending.count` as in-session blocking work only

**Status:** Accepted
**Date:** 2026-06-01

**Context:** The agent-UX review identified `pending.count` as the field every agent prompt anchors on. Under the new gate semantics, if it remained the raw total, agents would silently overcount and waste tokens trying to document foreign work.

**Decision:** `Count` reflects only the in-session blocking set. `gatherPrimeContext` calls both `GetPendingCommits` (drives `Count`) and `ExplainPending` (buckets `OutOfSession`/`Stale`); `buildPrimePending` was extracted to `prime_build.go` for the 350-line limit. Foreign-author commits surface as a count only — deliberately **not** in the `Commits` slice — because the review flagged that showing subject lines leads agents to hallucinate documentation for work that isn't theirs. The flip is invisible to in-session-only cases (`Count` unchanged) but reframes the contract for everyone.

**Consequences:**
- Agents target a meaningful zero, waste no tokens on foreign work, and can't fabricate what/why/how from leaked subjects.
- `ExplainPending` errors are deliberately non-fatal — a visibility-only feature must not break the correct in-session `Count` from the existing filtered path.
- The compact human-output "n out-of-session commits skipped" diagnostic was deferred to the post-commit note (ADR-10); only the JSON schema change landed here.

## ADR-10: Surface stale-self auto-skips as a non-blocking post-commit note

**Status:** Accepted
**Date:** 2026-06-01

**Context:** The agent-UX review named one failure mode unrecoverable in-session: a marathon session running past the 24h window silently stale-skips its *own* work, and the agent never knows. The reframe forbids blocking on it, so visibility had to come without a gate.

**Decision:** `classifyPostCommitState` walks `ExplainPending` and buckets reasons into actionable (in-session — the existing "document this commit" nudge) and stale-self (the new note); both fire independently and silently when their count is zero. The note references `DefaultSessionWindow` so the operator sees which threshold tripped, but does **not** block — blocking would defeat the lenience reframe. Rejected listing the stale-self SHAs (the list could grow large and noisy in long sessions; count plus a pointer to `--explain` is the balance) and rejected per-session rate-limiting (deferred — a count-based note doesn't grow with repetition). Foreign-author skips stay silent (the operator already chose not to be that author). `TIMBERS_SKIP_CROSS_AGENT_DEBT=1` short-circuits both surfaces.

**Consequences:**
- The unrecoverable failure mode is now visible without re-introducing a blocking gate.
- `ExplainPending`'s full-DAG display range is the right scope — it includes side-branch work relevant for visibility, vs the first-parent-strict gate range.
- No SHA list in the note — the operator must run `--explain` to see which commits were skipped.

## ADR-11: Keep `timbers log` output terse by default; render the readable panel only on demand

**Status:** Accepted
**Date:** 2026-06-08

**Context:** Laura, a timbers user, couldn't scan log output — `output.KeyValue` rendered fields with no key alignment and no value wrapping, so long Why/How/Notes dumped as one soft-wrapped line and fields bled together. The fork: make the readable panel the default `log` output, or keep `log` terse and surface the panel elsewhere.

**Decision:** Terse default + on-demand panel. Agents call `timbers log` without `--json`, so a default box would cost context tokens on every commit for no agent benefit. Added a shared `output.FieldsBox` renderer that finally wires up the previously-unused `output.Box` helper (rounded border at TTY, borderless when piped; keys align to a common column, values wrap with a hanging indent under the value column). Dropped the originally-proposed `--preview` flag once it was clear `timbers show --latest` already renders the same panel — fewer flags, and no semantically odd "preview that also writes." Folded in two bug fixes (dry-run dropped Notes; diffstat was formatted differently in `show` vs dry-run).

**Consequences:**
- `show` and `log --dry-run` are scannable, with no per-commit token cost for agents.
- Reuses the dormant `output.Box` helper; one renderer serves both surfaces.
- Wide-rune wrapping (display-width vs byte-len) is still wrong — deferred to timbers-qbm.
- Terminal width falls back to 80 cols when `charmbracelet/x/term` can't detect it.
