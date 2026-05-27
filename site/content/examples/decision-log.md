+++
title = 'Decision Log'
date = '2026-05-27'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

## ADR-1: Scope the commit gate to the first-parent line over author attribution

**Status:** Accepted
**Date:** 2026-05-18

**Context:** Parallel agents (the osprey-strike setup) share one `.timbers/` directory through merges, and all commit under the same git identity. The full-DAG gate was blocking agent A from committing because of agent B's still-undocumented commits — firing on the wrong actor.

**Decision:** Scope the gate to HEAD's first-parent line (`LogFirstParent`) instead of the full DAG. Author-based attribution (option 1 in the original request) was considered and dropped because every agent commits as the same identity, so an author filter would no-op; first-parent is git-native and works regardless of identity config. A regression test then exposed a residual case — `git merge --no-ff` puts a merge commit M on the first-parent line — so `dropEmptyFileChanges` was added as a gate-only filter, since clean merges have empty file lists from `git diff-tree`. `TIMBERS_SKIP_CROSS_AGENT_DEBT` remains as an escape hatch for the narrow case where the merge commit itself touched source during conflict resolution.

**Consequences:**
- The gate no longer blocks an agent on a teammate's undocumented work merged into the branch.
- The display path (`timbers pending`) keeps the conservative empty=unknown rule, so empty merges still surface there for awareness — display and gate now diverge intentionally.
- Identity-based skip is off the table for this setup; any future per-author logic would need distinct git identities to mean anything.
- Conflict-resolution merges that touch source still require the env-var escape hatch — they are not auto-detected.

## ADR-2: Warn when a commit is pushed before it is logged

**Status:** Accepted
**Date:** 2026-05-20

**Context:** In osprey-strike a push-before-log race stranded an entry locally. The protocol said "commit, then log" but never emphasized that nothing should be pushed in between, and `timbers log` gave no signal even though the upstream state needed to detect the race was already available.

**Decision:** Make the commit→log→push ordering explicit with a "never push between them" callout, and have `timbers log` actively detect the race rather than only document against it. `IsPushedToUpstream` checks the documented commit against `@{u}` after `WriteEntry`, and `printer.Warn` fires when the documented commit is already upstream but its entry is not. The cheap correct fix was to read data that was "sitting right there" instead of adding new tracking.

**Consequences:**
- A stranded-entry situation now produces a visible warning instead of silent local drift.
- The check depends on a configured upstream ref; detached-HEAD or upstream-less states won't trigger it.
- It warns rather than blocks — ordering remains advisory, not enforced.

## ADR-3: Compose protocol text from package consts in `internal/protocol` over a single shared blob

**Status:** Accepted
**Date:** 2026-05-20

**Context:** The commit→log→push protocol text needed a single home, but two consumers render it differently — `cmd/timbers` emits the full PRIME document while `internal/mcp` needs only a subset.

