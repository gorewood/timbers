+++
title = 'Decision Log'
date = '2026-05-02'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

## ADR-1: Default skip rules — typed infrastructure filter over ledger-only special case

**Status:** Accepted
**Date:** 2026-05-02

**Context:** When timbers shipped, the pending filter only excluded commits touching `.timbers/`. After beads' auto-stage hook started writing `.beads/issues.jsonl` into timbers entry commits, the filter broke and timbers began documenting its own documentation work. Beyond that, default-deny pending was nagging operators on housekeeping files (`.gitignore`, `.editorconfig`, `*.lock`) that carry zero design intent.

**Decision:** Broaden the special-cased `.timbers/` prefix into a typed `skipRule` grammar (prefix/exact/suffix) with a curated default list. Defaults cover timbers' own infrastructure (`.timbers/`, `.beads/`), narrowly-scoped housekeeping files (specific exact paths, not whole directories), and lockfiles across major ecosystems (`package-lock.json`, `pnpm-lock.yaml`, `yarn.lock`, `go.sum`, `Cargo.lock`, `Gemfile.lock`). Design review explicitly excluded `.github/`, `CHANGELOG.md`, and `.claude/` from defaults because their contents are often substantive (workflows, slash commands, project-specific conventions).

**Consequences:**
- `timbers pending` no longer nags on housekeeping or lockfile-only commits, eliminating the bulk of friction with no opt-in.
- The typed rule grammar is reusable for the per-repo `.timbersignore` feature (ADR-2).
- Fixed a latent `strings.HasPrefix` bug where `.gitignore` matched `.gitignores`.
- Lockfile-only commits CAN be substantive (manual transitive override, security patch); accepted that risk because the file-level filter only triggers on lockfile-ONLY commits, which are rare when paired with manifest changes.
- Project-specific conventions belong in per-repo `.timbersignore`, not built-in defaults.

## ADR-2: Per-repo `.timbersignore` at repo root, extending built-in defaults

**Status:** Accepted
**Date:** 2026-05-02

**Context:** The hardcoded default skip list (ADR-1) cannot cover every repo's housekeeping — `vendor/`, custom dep dirs, project-specific generated files. Each repo needed an opt-out mechanism without code changes, and the file's location had to be discoverable by users coming from the `.gitignore` / `.dockerignore` / `.npmignore` convention.

**Decision:** Add a repo-root `.timbersignore` (newline-delimited, `#` comments) that uses the same `skipRule` grammar as built-in defaults. Rules load once at `NewStorage` construction and merge with compiled defaults. The file lives at the repo root (not inside `.timbers/`) to match every other `*ignore` convention — dotfile-in-a-dot-dir was nonstandard and undiscoverable.

**Consequences:**
- Each repo can opt into additional skips without forking timbers.
- Loader errors fall back to defaults silently — a malformed `.timbersignore` should never block enforcement, which would invert the gate.
- TOML/YAML config was considered but rejected: would have added a parser dep for one feature.
- Doublestar `**` globs deferred — prefix+suffix covers `vendor/`, `*.lock`, `third_party/`, the bulk of demand; can layer in later if real cases emerge.
- `NewStorage` now does I/O at construction; documented in godoc rather than refactored to lazy load, because lazy loading would hide state behind first-use side effects for one tiny file open.

## ADR-3: Surface infrastructure-skipped count in `timbers status`

**Status:** Accepted
**Date:** 2026-05-01

**Context:** With ADR-1 and ADR-2 expanding default-skip behavior and adding per-repo extension, skip relaxations needed a feedback channel. Without visibility, operators can't tell when their `.timbersignore` is over-skipping and silently hiding substantive work.

**Decision:** Add a one-line surface in `timbers status` showing infrastructure-skipped commit count since the latest entry. Visible only behind `--verbose` for human display (per reviewer, to avoid surprising latency); JSON output always emits `infra_skipped_since_entry` for machine consumers. Errors and stale-anchor cases collapse to 0 silently.

**Consequences:**
- Cheap visibility surface that catches drift without building anti-abuse machinery.
- Status remains a visibility tool, not enforcement — a noisy error here would invert the goal.
- The check walks `(latestAnchor..HEAD)` on every `--verbose` invocation; acceptable because work is bounded by entry cadence.

