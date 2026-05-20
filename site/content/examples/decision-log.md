+++
title = 'Decision Log'
date = '2026-05-20'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

## ADR-1: First-parent gate scope for parallel-agent flows

**Status:** Accepted
**Date:** 2026-05-18

**Context:** Parallel agents working in the same repo share `.timbers/` via merges. The original full-DAG pending gate fired on agent A whenever agent B had undocumented commits anywhere in history, blocking the wrong actor. The team considered author-based attribution as one option, but all agents in the target setup commit under the same git identity, making author filters a no-op.

**Decision:** Scope the gate to the first-parent line via a new `LogFirstParent` git primitive and a `GetGatePendingCommits` path that applies a `gateStrict dropEmptyFileChanges` filter. The display path (`timbers pending`) keeps the conservative "empty file list = unknown" rule so clean merges still surface for awareness, but the gate ignores them since they add no work to this branch's first-parent line. A `TIMBERS_SKIP_CROSS_AGENT_DEBT` env var provides an escape hatch for the narrower case where a merge commit itself touches source (conflict resolution).

**Consequences:**
- Agent A is no longer blocked by agent B's undocumented work on side branches.
- Gate and display paths now have asymmetric semantics — gate is permissive on empty-file merges, display is conservative. Documented but easy to miss.
- Merge commits with non-trivial conflict-resolution content still require the env-var bypass.
- Author-based attribution is foreclosed as a primary strategy; first-parent is git-native and identity-agnostic.

## ADR-2: Unify post-commit and pending gating through a single actionable check

**Status:** Accepted
**Date:** 2026-05-10

**Context:** A real bug in `gorewood/vellum` showed the post-commit hook nudging agents to log `.beads/issues.jsonl`-only commits that `timbers log` itself would refuse — the hook and the log command had drifted to two different definitions of "actionable." Narrower fix: inline a `HasPendingCommits` check in `runPostCommitHook`.

**Decision:** Extract a `hasActionablePending()` helper combining `IsRepo`, `IsInteractiveGitOp`, `.timbers/` existence, storage construction, and `HasPendingCommits`. Both pre-commit and post-commit hooks route through it, eliminating the divergence at the root rather than patching one call site. The broader extraction was chosen specifically so that any future hook (e.g., post-rewrite) inherits the same gate for free.

**Consequences:**
- Hook output, `timbers pending`, and `timbers log` can no longer disagree about what counts as undocumented.
- Adding a new hook costs one helper call instead of re-deriving the gate logic.
- Test harness needed a `seedFile` escape hatch so `.timbersignore` could be baked into the initial commit (otherwise adding it as a separate commit makes the ignorefile itself actionable).

## ADR-3: Drop the Dolt remote; embedded JSONL is the single sync channel

**Status:** Accepted
**Date:** 2026-05-08

**Context:** The previous agent skipped committing auto-staged `.beads/issues.jsonl` because bd 1.0.x rewrote the file with `_type`-prefixed records and reordered entries, which read as corruption. The Dolt remote had been unused since 2026-04-29 and `bd dolt push/pull` no-op without one. Two channels (Dolt remote + JSONL) were creating the misread.

**Decision:** Remove the Dolt SQL remote entirely. `.beads/issues.jsonl` (committed) is the single source of truth; `.beads/dolt/` (gitignored) is local cache. Document the bd 1.0.x normalized JSONL shape in `AGENTS.md` so future agents recognize reordering as correct behavior, and add a `bd export | diff` drift-recovery recipe. Drop `bd dolt push` from the session-close workflow.

**Consequences:**
- One sync channel removes the "trust the rewrite" ambiguity.
- A Dolt remote can be re-added with one command if multi-machine federation is ever needed — no architectural lock-in.
- Agents must accept that bd may reorder and rewrite JSONL on every export; reverting auto-staged changes is now explicitly forbidden.

## ADR-4: Compact prime as default, full guide behind `--full`

**Status:** Accepted
**Date:** 2026-05-07

**Context:** The default `timbers prime` injection at session start was spending significant context on repeated coaching text that agents had already internalized within a project. The operational ledger safeguards (anchor state, pending list, stale-anchor warnings) were the load-bearing content; the prose around them was tax.

**Decision:** Ship a compact v2 renderer as the default for hook-driven session-start injection. Keep the full guide accessible behind `--full`/`guide` for first-time onboarding and explicit recall. Align stale-anchor prime output with `timbers pending` so the two surfaces tell the same story.

**Consequences:**
- Per-session context cost drops materially for established projects.
- New users no longer see the full coaching by default — discoverability of `--full` becomes important.
- Subsequent follow-ups (ADR-5) had to restore some affordances that the first cut over-compressed.

## ADR-5: Preserve agent affordances in compact prime output

**Status:** Accepted
**Date:** 2026-05-08

