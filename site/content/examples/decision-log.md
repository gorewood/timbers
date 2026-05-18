+++
title = 'Decision Log'
date = '2026-05-17'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

## ADR-1: Per-repo `.timbersignore` for skip-rule extension, placed at repo root

**Status:** Accepted
**Date:** 2026-05-02

**Context:** The hardcoded default skip list (built into the binary) could not cover every repo's housekeeping needs — vendor directories, lockfiles, custom dependency paths vary by ecosystem. Operators needed a way to opt-out additional patterns without code changes, while keeping the defaults safe and unconfigurable. The initial location for the config file (`.timbers/.timbersignore`) was a dotfile inside a dot-directory — nonstandard relative to `.gitignore`, `.dockerignore`, `.npmignore`, all of which sit at the repo root.

**Decision:** Add a `.timbersignore` file using the existing skipRule grammar from the built-in defaults, loaded once at `NewStorage` construction and merged with `compiledDefaultSkipRules`. Place the file at the repo root rather than inside `.timbers/`, deriving the path in `NewStorage` as `filepath.Dir(files.Dir())`. Rejected TOML (would have added a parser dep for one feature) and doublestar globs (`**` support unnecessary when prefix+suffix covers vendor/, *.lock, third_party/). Rejected dropping the leading dot (`.timbers/timbersignore`) because the `*ignore` convention IS the leading dot.

**Consequences:**
- Per-repo configuration extends defaults without modifying the binary.
- File discoverability matches every other ignore-file in the ecosystem.
- Loader errors fall back to defaults silently — a malformed `.timbersignore` cannot block enforcement (would invert the gate).
- `NewStorage` now does I/O at construction time (opens the ignorefile); documented in godoc rather than refactoring to lazy-load, on the grounds that lazy paths hide state behind first-use side effects.
- No `**` glob support; users with deeper structural needs must wait for explicit demand.

---

## ADR-2: Surface infra-skipped commit count in `timbers status` as visibility, not enforcement

**Status:** Accepted
**Date:** 2026-05-01

**Context:** Relaxing skip rules (per ADR-1) created a new failure mode: an over-broad `.timbersignore` could silently drop substantive commits from pending detection without the operator noticing. The team needed a feedback channel before shipping the relaxation, but did not want to build anti-abuse machinery — just enough visibility to catch drift.

**Decision:** Add `Storage.CountInfraSkippedSinceLatest` walking `(latestAnchor..HEAD)` through the same skipRule set used for pending. `timbers status --verbose` adds a key/value line; JSON always includes `infra_skipped_since_entry` per snake_case convention. Reviewer requested `--verbose`-only for human display to avoid surprising latency — complied. Errors and stale-anchor cases collapse to 0 silently.

**Consequences:**
- Operators can detect over-skipping by reading status output, without enforcement firing.
- JSON consumers always get the field; human readers see it only when asking for verbose output.
- No correctness signal — a high count is informational, not an error condition.
- Required splitting `skipcount.go` out of `storage.go` to stay under 350-line file budget.

---

## ADR-3: Auto-skip reverts of already-documented commits from pending detection

**Status:** Accepted
**Date:** 2026-05-01

**Context:** Revert commits add no new design intent when the reverted commit is already documented — the original entry IS the audit trail. Requiring a fresh entry for every revert inflates ledger noise without capturing information that isn't already present.

**Decision:** Detect `Revert "..."` subject + `This reverts commit <sha>` body trailer via regex in `internal/ledger/revert.go`, cross-reference the SHA against every entry's `Workset.Commits`, and skip the revert when documented. Integrated alongside infrastructure filtering in `filterLedgerOnlyCommits` via a shared `filterByRules` helper. All failure modes (squashed reverts, GC'd SHAs, undocumented originals, multi-revert with mixed coverage) fall back to "normal pending" — for multi-revert, any undocumented SHA keeps the revert pending. A `--revert` flag for new entries was deliberately deferred — auto-skip alone covers the common case.

