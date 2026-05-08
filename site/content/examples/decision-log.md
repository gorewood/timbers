+++
title = 'Decision Log'
date = '2026-05-07'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

## ADR-1: Drop colons from ledger filenames to unblock `go install`

**Status:** Accepted
**Date:** 2026-04-29

**Context:** A user reported that `go install ...@latest` failed for every tagged version of timbers from v0.16.x through v0.17.x. Go's module zip format and proxy reject paths containing colons, and ledger entry filenames embedded the full ISO 8601 timestamp (e.g. `tb_2026-04-29T16:04:31Z_b60c77.json`). Every release tag pointed to a tree containing colon-named files, blocking distribution outright.

**Decision:** Adopt a filesystem-only encoding that flips `HH:MM:SS` separators to dashes for storage while keeping the canonical entry ID (used in CLI args and JSON content) intact. Added `IDToFilename`/`FilenameToID` helpers, made `entryPath` emit dashed names, kept `legacyEntryPath` as a read fallback for indefinite back-compat, and added `timbers doctor --fix` to bulk-migrate existing repos. Considered mangling the ID itself (drop colons everywhere) and rejected — the canonical timestamp is meaningful and used in many places; only the filesystem encoding needed to change. Considered a hard cutover (delete legacy reads after the next tag) and rejected — forks and downstream consumers may still hold colon-named entries.

**Consequences:**
- `go install github.com/.../timbers@latest` works for all post-v0.18 tags.
- Pre-v0.18 colon-named files in git history remain readable forever; new writes are dashed and a transparent on-write cleanup deletes the colon sibling.
- Indefinite read-time fallback adds a small amount of permanent code surface to `entryPath`/`legacyEntryPath`.

## ADR-2: Typed skip-rule grammar with exact-path matching, expanded defaults

**Status:** Accepted
**Date:** 2026-05-01

**Context:** Pending-commit detection was running on a flat `infrastructurePrefixes` list using `strings.HasPrefix`, which had two problems: it was nagging on housekeeping files (`.gitignore`, `.editorconfig`, narrowly-scoped `.github/` metadata) that carry zero design intent, and the prefix match contained a latent bug — `.gitignore` would also match `.gitignores`. The upcoming `.timbersignore` feature was already telegraphed to reuse the same matcher, so any rule-grammar shape chosen here would set the precedent.

**Decision:** Replaced the prefix-only list with a typed `skipRule` (prefix / exact / suffix variants) in a new `skiprules.go`. Expanded defaults to cover the housekeeping cases — but design review pushed back on `.github/` as a directory (workflows/ are substantive) and on `CHANGELOG.md` (project-specific convention; belongs in user `.timbersignore`, not built-ins); `.claude/` was also dropped because slash commands are substantive. The exact-path variant fixed the `.gitignores` collision in passing.

**Consequences:**
- Default-deny stops nagging on common housekeeping files; ~10–15% reduction in pending-friction with no opt-in.
- `skipRule` types are reusable by `.timbersignore` parsing, which lets the per-repo extension layer onto built-ins instead of duplicating matcher code.
- The default list is deliberately conservative and not user-configurable for the common case — repo-specific rules belong in `.timbersignore` (see ADR-5).

## ADR-3: Surface infrastructure-skipped count via `--verbose` status

**Status:** Accepted
**Date:** 2026-05-01

**Context:** Once skip rules started relaxing what counts as "pending" (ADR-2, plus the upcoming `.timbersignore`), there had to be a feedback channel — without visibility, a user's custom rules could over-skip and erode the ledger's coverage silently. The question was whether to build anti-abuse machinery or settle for cheap visibility.

**Decision:** Added `Storage.CountInfraSkippedSinceLatest` walking `(latestAnchor..HEAD)` through the same `skipRule` set used for pending. `timbers status --verbose` adds a one-line key/value; JSON always includes `infra_skipped_since_entry` for machine consumers per the snake_case convention. Reviewer asked for `--verbose`-only on the human side to avoid surprising default latency; complied. Errors and stale-anchor cases collapse to 0 silently — status is visibility, not enforcement, and a noisy error here would invert the goal.

