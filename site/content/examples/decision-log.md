+++
title = 'Decision Log'
date = '2026-03-22'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

# Timbers Decision Log — 2026-03-04 to 2026-03-13

## ADR-1: Append-Section Over Backup-and-Chain for Hook Installation

**Context:** Timbers needed to install git hooks in repos where other tools (beads, husky) might already own the hook file. The original strategy was backup-and-chain: rename the existing hook to `.backup`, install timbers' hook, and have it exec the backup at the end. This took ownership of the hook file and conflicted with tools like beads that use `core.hooksPath`. The question was how to coexist in multi-tool environments without stomping on other tools' hooks.

**Decision:** Replace backup-and-chain with an append-section strategy. Timbers appends a clearly-delimited section to the existing hook file rather than replacing it. A four-tier environment classification (uncontested/existing/known-override/unknown-override) determines messaging and behavior. For symlinks and binaries, timbers refuses with guidance rather than maintaining a second code path. Interactive prompting during `init` was rejected — steering via Claude Code's Stop hook is the primary enforcement mechanism, git hooks are a bonus.

**Consequences:**
- Positive: Eliminates hook ownership conflicts with beads and other tools. No more `.backup` file management. Simpler code path — no rename-and-restore logic.
- Positive: Tiered messaging means `init` and `doctor` give contextually appropriate guidance instead of silent skips.
- Negative: Timbers can no longer guarantee its hook section runs first or last — execution order depends on position in the file.
- Negative: Refuses to handle symlinks/binaries, requiring manual intervention in those environments.

## ADR-2: Respect `core.hooksPath` as Band-Aid Pending Plugin Mechanism

**Context:** Beads sets `core.hooksPath=.beads/hooks/` to redirect git hooks away from `.git/hooks/`. Timbers was hardcoding `.git/hooks/` for hook installation, meaning hooks were installed where git never reads them. Both tools want to own the pre-commit hook, and `core.hooksPath` is winner-take-all.

**Decision:** `GetHooksDir` now reads `git config core.hooksPath` and falls back to `.git/hooks/`. This is explicitly a band-aid — the real fix requires beads to implement a plugin/chain mechanism so timbers can register without owning the hook file. For now, timbers installs to wherever `core.hooksPath` points and `--chain` backs up whatever was there.

**Consequences:**
- Positive: Hooks actually work in beads-managed repos immediately.
- Negative: Still a single-owner model — whichever tool installs last wins. Fragile across tool version upgrades that reinstall hooks.
- Constraint: Future beads versions need a plugin mechanism to resolve this properly. This decision creates a known dependency on upstream.

## ADR-3: Accuracy Over Speed for `HasPendingCommits`

**Context:** The original `HasPendingCommits` used a fast path (~15ms) that simply compared HEAD against the anchor commit. The implementation plan noted "may false-positive on ledger-only commits; acceptable trade-off for ~15ms." Testing in a real repo immediately revealed this blocked every session end, because `timbers log` auto-commits an entry file which changes HEAD — making the Stop hook useless.

**Decision:** Delegate to `GetPendingCommits` which filters ledger-only commits via `CommitFilesMulti` batch lookup. This adds one more git subprocess per call (~85ms slower) but runs once per session. The fresh-repo exemption is preserved by checking `latest==nil`.

**Consequences:**
- Positive: Stop hook actually works — no false-positive blocking on every session end.
- Positive: Correctness is non-negotiable for a gate that blocks agent workflows.
- Negative: ~100ms per invocation instead of ~15ms. Acceptable since it runs once per session, not in a hot path.
- Lesson: "Acceptable trade-off" in a plan needs validation against real usage before shipping. The fast path saved 85ms but made the feature useless.

## ADR-4: Unique Ports Over Default for Cross-Repo Dolt Isolation

**Context:** Multiple repos using beads with Dolt all defaulted to port 3307. Cross-repo collisions caused misleading errors ("port in use"), and Claude Code's sandbox blocking localhost TCP was misdiagnosed as stale PIDs — two different failure modes with the same symptom.

**Decision:** Moved from hardcoded port 3307 to unique ports per repo (initially 3308, then beads 0.59 introduced hash-derived ports, and 0.60 moved to OS-assigned ephemeral ports). Hardcoded overrides were removed because they defeat the collision-prevention mechanism.

**Consequences:**
- Positive: Eliminates cross-repo port collisions entirely with ephemeral ports.
- Positive: No configuration needed — ports are assigned automatically.
- Negative: Port is no longer predictable, which complicates manual debugging (must check metadata to find the active port).
- Negative: Required three iterations (hardcoded → hash-derived → ephemeral) to reach the right design.

## ADR-5: Chained Hook Must Check Exit Code Before Delegating

**Context:** When timbers' pre-commit hook was chained with a backup hook, it unconditionally `exec`'d the backup after running. This meant timbers' blocking behavior (exit non-zero on pending commits) was silently overridden — the backup hook's exit code replaced timbers', so commits always succeeded regardless of pending state.

**Decision:** Capture the exit code after the timbers hook runs and exit early if non-zero, before exec'ing the backup hook.

**Consequences:**
- Positive: Timbers can actually block commits when pending entries exist.
- Negative: If timbers has a bug that incorrectly returns non-zero, the chained hook never runs. Timbers becomes a hard gate in the chain.

## ADR-6: Full Command Syntax in Stop Hook Reason Strings

**Context:** The Claude Code Stop hook fires when agents have undocumented commits. The original reason string said just "timbers log" — agents receiving this would fail because they didn't know the required `--why`/`--how` flags.

**Decision:** Expanded the reason string to include `timbers pending` plus the full `timbers log` syntax with placeholder arguments. The hook message must be actionable without external documentation.

**Consequences:**
- Positive: Agents can act on the hook message without consulting docs or prior context.
- Negative: Longer reason string — more tokens consumed in agent context for every Stop hook firing.
- Principle: Agent-facing messages must be self-contained and executable. A command name without its required flags is not actionable.

## ADR-7: Backup State Excluded from Git Tracking

**Context:** `.beads/backup/` contained timestamps and Dolt commit hashes that changed on every `bd` operation. These files were tracked in git, creating noisy diffs on every beads interaction.

**Decision:** Gitignore `.beads/backup/` and untrack existing files. Backup state is machine-local runtime data, not project state.

**Consequences:**
- Positive: Eliminates noisy diffs from routine beads operations.
- Positive: Reduces accidental commits of machine-specific state.
- Negative: Backup state is no longer recoverable from git history if the local machine is lost. This is acceptable because backups are reconstructible from the Dolt remote.