## ADR-4: Auto-skip reverts of already-documented commits from pending

**Status:** Accepted
**Date:** 2026-05-01

**Context:** Revert commits genuinely add no new design intent when the reverted commit is already documented — the original entry IS the audit trail. Requiring a fresh entry for every revert just inflates ledger noise.

**Decision:** Detect `Revert "..."` subjects with `This reverts commit <sha>` body trailers via regex, cross-reference the SHA against every entry's `Workset.Commits`, and skip the revert when documented. Use prefix matching for short-SHA tolerance. All failure modes (squashed reverts, GC'd SHAs, undocumented originals, multi-revert with mixed coverage) fall back to "normal pending" — for multi-revert, any single undocumented SHA keeps the whole commit pending.

**Consequences:**
- The common case (single revert of a documented commit) eliminates ledger noise without operator intervention.
- Conservative fallback avoids silent loss of context when revert state is ambiguous.
- A `--revert` flag for new entries was considered for the manual case but deferred — auto-skip alone covers the common case; manual entry can be added if real demand emerges.
- Short-SHA prefix matching has a small collision risk; mitigated by raising the minimum match length to 12 chars after code review.

## ADR-5: Filename grammar — drop colons from on-disk encoding, preserve canonical ID

**Status:** Accepted
**Date:** 2026-04-29

**Context:** A user reported `go install ...@latest` failing for all `v0.16.x` and `v0.17.x` tags. Root cause: `.timbers/*.json` filenames contained colons from ISO 8601 timestamps (`HH:MM:SS`), which break Go's module zip format and proxy. The fix had to be backward-compatible with the 146 existing entries on the timbers repo and any entries written by older binaries in downstream forks.

**Decision:** Add `IDToFilename` / `FilenameToID` helpers that flip only the `HH:MM:SS` separators in the on-disk filename, while keeping the canonical entry ID unchanged. New writes use the dashed filename; reads accept both; on-write cleanup removes legacy siblings transparently. Bulk migration via `FileStorage.MigrateLegacyFilenames` exposed through `timbers doctor --fix`.

**Consequences:**
- `go install` works on tagged versions again.
- Backward-compat reads remain indefinitely — forks and downstream consumers may have entries written by older binaries and shouldn't need to upgrade in lockstep.
- Mangling the ID itself (drop colons everywhere, including in CLI args and JSON content) was rejected — the canonical ISO 8601 timestamp is meaningful and load-bearing in many places; only the filesystem encoding needed to change.
- A hard cutover (delete legacy support entirely once tagged) was rejected to preserve the decoupling between filesystem encoding and entry identity.

## ADR-6: Durable ADR numbering via caller-supplied `--var` offset

**Status:** Accepted
**Date:** 2026-04-20

**Context:** ADRs are meant to be stable identifiers — `ADR-12` should always mean the same decision. The decision-log template's hardcoded restart-at-1 broke that, causing external references to rot silently when the file was regenerated. The fix had to keep `draft` as a pure renderer (no state, no embedded counter logic).

**Decision:** Add a repeatable `--var key=value` flag to `draft` that exposes caller-supplied variables under `{{vars.key}}` in templates. Add a `vars:` map to template frontmatter for per-template defaults; `decision-log` sets `starting_number` default to 1. The justfile recipe greps the max `ADR-N` from the target file and passes `next` via `--var`, making the output file (version-controlled markdown) the single source of truth for the counter. Hardened `parseVars` with a duplicate-key check after review flagged silent last-wins as a scripting footgun.

**Consequences:**
- ADR identifiers stay stable across regenerations because the offset is computed from the durable artifact itself.
- The `{{vars.*}}` namespace prevents callers from accidentally shadowing built-in tokens like `{{date}}`.
- Per-template defaults live in frontmatter (not `render.go`) to avoid coupling the render package to specific template variables.
- Storing the counter in `.timbers/state.json` was rejected — two sources of truth means they can drift, and regenerating an ADR file would orphan the counter.
- Baking `--continue-from <file>` into `draft` itself was rejected — that would embed decision-log-specific parsing logic in a generic command; the justfile recipe is the right place for that sugar.

## ADR-7: Template tuning — artifact-appropriate signal over uniform operator-voice