**Consequences:**
- Users can tell when their `.timbersignore` is over-skipping by watching the count drift up between entries.
- Default `timbers status` stays fast and quiet; latency cost is paid only by the user who asks for `--verbose`.
- No enforcement layer — the count is an indicator, not a gate. If a repo's count grows pathologically, that's a signal for human judgement, not an automated correction.

## ADR-4: Auto-skip reverts of documented commits

**Status:** Accepted
**Date:** 2026-05-01

**Context:** Revert commits were producing pending-detection noise even when the original commit was already documented in the ledger. The original entry IS the audit trail for that work; requiring a fresh entry on the revert just inflated ledger volume without adding design intent. The risk was that a revert of an *undocumented* commit shouldn't be silently swallowed, and edge cases (squashed reverts, GC'd SHAs, multi-revert with mixed coverage) had to fail safe.

**Decision:** Added `internal/ledger/revert.go` to detect `Revert "..."` subjects with a `This reverts commit <sha>` body trailer (regex), cross-reference the SHA against every entry's `Workset.Commits`, and skip when documented. SHA matching uses prefix to tolerate short SHAs. Reviewer required all failure modes (squashed/GC'd/undocumented/mixed-coverage) to fall back to "normal pending" — chose the conservative path that any undocumented SHA in a multi-revert keeps the whole commit pending. The `--revert` flag for new entries was deliberately deferred per design review; auto-skip alone covers the common case.

**Consequences:**
- Revert-on-documented-commit no longer adds pending noise; the original entry remains the audit record.
- Failure modes default to "still pending," so an undocumented revert is never silently swallowed.
- Manual `--revert` semantics remain a future bead if real demand emerges; not building it now keeps the surface small.

## ADR-5: Per-repo `.timbersignore` at repo root extends built-in skip rules

**Status:** Accepted
**Date:** 2026-05-02

**Context:** The hardcoded default skip list (ADR-2) couldn't cover every repo's housekeeping — `vendor/`, custom dependency dirs, project-specific lockfiles. A configurable extension was needed without code changes per repo, but defaults themselves should remain unconfigurable so the common case stayed safe. There was also a placement question: every `*ignore` file in the wider ecosystem (`.gitignore`, `.dockerignore`, `.npmignore`) lives at the repo root, but the initial implementation put `.timbersignore` inside `.timbers/`.

**Decision:** Added `internal/ledger/timbersignore.go` parsing newline-delimited entries with `#` comments using the existing `skipRule` grammar from ADR-2; rules load once at `NewStorage` and merge with the built-in defaults. Considered TOML for symmetry with a future `.timbers/config.yaml` and rejected — would have added a parser dep for one feature, and the `.timbersignore` name was already telegraphed by the existing storage comment. Considered doublestar `**` globs and rejected — prefix+suffix already handles `vendor/`, `*.lock`, `third_party/`, which is the bulk of demand. Loader errors fall back to defaults silently because a malformed file should never block enforcement. Initial location at `.timbers/.timbersignore` was caught pre-release and moved to repo root the next day to match every other `*ignore` file in the ecosystem (rejected `.timbers/timbersignore` without the leading dot — even more nonstandard since the leading dot IS the convention).

**Consequences:**
- Repos can opt out of pending-detection for their specific housekeeping without code changes.
- Defaults remain fixed and conservative; per-repo customization layers on top via the same `skipRule` types.
- `NewStorage` now does a small amount of I/O at construction (one `os.Open`); chose to document this rather than refactor to lazy load, because lazy paths hide state behind first-use side effects.
- Doublestar globs and TOML config can be added later if real demand emerges — neither is foreclosed.

## ADR-6: Default-skip lockfiles across major ecosystems

**Status:** Accepted
**Date:** 2026-05-02

**Context:** Isolated lockfile-only commits (manual conflict resolution, auto-rebase byproducts) carried zero design intent but were still triggering pending detection. The structural question was whether adding lockfiles to defaults would silently swallow legitimate substantive lockfile changes (manual transitive override, security patch) — and the answer was that the existing file-level filter only triggers on lockfile-*only* commits, which makes those legitimate cases rare in practice and still catchable via the `infra_skipped_since_entry` surface from ADR-3.

