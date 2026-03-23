+++
title = 'Decision Log'
date = '2026-03-22'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

# Decision Log

## ADR-1: Tiered Environment Classification for Hook Installation

**Context:** Timbers was "stompy" in multi-tool environments — its backup-and-chain strategy for git hooks took ownership of hook files, conflicted with beads' `core.hooksPath`, and silent skips left agents unaware of their options. The question was how to coexist gracefully when other tools also want to own the pre-commit hook.

**Decision:** Classify the hook environment into four tiers (uncontested / existing / known-override / unknown-override) and use an append-section strategy instead of backup-and-chain. Refuse symlinks and binaries with guidance rather than maintaining a rename-and-chain fallback. Steering via Claude Code's Stop hook is the primary enforcement mechanism; git hooks are a bonus, so `init` does not prompt for hook installation in Tier 1.

**Consequences:**
- Timbers no longer overwrites or takes ownership of hook files, eliminating conflicts with beads and other tools
- Append-section eliminates the need for `.original` / `.backup` rename bookkeeping entirely
- Symlinks and compiled binaries are explicitly unsupported — users with exotic hook setups get guidance, not silent corruption
- Adding an interactive hook prompt to `init` was rejected as a UX regression, meaning uncontested environments get hooks silently without user opt-in
- Doctor and `hooks status` now give tier-aware messaging, so agents and users can understand why hooks aren't active

## ADR-2: Install Hooks to `core.hooksPath` as a Band-Aid

**Context:** Beads sets `core.hooksPath=.beads/hooks/`, redirecting git away from `.git/hooks/`. Timbers was hardcoding `.git/hooks/` for hook installation, so hooks were installed where git never reads them. The deeper problem: both beads and timbers want to own the pre-commit hook, and `core.hooksPath` is winner-take-all.

**Decision:** Read `core.hooksPath` from git config and install there, falling back to `.git/hooks/`. Acknowledged as a band-aid — the real fix requires beads to provide a plugin/chain mechanism so timbers can register without owning the hook file.

**Consequences:**
- Hooks now actually execute in beads-managed repos, fixing a silent failure
- `--chain` correctly backs up whatever exists at the resolved hooks directory
- The underlying ownership conflict remains: whichever tool writes last wins the pre-commit hook
- Creates implicit coupling to beads' directory layout — if beads changes its hooks path, timbers follows automatically via git config, but the append-section content may need updating
- Documents a known architectural debt that should be resolved by a beads plugin mechanism

## ADR-3: Three-Voice Essay Structure for Devblog Template

**Context:** The devblog template used a Carmack `.plan`-style stream-of-consciousness format. This produced flat recaps — readable but lacking narrative arc or distinct takeaways. The question was whether to iterate on the `.plan` style or switch to a fundamentally different structure.

**Decision:** Replace stream-of-consciousness with a three-voice essay structure (Storyteller / Conceptualist / Practitioner) using a Hook → Work → Insight → Landing scaffold. Named voices give the LLM generating the post distinct perspectives to inhabit rather than a single monotone recap.

**Consequences:**
- Posts have clearer narrative structure with identifiable takeaways
- The three-voice constraint forces richer analysis — the Conceptualist voice must find a generalizable insight, the Practitioner must give concrete guidance
- Template is more opinionated, which means less flexibility for entries that don't naturally decompose into three perspectives
- Adds a no-headers constraint after test generation revealed excessive markdown structure in output
- All existing site posts were regenerated, creating a visual consistency break with any cached or archived versions

## ADR-4: Hash-Derived Ports Over Hardcoded for Dolt Server

**Context:** Beads' Dolt server initially used hardcoded port 3307, causing cross-repo collisions when multiple projects ran simultaneously. A temporary fix assigned port 3308, but beads 0.59 introduced hash-derived ports as a general solution. The question was whether to keep the explicit port override or adopt the hash-derived mechanism.

**Decision:** Remove hardcoded port overrides and adopt beads 0.59's hash-derived port assignment. Hardcoded overrides defeat the collision-prevention mechanism — they're the problem masquerading as the solution.

**Consequences:**
- Each repo gets a deterministic but unique port derived from its path, eliminating cross-repo collisions without manual coordination
- Port numbers are no longer human-memorable, making manual `dolt sql-client` connections slightly harder to debug
- Configuration files (`metadata.json`, `config.yaml`) are simpler — no port fields to manage
- Requires beads 0.59+, creating a minimum version dependency
- Previously documented port 3308 workarounds in `CLAUDE.md` became obsolete and needed cleanup

## ADR-5: Full Command Syntax in Stop Hook Reason Strings

**Context:** The Claude Code Stop hook fires when timbers detects undocumented commits, telling the agent to run `timbers log`. But agents receiving just the command name would fail — they need the full invocation syntax including required flags (`--why`, `--how`) with placeholder arguments to produce a valid command.

**Decision:** Expand the Stop hook reason string to include `timbers pending` for diagnosis plus the full `timbers log` syntax with placeholder arguments, making the hook output directly actionable.

**Consequences:**
- Agents can parse the reason string and construct a valid `timbers log` invocation without consulting documentation
- The reason string is longer and more verbose for human readers
- Placeholder arguments (e.g., `"what you did"`) set expectations about required fields, reducing retry loops from missing-argument errors
- Couples the hook output format to the CLI's argument syntax — flag renames require updating the hook reason string