**Consequences:**
- Reverts of documented work no longer require fresh entries.
- Conservative fallback means undocumented reverts still surface in pending.
- SHA matching uses prefix match for short-SHA tolerance; minimum length raised to 12 chars (per code-review feedback) to make collisions vanishingly unlikely.
- Manual `--revert` flow remains absent; will need a separate bead if real demand emerges.

---

## ADR-4: Default-skip lockfiles across six major ecosystems

**Status:** Accepted
**Date:** 2026-05-02

**Context:** Isolated lockfile-only commits (manual conflict resolution, auto-rebase byproducts) carry zero design intent. The existing file-level filter already keeps lockfile changes pending when paired with a manifest change, making the structural addition safe. A survey of osprey-strike commits grounded the choice in real-world friction patterns.

**Decision:** Add six suffix patterns (`*package-lock.json`, `*pnpm-lock.yaml`, `*yarn.lock`, `*go.sum`, `*Cargo.lock`, `*Gemfile.lock`) to `defaultSkipPatterns`. The existing `skipSuffix` matcher handles them with no matcher changes. Excluded the long tail (poetry.lock, Pipfile.lock, composer.lock, mix.lock, pubspec.lock) — the six included cover the bulk of agent-driven repos, and the long tail can opt in via `.timbersignore` (per ADR-1).

**Consequences:**
- Lockfile-only commits stop triggering pending nudges in the common case.
- A genuinely substantive lockfile-only change (manual transitive override, security patch) becomes invisible to the gate — accepted because (a) the file-level filter only triggers when no other files change and (b) `timbers status --verbose` (per ADR-2) catches drift.
- Long-tail ecosystems must configure manually.

---

## ADR-5: Tune draft templates per-artifact, rejecting uniform operator-voice

**Status:** Accepted
**Date:** 2026-05-02

**Context:** The default devblog template was producing dry, hero-voice prose that erased the human-agent collaboration shape of how work actually gets done. The fix could have been applied uniformly across all seven builtin templates, but each artifact has different signal needs — readers of a changelog or release notes want neutral facts at a specific abstraction level, not operator commentary.

**Decision:** For the devblog template, add an Audience section, a fourth Collaborator voice, tone-calibration rows, and anti-patterns for hero/tour-guide voice. Add a `<operator-voice>` section to `prime_workflow.go` with two habits (intent + collaboration) and a "don't fabricate" guardrail. For the remaining six templates, diagnose per-artifact and tune individually: changelog gets Added/Changed disambiguation + strict exclusion list; decision-log gets Status/Date/supersession + operator-intent in Context; pr-description gets size adaptation + collaboration callout + test-plan honesty; release-notes gets strict "user-observable" filter + breaking-change instructions; sprint-report gets Friction/Carry-overs + Highlights criteria; standup gets Asks for help + time-burn texture. Explicitly rejected injecting operator-voice uniformly. Also rejected a structured `agent_involvement: high|medium|low` field — too much taxonomy, encourages fabrication; the existing notes field handles it when used well.

**Consequences:**
- Each template now produces signal calibrated to its reader (reviewer, end user, PM, teammate).
- The ADR `Status` field is the most structurally significant change — without it, decision logs accumulated with no scaffolding for reversals.
- Test-plan honesty in PR description pushes back on the "Tests pass" fake-verified-claim failure mode.
- Templates that share concerns (collaboration awareness in PR/ADR/sprint/standup) now repeat similar guidance — accepted over consolidating into prime workflow.
- Length budget for devblog moved 800 → 700; expect shorter generated posts.

---

## ADR-6: Default PR descriptions to timbers entries when present

**Status:** Accepted
**Date:** 2026-05-02

**Context:** Agents opening PRs without explicit body instructions were defaulting to ad-hoc summaries that drift from the session's documented intent. When entries exist for the branch, the `pr-description` template is the natural source-of-truth and produces a more consistent body than ad-hoc agent reasoning. The risk was over-aggression — forcing template use when the operator wanted a one-line PR body for trivial work.