**Context:** The first compact prime cut (ADR-4) elided full entry IDs in favor of an ellipsis form, hid custom `PRIME.md` content entirely, and hardcoded `Mode=full` in the JSON branch. Real usage showed the ellipsis IDs were unresolvable when pasted into `timbers show`, custom workflows became invisible, and JSON consumers got dishonest mode reporting.

**Decision:** Restore full `tb_<ts>_<sha>` IDs (≈50 bytes/entry cost accepted for paste-into-`timbers show` ergonomics). `loadWorkflowContent` now returns whether `PRIME.md` was overridden, so compact emits a hint and JSON exposes `custom_workflow`. Set `Mode=full` only after flag interpretation so JSON honestly reports the requested mode. Align compact health truncation to 96 chars to match entries.

**Decision refines ADR-4:** Keep compact as default, but treat agent-resolvable IDs and custom-workflow visibility as non-negotiable affordances rather than optional flourishes.

**Consequences:**
- Slightly larger compact payload than the first cut, but still well under the full-guide cost.
- Hint-over-auto-merge for custom `PRIME.md` content keeps compact tight while flagging that customization exists — future renderers must respect the same separation.
- The Mode-field bug fix prevents the same class of "hardcoded before interpretation" leaks in MCP/JSON consumers.

## ADR-6: Default-skip lockfiles across major ecosystems

**Status:** Accepted
**Date:** 2026-05-02

**Context:** Isolated lockfile-only commits from manual conflict resolution or auto-rebase byproducts carry zero design intent, but appeared in pending lists across agent-driven repos. A lockfile change unaccompanied by a manifest change *can* be substantive (manual transitive override, security patch), so the call wasn't trivial.

**Decision:** Add six suffix patterns to `defaultSkipPatterns` covering npm, pnpm, yarn, Go, Cargo, and Bundler. Accept the residual risk because (a) the file-level filter only triggers on lockfile-ONLY commits — paired manifest changes still surface, and (b) `timbers status`'s `infra_skipped_since_entry` field gives a drift-detection surface. The long tail (poetry.lock, Pipfile.lock, composer.lock, mix.lock, pubspec.lock) can opt in via `.timbersignore`.

**Consequences:**
- Eliminates the most common source of low-signal pending noise.
- Substantive lockfile-only commits (security patches with no manifest bump) get silently skipped; the user must add an explicit entry or remove the pattern in `.timbersignore`.
- Establishes the pattern that defaults cover the dominant case and `.timbersignore` handles the long tail, rather than maintaining an exhaustive default list.

## ADR-7: `.timbersignore` at repo root, not inside `.timbers/`

**Status:** Accepted
**Date:** 2026-05-02

**Context:** The initial implementation placed `.timbersignore` inside `.timbers/`. The team caught this before any release tag — only one commit on main carried the inner location. Every other *ignore file in the ecosystem (`.gitignore`, `.dockerignore`, `.npmignore`) lives at the repo root.

**Decision:** Move `.timbersignore` to the repo root. Derive its location in `NewStorage` as `filepath.Dir(files.Dir())` since `FileStorage.Dir` is the `.timbers/` directory. Reject the alternative of dropping the leading dot (`.timbers/timbersignore`) because the `*ignore` convention *is* the leading dot. Reject embedding skip rules in a future `.timbers/config.yaml` as premature config-system construction for a single feature with a perfectly good format.

**Consequences:**
- Discoverability matches ecosystem expectations.
- Zero migration cost since no release shipped the inner location.
- Forecloses any future "all timbers config under `.timbers/`" purist design — accepted because ecosystem familiarity wins over directory-locality.

## ADR-8: Operator-voice coaching in prime, not per-template injection

**Status:** Accepted
**Date:** 2026-05-02

**Context:** A real custom-template devblog from a downstream worktree read with invisible-agent "I"-everywhere voice, tour-guide pacing, and no surfaced collaboration. The default devblog was producing hero-voice prose that erased the human-agent collaboration shape of how work actually gets done. Two placement options: bake the coaching into every relevant template, or put it once in prime and let templates inherit.

**Decision:** Add an `<operator-voice>` section to `defaultWorkflowContent` in `prime_workflow.go` with two habits (intent, collaboration) and a "don't fabricate" guardrail. Tune the devblog template separately with audience callout, fourth Collaborator voice, tone-calibration rows, and explicit hero-voice/tour-guide anti-patterns. Length budget moved 800 → 700.

**Consequences:**
- One canonical narrative-shape source rather than fragmented per-template instructions.
- Collaborator voice ships as default-on because the bias toward solo-developer narrative is the default failure mode.
- Templates that should *stay* neutral (changelog, release notes) are insulated — operator-voice lives at prime level, not in every template.
- "Don't fabricate" guardrail accepts that thin entries should produce thin narratives, not invented collaboration moments.

## ADR-9: Per-artifact template tuning, no uniform operator-voice injection

