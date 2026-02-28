+++
title = 'Pr Description'
date = '2026-02-27'
tags = ['example', 'pr-description']
+++

Generated with `timbers draft pr-description --last 20 | claude -p --model opus`

---

## Why

This PR ships the v0.9.0 coaching rewrite and v0.10.0 content safety features. The coaching system was rewritten because generic rules like "be specific" weren't producing useful `--why` fields — agents need motivated rules with examples to internalize the difference between feature descriptions and design verdicts. Content safety was added after realizing entries are committed to git and could leak secrets in public repos.

## Design Decisions

- **Motivated rules over imperatives**: Every coaching rule now explains *why* it matters, not just what to do. This follows the Opus 4.6 prompt guide finding that motivation enables generalization — agents apply the principle to novel situations instead of pattern-matching on examples alone.
- **5-point notes trigger over vague guidance**: Replaced "use when you made a real choice" with a concrete checklist (2+ approaches, rejected obvious, surprise, lock-in, non-obvious). Concrete triggers reduce false negatives without being prescriptive.
- **XML tags in coaching strings**: Adopted despite initial concerns about format contracts — coaching is a Go string constant consumed only by LLMs, so there's no API boundary to break. XML tags give section boundaries that models parse reliably.
- **`AgentEnv` registry pattern**: Decoupled `init`/`doctor`/`setup` from Claude-specific assumptions. Chose registry over interface dispatch because new agent environments should be additive (register, don't modify).

## Risk & Reviewer Attention

- `prime_workflow.go` coaching content is a Go string constant with embedded XML — verify the escaping is correct and the closing tags match.
- The `AgentEnv` registry in `internal/setup/` is new infrastructure. Check that `init` and `doctor` correctly iterate registered environments and that the Claude environment registers itself via `init()`.
- Content safety validation in `timbers log` rejects entries with patterns matching API keys — verify the regex doesn't false-positive on legitimate hex strings in commit SHAs.

## Scope

Concentrated in the CLI layer (`cmd/timbers/`) and setup infrastructure (`internal/setup/`). Templates, storage, and git operations are untouched. The coaching changes are pure content — no structural changes to how prime output is assembled.

## Test Plan

- Run `timbers prime` and verify coaching sections include BAD/GOOD examples and motivation text
- Run `timbers doctor` and verify `AgentEnv` checks appear in the INTEGRATION section
- Run `timbers log` with a test entry containing `sk-ant-...` and verify it's rejected
- Run `just check` to confirm lint and tests pass