**Decision:** Added six suffix patterns to `defaultSkipPatterns`: `*package-lock.json`, `*pnpm-lock.yaml`, `*yarn.lock`, `*go.sum`, `*Cargo.lock`, `*Gemfile.lock`. The existing `skipSuffix` matcher handles them without changes. Tests cover both the lockfile-only cases and the manifest cases (`package.json`, `go.mod`, `Cargo.toml`) that must stay pending when lockfiles change alongside them. Considered the long tail (`poetry.lock`, `Pipfile.lock`, `composer.lock`, `mix.lock`, `pubspec.lock`) and left them for `.timbersignore` opt-in — the six included cover the bulk of agent-driven repos.

**Consequences:**
- Fewer pending-detection false positives on auto-rebase byproducts and conflict resolutions.
- Manifest-bundled lockfile changes still surface for documentation because the file-level filter requires a lockfile-only commit to skip.
- Long-tail ecosystems (Python/PHP/Elixir/Dart) need to add their lockfiles via `.timbersignore`.
- Substantive lockfile-only changes (security patches without manifest bumps) will be silently skipped; the visibility surface from ADR-3 is the only feedback channel for catching that drift.

## ADR-7: Per-artifact voice calibration for draft templates

**Status:** Accepted
**Date:** 2026-05-02

**Context:** Two related observations forced a template-design decision. First, a real custom-template devblog from a sibling project read with invisible-agent "I" everywhere, tour-guide pacing, and resolution-voice on every beat — erasing the human-agent collaboration that's increasingly the actual shape of how work gets done. Second, fixing devblog by adding operator-voice and a `Collaborator` voice raised the question: should the same treatment apply to changelog, ADR, PR description, release notes, sprint report, and standup templates? Each has different readers calibrating differently.

**Decision:** Differential treatment per artifact. Devblog gets explicit operator-voice and collaboration-aware narrative (added `Audience` section, fourth `Collaborator` voice, two new anti-patterns for hero-voice and tour-guide voice; tightened length 800→700). Changelog and release notes stay deliberately neutral — readers want a list of facts at a specific abstraction level, and operator-voice would feel performative there. PR description, ADR `Context`, sprint-report, and standup get collaboration awareness because their readers (reviewers, future maintainers, PMs, teammates) calibrate differently knowing how the work was actually built. The most structurally significant change is the new `Status`/`Date` fields on ADRs — without them, decision logs accumulated with no scaffolding for noting "this was superseded by X." A separate `agent_involvement` entry field was considered and rejected as too much taxonomy that would encourage fabrication; the existing `notes` field handles this fine when used well. A subsequent Codex second-opinion review identified "soft fabrication" (invented emotion, theme, vague benefit from thin entries) as the unifying failure mode across templates and patches were accepted to harden every template against that specific risk.

**Consequences:**
- Operator-voice and collaboration are now defaults in devblog, opt-out via custom templates, rather than something users have to discover.
- Neutral artifacts (changelog, release notes) stay readable as fact lists.
- ADRs gain proper `Status`/supersession structure, enabling reversal-tracking — the absence of this would have made it impossible to record "ADR-N is superseded by ADR-M."
- Template maintenance now has an explicit framing — "soft fabrication" — for evaluating any future addition.

## ADR-8: Coach agents to draft PR descriptions from ledger entries

**Status:** Accepted
**Date:** 2026-05-02

**Context:** Agents opening PRs without an explicit body instruction were defaulting to ad-hoc summaries that drifted from the session's documented intent. When ledger entries already exist on the branch, the `pr-description` template is the natural source-of-truth and produces a more consistent body than the agent reasoning from scratch. The question was whether to make this an "always draft from ledger" rule or something more conditional.

**Decision:** Added a `<pr-authoring>` section to `defaultWorkflowContent` in `prime_workflow.go` listing explicit when-applies (open-PR ask without dictated body, entries exist in branch range) and when-doesn't (operator dictates body, no entries, trivial PR) conditions, the recommended pipe-through-`claude` flow, and an empty-section signal — missing `Design Decisions` means thin entries, not a license to fabricate. Considered a more aggressive "always draft from ledger unless told otherwise" rule and rejected — it would override operators who want a one-line PR body for trivial work.