**Status:** Accepted
**Date:** 2026-05-02

**Context:** After tuning the devblog (ADR-8), the obvious next move was applying operator-voice across all six remaining builtin templates. Reviewer pushback: changelogs and release notes are deliberately neutral artifacts where readers want facts at a specific abstraction level. Performative voice would miss what those readers need.

**Decision:** Diagnose each template separately rather than applying a uniform treatment. Changelog gets Added/Changed disambiguation and a strict exclusion list. Decision-log gets Status/Date/supersession fields and operator-intent in Context. PR description gets size-adaptive guidance, collaboration callout, and explicit test-plan honesty rules. Release notes get a "user-observable" filter, "what should I do" for breaking changes, and anti-fabrication. Sprint-report gets Friction/Carry-overs and Highlights criteria. Standup gets Asks-for-help and time-burn texture.

**Decision refines ADR-8:** Operator-voice belongs in templates whose readers (reviewers, future maintainers, PMs, teammates) calibrate based on collaboration shape — not in neutral artifacts.

**Consequences:**
- The Status field on ADRs is the structurally most significant change — without it, decision logs accumulate without any way to capture reversals.
- Test-plan honesty in PR descriptions explicitly names "Tests pass" as a fabrication risk; the cheapest counter is naming the failure mode directly.
- A proposed structured `agent_involvement: high|medium|low` entry field was rejected as too much taxonomy that would encourage fabrication.
- Templates now diverge in shape — fewer cross-template invariants to rely on when refactoring the rendering layer.

## ADR-10: Skip-authors via `.timbersignore` `author:` lines, not a separate file

**Status:** Accepted
**Date:** 2026-05-20

**Context:** The osprey-strike friction report flagged merge SHAs appearing in pending with no obvious next action, and the impending `q-redshifted` autofix pipeline needed a bot-author skip path. Initial implementation: a dedicated `.timbers/skip-authors` file. Pushback: file sprawl, two sources of truth for repo skip config.

**Decision:** Fold author globs into `.timbersignore` as `author:<glob>` lines parsed by `classifyTimbersIgnoreLine`, which yields both rule shapes from one parser. Mirrors the `.gitignore` family convention of one ignorefile carrying multiple rule types. Document the edge case explicitly: the `author:` prefix collides with literal paths starting with `author:` (extremely rare since `:` is forbidden in Windows filenames).

**Consequences:**
- Single source of truth for repo skip config.
- Documentation for `.timbersignore` must now cover both path globs and author globs; examples needed for exact-name, email-domain, and the GitHub-bot prefix-wildcard workaround for `filepath.Match`'s character-class semantics.
- Literal `author:`-prefixed paths cannot be ignored without escaping — acceptable given the rarity.

## ADR-11: `timbers ack` for honest skip-with-reason

**Status:** Accepted
**Date:** 2026-05-20

**Context:** Agents facing undocumented commits they shouldn't log (third-party merges, mechanical rewrites, work outside their scope) had two bad options: fabricate an entry to clear the gate, or bypass with `--no-verify`. Neither leaves an honest trail.

**Decision:** Add `timbers ack` to record an explicit skip-with-reason, stored under `.timbers/YYYY/MM/DD/ack_*.json` with `kind=ack` under the `timbers.devlog/v1` schema. Thread `AckedSet` through `filterCommits` in parallel to `docSet` so a single pending-check scan covers both documented and acknowledged commits.

**Consequences:**
- Agents get a third option that's honest about the decision and persists the reasoning.
- The `kind` discriminator on stored records means consumers must now distinguish entries from acks; export/query paths need to filter appropriately.
- Reviewers can inspect `ack_*.json` to validate skip decisions instead of seeing unexplained `--no-verify` bypasses.

## ADR-12: Push-before-log detection via upstream comparison

**Status:** Accepted
**Date:** 2026-05-20

**Context:** A push-before-log race in osprey-strike stranded a ledger entry locally — the protocol said "commit, log" but didn't bold the no-push-between-them rule, and `timbers log` gave no signal despite having all the data needed to detect the race.

**Decision:** Add `IsPushedToUpstream` in `internal/git` that checks the docs anchor against `@{u}` after `WriteEntry`. `printer.Warn` fires when the documented commit is already on upstream but the entry isn't yet. Rewrite the protocol text with an explicit "never push between" callout. Move shared protocol/stale-anchor sections into an `internal/protocol` package, composed via const+const concatenation by both `cmd/timbers` (full PRIME doc) and `internal/mcp` (subset).

**Consequences:**
- Stranded-entry race now produces an audible warning rather than silent loss.
- Protocol text composition uses compile-time concatenation — no runtime cost, but consumers can't dynamically reshape the text.
- Initial single-file approach to protocol text was rejected because different consumers need different subsets; const composition gives that without runtime concat overhead.
