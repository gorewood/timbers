+++
title = 'Decision Log'
date = '2026-05-28'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

## ADR-1: Pending-detection response: diagnostic-first, defer algorithm change

**Status:** Accepted
**Date:** 2026-05-21

**Context:** Laura reported pending detection "scrambling" on v0.22.0 against a side-branch merge topology. The initial plan was a first-parent-aware latest-entry algorithm change. An independent reviewer in a separate context caught that the proposed algorithm would regress the common solo-dev-feature-branch workflow already covered by `TestBranchMerge_EntryOnBranch_NoneOnMain` — entries on a feature branch that later merges would lose their anchor entirely. A second reviewer dug into Laura's transcript and identified `--batch` picking a side-branch SHA in `commits[0]` as the actual culprit, not the docSet algorithm.

**Decision:** Reverse the algorithm-change plan. Phase 0 ships diagnostic-only: `IsOnFirstParentLine` helper, `LatestAnchorOffFirstParent` disambiguation from stale-anchor, and surfacing the topology in `pending` and `doctor`. The current code already passes the codified contract test — the friction was opacity, not incorrectness. Two follow-up patches landed targeted fixes: `pickBatchAnchor` prefers first-parent-line commits in `--batch` grouping, and `filterByRules` gained direct `docSet` membership filtering plus an off-first-parent gate fallback via `anchorShortCircuit`. The originally-planned algorithm rewrite was dropped.

**Consequences:**
- Diagnostic surface tells users when the topology, not the code, is the cause
- Solo-dev-feature-branch flows preserved; no regression
- `pickBatchAnchor` + gate fallback resolve Laura's class without algorithm-level changes
- Multi-reviewer, multi-context collaboration caught the regression a single context missed
- Residual cross-agent-debt cases (where `pickBatchAnchor` falls back to `commits[0]`) remain visible via the diagnostic surface but unresolved

---

## ADR-2: `msg:` subject globs over path-based skip for housekeeping commits

**Status:** Accepted
**Date:** 2026-05-27

**Context:** Release changelog commits needed to be excluded from pending detection. Path-based skipping (`CHANGELOG.md` + `site/`) was the first instinct but could not work: `filterByRules` requires ALL files in a commit to match a skip rule, and the release commit also touches `site/layouts/index.html` (the version badge). Path-skipping that file would have globally hidden legitimate landing-page edits.

**Decision:** Add a third rule kind to `.timbersignore`: `msg:` commit-subject globs, mirroring the existing `author:` glob design. Release commits skip via `msg:chore: changelog for v*`. Allowlisted `.timbersignore` in `.gitignore` so every clone and CI reads the same config — without this, the skip would have been silently clone-local.

**Consequences:**
- Precise targeting of housekeeping subjects without hiding files globally
- Generalizes to other subject-driven housekeeping (version bumps, release commits)
- Three rule kinds now: path glob, `author:`, `msg:` — parser and `classifyByIdentity` carry a third branch each
- `filterByRules` refactored to reuse `classifyByIdentity`, single-sourcing skip semantics with the debug trace

---

## ADR-3: Migrate beads to embedded Dolt + canonical `refs/dolt/data` sync

**Status:** Accepted
**Date:** 2026-05-27

**Context:** The repo was in a broken hybrid beads state: server-mode Dolt could not persist under the sandbox (it re-imported JSONL on every call), a stale CLI remote and three-week-old `refs/dolt/data` existed while steering claimed "no remote", and JSONL was the only current source of truth (server Dolt last modified Mar 22; `refs/dolt/data` topped out May 7). The osprey-strike repo had already migrated to the canonical embedded model.

**Decision:** Align to the canonical embedded Dolt + `refs/dolt/data` sync model. Rebuild `embeddeddolt/` from the trusted 60-issue JSONL via `bd bootstrap`, drop and re-seed `refs/dolt/data`, flip `export.auto`/`export.git-add` to false, untrack and gitignore `issues.jsonl`, rewrite AGENTS.md sync sections. Followed osprey-strike's validated migration doc as the playbook, adapted for the server→embedded delta osprey did not have.

