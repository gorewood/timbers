+++
title = 'Decision Log'
date = '2026-06-01'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

## ADR-1: Document the existing `ack` path for rebase-relink over building content-matching

**Status:** Accepted
**Date:** 2026-05-27

**Context:** A feature request arrived with five proposals for relinking ledger entries after a rebase shifts commit SHAs, the requester favoring automatic content-matching (their option A). The blocking constraint: timbers has no content-matching today — `docSet` holds literal SHAs only, so the preferred option would be built from scratch.

**Decision:** Ship documentation of the *existing* `ack` path — which already produces a structured record that counts as documented — rather than build new matching machinery. The reviewer's preference order was inverted to D→B→(maybe A) on Gall's Law grounds: the naive byte-identical-diff/reflog form of content-matching is unsound (rebases shift context lines; reflog is absent on fresh clones/CI), and a sound version requires storing `patch-id` at log time, i.e. a schema change. Added a `RebaseRelinkGuidance` const in `internal/protocol`, wired into both `prime_workflow.go` and `mcp/helpers.go`, replaced generic `ack '...'` reasons with a copy-pasteable `rebased; content in <entry-id>` form, and locked the section with a protocol test.

**Consequences:**
- Cheapest correct fix — no new mechanism, ships immediately.
- Graduates to a typed `ack --to <entry>` only if the free-text link proves fragile; `patch-id` gating built only with real volume evidence.
- No automatic relink — the operator must manually `ack`.
- Distinct from the existing stale-anchor case: there the anchor is GC'd and self-heals; here it stays reachable and does *not* self-heal, which is why a manual record is needed.

## ADR-2: Skip release/housekeeping commits by subject glob (`msg:`) over path-based rules

**Status:** Accepted
**Date:** 2026-05-27

**Context:** Release changelog commits should be excluded from pending detection, but a path-based rule cannot isolate them: `filterByRules` only skips a commit when *every* file matches a rule, and the release commit also touches the version-badge source (`site/layouts/index.html`). Path-skipping that file would silently hide legitimate landing-page edits.

**Decision:** Add a third `.timbersignore` rule kind, `msg:`, matching commit subjects (mirroring the existing `author:` glob design), and skip releases via `msg:chore: changelog for v*`. Message matching is the precise tool and generalizes to other housekeeping subjects (version bumps). `filterByRules` was refactored to reuse `classifyByIdentity` (new reason `message`), which both single-sources skip semantics with the `TIMBERS_DEBUG` trace and flattens the loop under the complexity budget. Also discovered `.timbersignore` was being silently gitignored by the repo's allowlist `.gitignore` — fixed, so every clone/CI reads the same config.

**Consequences:**
- Precise subject-based skip without hiding a file's real edits elsewhere.
- Skip semantics single-sourced through `classifyByIdentity`.
- Subject and author globs are matched literally, so a glob like `author:dependabot[bot]` reads as a character class and silently no-ops (later guarded by a `doctor` lint).
- Exposed a follow-up bug: `status`'s `countAutoSkipped` tally had drifted onto a parallel inline loop never updated for the `msg:` rule and undercounted matches; fixed by delegating it to the same `classifyByIdentity` path so visibility and the gate share one source of truth.

## ADR-3: Verify releases by installing the published artifact as a normal consumer

**Status:** Accepted
**Date:** 2026-05-27

**Context:** `just release` tagged and pushed, but verifying that the full pipeline (CI build → GH release → `install.sh`) actually works was left to a manual follow-up, and the local binary could drift from the published one. The initial framing considered a cron-based follow-up.

**Decision:** After `git push`, poll the GH release endpoint (`gh release view <tag> --jq assets length`, ~10m deadline, 15s interval) and on publish run `just install-release`, verifying the installed `--version` contains the tag. A poll-and-install at the tail of `just release` was chosen over a cron job — simpler, no moving parts, and CI can't install on the dev's laptop anyway, so it has to be local. `install-release` stays the real consumer path (rather than pinning `VERSION`) per the "as if a normal consumer" intent.

**Consequences:**
- Each release sanity-checks the entire publish+install pipeline against the official artifact, not an identical local build.
- Keeps the local binary current without a manual step.
- Relies on `.goreleaser draft:false` so `latest` equals the just-pushed tag once assets appear.
- On poll timeout the release is already pushed, so the recipe prints a recovery hint rather than failing.

## ADR-4: Beads sync — embedded Dolt + canonical `refs/dolt/data` over the server-mode JSONL hybrid

**Status:** Accepted
**Date:** 2026-05-27

**Context:** The repo sat in a broken hybrid state: server-mode Dolt couldn't persist under the sandbox (it re-imported JSONL on every call), and a stale CLI remote plus a 3-week-old `refs/dolt/data` existed while the steering docs claimed "no remote." JSONL was the only current source of truth (server Dolt last modified Mar 22; `refs/dolt/data` topped out May 7).

**Decision:** Align to the canonical embedded model used by osprey-strike: embedded Dolt at `.beads/embeddeddolt/` (gitignored), `.beads/issues.jsonl` as a passive export, sync via `refs/dolt/data` on origin (`bd dolt push`/`pull`). Rebuilding from JSONL was judged safe and correct despite losing Dolt audit history, since JSONL was the only current truth. Followed osprey-strike's validated migration doc as the playbook, adapting for the server→embedded delta osprey didn't have.

