+++
title = 'Decision Log'
date = '2026-05-27'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

## ADR-1: Keep full entry IDs and a customization hint in compact prime output over maximum terseness

**Status:** Accepted
**Date:** 2026-05-08

**Context:** The compact `timbers prime` output had shipped to cut the session-context payload, but a follow-up pass found it had traded away agent affordances to do so: it printed ellipsis-truncated entry IDs that couldn't be pasted into `timbers show`, hid that a custom `PRIME.md` workflow was in effect, and the JSON branch hardcoded `Mode=full` before the flag was even interpreted. The fork was whether terseness should win outright or whether specific affordances were worth their byte cost.

**Decision:** Favor affordance and honesty over raw size. Removed `compactEntryID` so the full `tb_<ts>_<sha>` ID prints — ~50 bytes per session across three entries — to restore paste-into-`timbers show` ergonomics. Emit a hint when `PRIME.md` is overridden rather than auto-merging its content into compact, keeping the view tight while flagging that customization exists. Set `Mode` from the flag before the JSON branch so JSON reports the mode that was actually requested.

**Consequences:**
- Compact output costs ~50 bytes more per session, buying resolvable, paste-ready IDs.
- Custom-workflow users get a discoverability signal without bloating the default compact view.
- The `Mode` fix was a structural bug repair — the hardcoded value would otherwise have leaked into MCP/JSON consumers.
- Does not surface full `PRIME.md` content in compact mode; only a hint. Users who want the full custom workflow must ask for full mode.

## ADR-2: Drop the Dolt remote in favor of a single JSONL sync channel

**Status:** Accepted
**Date:** 2026-05-08

**Context:** A previous agent had skipped committing auto-staged `.beads/issues.jsonl`, misreading a one-time bd 1.0.x schema flip (records gaining a `_type` discriminator and reordering) as corruption. The repo carried two potential sync channels — a Dolt SQL remote and the committed JSONL — but the Dolt remote had been unused since 2026-04-29, and `bd dolt push/pull` no-op without one. The choice was to keep the remote "just in case" or delete it to leave one unambiguous channel.

**Decision:** Delete the Dolt remote. The gitignored `.beads/dolt/` working DB plus the committed JSONL already give full reconstruction, so a second channel added confusion without capability. Documented embedded-only mode in `AGENTS.md`, instructed agents to commit bd's `_type`-prefixed JSONL rewrites without reverting, and added a `bd export | diff` drift-recovery recipe. A future remote can be re-added with one command if multi-machine federation is ever actually needed.

**Consequences:**
- One sync channel (committed JSONL) removes the misread that stranded bead state locally.
- `bd dolt push` drops out of the session-close workflow.
- Agents must trust bd's JSONL rewrites; `bd export -o /tmp/x && diff` is the verification path when the file looks wrong.
- Multi-machine Dolt federation is unavailable until the remote is re-added — a deliberate deferral, not a loss.

## ADR-3: Share one `hasActionablePending()` gate across hooks over inlining a narrow fix

**Status:** Accepted
**Date:** 2026-05-10

**Context:** In gorewood/vellum, commits touching only `.beads/issues.jsonl` triggered the post-commit hook's "document this" nudge, even though `timbers log` would refuse them as non-actionable — a contradictory signal for agents. Root cause: `runPostCommitHook` never checked `HasPendingCommits` before printing. The narrow fix was to inline that one check in the post-commit path; the broader option was to extract a single shared definition of "actionable."

**Decision:** Extract `hasActionablePending()` — combining `IsRepo`, `IsInteractiveGitOp`, `.timbers/` existence, storage construction, and `HasPendingCommits` — and route both pre-commit and post-commit hooks through it. The helper eliminates the divergence at its root rather than patching one symptom: pending, log, and both hooks now share one definition, and a future third hook (e.g. post-rewrite) inherits the gate for free.

**Consequences:**
- Hooks and `timbers log` can no longer disagree about what counts as undocumented.
- New hooks get correct gating without re-deriving it; the parity test reads as a single assertion.
- The test harness needed a `seedFile` escape hatch so `.timbersignore` can be baked into the initial commit — otherwise adding it as a separate commit makes *that* commit actionable.
- Does not change what "actionable" means; only unifies where it is decided.

## ADR-4: Scope the commit gate to the first-parent line over author-based attribution

**Status:** Accepted
**Date:** 2026-05-18

