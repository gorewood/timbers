# Timbers Dogfood Evaluation: Agentic DX Findings

## Overview

Three-phase evaluation of timbers' developer experience for AI agents, conducted
in a git worktree (`dogfood/tag-filtering`) with real development tasks delegated
to subagents. The gold standard comparison is beads, which inspired timbers' API.

**Date**: 2026-02-09
**Branch**: `dogfood/tag-filtering` (worktree at `../timbers-dogfood`)
**Model**: claude-sonnet-4-5 (subagents), claude-opus-4-6 (orchestrator/director)
**Beads**: timbers-2qw (epic), timbers-oxq (tag filter), timbers-6e4 (amend)

---

## Methodology

| Phase | Task | Timbers Integration | Purpose |
|-------|------|---------------------|---------|
| 1 | Add `--tag` to query | Documentation-only (mentioned in prompt) | Baseline: will agents use timbers voluntarily? |
| 2 | Add `timbers amend` command | Pre-injected `timbers prime` output | Test: does automatic injection change behavior? |
| 3 | Add `--tag` to export (continuation) | Pre-injected `timbers prime` + prior entries | Consumption test: do prior entries help orientation? |

Each subagent received the same model (Sonnet), same repo, same quality gates.
The only variable was how timbers context was provided.

---

## Key Findings

### Finding 1: Automatic Injection Is Non-Negotiable

**Severity: Critical**

| Metric | Phase 1 (docs-only) | Phase 2 (prime injected) | Phase 3 (prime injected) |
|--------|---------------------|--------------------------|--------------------------|
| Ran `timbers prime`? | No | N/A (pre-injected) | N/A (pre-injected) |
| Ran `timbers pending`? | No | Yes | Yes |
| Ran `timbers log`? | No | Yes | Yes |
| Followed close protocol? | No | Yes | Yes |

**Phase 1**: The subagent was told "This project uses timbers for capturing
development context" and given the binary path. It completed excellent work
(tag filtering with OR semantics, tests, clean code) but **never touched timbers**.
The "why" (OR semantics rationale) went into the commit message body instead.

**Phase 2**: With `timbers prime` output pre-injected at the top of the prompt
(simulating the SessionStart hook), the subagent followed the session close
protocol, ran `timbers pending`, and created a ledger entry.

**Phase 3**: Same injection pattern, same result — agent used timbers.

**Conclusion**: The `timbers setup claude` SessionStart hook is the difference
between 0% and 100% adoption. Documentation-only integration is insufficient.
This validates the agent-dx-guide's "automatic injection" principle and matches
beads' success pattern (beads is auto-injected via orchestrator skill).

**Recommendation**: Make `timbers setup claude` part of `timbers init` by default
(it already is, but emphasize this in onboarding). Consider adding a `--strict`
mode to the pre-commit hook that blocks when entries are missing.

---

### Finding 2: "Why" Enrichment Is Partial

**Severity: Medium**

The core value proposition of timbers is capturing session context that would
otherwise be lost — the "enriched why." Here's how each phase performed:

**Phase 1 entry** (written by director, not subagent):
> "Chose OR semantics over AND because agents typically want broad discovery —
> AND can be composed with multiple queries. Cobra StringSliceVar handles both
> repeated flags and comma separation for free."

This captures the *reasoning behind design decisions* — exactly what timbers aims for.

**Phase 2 entry** (written by subagent):
> "Users needed ability to fix typos, add missing context, or update summary
> fields in existing entries without recreating them"

This is a *feature description*, not enriched session context. The actual design
decisions (separate command vs --force flag, no audit trail, partial updates)
went into the commit message body, not the timbers entry.

**Phase 3 entry** (written by subagent):
> "Export command needed tag filtering capability to match query command
> functionality. Users expect consistent filtering options across all commands
> that retrieve entries."

Again, a feature description. Not wrong, but not capturing the decision context
that makes timbers entries valuable.

**Analysis**: Agents default to writing *what* they built (feature description)
rather than *why they made specific choices* (session context). The what/why/how
structure helps, but agents fill "why" with product rationale rather than
implementation rationale.

**Recommendation**: The `timbers prime` workflow instructions should explicitly
coach agents on what makes a good "why":
```
# What makes a good "why"
BAD:  "Users needed this feature" (product rationale — already in the ticket)
GOOD: "Chose OR semantics over AND because agents need broad discovery;
       AND can be composed via multiple queries" (design decision context)
```

Consider adding `--why` hint text or examples in `timbers log --help`.

---

### Finding 3: Entry-Commit Ordering and Misdirected Work

**Severity: Medium**

Phase 3 subagent created a timbers entry in the worktree but committed the code
to main instead of the worktree branch. The code works (`just check` passes),
but the timbers entry in the worktree references an anchor commit on a different
branch — creating a state where the entry and its documented code are in
different locations.

Additionally, the entry was created in the same session as the code — meaning
if the implementation had failed, a "phantom entry" would exist documenting
work that doesn't compile. The current protocol doesn't enforce ordering.

**Root cause**: (1) Subagent navigated to the wrong directory for git operations.
(2) `timbers log` writes a git note independently of git commit — there's no
enforcement that the anchor commit's code actually works.

**Recommendation**: Add guidance to the session close protocol to log AFTER commit:
```
# Session Close Protocol (revised order)
1. just check                    (verify quality gate)
2. git commit                    (commit code)
3. timbers log "..." --why "..." (document committed work)
4. timbers notes push            (sync ledger)
```

