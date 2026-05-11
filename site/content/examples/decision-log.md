+++
title = 'Decision Log'
date = '2026-05-10'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

## ADR-1: Expand default infrastructure skip rules with typed grammar

**Status:** Accepted
**Date:** 2026-05-01

**Context:** The default-deny pending checker was nagging on housekeeping commits (`.gitignore`, `.editorconfig`, narrow `.github/` metadata) that carry no design intent. The existing matcher was a `strings.HasPrefix` over a flat `infrastructurePrefixes` list — both too narrow (no exact-path or suffix matching) and silently buggy (`.gitignore` would match `.gitignores`). Design review wanted to relax defaults *and* set up a grammar reusable by the per-repo extension feature coming next.

**Decision:** Replace prefix-only matching with a typed `skipRule` (prefix / exact / suffix) in a new `skiprules.go`. Expand the default set to cover dotfile housekeeping, but reject reviewer-proposed entries that would over-skip: `.github/` (workflows are substantive), `CHANGELOG.md` (project-specific convention — belongs in user `.timbersignore`), and `.claude/` (slash commands are substantive). The new exact-path rule type fixes the latent `HasPrefix` bug as a side effect.

**Consequences:**
- Eliminates ~10–15% of pending-check friction with no user opt-in required.
- The typed grammar is the foundation for `.timbersignore` (ADR-2) and lockfile defaults (ADR-5) — both reuse `skipRule` directly.
- Exact-path semantics close a real correctness bug in prefix matching.
- Substantive-vs-housekeeping boundary still requires judgment per addition; the curated default list is deliberately conservative.

---

## ADR-2: Per-repo `.timbersignore` at repo root extends built-in skip rules

**Status:** Accepted
**Date:** 2026-05-02

**Context:** The hardcoded default skip list (ADR-1) cannot cover every repo's housekeeping — `vendor/`, `*.lock`, project-specific dep dirs vary widely. Some mechanism for per-repo extension was needed, but the choice of format and location had downstream cost: a config-system would be premature for one feature, and a non-standard location would be undiscoverable.

**Decision:** Add `.timbersignore` at the repo root, parsed with the `skipRule` grammar from ADR-1 (newline-delimited, `#` comments). Rules merge with defaults at `NewStorage` construction. Loader errors fall back to defaults silently — a malformed ignorefile must never block enforcement, since that would invert the gate. Reject TOML (would add a parser dep for one feature) and `**` globs (prefix+suffix covers the bulk of demand). Location originally landed inside `.timbers/.timbersignore`; moved to repo root before any release tag because every other `*ignore` file in the ecosystem (`.gitignore`, `.dockerignore`, `.npmignore`) lives at the root and users would never look inside `.timbers/`.

**Consequences:**
- Repos can opt out of arbitrary patterns without code changes.
- Defaults remain safe and unconfigurable for the common case — extension is additive only.
- `NewStorage` now does I/O at construction (one `os.Open` of a tiny file); documented in godoc rather than refactored to lazy-load, because lazy paths hide state behind first-use side effects.
- `**` globs deferred; can layer in later if real demand emerges.

---

## ADR-3: Auto-skip reverts of already-documented commits

**Status:** Accepted
**Date:** 2026-05-01

**Context:** Revert commits add no new design intent when the reverted commit is already documented — the original entry IS the audit trail. Requiring a fresh entry for every revert inflates ledger noise without adding signal. But auto-skipping is risky: squashed reverts, GC'd SHAs, undocumented originals, and multi-revert commits with mixed coverage all need careful handling.

**Decision:** Detect `Revert "..."` subjects with `This reverts commit <sha>` body trailers via regex, cross-reference the SHA against every entry's `Workset.Commits`, and skip when documented. Integrate alongside infrastructure filtering in a shared `filterByRules` helper. Take the conservative path on every failure mode: any undocumented SHA in a multi-revert keeps the commit pending; squashed reverts and GC'd SHAs fall back to normal pending. Defer the symmetric `--revert` flag for new entries — auto-skip alone covers the common case.

**Consequences:**
- Eliminates a class of "fresh entry just to say undo" friction.
- SHA matching uses prefix match for short-SHA tolerance even though `git revert` defaults to 40-char SHAs.
- Code review later required raising the short-SHA minimum to 12 chars to avoid collision risk — accepted; matcher tightened.
- Manual `--revert` opt-in stays available as a future bead if demand emerges.

---

## ADR-4: Surface infrastructure-skipped count in `status --verbose`

**Status:** Accepted
**Date:** 2026-05-01

**Context:** Relaxing skip rules (ADR-1, ADR-2, ADR-5) creates a new failure mode: users can't tell when their `.timbersignore` is over-skipping. Without a feedback channel, drift accumulates silently. But adding heavy anti-abuse machinery would invert the goal of frictionless skipping.

**Decision:** Add `Storage.CountInfraSkippedSinceLatest`, which walks `(latestAnchor..HEAD)` through the same `skipRule` set used for pending and returns the count. Surface in `status --verbose` only for human display (reviewer flagged latency surprise on the default path); always emit `infra_skipped_since_entry` in JSON for machine consumers. Collapse errors and stale-anchor cases to `0` silently — status is visibility, not enforcement, and a noisy error here would invert the goal.

