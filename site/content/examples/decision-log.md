+++
title = 'Decision Log'
date = '2026-04-20'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

## ADR-1: Caller-Supplied Template Variables via `--var` Flag with Namespacing

**Context:** The `draft` command needed a way for callers to inject arbitrary values into templates (e.g., a starting ADR number) without baking domain-specific logic into the command itself. Options included a generic flat `--var` flag, a template-owned counter in `.timbers/state.json`, or special-case flags per template.

**Decision:** Added a repeatable `--var key=value` flag, threaded through `RenderContext.Vars` and substituted under the `{{vars.key}}` namespace after built-in tokens. Rejected a state file because it creates two sources of truth that can drift (regenerating output orphans the counter); the rendered markdown is inherently durable and version-controlled, so making the caller extract state from it keeps one source of truth. Namespacing under `{{vars.*}}` (rather than a flat namespace) was chosen to keep built-in tokens sacred so callers cannot accidentally shadow them.

**Consequences:**
- Templates remain pure renderers with no per-template command surface area.
- Callers own durability — any state-like behavior is expressed through the output file, not a sidecar.
- Flat `--var foo=bar` calls always resolve as `{{vars.foo}}`, which is slightly more verbose than an unnamespaced design but eliminates a class of shadowing bugs.
- Unknown `{{vars.*}}` tokens remain literal, matching existing token behavior; this silently tolerates typos.

## ADR-2: Per-Template Default Variables in Frontmatter

**Context:** With `--var` in place, the decision-log template needed a default `starting_number` of 1 when the caller omitted it. Options included hardcoding defaults in `internal/draft/render.go`, or letting each template declare its own defaults.

**Decision:** Added a `vars:` map to template frontmatter for per-template defaults. Render applies caller-supplied vars first, then template defaults fill remaining `{{vars.*}}` tokens. Briefly considered a `--continue-from <file>` flag on `draft` itself, but that would embed decision-log-specific parsing logic into a generic command; the justfile recipe is the right place for that sugar.

**Consequences:**
- The render package stays decoupled from specific template variables — new templates can declare defaults without touching Go code.
- Template authors have a clear, local place to document expected variables.
- Callers can still override any default via `--var`.
- Two layers of resolution (caller vars, then template defaults) means debugging a wrong value requires checking both sources.

## ADR-3: Reachability Check for Stale Anchor Detection

**Context:** `GetPendingCommits` treated an anchor as stale only when the SHA was completely absent from the object store. After `git pull --rebase`, old objects linger for up to two weeks before GC, so `git log` succeeded against phantom SHAs and produced spurious "pending" commits for work already documented on the rebased branch. A user (Noam) reported this in the wild.

**Decision:** Before running `git log`, check `git merge-base --is-ancestor` to verify the anchor is reachable from `HEAD`. Added `IsAncestorOf` to the `GitOps` interface. Existence alone is insufficient; reachability is the correct invariant for "the anchor is still part of this branch's history."

**Consequences:**
- Rebased/cherry-picked history no longer produces silent phantom pending lists.
- The self-heal path now fires immediately on rebase rather than waiting weeks for GC.
- Adds one `git` subprocess call per pending check; negligible at normal repo scales.
- Slightly expands the `GitOps` interface surface.

## ADR-4: Separate Commits for Ledger Entries (Preserved by Design)

**Context:** Users surfaced that `timbers log` producing its own commit creates visible noise in `git log`. Alternatives evaluated across two council rounds: a side-branch for entries (rejected — breaks on clone, regressions on push), amending entries into the feature commit (rejected — <20% reliable and breaks `filterLedgerOnlyCommits`), and a stage-and-flush mechanism (viable but unnecessary).

**Decision:** Keep entries as separate commits. The whole pending-detection and ledger-only filtering design depends on that separation; the noise is cosmetic and already mitigated for agents via `--invert-grep` filtering and for humans via the `git log` alias suggestion. Proactively documented the rationale in `docs/design-decisions.md`, `timbers prime` output, and `timbers log --help` to head off repeated criticism.

**Consequences:**
- Reliable pending detection and clean filtering remain intact.
- `git log` shows ~2x commit count in active sessions; users must learn the filter idiom.
- The rationale is now discoverable without requiring users to open an issue.
- Future proposals to collapse entries into feature commits face an explicit, documented bar to clear.

## ADR-5: Suppress Hooks During Interactive Git Operations

**Context:** During `git rebase`, `git merge`, `git cherry-pick`, and `git revert`, the pre-commit pending-check hook created a deadlock: the agent was blocked from continuing the rebase by the pending check, and could not satisfy the check by running `timbers log` because commits are forbidden mid-rebase.

**Decision:** Added `git.IsInteractiveGitOp()` which inspects `.git/` state files (e.g., `rebase-merge/`, `MERGE_HEAD`, `CHERRY_PICK_HEAD`). All hooks early-return when an interactive op is in progress. Code review caught that the relative `gitDir` path had to be resolved against the repo root and that `pending.go` was still running `GetPendingCommits` before the interactive check; both were fixed before merge.

**Consequences:**
- Rebase/merge/cherry-pick/revert workflows no longer deadlock.
- Brief blind spot: work committed mid-rebase bypasses the pending check. This is acceptable because the check re-engages on the next normal commit.
- `GitOps` gains a filesystem-level surface (reading `.git/` state files) that must stay in sync with git internals across versions.

## ADR-6: Broaden Ledger-Only Filter to Infrastructure-Only

**Context:** The `beads` 0.63 auto-stage hook started including `.beads/issues.jsonl` in commits. Since `timbers log` commits now contained both `.timbers/` *and* `.beads/` files, the `isLedgerOnlyCommit` filter treated them as mixed and broke the timbers-on-timbers loop.

**Decision:** Renamed and generalized the predicate to `isInfrastructureOnlyCommit` with a configurable prefix list (`.timbers/`, `.beads/`). Rather than hardcoding a second tool's directory, the prefix list is the extension point for future sibling tools that piggyback on the same commit.

**Consequences:**
- Coexistence with other git-native infrastructure tools no longer requires bespoke patches.
- The filter's semantics shift from "ledger-only" to "infrastructure-only" — a subtle expansion that could mask legitimate mixed commits if a future tool writes there intentionally.
- The prefix list becomes a small piece of configuration that must be maintained as the ecosystem evolves.

## ADR-7: Union Anchor and Diff Discovery for `--range`

**Context:** The v0.15.3 `--range` fallback to diff-based discovery only triggered when the anchor-based lookup returned zero matches. Partial-stale ranges (some valid anchors, some stale) short-circuited the fallback and silently dropped the entries behind the stale anchors.

**Decision:** Always run both anchor-based and diff-based discovery and union the results by entry ID. The short-circuit optimization was unsound because the zero-match heuristic doesn't distinguish "fully resolved" from "partially resolved."

**Consequences:**
- `--range` is now correct in the partial-stale case that users actually hit after rebases.
- Slightly more work per call (two discovery passes unconditionally), but both were already cheap.
- Entry ID union is the right dedup key because entries are content-addressed, not position-addressed.
