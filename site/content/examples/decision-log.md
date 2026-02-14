+++
title = 'Decision Log'
date = '2026-02-14'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 25 | claude -p --model opus`

---

## ADR-1: Auto-Commit Entry Files on `timbers log`

**Context:** After switching from git-notes to `.timbers/` directory storage, running `timbers log` would create and stage an entry file but leave it uncommitted. Users had to remember a manual `git commit` step, and the staged-but-uncommitted gap caused confusion — entry files sitting in the index could get swept into unrelated commits or lost on branch switches.

The original git-notes design didn't have this problem because notes lived in a separate ref. The `.timbers/` approach introduced it but never added an automatic commit step. Two alternatives were considered: committing on the working branch (simple, filesystem-visible) or storing entries on a separate branch (like beads/entire.io, cleaner separation but invisible to `timbers prime`/`draft` without worktree indirection).

**Decision:** `timbers log` auto-commits the entry file using `git commit -m ... -- <path>` with pathspec scoping. Entries stay on the working branch for filesystem visibility. The commit is scoped with `--` to prevent sweeping other staged files into the timbers commit.

**Consequences:**
- Positive: Eliminates the manual commit step and the staged-but-uncommitted gap entirely
- Positive: Agent DX preserved — `timbers prime` and `timbers draft` can read entries without worktree indirection
- Positive: Pathspec scoping (`--`) is a safety mechanism that prevents accidentally committing unrelated staged work
- Negative: Every `timbers log` creates a commit on the working branch, adding noise to `git log`
- Constraint: Entries must remain as working-branch files rather than a separate ref, which couples the ledger to the branch history

---

## ADR-2: `--color` Flag Over Full Theme Configuration

**Context:** Users on Solarized Dark terminals reported that color 8 (bright black) used for dim/hint text was completely invisible. Terminals don't expose their color scheme to applications, so automatic detection isn't possible. The options ranged from a simple override flag (`--color never/auto/always`) to full theme configuration via environment variables and config files. Lipgloss v1.1.0 also offers `AdaptiveColor` and `HasDarkBackground()` that weren't yet in use.

**Decision:** Ship a global `--color` persistent flag with `never/auto/always` values, plumbed through `ResolveColorMode` to all `NewPrinter` call sites. Defer `AdaptiveColor` and `HasDarkBackground()` to a future iteration.

**Consequences:**
- Positive: Covers ~95% of terminal compatibility issues with minimal implementation surface
- Positive: Persistent flag means users set it once, not per-command
- Negative: Users with exotic color schemes still can't customize individual colors — only disable them entirely
- Constraint: Full theme support deferred, creating a known gap for users who want colors but different ones

---

## ADR-3: Registry Pattern Over Switch Statement for Agent Environments

**Context:** Timbers needed to support multiple agent environments (Claude Code, and eventually Gemini, Codex, etc.) for `init`, `doctor`, and `uninstall` commands. Two patterns were considered: a centralized switch statement where each addition touches multiple files, or a registry pattern where each environment is self-contained in a single file and registers itself via `init()`.

**Decision:** Registry pattern with an `AgentEnv` interface (`Detect`/`Install`/`Remove`/`Check` methods). `ClaudeEnv` wraps existing setup functions. New environments register via `init()` with stable ordering maintained by the registry.

**Consequences:**
- Positive: Adding a new agent environment (Gemini, Codex) is a single-file task — no changes to existing code
- Positive: Interface methods map naturally to what `doctor`/`setup`/`init` already need
- Negative: One file per environment adds slight file count overhead
- Negative: `init()` registration makes the dependency graph implicit — you can't see what's registered without checking each file
- Enables: `doctor` iterates `AllAgentEnvs()` for health checks; `uninstall` gathers all detected envs via `AgentEnvState` slice

---

## ADR-4: Notes Field Coaches by Question, Not by Structure

**Context:** The ledger had `what`/`why`/`how` fields but no way to capture the *journey* to a decision — the alternatives considered, surprises encountered, reasoning chains. A council deliberated the design. The key question was how to coach agents to write good notes: provide a structured template (headings like "## Alternatives", "## Decision") or use the same question-based coaching that proved effective for the `why` field.

**Decision:** Notes captures deliberation as free-form "thinking out loud," coached by question ("What would help someone revisiting this decision in 6 months?") rather than by imposed structure. The `why` field holds the verdict; `notes` holds the journey. A concrete 5-point trigger checklist determines when notes are warranted: 2+ viable approaches, rejected an obvious approach, encountered surprise, creates lock-in, or non-obvious to a teammate.

**Consequences:**
- Positive: Follows the proven pattern — `why` coaching by question already produced high-quality verdicts
- Positive: Free-form notes are richer for template output — the decision-log template can extract genuine fork-in-the-road context
- Positive: Trigger checklist prevents both over-use (form-filling on every commit) and under-use (never writing notes)
- Negative: Free-form means inconsistent structure across entries, harder to parse programmatically
- Constraint: `why` and `notes` must be clearly differentiated in coaching or agents will put journey content in `why`

---

## ADR-5: Motivated Rules Over Imperative Density in Coaching

**Context:** The prime coaching text used 11 instances of `MUST`/`CRITICAL` to enforce rules. Analysis of the Opus 4.6 prompt guide revealed that imperative density causes overtriggering — the model treats everything as equally critical and can't prioritize. A council debated three approaches: generic clarity improvements, Opus-specific tuning, and a pragmatist middle ground. All three converged on the same conclusion.

**Decision:** Replace imperative shouting with motivated rules — each rule explains *why* it exists, enabling the model to generalize correctly. Added XML section tags for structure, concrete BAD/GOOD examples, and calm framing. No model-specific coaching variants needed because good coaching IS Opus-optimized coaching.

**Consequences:**
- Positive: Models generalize better when they understand the reason behind a rule, not just the demand
- Positive: Single coaching text works across models — no maintenance burden of model-specific variants
- Positive: XML tags provide clear section boundaries at zero cost (coaching is a Go string consumed only by LLMs)
- Negative: Motivated rules are longer than bare imperatives — coaching text grows
- Constraint: BAD/GOOD examples must be maintained as the tool evolves or they become misleading

---

## ADR-6: Stdin-Based Hook Input Over Environment Variables

**Context:** The PostToolUse hook that reminded users to run `timbers log` after `git commit` had been silently broken since creation. It read `$TOOL_INPUT` from an environment variable, but Claude Code hooks receive JSON on stdin — `$TOOL_INPUT` was always empty. Two fix approaches: parse stdin with `jq` to extract the specific `tool_input.command` field, or grep the raw JSON blob from stdin for `git commit`.

**Decision:** Read stdin directly with `grep` rather than parsing with `jq`. Also added upgrade logic (`hasExactHookCommand` detection + `removeTimbersHooksFromEvent` cleanup) so reinstall replaces stale hooks rather than skipping them.

**Consequences:**
- Positive: No dependency on `jq` — works on any system with standard Unix tools
- Positive: Upgrade logic prevents broken hooks from persisting forever (the old skip-if-any-exists behavior)
- Negative: Grepping raw JSON for `git commit` is technically imprecise — could false-positive on a commit message mentioning "git commit," though the risk is negligible in practice
- Constraint: Any future hook input format changes in Claude Code require updating the stdin reading approach

---

## ADR-7: GITHUB_TOKEN Workflow Chaining via `workflow_dispatch`

**Context:** The devblog CI workflow commits generated blog posts and pushes them, but the subsequent GitHub Pages deployment never triggered. Root cause: pushes made with `GITHUB_TOKEN` intentionally don't fire `on: push` workflows — this is GitHub's infinite-loop prevention. The devblog posts were being committed but never published.

**Decision:** Chain workflows explicitly using `workflow_dispatch`. The devblog workflow triggers `pages.yml` via the GitHub API after a successful push, with `actions: write` permission and a committed-gate check.

**Consequences:**
- Positive: Blog posts now reliably deploy after generation
- Positive: Explicit chaining is visible and auditable — no hidden coupling via push events
- Negative: Requires `actions: write` permission, which is broader than the workflow otherwise needs
- Constraint: Any new workflow that depends on devblog pushes must also be explicitly chained — implicit `on: push` triggers will never fire from `GITHUB_TOKEN` commits