**Consequences:**
- Cheap one-line visibility catches skip-rule drift without building enforcement.
- `--verbose`-only on human path keeps default `status` fast.
- JSON always carries the field so MCP/automation can monitor without per-call flags.
- Code review caught a divergence between the count helper and the actual filter; unified later via `filterCommits` so visibility tracks reality.

---

## ADR-5: Default-skip lockfiles across major ecosystems

**Status:** Accepted
**Date:** 2026-05-02

**Context:** Isolated lockfile-only commits — manual conflict resolution, auto-rebase byproducts — carry zero design intent. The file-level filter from ADR-1 means lockfiles paired with a manifest change stay pending automatically, so default-skipping is structurally safe for the lockfile-only case. The question was scope: how many ecosystems to cover.

**Decision:** Add six suffix patterns to `defaultSkipPatterns`: `*package-lock.json`, `*pnpm-lock.yaml`, `*yarn.lock`, `*go.sum`, `*Cargo.lock`, `*Gemfile.lock`. These cover the bulk of agent-driven repos; the long tail (`poetry.lock`, `Pipfile.lock`, `composer.lock`, `mix.lock`, `pubspec.lock`) can opt in via `.timbersignore`. Tests cover both the lockfile cases and the manifests that must stay pending (`package.json`, `go.mod`, `Cargo.toml`).

**Consequences:**
- Removes a common class of agent-driven friction for the six biggest ecosystems.
- Accepts a residual risk: a lockfile change unaccompanied by a manifest CAN be substantive (manual transitive override, security patch). Mitigated by (a) the file-level filter only triggering on lockfile-only commits, which are rare, and (b) the `infra_skipped_since_entry` visibility surface (ADR-4).
- Long-tail ecosystems must opt in per-repo — explicit non-coverage.

---

## ADR-6: Calibrate draft template signal per-artifact rather than uniformly

**Status:** Accepted
**Date:** 2026-05-02

**Context:** A read of a real custom-template devblog from a sister repo surfaced a default-on bias toward solo-developer hero voice — invisible agents, tour-guide pacing, resolution-voice on every beat. The instinct was to inject operator-voice and collaboration awareness uniformly across all seven `draft` templates. But the underlying audiences differ: changelog and release-notes readers want neutral facts at a specific abstraction level, while PR descriptions, ADRs, sprint reports, and standups have readers (reviewers, maintainers, PMs, teammates) who calibrate differently when they know how the work was built.

**Decision:** Reject uniform operator-voice. Per-template diagnosis instead: devblog gets an explicit Collaborator voice, operator-intent rows, and length budget tightened 800 → 700. Six other templates get template-specific tuning — Added/Changed disambiguation for changelog; Status/Date/supersession for decision-log (no decision tracking without it); size adaptation and test-plan honesty for PR description; user-observable filter for release-notes; Friction/Carry-overs/Highlights criteria for sprint-report; Asks-for-help and time-burn texture for standup. After a Codex second-opinion review, accept the framing that "soft fabrication" (emotion, theme, vague benefit) is the same failure mode as test-plan invention at lower visibility — add per-template anti-fabrication clauses for soft content targets. Reject Codex's suggestion to soften literary-scaffolding voices (Atwood, Fowler, Orosz) — they're directional guidance the LLM reads as voice, not mandatory mood.

**Consequences:**
- Each template now carries audience-aware coaching matched to its readers.
- ADR Status field is the most structurally significant change — without it, decision logs accumulated with no scaffolding for noting reversals.
- "Empty section signals thin entries, not a license to fabricate" rule converts a fabrication risk into a feedback loop.
- Operator-voice does NOT propagate to changelog or release-notes; those stay neutral.
- A structured `agent_involvement` field on entries was considered and rejected — too much taxonomy, encourages fabrication; the existing `notes` field handles it when used well.

---

## ADR-7: Coach agents to draft PR descriptions from timbers entries by default

**Status:** Accepted
**Date:** 2026-05-02

**Context:** Agents opening PRs without explicit body instructions were defaulting to ad-hoc summaries that drift from the session's documented intent. When entries exist for the branch range, the `pr-description` template is the natural source of truth — but a blanket "always draft from ledger" rule would override operators who want a one-line body for trivial work.

**Decision:** Add an `<pr-authoring>` section to `defaultWorkflowContent` in `prime_workflow.go` with explicit when-applies / when-not lists: applies when there's an open-PR ask without a dictated body and entries exist in the branch range; does not apply when the operator dictates the body, no entries exist, or the PR is trivial. Recommend the `timbers draft pr-description | claude -p --model opus` pipe. Include an empty-section signal — missing Design Decisions means thin entries, not a license to fabricate. Place this in prime workflow rather than the `pr-description` template itself because the guidance is about WHEN to invoke, not HOW the template works.