**Consequences:**
- PR bodies get consistent shape and don't drift from the session's recorded intent.
- The empty-section signal converts a fabrication risk into a feedback loop: thin entries surface as missing template sections, prompting either better entries or honest acknowledgement that the section doesn't apply.
- Guidance lives in prime workflow rather than the `pr-description` template itself because it's about WHEN to invoke the template, not HOW the template works — keeps templates focused on output shape and prime workflow focused on agent behavior.

## ADR-9: Compact prime output as default, full guide on demand

**Status:** Accepted
**Date:** 2026-05-08

**Context:** Default `timbers prime` injection at session start was spending agent context on repeated coaching content — every session re-paying the cost of explanatory guidance the agent had already absorbed in prior sessions. The trade-off was between context efficiency and the educational value of seeing the full guide on every session-start. Compact output also had to preserve operator affordances: agents needed to be able to paste IDs into `timbers show`, and any custom `PRIME.md` the repo defined had to remain discoverable.

**Decision:** Added a compact v2 renderer that preserves only operational ledger safeguards (full entry IDs, recent work summaries, anchor health), moved the full guide behind `--full`/`guide`, and updated Claude hooks to use hook (compact) mode. `compactEntryID` was tried and removed — the ellipsis form was unresolvable when pasted into `timbers show`, so full `tb_<ts>_<sha>` IDs are kept; the ~50-byte cost per session (3 entries) is worth the paste-into-tool ergonomics. `loadWorkflowContent` now returns whether `PRIME.md` was overridden so compact emits a hint rather than auto-merging the custom content (keeps compact tight while flagging that customization exists). JSON output reports the requested mode honestly via a structural fix — the `Mode` field had been hardcoded before flag interpretation, which would have leaked into MCP/JSON consumers. Compact health truncation aligned to 96 chars to match entries.

**Consequences:**
- Default session-start injection is dramatically smaller, leaving more context for the actual work.
- Full guide is one flag away (`--full`) for newcomers or when re-reading is needed.
- IDs in compact output remain copy-pasteable, preserving the `prime → show` workflow.
- Custom `PRIME.md` content is signaled via hint rather than embedded, requiring agents to know to follow the hint — accepts a small discoverability cost in exchange for compact output staying compact.

## ADR-10: Drop Dolt remote in favor of JSONL-only sync

**Status:** Accepted
**Date:** 2026-05-08

**Context:** Beads 1.0+ runs an embedded Dolt server locally; `.beads/dolt/` is the working DB (gitignored) and `.beads/issues.jsonl` is the committed source of truth. The repo had also been carrying a Dolt SQL remote (`origin → git+ssh://...gorewood/timbers.git`), unused since 2026-04-29. A previous agent had read the dual-channel setup (committed JSONL + Dolt remote) and skipped committing auto-staged JSONL changes, mistaking bd 1.0.x's JSONL schema flip (records gained a `_type` discriminator and reordered) for corruption. Verification with `bd export -o /tmp/x.jsonl && diff` against `.beads/issues.jsonl` confirmed the file matched the local DB and was safe to commit — but the dual-channel ambiguity was the root cause of the misread.

**Decision:** Removed the Dolt SQL remote. Patched `AGENTS.md`'s sync-model bullet to call out embedded-only mode, told agents to commit bd 1.0.x's `_type`-prefixed JSONL rewrites without reverting, added a `bd export | diff` drift-recovery recipe, and dropped `bd dolt push` from the session-close workflow. Considered keeping the remote "just in case" and rejected — the gitignored `.beads/dolt/` working DB plus the committed JSONL gives full reconstruction; a Dolt remote can be re-added with one command if multi-machine federation is ever needed.

**Consequences:**
- One sync channel, one source of truth — agents have fewer ways to misread the system state.
- `bd dolt push/pull` are no-ops on this repo (per fix #3194 they exit 0 with informational messages); leave them alone in injected blocks.
- Multi-machine federation via Dolt is no longer an off-the-shelf feature; would require re-adding the remote.
- `AGENTS.md` now carries explicit guidance to commit auto-staged JSONL rewrites and not revert them, closing the failure mode that triggered this change.