**Decision:** Extract the protocol and stale-anchor text into an `internal/protocol` package as separate `const`s that each consumer concatenates as needed. A single-file approach (inspired by the same session's move to fold skip-authors into `.timbersignore`) was considered and rejected: the workflow text is fundamentally compositional — different consumers need different subsets — and `const`+`const` concatenation gives compile-time composition with no runtime concat cost.

**Consequences:**
- Both consumers draw from one source, so wording can't drift between PRIME and MCP output.
- Adding a new consumer means assembling the subset it needs from const pieces rather than copy-editing prose.
- Section ordering is now verifiable: a protocol sanity test asserts checklist position, because a prose-only check would miss a reordered checklist.

## ADR-4: Introduce `ack` for honest skip-with-reason over fabricated entries or `--no-verify`

**Status:** Accepted
**Date:** 2026-05-20

**Context:** osprey-strike's impending autofix pipeline (q-redshifted) plus ordinary housekeeping commits created a class of commits that legitimately don't warrant a full what/why/how entry. The only existing ways to clear them were fabricating an entry or bypassing the gate with `--no-verify`.

**Decision:** Add `timbers ack`, which records a structured `kind=ack` document under `.timbers/YYYY/MM/DD/` that counts as documented. `AckedSet` threads through `filterCommits` parallel to `docSet` in the same single scan per pending check. This gives an honest skip-with-reason instead of a fabricated rationale or an untracked bypass.

**Consequences:**
- Commits can be cleared from pending with a recorded reason, preserving an audit trail that `--no-verify` would not.
- Ack records are first-class ledger files, so they sync and replicate like entries.
- A new commit classification (acked) now sits alongside documented/skipped, surfaced as a distinct classify-reason in the `TIMBERS_DEBUG` trace.

## ADR-5: Consolidate commit-skip config in `.timbersignore` via `author:` globs over a dedicated skip-authors file

**Status:** Accepted
**Date:** 2026-05-20

**Context:** The autofix pipeline needed to skip bot-authored commits. The first implementation used a dedicated `.timbers/skip-authors` file.

**Decision:** After the operator pushed back on file sprawl, fold author-skip into `.timbersignore` as `author:<glob>` lines parsed by `classifyTimbersIgnoreLine` — the same parser yields both path and author rule shapes. This mirrors the `.gitignore` family and gives one source of truth for repo skip config. The operator's sprawl objection is the moment that reshaped this from a new file into a rule kind.

**Consequences:**
- One file holds all skip configuration; there's no second config surface to discover or keep in sync.
- The `author:` prefix collides with literal paths beginning `author:` — accepted as extremely rare, since `:` is forbidden in Windows filenames.
- Glob semantics create a footgun: `author:dependabot[bot]` reads `[bot]` as a character class and silently matches nothing, so GitHub bots need a prefix-wildcard workaround — later guarded by a `doctor` lint that flags literal-looking `[..]` classes in `author:`/`msg:` globs.

## ADR-6: Side-branch anchors — diagnose and target the real gap over rewriting anchor selection

**Status:** Accepted
**Date:** 2026-05-21

**Context:** Laura hit friction on v0.22.0 that read as "pending scrambling." The author's first plan was to make the latest-entry anchor selection first-parent-aware — an algorithm change.

**Decision:** Ship diagnostics first and reject the algorithm rewrite. An independent reviewer caught the original plan as a regression for the common solo-dev "entry on feature branch, then merge" flow (already exercised by `TestBranchMerge_EntryOnBranch_NoneOnMain`): a first-parent-aware latest entry would have lost the anchor entirely. v0.22.1 shipped Phase 0 — pure diagnostics (`IsOnFirstParentLine`, `LatestAnchorOffFirstParent`) plus a regression contract test that current code already passed, which was the empirical signal that the docSet algorithm was correct and the friction was opacity, not incorrectness. A second reviewer then traced Laura's transcript to the actual culprits, fixed in v0.22.2: (1) `--batch` picked `commits[0]`, which could be a side-branch SHA — `pickBatchAnchor` now prefers the first commit on HEAD's first-parent line, falling back to `commits[0]` only for the pure cross-agent-debt case; (2) `filterByRules` used `docSet` only for revert detection, never for direct "is this commit documented" membership — adding that membership check plus an `anchorShortCircuit` gate fallback drives Laura's class to zero pending. Neither fix alone was sufficient.

**Consequences:**
- The core docSet algorithm is unchanged; fixes are localized to batch-anchor selection and filtering coverage.
- `pickBatchAnchor` resolves HEAD best-effort and degrades to legacy `commits[0]` on git error rather than failing the batch run.
- "documented" is now a distinct classify-reason in the `TIMBERS_DEBUG` trace and the auto-skipped count.
- The residual pure-cross-agent-debt case (no group commit on the first-parent line) still falls back to `commits[0]`, surfaced by the v0.22.1 diagnostic rather than resolved.

## ADR-7: Skip release-changelog commits by commit-subject glob (`msg:`) over path matching

**Status:** Accepted
**Date:** 2026-05-27

**Context:** Release commits kept appearing in pending and needed an automatic skip. The obvious approach was a path-based rule on `CHANGELOG.md` and `site/`.

**Decision:** Add a `msg:<glob>` rule kind that skips commits by subject, and skip releases via `msg:chore: changelog for v*`. Path-based skip can't work: `filterByRules` requires *every* file in a commit to match a rule, and the release commit also touches `site/layouts/index.html` (the version badge) — a path rule broad enough to skip it would also hide legitimate landing-page edits. The `msg:` design mirrors the existing `author:` glob as a third `ignoreLine` kind on the same parse/thread path, and `filterByRules` was refactored to reuse `classifyByIdentity` so skip semantics are single-sourced with the debug trace. `.timbersignore` was also found to be gitignored by the repo's allowlist `.gitignore` and explicitly re-allowed, so the rule reads the same in every clone and CI.

**Consequences:**
- Release commits are skipped precisely without suppressing real edits to files they happen to touch.
- The mechanism generalizes to other housekeeping subjects (version bumps, release commits).
- Skip config is now genuinely shared; had the gitignore issue gone unnoticed, the rule would have been silently clone-local.

## ADR-8: Document `ack` as the rebase-relink path over building automatic content-matching

**Status:** Accepted
**Date:** 2026-05-27

**Context:** A five-proposal feature request asked for rebase-relink — re-associating an entry whose anchor SHA changed under rebase — with the requester preferring automatic content-matching (proposal A).

**Decision:** Document that `ack` already satisfies the requirement (a structured record that counts as documented) rather than build new machinery, and invert the requester's preference order to D→B→maybe-A on Gall's Law grounds. timbers has no content-matching today — `docSet` is literal SHAs — so proposal A would be built from scratch, and its naive byte-identical-diff/reflog form is unsound: rebases shift context lines, and reflog is absent on fresh clones and CI. A sound version needs a patch-id stored at log time, i.e. a schema change. So: ship the doc now (a `RebaseRelinkGuidance` protocol section in `prime` and MCP, plus a copy-pasteable `rebased; content in <entry-id>` ack reason), graduate to a typed `ack --to <entry>` only if the free-text link bites, and build patch-id gating only with real volume evidence.

**Consequences:**
- The friction is resolved with documentation and a const — no schema change.
- Rebase-relink stays a manual, free-text convention; there is no structural link or validation yet.
- This is distinct from the existing stale-anchor case: there the anchor is GC'd and self-heals, here it stays reachable and does not — which is why the two needed separate guidance.
- Patch-id storage and typed relink are explicitly deferred, gated on evidence that the manual path is insufficient.

## ADR-9: Detect shadowed `timbers` binaries with a `doctor` check using version-token comparison

**Status:** Accepted
**Date:** 2026-05-27

**Context:** During the v0.22.3 release a `dev` go-install binary in mise's GOBIN shadowed the current `~/.local/bin` build on PATH, so `timbers ack` wrote its record but the auto-commit was gate-blocked by the stale hook — surfacing only as a confusing "failed to commit ack file."

**Decision:** Add a `doctor` "Binary Shadowing" check that enumerates `timbers` binaries on PATH (deduped by resolved path) and warns when the first reports a different version token than a shadowed one; `WriteAck`'s commit-failure path now explains the staged-but-uncommitted state and points at upgrade + `doctor` + `git commit` recovery. Surfacing the divergence proactively was judged to beat debugging the confusing downstream error. The comparison uses the version *token*, not the full `--version` string, so two builds of the same version with different commit/date suffixes don't false-warn, and `?` tokens are skipped to avoid false positives on broken shims.

**Consequences:**
- A stale shadowing binary is surfaced proactively instead of through a misleading commit error.
- The check is heuristic on the version token; same-version-different-build shadows are intentionally not flagged.

## ADR-10: Migrate beads to embedded Dolt with canonical `refs/dolt/data` sync

**Status:** Accepted
**Date:** 2026-05-27

**Context:** The repo was in a broken hybrid beads state: server-mode Dolt couldn't persist under the sandbox (it re-imported JSONL on every call), and a stale CLI remote plus a three-week-old `refs/dolt/data` existed while the steering docs claimed "no remote."

**Decision:** Migrate to embedded Dolt + canonical `refs/dolt/data` sync, matching osprey-strike's validated model — judged the only coherent fix for the hybrid state. JSONL was the only current source of truth (server Dolt last modified Mar 22, `refs/dolt/data` topped out May 7), so rebuilding from the trusted 60-issue JSONL was safe despite losing Dolt audit history. Mechanics: stop/kill the server, set aside the stale DB and backup, flip `metadata.json` to embedded, `bd bootstrap` to rebuild, drop and re-seed `refs/dolt/data`, set `export.auto`/`git-add` false, untrack and gitignore `issues.jsonl`. osprey-strike's migration doc was the playbook, adapted for the server→embedded delta it didn't cover.

**Consequences:**
- Bead state persists correctly under the sandbox; no more empty-DB re-imports on every call.
- Dolt audit history before the rebuild is lost — accepted because JSONL was the only live source anyway.
- Bootstrap detection order matters: `refs/dolt/data` had to be dropped *and* `.beads/backup` set aside so bootstrap fell through to the current JSONL instead of cloning stale state.
- Hand-edits to the Sync/Landing-the-Plane sections landed inside bd's managed markers; relocating them is tracked as follow-up (timbers-069), not resolved here.

## ADR-11: Verify releases by polling and installing the published artifact over a cron follow-up

**Status:** Accepted
**Date:** 2026-05-27

**Context:** After `just release` tagged and pushed, the local binary stayed stale and nothing exercised the published pipeline end to end. The operator's initial framing was a cron follow-up.

**Decision:** Append a bounded poll loop to `just release` (`gh release view <tag> --jq` assets length, ~10m deadline, 15s interval) that, once assets appear, runs `just install-release` and verifies the installed `--version` contains the tag. Installing the official published artifact — not the identical local build — sanity-checks the whole pipeline (CI build, GH release, `install.sh`). Poll-and-install at the tail of `release` was chosen over the operator's cron idea because it has no moving parts and CI can't install on the dev's laptop anyway, so it has to be local. `install-release` stays the real consumer path (vs pinning `VERSION`) per the operator's "as if a normal consumer" intent.

**Consequences:**
- Every release exercises the consumer install path, catching pipeline breakage immediately.
- On timeout the recipe prints a recovery hint rather than failing — the tag is already pushed, so the release isn't blocked.
- Relies on `.goreleaser draft:false` so `latest` equals the just-pushed tag once assets publish.

## ADR-12: Make `just release` site-example regeneration non-fatal

**Status:** Accepted
**Date:** 2026-05-27

**Context:** A transient flake in the parallel `claude -p` site-content generation (the decision-log example) aborted an entire version release — an ancillary content step took down the core ship-a-tag flow.

**Decision:** Wrap the `just examples` step in the release recipe with `|| echo WARNING` so a failure warns and the release proceeds to commit/tag/push; examples regenerate separately. A transient LLM hiccup must not block shipping a tag.

**Consequences:**
- Releases no longer abort on ancillary content-generation flakes.
- Site examples can lag the freshly shipped tag until regenerated separately — accepted as the cost of decoupling.