**Status:** Accepted
**Date:** 2026-05-02

**Context:** A real custom-template devblog from a downstream worktree read as dry, hero-voice prose: invisible agent, tour-guide pacing, no surfaced disagreement. The diagnosis raised a broader question — should the same operator-voice and collaboration-aware coaching apply to all draft templates? Each artifact has a different reader and different signal needs.

**Decision:** Tune each template per-artifact rather than apply operator-voice uniformly. Devblog gets a fourth Collaborator voice, anti-patterns for hero-voice and tour-guide voice, and a tightened length budget (800 → 700). Changelog and release-notes stay deliberately neutral — readers want a list of facts at a specific abstraction level, and operator-voice there would be performative. PR description, ADR Context, sprint-report, and standup get collaboration awareness because their readers (reviewers, future maintainers, PMs, teammates) calibrate differently when they know how the work was actually built. The ADR template gains a `Status` field with explicit supersession semantics. An `<operator-voice>` section in `prime_workflow.go` carries the broader principle (intent + collaboration habits, with an anti-fabrication guardrail) into ad-hoc writing.

**Consequences:**
- Templates produce artifact-appropriate output instead of one-size-fits-all coaching.
- The Status field on ADRs is the most structurally significant change — without it, decision logs accumulate with no way to capture reversals.
- A separate "collaboration template" was rejected — would fragment the surface; baking collaboration into existing templates keeps one canonical narrative shape per artifact.
- A structured `agent_involvement: high|medium|low` entry field was rejected — too much taxonomy, encourages fabrication; existing notes field handles this when used well.
- Test-plan honesty in PR description ("Tests pass" as a fake verified claim) was named explicitly because LLM laziness here is a real failure mode.

## ADR-8: Anti-fabrication as a continuum across template families

**Status:** Accepted
**Date:** 2026-05-02

**Context:** A second-opinion review of the tuned templates (ADR-7) surfaced 14 points across seven templates. The unifying observation: the codebase had been treating fabrication risk as binary (test-plan invention, metric invention) when softer kinds (emotion, theme, vague benefit) are the same failure mode at lower visibility.

**Decision:** Accept the framing that fabrication risk is continuous and patch each template's specific soft-content target. Nine targeted patches landed: changelog header rule, ADR consequences tightening + date semantics, devblog emotion anti-fabrication + dropped word floor, PR tiny-PR risk override + design-decision collaboration, release-notes incomplete-migration handling + soft-benefit tightening, sprint-report theme threshold + chronology-last-resort, standup stale-next-steps. Two suggestions were rejected: weakening devblog literary scaffolding (the named voices do directional work the LLM reads as guidance, not mandatory mood) and merging the extension-author audience.

**Consequences:**
- Every template now has at least one explicit anti-fabrication clause targeting its specific soft-content failure mode.
- The codex review functioned as adversarial sparring, not authority — being explicit about what was rejected and why is part of the value of the exercise.
- Template versions bumped only where material changes landed.

## ADR-9: PR-authoring default — draft from ledger entries, empty section as feedback signal

**Status:** Accepted
**Date:** 2026-05-02

**Context:** Agents opening PRs without explicit body instructions were defaulting to ad-hoc summaries that drifted from the session's documented intent. When timbers entries exist for the branch, the `pr-description` template is the natural source of truth and produces a more consistent body than the agent reasoning from scratch.

**Decision:** Add a `<pr-authoring>` section to `defaultWorkflowContent` in `prime_workflow.go` listing when this default applies (open-PR ask without dictated body, entries exist in branch range), when it doesn't (operator dictates body, no entries, trivial PR), the recommended pipe-through-claude flow, and an empty-section signal — a missing Design Decisions section means thin entries, not a license to fabricate. Guidance lives in prime workflow rather than the `pr-description` template itself because it's about WHEN to invoke the template, not HOW the template works.

**Consequences:**
- PR bodies stay anchored to documented intent when entries exist.
- An aggressive "always draft from ledger unless told otherwise" rule was rejected — would override operators who want a one-line body for trivial work.
- The empty-section signal converts a fabrication risk (LLM filling Design Decisions with vague restatements) into a feedback loop (entries were thin; either improve them or accept that the section doesn't apply).