**Consequences:**
- Works under the sandbox; embedded avoids the TCP dependency server mode required
- Loses Dolt audit history (rebuild-from-JSONL); acceptable because JSONL was authoritative anyway
- Bootstrap detection order matters — dropping `refs/dolt/data` AND setting aside `.beads/backup` was required so bootstrap fell through to current JSONL instead of stale state
- Fresh clones onboard via `bd bootstrap` with no JSONL gymnastics
- AGENTS.md hand-edits landed inside bd's managed markers; relocation tracked as separate work

---

## ADR-4: Document ack-for-rebase pattern over building content-matching relink

**Status:** Accepted
**Date:** 2026-05-27

**Context:** A feature request proposed five alternatives for handling rebased commits whose old SHAs no longer exist on the first-parent line but whose content lives in the new history. The requester's preferred option was "auto content-match relink" via byte-identical-diff or reflog heuristics. Audit found timbers has no content-matching today — `docSet` is literal SHAs — so this would be built from scratch. Naive byte-identical-diff is unsound (rebases shift context lines); reflog is absent on fresh clones and CI; a sound version would require patch-id stored at log time, which is a schema change.

**Decision:** Document the existing path instead of building new machinery. `ack` already satisfies the rebase-relink requirement — a structured record that counts as documented. Added a `RebaseRelinkGuidance` protocol section wired into both prime and MCP compose sites, sharpened pending/doctor ack hints with a copy-pasteable `rebased; content in <entry-id>` form, and added a DX-guide subsection. Inverted the requester's preference order to D→B→maybe-A per Gall's Law: ship the doc now, graduate to typed `ack --to <entry>` only if free-text linking proves insufficient, build patch-id gating only with real volume evidence.

