+++
title = 'Decision Log'
date = '2026-04-29'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

## ADR-1: Indefinite Backward-Compat Reads for Legacy Colon-Filename Entries

**Context:** Ledger files used ISO 8601 timestamps with colons in their names (`.timbers/*.json`), which break Go's module zip format and proxy — blocking `go install ...@latest` for all v0.16.x and v0.17.x. The fix required renaming files, but pre-v0.18 entries on forks and downstream consumers still use the old encoding. Options: hard cutover (drop legacy reads after a tag) vs. indefinite read compatibility.

**Decision:** Adopt indefinite legacy-read support. Canonical writes use dashed filenames via `IDToFilename`; reads fall back to `legacyEntryPath` for colon-named files; on-write transparently cleans up legacy siblings; bulk migration via `FileStorage.MigrateLegacyFilenames` and `timbers doctor --fix`. The ID itself (containing the canonical ISO 8601 timestamp) is preserved unchanged — only the filesystem encoding flips `HH:MM:SS` separators.

**Consequences:**
- Forks and downstream consumers don't need to upgrade in lockstep; pre-v0.18 entries in git history remain readable forever.
- The canonical timestamp stays meaningful in CLI args, JSON content, and IDs (rejected alternative: mangle the ID everywhere).
- Legacy fallback code lives in the codebase indefinitely, slightly increasing storage-layer surface area.
- Migration is opt-in via `doctor --fix`; repos that never migrate accumulate dual-format reads but stay correct.

## ADR-2: Per-Template Default Vars in Frontmatter Over Hardcoded Defaults in Render

**Context:** The decision-log template needed a default `starting_number` of 1 when callers don't supply one. The render package could either bake template-specific defaults in code or templates could declare their own defaults.

**Decision:** Add a `vars:` map to template frontmatter. Render applies caller-supplied `--var` values first, then fills remaining `{{vars.*}}` tokens from frontmatter defaults. The `render` package stays generic.

**Consequences:**
- Render package isn't coupled to specific template variables; new templates can declare their own defaults without touching Go code.
- Template authors own their own default surface, keeping concerns local.
- Two layers of substitution (caller, then template) — slightly more logic in render, but the layering is explicit.

## ADR-3: Caller-Owned ADR Numbering Offset Over Internal Counter

**Context:** ADRs need stable identifiers — `ADR-12` should always mean the same decision. Hardcoded restart-at-1 broke that and caused external references to rot silently. Two options for tracking the next number: a `.timbers/state.json` counter, or extracting the offset from the existing output file.

**Decision:** The caller (e.g., a `just decision-log` recipe) greps the max `ADR-N` from the target file and passes the next number via `--var starting_number=N`. No internal counter. A generic `--var key=value` flag on `draft` carries arbitrary caller-supplied template variables, namespaced under `{{vars.key}}` to avoid shadowing built-ins like `{{entry_count}}`.

**Consequences:**
- One source of truth: the version-controlled markdown file itself. No drift between counter and output.
- Regenerating an ADR file doesn't orphan a counter.
- `draft` stays a pure renderer; ADR-specific parsing lives in the justfile recipe where it belongs.
- Callers must extract the offset themselves (small shell idiom), but namespacing keeps built-in vars sacred.

## ADR-4: Duplicate-Key Error in `parseVars` Over Last-Wins

**Context:** The `--var` flag is repeatable, so `parseVars` had to handle multiple values for the same key. Silent last-wins is a footgun under scripting where duplicates often signal a generation bug.

**Decision:** Return an error on duplicate keys. Add table-driven test coverage. Separately, in the `just decision-log` recipe, move the fallback inside the pipeline as `|| echo ""` instead of `|| true` outside `$()` — making empty-file and no-match cases unambiguous under `pipefail`.

**Consequences:**
- Scripts surface generation bugs immediately rather than silently using the last value.
- Slightly stricter input contract — callers passing the same `--var` twice now fail loudly.
- The pipeline idiom is more robust to shell strictness settings.

