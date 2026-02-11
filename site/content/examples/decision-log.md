+++
title = 'Decision Log'
date = '2026-02-10'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 30 --model opus`

---

## ADR-1: Project-Level Claude Hooks Over Global Hooks

**Context:** The `timbers setup claude` command originally installed hooks at the global level. This meant the `timbers prime` hook ran in every repository, even uninitiated ones where it was a no-op. More critically, global git hooks conflicted with other tools like `beads` that rely on `pre-commit` for critical operations. Since timbers already requires per-repo `timbers init`, the global scope was a mismatch.

**Decision:** Default hook installation was switched from global to project-level, with `--global` available as an explicit opt-in flag. Git hooks were also changed from default-on (`--no-hooks` to disable) to opt-in (`--hooks` to enable).

**Consequences:**
- Positive: Eliminates interference with other tools that use global git hooks
- Positive: No wasted execution of `timbers prime` in repos that haven't been initialized
- Positive: Scope of hook installation matches the scope of `timbers init` (per-repo)
- Negative: Users who want timbers active everywhere must explicitly pass `--global`
- Negative: Opt-in `--hooks` means fewer users will discover the git hook integration unless documentation guides them to it

---

## ADR-2: Rename `prompt` Command to `draft`

**Context:** The command for generating documents (changelogs, release notes, etc.) from ledger entries was originally named `prompt`, reflecting its LLM-prompting mechanics. The choice was between keeping the technically accurate name and adopting one that communicated the user-facing action.

**Decision:** Renamed `prompt` to `draft` because "prompt" is developer/LLM jargon that undersells the document-generation capability. "Draft" reads as the action it performs — *draft a changelog*, *draft release notes* — making it self-documenting to users unfamiliar with the LLM pipeline underneath.

**Consequences:**
- Positive: More intuitive command naming for non-LLM-savvy users
- Positive: Better discoverability — agents and users understand `draft` as a generative action
- Negative: Breaking rename for anyone who scripted against `timbers prompt`
- Negative: Loses the direct signal that this command constructs an LLM prompt, which may confuse contributors working on the internals

---

## ADR-3: Markdown Output Over JSON for Changelogs

**Context:** The `timbers changelog` command needed an output format. The two main candidates were structured JSON (consistent with `timbers export`) and Markdown. The intended audience for changelogs is humans reading release notes, not machines consuming data.

**Decision:** Default to Markdown output because changelogs are human-readable documents, not machine data. JSON is available as an alternative output path but is not the default.

**Consequences:**
- Positive: Output is immediately usable — paste into a release, commit as `CHANGELOG.md`, or pipe to further formatting
- Positive: Matches the conventions of Keep a Changelog and GitHub Releases
- Negative: Downstream tooling that wants structured data must explicitly request `--json` or use `export` instead
- Negative: Markdown formatting choices (heading levels, grouping) become part of the tool's contract and harder to change later

---

## ADR-4: Group-by-Tag with Duplication Over Primary-Tag Assignment for Changelogs

**Context:** When an entry has multiple tags, the changelog grouping had two options: assign each entry to a single "primary" tag section, or duplicate entries across all matching tag sections. The trade-off is between deduplication (cleaner, shorter output) and discoverability (readers scanning a single section see everything relevant).

**Decision:** Duplicate entries across all matching tag groups because discoverability in context matters more than deduplication. A reader scanning the "api" section should see all API-related changes, even if some also appear under "bugfix."

**Consequences:**
- Positive: Each tag section is self-contained — readers don't miss relevant entries
- Positive: No need for a "primary tag" heuristic, which would be fragile and opinionated
- Negative: Changelogs are longer; repeated entries may confuse readers who read top-to-bottom
- Negative: Entry counts per section don't sum to total entries, which can be misleading in metrics

---

## ADR-5: OR Semantics for Tag Filtering Over AND

**Context:** The `--tag` flag on `timbers query` (and later `export`) needed to define how multiple tags combine. The choice was between OR semantics (entry matches if it has *any* specified tag) and AND semantics (entry must have *all* specified tags).

**Decision:** Chose OR semantics because agents and users typically want broad discovery — finding everything related to any of several topics. AND semantics can be composed by running multiple queries, but OR cannot be easily reconstructed from AND-only filtering. Additionally, Cobra's `StringSliceVar` handles both repeated flags and comma separation for free, making multi-tag OR natural.

**Consequences:**
- Positive: Better for exploratory discovery — broad net catches more relevant entries
- Positive: Simpler mental model for the common case of "show me anything tagged X or Y"
- Negative: No built-in way to express "entries tagged both X and Y" in a single query
- Negative: Results may be noisy when tags are broad; users must post-filter manually for intersection queries

---

## ADR-6: `.env.local` File Fallback for API Keys Over Environment-Only Configuration

**Context:** Timbers needs API keys (e.g., `ANTHROPIC_API_KEY`) to call LLM providers. Setting these as environment variables is standard, but Claude Code specifically conflicts with `ANTHROPIC_API_KEY` in the environment — it confuses the key with its own OAuth flow. An alternative delivery mechanism was needed.

**Decision:** Introduced `.env.local` (and `.env`) file support, loaded as fallback — file values are only set for variables *not already in the environment*. This is loaded in the root command's `PersistentPreRunE`.

**Consequences:**
- Positive: Cleanly sidesteps the Claude Code / `ANTHROPIC_API_KEY` conflict
- Positive: Environment variables still take precedence, preserving twelve-factor conventions
- Positive: `.env.local` is gitignore-friendly, reducing risk of key leakage
- Negative: Adds a file-loading step to every command invocation
- Negative: Users must understand the precedence order (env > `.env.local` > `.env`) to debug key resolution issues
- Negative: Yet another dotfile in the project root

---

## ADR-7: Centralized Config Directory with XDG and Cross-Platform Fallbacks

**Context:** The config path (`~/.config/timbers`) was hardcoded in two places — the env loader and the template resolver. This was wrong on Windows (where `AppData` is conventional) and didn't respect `XDG_CONFIG_HOME` on Linux. The alternatives were: keep the hardcoded path, add a single env var override, or implement full cross-platform resolution.

**Decision:** Created an `internal/config` package with a `Dir()` function that checks `TIMBERS_CONFIG_HOME` → `XDG_CONFIG_HOME` → Windows `AppData` → `~/.config/timbers` fallback. Both the env loader and template resolver now call `config.Dir()`.

**Consequences:**
- Positive: Correct behavior on Windows without special-casing at call sites
- Positive: Respects XDG conventions on Linux, fitting into existing dotfile management
- Positive: `TIMBERS_CONFIG_HOME` override enables testing and non-standard setups
- Positive: Single source of truth eliminates path drift between components
- Negative: More complex resolution logic to understand and debug
- Negative: Config location is now less predictable — users must run `timbers doctor` or read docs to find where config actually lives

---

## ADR-8: Default to All Entries for Changelog Unlike Export/Query

**Context:** The `export` and `query` commands require explicit scoping (`--last N`, `--since`, commit ranges). The `changelog` command faced the same design question: should it require explicit scoping, or default to all entries?

**Decision:** Default to all entries for `changelog` because a changelog without content is useless, and requiring `--last 999` or similar is hostile UX. This deliberately diverges from `export`/`query` conventions where explicit scoping prevents accidentally dumping large datasets.

**Consequences:**
- Positive: Zero-argument `timbers changelog` produces a useful, complete document immediately
- Positive: Matches user expectation — "generate my changelog" means "all of it"
- Negative: Inconsistent defaulting across commands may confuse power users
- Negative: Could be slow or produce very large output in repos with thousands of entries — no guard against accidental large dumps

---

## ADR-9: Init Creates Empty Notes Ref So Prime Works Immediately

**Context:** After `timbers init`, the `timbers prime` command (which injects workflow context into agent sessions) would silently exit if no git notes ref existed yet. Three independent reviewers identified this as the highest-priority adoption blocker: new users would run `init`, then `prime` would appear broken with no feedback.

**Decision:** `timbers init` now creates an empty notes namespace via git plumbing (`mktree` + `commit-tree` + `update-ref`) as a step before remote configuration. This ensures the notes ref exists immediately after init, even before any entries are logged.

**Consequences:**
- Positive: `prime` works immediately after `init` — no silent failure, no stalled onboarding
- Positive: Eliminates the confusing gap between "initialized" and "actually usable"
- Negative: Creates a notes ref with an empty tree commit, which is a slightly unusual git state
- Negative: Adds git plumbing calls to the init path, making init marginally more complex and harder to test