**Decision:** Add a `<pr-authoring>` section to `defaultWorkflowContent` in `prime_workflow.go` listing when this default applies (open-PR ask without dictated body, entries exist in branch range), when it does not (operator dictates body, no entries, trivial PR), the recommended pipe-through-claude flow, and an empty-section signal — missing Design Decisions means thin entries, not a license to fabricate. PR-authoring guidance lives in prime workflow rather than the pr-description template itself because it concerns WHEN to invoke the template, not HOW the template works.

**Consequences:**
- PRs default to the ledger when it has content, drift-free.
- Operators with explicit body instructions retain control.
- The empty-section signal converts a fabrication risk (LLM filling Design Decisions with vague restatements) into a feedback loop (entries were thin; beef them up or accept the section doesn't apply).
- Trivial PRs (typo fixes) are explicitly carved out — no machinery forces template use.

---

## ADR-7: Treat soft-content fabrication as the same risk as test-plan fabrication

**Status:** Accepted
**Date:** 2026-05-02

**Context:** An independent codex-review pass on the seven builtin templates flagged 14 issues. The unifying observation: fabrication risk had been treated as binary (test-plan invention, metric invention) when the softer kind (emotion, theme, vague benefit) is the same failure mode at lower visibility. The team needed to harden against soft fabrication without forcing templates into stilted compliance.

**Decision:** Triage the 14 review points and accept 9 hardening clauses: changelog header rule; ADR consequences tightening + date semantics; devblog emotion anti-fabrication + dropped word floor; PR tiny-PR risk override + design-decision collaboration; release-notes incomplete-migration handling + soft-benefit tightening; sprint-report theme threshold + chronology-last-resort; standup stale-next-steps. Reject 2: weakening devblog literary scaffolding (the named voices like Atwood/Fowler/Orosz do directional work the LLM reads as guidance, not mandatory mood) and merging extension-author audience. Bump template versions only where changes are material. Treat the codex agent as sparring, not adjudicating — being explicit about rejections is part of the value.

**Consequences:**
- Every accepted patch is a hardened anti-fabrication clause for a specific soft-content target.
- Literary scaffolding (named voices) preserved in devblog despite review pressure.
- Templates carry version bumps where they materially changed, leaving unchanged templates at prior versions.
- The framing — fabrication risk as a spectrum, not binary — becomes part of how future template work gets reviewed.

---

## ADR-8: Compact `timbers prime` output by default; full guide behind `--full`

**Status:** Accepted
**Date:** 2026-05-08

**Context:** Default prime injection was spending session context on repeated coaching content that established agents had already absorbed. The session-start payload needed to shrink, but compact output could not strip operational safeguards (ledger gates, pending hints) or break agent affordances (resolvable IDs, custom-workflow visibility).

**Decision:** Add a compact v2 renderer as the default, move the full guide behind `--full`/`guide`, update Claude hooks to use hook mode, and align stale-anchor prime output with pending. Follow-up adjustments: keep full `tb_<ts>_<sha>` IDs in compact (the ellipsis form was unresolvable for `timbers show`); have `loadWorkflowContent` return whether `PRIME.md` was overridden so compact emits a hint and JSON exposes `custom_workflow`; set `Mode=full` before the JSON branch in `runPrime` (was structurally hardcoded before flag interpretation, would have leaked into MCP/JSON consumers); align compact health truncation to 96 chars to match entries.

**Consequences:**
- Smaller session-context payload as the default; coaching available on request.
- Full IDs cost ~50 bytes per session (3 entries) but preserve paste-into-`timbers show` ergonomics.
- Custom `PRIME.md` content is signaled via hint rather than auto-merged into compact — keeps compact tight while flagging that customization exists.
- JSON consumers now report the requested mode honestly.

---

## ADR-9: Beads sync via committed JSONL, no Dolt remote

**Status:** Accepted
**Date:** 2026-05-08

**Context:** A prior agent skipped committing auto-staged `.beads/issues.jsonl` because a one-time bd 1.0.x schema flip (records gained `_type` discriminator and reordering) made the export look "wrong". The Dolt remote had been unused since 2026-04-29, and `bd dolt push/pull` no-op without one anyway. Keeping the unused remote was creating a misread without providing value.

**Decision:** Remove the bd SQL remote (`origin → git+ssh://...gorewood/timbers.git`) and patch `AGENTS.md` to call out embedded-only mode. Instruct agents to commit bd 1.0.x's `_type`-prefixed JSONL rewrites without reverting. Add a `bd export | diff` drift-recovery recipe. Drop `bd dolt push` from the session-close workflow. The gitignored `.beads/dolt/` working DB plus the committed JSONL gives full reconstruction; a future Dolt remote can be re-added with one command if multi-machine federation is ever needed.

**Consequences:**
- Single sync channel — `git push` of `.beads/issues.jsonl` is authoritative.
- No more "Push/Pull complete" output that transfers nothing.
- Schema rewrites in `.beads/issues.jsonl` are correct behavior, not corruption — documented for future agents.
- Multi-machine federation requires re-adding the remote (one command, but a deliberate step).

---

## ADR-10: Unify hook gating via `hasActionablePending` helper

**Status:** Accepted
**Date:** 2026-05-10

**Context:** A real bug in gorewood/vellum surfaced where `.beads/issues.jsonl`-only commits were triggering the post-commit hook's "log this commit" nudge — but `timbers log` itself refused to document the commit because the pending gate already filtered it out. The pre-commit hook, post-commit hook, and `timbers pending` all needed one definition of "actionable" or agents would receive contradictory signals.

**Decision:** Extract `hasActionablePending()` combining `IsRepo`, `IsInteractiveGitOp`, `.timbers/` existence, storage construction, and `HasPendingCommits` checks. Route both pre-commit and post-commit hooks through it. Reject the narrower fix (inline the `HasPendingCommits` gate in `runPostCommitHook` only) — the helper extraction eliminates the divergence root cause and any third hook added later (post-rewrite?) gets the same gate for free. Add `cmd/timbers/hook_run_test.go` with table-driven cases plus a pre-commit parity test.

**Consequences:**
- One source of truth for "is there work to document?"
- Future hooks get correct gating by default.
- Required a `seedFile` test-harness escape hatch so `.timbersignore` can be baked into the initial commit — otherwise adding it as a separate commit makes IT actionable.
- Parity test reads as one assertion, catching future divergence cheaply.

---

## ADR-11: Scope cross-agent timbers gate to first-parent line + env-var escape hatch

**Status:** Accepted
**Date:** 2026-05-18

**Context:** Parallel agents share `.timbers/` via merges. The original full-DAG gate blocked agent A on commits that agent B had made and not yet documented — firing on the wrong actor. The team considered two options: author-based attribution (filter by git identity) or git-native first-parent scoping. All agents in this user's setup commit as the same git identity, so author filter would no-op.

**Decision:** Add `LogFirstParent` to the git layer and refactor `GetPendingCommits` behind a `firstParent` bool. Add `GetGatePendingCommits` + `gateStrict` `dropEmptyFileChanges` filter. Route `HasPendingCommits` through the gate path. Add an `envTruthy` short-circuit in `hasActionablePending` for `TIMBERS_SKIP_CROSS_AGENT_DEBT`. The original plan was first-parent + env var only, but regression testing exposed that `git merge --no-ff` creates a merge commit M on the first-parent line that was still blocking; `dropEmptyFileChanges` was added as a gate-only filter because clean merges and `--allow-empty` commits have empty file lists from `git diff-tree`'s default combined diff. The display path (`timbers pending`) keeps the conservative empty=unknown rule for awareness. The env var remains as escape hatch for the narrower case where the merge commit itself touched source (conflict resolution).

**Consequences:**
- Agent A no longer blocks on agent B's undocumented commits on side branches.
- Clean `--no-ff` merges and `--allow-empty` commits stop blocking the gate while still appearing in `timbers pending`.
- Conflict-resolution merges (where the merge commit touched files) still block — escape hatch is intentional.
- Builds on ADR-10's unified gating: the env-var short-circuit lives in `hasActionablePending`.
- Reviewer flagged misleading env-var docs and a contradictory test name — both corrected before commit.