Consider a `timbers log --verify-committed` flag that checks the anchor commit
is reachable from HEAD before writing the entry.

---

### Finding 4: Subagents Consume Prior Entries Effectively

**Severity: Positive**

Phase 3 subagent was instructed to use `timbers show` to understand the prior
tag filtering work. The summary confirms it read the entry and understood the
OR semantics decision, then correctly replicated the same pattern for export.

**However**: The subagent's implementation failed (modified shared function
signatures without updating all callers), suggesting that reading the entry
gave it understanding of *design decisions* but not *implementation details*
sufficient to avoid breaking changes.

**Conclusion**: Timbers entries are useful for orientation ("what was done and
why") but don't replace code reading for implementation details. This is the
correct role — timbers captures the context that code alone doesn't reveal.

---

### Finding 5: Comparison with Beads DX

**Pattern-level comparison** (not tool-level, since they serve different purposes):

| Pattern | Beads | Timbers | Assessment |
|---------|-------|---------|------------|
| `prime` context injection | Excellent — auto-injected via orchestrator skill | Good — works when SessionStart hook active | Beads has deeper integration (skill-level), timbers relies on hook |
| `pending`/`ready` workflow | `bd ready` integral to orchestrator loop | `timbers pending` works but skipped without injection | Same pattern, same dependency on injection |
| `doctor` health check | Comprehensive, used proactively | Good, but not tested in this evaluation | Comparable |
| Error recovery | Structured JSON + hints | Structured JSON + hints | Comparable |
| Token efficiency | `bd close <id1> <id2>` batch close | `timbers log --batch` batch documentation | Different operations, both efficient |
| Close protocol | Auto-synced via hooks | Checklist in prime output | Beads hooks are more automatic |

**Key difference**: Beads is woven into the orchestrator's workflow firmware
(the `dm-work:orchestrator` skill automatically calls `bd prime`, `bd ready`,
`bd sync`). Timbers relies on the lighter-touch SessionStart hook injection.
This works but is more fragile — if the hook isn't installed, timbers is invisible.

**Recommendation**: Consider a timbers skill for Claude Code that integrates
at the same depth as beads' orchestrator integration.

---

## Friction Catalog

| # | Friction Point | Severity | Evidence | Recommendation |
|---|----------------|----------|----------|----------------|
| F1 | Without injection, agents skip timbers entirely | Critical | Phase 1: 0% usage | Ensure `setup claude` is default in `init` |
| F2 | "Why" fields capture feature rationale, not design decisions | Medium | Phase 2-3: generic whys | Add coaching to prime output + log help text |
| F3 | Entries can be created for uncommitted work | Medium | Phase 3: phantom entry | Add close protocol ordering guidance; consider `--verify-committed` |
| F4 | Shared helper refactoring breaks callers | Low | Phase 3: compilation errors | Not a timbers issue — general agent coding problem |
| F5 | `timbers prime` output doesn't show entry details | Low | Phase 3 was told to use `show` separately | Consider `--verbose` flag on prime that includes why/how of recent entries |

---

## Metrics Summary

| Metric | Value |
|--------|-------|
| Phases executed | 3 |
| Subagent sessions | 3 |
| Commits produced | 2 (Phase 1 + Phase 2) |
| Timbers entries created by agents | 2 (Phase 2 + Phase 3) |
| Timbers entries created by director | 1 (Phase 1 retrospective) |
| Quality gate passes | 3/3 (Phase 3 code works but committed to wrong branch) |
| Voluntary timbers usage (no injection) | 0% |
| Timbers usage with injection | 100% |
| "Why" enrichment score (0=paraphrase, 1=minor, 2=significant) | Phase 1 (director): 2, Phase 2: 1, Phase 3: 0.5 |

---

## Recommendations (Prioritized)

### P0: Must-Do
1. **Coach "why" quality in prime output** — Add explicit guidance on what makes
   a valuable "why" vs a feature description. This is the core value prop.
2. **Enforce close protocol ordering** — Log after commit, not before.

### P1: Should-Do
3. **Add `--verify-committed` flag** — Prevent phantom entries by checking anchor
   is reachable from HEAD.
4. **Enrich prime `--verbose` mode** — Include why/how of recent entries so new
   sessions get design decision context without extra `show` calls.
5. **Consider a timbers Claude Code skill** — Deeper integration like beads'
   orchestrator skill, not just a SessionStart hook.

### P2: Nice-to-Have
6. **Add "why" examples to `timbers log --help`** — Show the difference between
   product rationale and design decision context.
7. **Validate entry quality** — Optional lint/check that scores "why" richness
   and warns if it looks like a commit paraphrase.

---

## Conclusion

Timbers' agentic DX works when the integration stack is active. The SessionStart
hook (simulated by pre-injecting `timbers prime`) changes agent behavior from
"completely ignores timbers" to "follows the full workflow." This validates the
agent-dx-guide's core thesis: automatic injection beats documentation.

The main gap is "why" enrichment quality. Agents use timbers when prompted, but
write feature descriptions rather than capturing the session-specific design
decisions that are timbers' unique value. This is a coaching problem, not a tool
problem — the prime output needs to explicitly teach agents what a good "why"
looks like.

Compared to beads, timbers' integration is lighter (hook-based vs skill-based)
but follows the same successful patterns. The tool is ready for real-world use;
the improvement opportunities are in coaching and edge case handling, not in
fundamental architecture.