**Consequences:**
- Works under the sandbox (no localhost TCP), matches the org-standard model, and clones inherit it via committed `metadata.json` (`dolt_mode: embedded`) and `config.yaml`.
- Verified via a fresh-clone `bd bootstrap` round-trip.
- Lost Dolt audit history — acceptable here because that history was already stale.
- Bootstrap detection order is load-bearing: had to drop `refs/dolt/data` *and* set aside `.beads/backup` so bootstrap fell through to the current JSONL instead of cloning stale state.
- Hand-edits to the Sync/Landing-the-Plane sections landed inside `bd`'s managed markers (relocation tracked as timbers-069).

## ADR-5: Make the release's site-example regeneration non-fatal

**Status:** Accepted
**Date:** 2026-05-27

**Context:** `just release` regenerates site content via parallel `claude -p` calls. One of them (the decision-log) flaked on a transient LLM hiccup and aborted an entire version release.

**Decision:** A transient LLM failure in an ancillary step must not block shipping a tag. Wrapped `just examples` in the release recipe with `|| echo WARNING` so a failure warns and the release proceeds to commit/tag/push; examples regenerate separately.

**Consequences:**
- Releases are robust to transient LLM failures in ancillary content generation.
- Site examples can lag a release until regenerated separately; the printed warning is the only signal that they did.

## ADR-6: Pending detection — diagnose side-branch anchors and filter, not rewrite anchor selection

**Status:** Accepted
**Date:** 2026-05-28

**Context:** A user (Laura) hit apparent "pending scrambling" on v0.22.0 against a side-branch merge topology. The initial plan was a Phase 1 algorithm change (first-parent-aware latest-entry anchor selection). An independent reviewer in a separate context flagged that plan as a regression for the common "entry on a feature branch, then merge" workflow — already exercised by `TestBranchMerge_EntryOnBranch_NoneOnMain` — and supplied the exact counterexample.

**Decision:** Treat the friction as *opacity, not incorrectness*: the `docSet` algorithm was correct. Phase 0 shipped diagnostics only (`IsOnFirstParentLine`, `LatestAnchorOffFirstParent`, wired into `pending`/`doctor` JSON + human surfaces) plus a contract test capturing Laura's pathology — which passed on the unmodified code, the empirical signal that the algorithm change might be unnecessary. A second reviewer then traced the real root cause to `log --batch` naively picking `commits[0]`, which sometimes landed on a side-branch SHA; fixed with `pickBatchAnchor` (prefer the first commit on HEAD's first-parent line, fall back to `commits[0]`/legacy on git error). A remaining gap — the gate's `LogFirstParent` walking a structurally weird range, and `filterByRules` using `docSet` only for revert detection, never for direct membership — was closed by an `anchorShortCircuit` helper plus a direct `docSet[commit.SHA]` membership filter. Both fixes together drove Laura's class to zero pending; either alone was insufficient. Later refinements named the escape hatches: `pending` at count:0 notes an off-first-parent anchor, and `log --anchor` honors a single-commit range at zero pending with `--range` named as the explicit override. The Phase 1 algorithm change was confirmed unnecessary and dropped.

**Consequences:**
- Resolved the whole merge-topology friction class without an algorithm-level change, avoiding the regression-prone Phase 1.
- `documented` surfaced as a distinct classify reason in the `TIMBERS_DEBUG` trace and the auto-skip count.
- Three independent review passes (separate contexts) drove each correction; the decision was empirically gated on a passing contract test rather than designed up front.
- `pickBatchAnchor` still falls back to `commits[0]` for pure cross-agent-debt cases (no commit on the first-parent line); the off-first-parent diagnostic exists to surface that residual case rather than fix it.

## ADR-7: Refuse `timbers log` on a dirty tree over warn-and-proceed

**Status:** Accepted
**Date:** 2026-06-01

**Context:** v0.22.7 still warned-and-proceeded when the working tree was dirty. A field report from an agent in Constructured/osprey-strike observed two phantom entries in a single session: an aborted gated commit leaves staged changes in the index, then `timbers log` auto-commits an entry pathspec-scoped to the entry file only — the entry rides the old HEAD while the feature work stays unstaged. The existing warning text already told users to "commit first to avoid phantom entries"; the tool wasn't enforcing what it asked.

**Decision:** Replace the `printer.Warn` at `cmd/timbers/log.go:103` with a `UserError` return, guarded by `!flags.dryRun` so `--dry-run` stays usable for inspecting an entry mid-debug. The error names the likely trigger (aborted gated commit), the diagnostic (`git diff --cached`), and the `--dry-run` escape. Deliberately did **not** add `--allow-dirty`: the protocol has no case for logging uncommitted work, and the flag would reopen the exact footgun the refusal closes. Option B from the report (create the entry but skip the auto-commit) was rejected — the trigger *is* the gate aborting commits, so a deferred dirty entry would sit indefinitely and dirty entries compound.

**Consequences:**
- Enforces what the prior warning only requested; replaces the earlier warn-and-proceed behavior.
- `--dry-run` still works for inspecting an entry mid-debugging.
- No escape hatch for intentionally logging on a dirty tree — by design.
- Doesn't auto-detect the cross-agent debt that triggers the situation (filed as timbers-cs0; a gate-refusal hint about staged-changes-in-index as timbers-cwa).