**Consequences:**
- Zero new code surface for the immediate need
- Free-text reason field carries the link — readable by humans, not parseable by tools
- Distinct from the existing stale-anchor section (there the anchor is GC'd and self-heals; here it stays reachable and does not)
- Whether free-text linking holds up or pushes toward typed `ack --to <entry>` references remains an open observation
- Patch-id stored at log time deferred until real volume justifies the schema change

---

## ADR-5: Surface PATH-shadowing via `timbers doctor` rather than enforcing single-install

**Status:** Accepted
**Date:** 2026-05-27

**Context:** A stale `dev` go-install binary in mise's `GOBIN` shadowed the current `~/.local/bin` install during the v0.22.3 release. The git hook ran the stale binary, which did not recognize acks and silently blocked commits with a confusing "failed to commit ack file" error. The choice was between enforcing a canonical install location or detecting divergence at runtime and reporting it.

**Decision:** Add a "Binary Shadowing" check to `timbers doctor` that enumerates timbers binaries on PATH (deduped by resolved path) and warns when the first reports a different version token than a shadowed one. Compare version *tokens* (not full `--version` strings) so two builds of the same version with different commit/date suffixes do not false-warn. Skip `?` tokens to avoid false positives on broken shims. Also rewrote `WriteAck`'s commit-failure error to explain the staged-but-uncommitted state and point at upgrade + doctor + `git commit` recovery.

**Consequences:**
- Users discover stale-binary issues via a check they are already running, not by debugging mysterious gate failures
- Does not prevent shadowing — only surfaces it
- Escalation to install-time refusal remains open if shadowing keeps biting users

---

## ADR-6: `.timbersignore` discoverability over new filtering machinery

**Status:** Accepted
**Date:** 2026-05-27

**Context:** An agent had to research whether author-matching existed in `.timbersignore` and could silently write `author:dependabot[bot]` — a no-op glob, because `filepath.Match` treats `[..]` as a character class. The exemption lever existed but was invisible, and the footgun was undetectable until pending counts proved the rule was inert.

**Decision:** Make the existing surface discoverable rather than add new filtering machinery. Five touches in one commit: `doctor` lint flags literal-looking `[..]` classes in `author:`/`msg:` globs; `pending` prints a `.timbersignore` hint when non-empty; `pending --explain` classifies every commit via a new shared `pendingRange()` extracted from `getPendingCommits`; a `help timbersignore` topic and an `onboard` blurb document the three rule kinds and the canonical bot recipe.

**Consequences:**
- Research-and-trap becomes a one-liner for future agents
- `pendingRange()` extraction means the filtering gate and the new classifier share one range-resolution path; the 40K-line gate test suite is the regression net
- Four beads bundled into one commit because `pending.go` carries changes for two of them; a clean split would have needed `git add -p`
- Reinforces a pattern: surface existing levers before building new ones

---

## ADR-7: Release pipeline tolerates ancillary content-generation failures

**Status:** Accepted
**Date:** 2026-05-27

**Context:** `just release` regenerated site examples via parallel `claude -p` calls. A transient LLM hiccup on the decision-log generator aborted the entire release, even though the version tag and binary were ready to ship.

**Decision:** Wrap `just examples` in the release recipe with `|| echo WARNING` so a failure warns and the release proceeds to commit/tag/push. Examples regenerate separately.

**Consequences:**
- Transient LLM flakes can no longer block a version release
- Site-example freshness no longer guaranteed by the release pipeline; relies on the separate regeneration path being run
- Sets a precedent: ancillary tooling that can re-run independently should not be on the release critical path

---

## ADR-8: `just release` polls GH then installs the published build as a normal consumer

**Status:** Accepted
**Date:** 2026-05-27

**Context:** After tagging and pushing a release, the dev's local binary stayed at the pre-release build until manually reinstalled. Installing the *published* artifact — not the identical local build — would sanity-check the entire release pipeline (CI build, GH release, `install.sh`). The choice was between a cron follow-up or extending `just release` itself.

**Decision:** Append a bounded poll loop (`gh release view <tag> --jq 'assets|length'`, ~10m deadline, 15s interval) to `just release`. On publish, run `just install-release` and verify the installed `--version` contains the tag. On timeout, print a recovery hint since the release is already pushed. Cron was rejected because the install must happen on the dev's laptop, not in CI — a poll at the tail of `just release` has no moving parts.

**Consequences:**
- Every release tests the full consumer install path
- Dev's local binary stays current without a manual follow-up
- Adds seconds-to-minutes to `just release` while waiting for assets to publish
- Relies on `.goreleaser draft:false` so `latest` equals the just-pushed tag once assets appear

---

## ADR-9: `--anchor` honors single-commit documentation at 0 pending; `--range` is the escape hatch

**Status:** Accepted
**Date:** 2026-05-28

**Context:** The `--anchor` flag's name promises "use this anchor", but at 0 detected pending commits it refused to run — a contradiction that hit osprey-strike's off-first-parent/0-pending case. The user-visible refusal pointed at no clean recovery path.

**Decision:** When `--anchor` is set and pending is empty, `getLogCommits` falls back to `LogRange(anchor^, anchor)` to document the single commit. The refusal message for the remaining error cases names `--range` as the explicit escape hatch. Extracted commit-resolution helpers to `log_resolve.go` to stay under the file-length limit; a ref-aware mock and `TestLogAnchorBypassesZeroPending` lock the contract in.

**Consequences:**
- `--anchor` semantics now match its name in the 0-pending case
- `--range` becomes the canonical broader-escape-hatch in error messaging
- `--anchor` and `--range` overlap in capability (both can document a single commit); they diverge in intent rather than capability
- File-split into `log_resolve.go` driven by complexity budget

---

## ADR-10: Single source of truth for skip classification (`classifyByIdentity`)

**Status:** Accepted
**Date:** 2026-05-28

**Context:** v0.22.5 introduced the `msg:` glob rule (see ADR-2). A parallel inline identity chain in `countAutoSkipped` was not updated, causing `timbers status` to undercount housekeeping-skipped tallies when `msg:` globs matched. Both arch and code reviewers caught it independently. The fork: keep `countAutoSkipped`'s loop as a separate-but-symmetric structure (matching the historical `filterByRules` vs `ExplainPending` split), or collapse the visibility path onto the same classifier the gate uses.

**Decision:** Delegate identity classification in `countAutoSkipped` to the same `classifyByIdentity` chain `filterByRules` uses. The bug WAS that the loops drifted; preserving them as separate would invite the same class of bug on the next rule addition. Added `msg:` and `author:` regression cases to `TestCountInfraSkippedSinceLatest`.

**Consequences:**
- Visibility (status counts) and the gate (filtering decisions) cannot diverge on identity rules
- New rule kinds only need to be added in one place
- The remaining split between `filterByRules` and `ExplainPending` becomes harder to justify; future work may converge those as well