## ADR-5: Broaden Pending Filter to Infrastructure-Only, Not Just Ledger-Only

**Context:** A beads auto-stage hook caused timbers entry commits to also include `.beads/issues.jsonl`, breaking `isLedgerOnlyCommit` and creating a timbers-on-timbers feedback loop where ledger commits looked like new pending work.

**Decision:** Generalize `isLedgerOnlyCommit` to `isInfrastructureOnlyCommit` with a configurable prefix list (`.timbers/`, `.beads/`). Commits touching only those prefixes are excluded from pending.

**Consequences:**
- Tooling that auto-stages sibling infrastructure files no longer breaks the pending filter.
- The prefix list becomes a contract — adding new infra dirs requires updating it.
- Slightly fuzzier semantics (filter is no longer strictly "ledger only") in exchange for surviving real-world tool composition.

## ADR-6: Reachability Check for Stale Anchor Detection

**Context:** After `git pull --rebase`, old SHAs remain in the object store for ~2 weeks but aren't in HEAD's history. `git log <stale-anchor>..HEAD` succeeds with phantom results, causing timbers to show pending commits that were already documented on a feature branch (reported by Noam).

**Decision:** Add a `git merge-base --is-ancestor` reachability check before `git log` in `GetPendingCommits`. Add `IsAncestorOf` to the `GitOps` interface. Stale-anchor detection now fires on unreachable SHAs, not just absent ones.

**Consequences:**
- Rebase-induced phantoms are caught immediately rather than after object expiry.
- One additional git invocation per pending check — negligible cost.
- The `GitOps` interface grows by one method, which downstream test fakes must implement.

## ADR-7: Suppress Hooks During Interactive Git Operations

**Context:** Hooks running during rebase, merge, cherry-pick, or revert created a deadlock: the agent was blocked by a pending check mid-rebase and could neither log nor continue.

**Decision:** Add `git.IsInteractiveGitOp()` that checks `.git` state files (e.g., `rebase-merge`, `MERGE_HEAD`, `CHERRY_PICK_HEAD`, `REVERT_HEAD`). All hooks early-return during these operations.

**Consequences:**
- Mid-rebase agent workflows no longer deadlock.
- Pending state may briefly diverge during an interactive op; resolves on the next post-op hook fire.
- Hook entry points share a single guard, keeping the suppression rule consistent.

## ADR-8: Pass Detected Scope to `Install` for `doctor --fix`

**Context:** `Install(true)` hardcoded project-local scope, so retired hooks in *global* settings survived `doctor --fix` even though `Detect()` found them. The retired-event cleanup logic was correct — it just ran against the wrong file.

**Decision:** Thread the detected scope from `Detect()` into `Install(scope == "project")` in `checkAgentEnvStaleness`. The fix runs against whichever scope harbored the stale hook.

**Consequences:**
- `doctor --fix` now actually cleans up global-scope retirements.
- Install signature carries scope intent rather than assuming a default — clearer contract.
- Tests must cover both scopes for retired-event cleanup paths.

## ADR-9: Keep Separate Commits for Ledger Entries (Council-Evaluated)

**Context:** Users criticize git log noise from per-entry commits. Three alternatives were evaluated across two council rounds: side-branch (separate ref for ledger entries), amend (fold into prior commit), and stage+flush (batch ledger entries before pushing).

**Decision:** Keep separate commits. Side-branch was rejected (clone gap, push regression), amend was rejected (<20% success rate, breaks `filterLedgerOnlyCommits`), stage+flush was viable but unnecessary. Document the rationale in `docs/design-decisions.md`, surface a one-liner in `prime` output, and note it in `log --help`.

**Consequences:**
- `filterLedgerOnlyCommits` continues to work because the separation is structural.
- Git log noise persists but is cosmetic — already mitigated for agents and trivially filterable for humans (`git log --invert-grep --grep="^timbers: document"`).
- Documentation now answers the criticism proactively, reducing repeated user pushback.
- Locks in the per-commit cadence as a hook-enforced contract, not just convention.