**Consequences:**
- PRs default to draft-from-ledger when ledger material exists, producing more consistent bodies than agents reasoning from scratch.
- Operators retain override authority for trivial or dictated-body cases.
- The empty-section signal converts a known LLM failure mode (filling Design Decisions with vague restatements) into a feedback loop on entry quality.
- WHEN vs HOW separation keeps the template surface clean and the trigger guidance discoverable from `timbers prime` output.

---

## ADR-8: Compact `timbers prime` output by default; full guide behind `--full`

**Status:** Accepted
**Date:** 2026-05-08

**Context:** Default `timbers prime` injection was spending session-start context on repeated coaching that experienced agents had already internalized. Every session paid the full token cost regardless of whether the agent needed the full operating guide. The compact-by-default move had to preserve agent affordances (resolvable IDs, custom-workflow visibility) — early compact output dropped them.

**Decision:** Add a compact v2 renderer as the new default; move the full guide behind `--full`/`guide`. Update Claude hooks to use hook mode. Compact must preserve: full `tb_<ts>_<sha>` entry IDs (the ellipsis form was unresolvable when pasted into `timbers show`); a PRIME.md hint when custom workflow content overrides the default (rather than auto-merging custom content into compact output); JSON `Mode=full` reported honestly when requested; 96-char health-line truncation aligned with entry truncation. Set the JSON-mode flag *before* the JSON branch — earlier code path hardcoded it ahead of flag interpretation, which would have leaked stale `Mode` into MCP/JSON consumers.

**Consequences:**
- Per-session prime cost drops materially; the smaller payload is what 0.20.0 ships to external installs.
- Full guide still reachable when needed via `--full`.
- Custom PRIME.md customization stays visible (hint) without bloating compact output (auto-merge).
- Recovered ergonomics: paste-into-`timbers-show` works again with full IDs.

---

## ADR-9: Drop Dolt remote; embedded Dolt + JSONL transport as single sync channel

**Status:** Accepted
**Date:** 2026-05-08

**Context:** A prior agent session had skipped committing auto-staged `.beads/issues.jsonl` because they didn't trust bd 1.0.x's one-time JSONL schema flip (records gained a `_type` discriminator and reordering). Their fresh export matched HEAD, but the dual sync model — embedded Dolt + JSONL transport AND a configured Dolt SQL remote — gave them a plausible-looking alternative channel to second-guess against. The Dolt remote had been unused since 2026-04-29 and `bd dolt push/pull` no-op without one.

**Decision:** Remove the bd SQL remote (`origin → git+ssh://...gorewood/timbers.git`). Patch `AGENTS.md` to call out embedded-only mode, tell agents to commit bd 1.0.x's `_type`-prefixed JSONL rewrites without reverting, add a `bd export | diff` drift-recovery recipe, drop `bd dolt push` from session-close workflow. Verified correctness with `bd export -o /tmp/x.jsonl && diff` against `.beads/issues.jsonl` — identical. Reject keeping the remote "just in case" — the gitignored `.beads/dolt/` working DB plus the committed JSONL gives full reconstruction, and a future remote can be re-added with one command if multi-machine federation is ever needed.

**Consequences:**
- Single sync channel removes the misread that caused the prior skipped-commit incident.
- `.beads/issues.jsonl` is the unambiguous source of truth; agents have a clear drift-recovery recipe.
- `bd dolt push` is now a documented no-op in this repo's session-close — one less mandatory step.
- Multi-machine federation, if ever needed, requires re-adding the remote (one-command cost).

---

## ADR-10: Gate post-commit hook on actionable pending work via shared helper

**Status:** Accepted
**Date:** 2026-05-10

**Context:** A real bug in a sister repo (gorewood/vellum) showed `.beads/issues.jsonl`-only commits triggering the post-commit "log this" nudge, even though `timbers log` would refuse to document them. Pending, log, and the post-commit hook had drifted into three different definitions of "actionable" — the hook never called `HasPendingCommits` before printing. Divergence between these gates produces contradictory signals for agents: the hook nudges to log a commit that `timbers log` then refuses.

**Decision:** Extract a `hasActionablePending()` helper that combines `IsRepo`, `IsInteractiveGitOp`, `.timbers/` existence, storage construction, and `HasPendingCommits` checks. Route both pre-commit and post-commit hooks through it so they share one definition of actionable. Reject the narrower fix (inline the `HasPendingCommits` gate in `runPostCommitHook` only) — the helper eliminates the divergence at its root and any third hook added later (post-rewrite?) inherits the gate for free. Add `cmd/timbers/hook_run_test.go` with table-driven cases covering `.beads/`-only, `.timbers/`-only, lockfile-only, `.timbersignore`-skipped, substantive, missing-`.timbers/`, and pre-commit parity.

**Consequences:**
- Hook nudges and `timbers log` enforcement agree by construction.
- Future hooks (post-rewrite, etc.) get correct gating without separate plumbing.
- Test harness now has a `seedFile` escape hatch so `.timbersignore` can be baked into the initial commit — needed because adding it as a separate commit makes IT actionable.
- The "actionable" definition is now a single point of truth; changes propagate to every hook automatically.