**Context:** Parallel agents share `.timbers/` through merges, and the old full-DAG gate blocked agent A's commit on agent B's undocumented commits — firing on the wrong actor. The osprey-strike feedback proposed author-based attribution (skip other authors' commits), but in this user's setup all agents commit under the same git identity, so an author filter would no-op.

**Decision:** Scope the gate to HEAD's first-parent line via `LogFirstParent` rather than author. First-parent is git-native and works regardless of identity configuration, which author attribution cannot here. A regression test then exposed a residual case — `git merge --no-ff` puts a merge commit on the first-parent line that still blocked — so a gate-only `dropEmptyFileChanges` filter was added: clean merges and `--allow-empty` commits have empty file lists and add no work to this branch's line. The display path deliberately keeps the conservative empty=unknown rule so `timbers pending` still surfaces them. `TIMBERS_SKIP_CROSS_AGENT_DEBT` remains an escape hatch for the narrow case where a merge commit itself touched source (conflict resolution).

**Consequences:**
- Agents are no longer gated on commits outside their first-parent line, fixing the parallel-agent false block.
- Gate and display paths now diverge intentionally — empty merges drop from the gate but remain visible in `pending` for awareness.
- The env-var bypass covers conflict-resolution merges that legitimately carry source changes.
- Genuinely distinct-identity use cases stay unaddressed by design; first-parent was chosen precisely because identities are *not* distinct in this setup.

## ADR-5: Compose protocol text from `internal/protocol` const sections over a single shared file

**Status:** Accepted
**Date:** 2026-05-20

**Context:** A push-before-log race in osprey-strike stranded an entry locally — the protocol said "commit, log" but didn't make the no-push-between rule prominent, and `timbers log` gave no warning despite having the data to detect it. Fixing the wording meant touching workflow text that two surfaces consume: the full PRIME doc in `cmd/timbers` and a subset in `internal/mcp`. The question was whether to keep that text in one file or structure it for multiple consumers.

**Decision:** Move shared protocol and stale-anchor sections into an `internal/protocol` package as composable `const` sections, assembled differently by each consumer — `cmd/timbers` builds the full PRIME doc, `internal/mcp` takes a subset. The text is fundamentally compositional (different consumers need different subsets), and const+const concatenation gives compile-time composition with no runtime concat overhead, versus a single-file blob that every consumer would have to over-include. Alongside, `IsPushedToUpstream` lets `timbers log` warn when the documented commit is already on `@{u}` but the entry isn't, and the protocol gained an explicit "never push between" callout.

**Consequences:**
- Protocol wording lives in one package; both surfaces stay consistent without duplicating prose.
- New sections plug into the same compose sites — later, `RebaseRelinkGuidance` (ADR-9) wires into both `prime_workflow.go` and `mcp/helpers.go` this way.
- A protocol sanity test must assert ordering positions, since a prose-only check would miss a reordered checklist.
- The push-before-log race is now surfaced as a warning rather than silently tolerated.

## ADR-6: Fold skip-authors into `.timbersignore` author globs over a dedicated config file

**Status:** Accepted
**Date:** 2026-05-20

**Context:** The impending q-redshifted autofix pipeline needed a way to skip bot-authored commits from pending detection. The first implementation added a dedicated `.timbers/skip-authors` file; the operator pushed back on file sprawl during review.

**Decision:** Fold author skipping into `.timbersignore` as `author:<glob>` lines, parsed by `classifyTimbersIgnoreLine` (the same parser yields both path and author rule shapes). This mirrors the `.gitignore` family and gives one source of truth for repo skip config rather than a second file. The operator's pushback reshaped the call from a new file to a prefixed line.

**Consequences:**
- Repo skip configuration lives in one file, idiomatic to anyone who knows `.gitignore`.
- Author globs support exact-name, email-domain, and (via prefix-wildcard) GitHub-bot patterns, working around `filepath.Match`'s character-class semantics.
- Edge case: the `author:` prefix collides with literal paths beginning `author:` — accepted as extremely rare, since `:` is forbidden in Windows filenames.
- Empty or malformed globs fall through the existing silent-drop path for malformed rules.

## ADR-7: Provide `timbers ack` as an honest skip-with-reason over fabrication or `--no-verify`

**Status:** Accepted
**Date:** 2026-05-20

**Context:** In osprey-strike, Laura saw merge SHAs in `pending` with no obvious next action. The two existing responses to a commit you don't want to document were both bad: fabricate a ledger entry for it, or bypass the gate with `git commit --no-verify`. Neither leaves an honest record of the deliberate skip.

**Decision:** Add `timbers ack` — a structured skip-with-reason stored under `.timbers/YYYY/MM/DD/ack_*.json` with `kind="ack"` under `timbers.devlog/v1`. The acked set threads through `filterCommits` parallel to `docSet` (one scan per pending check), so an acked commit clears pending the way a documented one does, but the record states it was deliberately skipped and why — honest accounting instead of a fake entry or a silent bypass.

**Consequences:**
- A commit can be cleared from pending with a recorded reason, no fabrication.
- Ack records are structured `timbers.devlog/v1` documents, not out-of-band state.
- The acked set adds a second membership check, folded into the existing single scan rather than a new pass.
- Becomes the basis for the rebase-relink workflow (ADR-9), where an ack counts as a structured "documented" record.

## ADR-8: Resolve merge-topology pending friction with diagnostics and targeted fixes, not a pending-detection algorithm rewrite

**Status:** Accepted
**Date:** 2026-05-21

**Context:** Laura's v0.22.0 friction read as "pending scrambling." The initial plan (Phase 1) was to make the pending-detection algorithm first-parent-aware when selecting the latest entry's anchor. An independent reviewer in a separate context caught that plan as a regression: a first-parent-aware latest entry would lose the anchor entirely for the common "entry on a feature branch, then merge" workflow already exercised by `TestBranchMerge_EntryOnBranch_NoneOnMain` — and gave the exact counterexample. The real question became whether the algorithm was wrong at all.

**Decision:** Don't rewrite the algorithm — the `docSet` algorithm was correct; the gaps were diagnostic opacity and filtering coverage. Shipped in phases:
- **Phase 0 (diagnostic only):** `IsOnFirstParentLine` + `LatestAnchorOffFirstParent` surface side-branch anchors in `pending`/`doctor`, and a contract test codifies Laura's pathology. That test passing on *unchanged* code was the empirical signal that the friction was opacity, not incorrectness.
- **Targeted fixes:** A second reviewer traced Laura's transcript to the actual culprit — `--batch` mode picking `commits[0]`, which sometimes landed on a side-branch SHA. `pickBatchAnchor` now returns the first commit on HEAD's first-parent line, falling back to `commits[0]` for the pure cross-agent-debt case. A gate fallback (`anchorShortCircuit`) plus a direct `docSet[commit.SHA]` membership check in `filterByRules` (which had used `docSet` only for revert detection) were both required *together* to get Laura's class to zero pending. The planned algorithm change was confirmed unnecessary.

Two reviewers reshaped this twice — first killing the algorithm change, then locating the true root cause in `--batch` anchor selection.

**Consequences:**
- Laura's merge-topology friction resolves with small, local fixes instead of a risky rewrite; the "entry on feature branch then merge" workflow is preserved.
- `--batch` entries now anchor on the first-parent line; pure cross-agent-debt cases fall back to `commits[0]` and surface via the off-first-parent diagnostic.
- Direct `docSet` membership surfaces "documented" as a distinct classify-reason in the `TIMBERS_DEBUG` trace and auto-skip count.
- The fix was later backed by a `pickBatchAnchorWith` pure-function variant with seven mixed-topology test cases, closing the post-hoc review gap that the original fix shipped without.
- Phasing let the v0.22.1 diagnostic surface deliver value on its own before the v0.22.2 fixes landed.

## ADR-9: Document the existing ack path for rebase-relinking over building content-matching

**Status:** Accepted
**Date:** 2026-05-27

**Context:** A five-proposal feature request asked timbers to handle rebased commits whose SHAs changed but whose content is preserved — an entry's `docSet` holds literal SHAs that no longer exist in history. The requester's preferred option (A) was automatic content-match relinking. But timbers has no content-matching today, so A would be built from scratch, and its naive byte-identical-diff/reflog form is unsound: rebases shift context lines, and reflog is absent on fresh clones and CI. A sound version needs patch-id stored at log time — a schema change.

**Decision:** Don't build content-matching now. `ack` (ADR-7) already satisfies the requirement — it produces a structured record that counts as documented — so the cheapest correct fix is documenting that path. Added a `RebaseRelinkGuidance` protocol section (wired into both prime and MCP compose sites per ADR-5), sharpened the `pending`/`doctor` ack hints from a generic `...` reason to a copy-pasteable `rebased; content in <entry-id>` form, and locked the section with a protocol test. Inverted the requester's preference order to D→B→maybe-A per Gall's Law: ship the doc now, graduate to a typed `ack --to <entry>` only if the free-text link bites, and build patch-id gating only with real volume evidence.

**Consequences:**
- Rebased-and-content-preserved commits get a documented, working relink path immediately, with no new code paths.
- The unsound auto-match (byte-diff/reflog) is avoided; the sound version (patch-id at log time) is deferred behind an evidence gate.
- Distinct from the stale-anchor case: there the anchor is GC'd and self-heals; here it stays reachable and does not, so it needs explicit handling.
- Follow-ups filed and deferred: typed `ack --to <entry>` (P3) and patch-id gating (P4).
