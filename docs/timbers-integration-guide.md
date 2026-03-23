# Timbers Integration Guide for Plugin Maintainers

Guide for integrating timbers into developer workflow tooling (repo-init, repo-health, worktrees, steering docs). Written for the dm plugin maintainer.

## The Anchor Problem

Every timbers entry has an **anchor commit** — the HEAD SHA when the entry was created. `timbers pending` uses this to find undocumented work:

```
git log <latest_anchor>..HEAD
```

This breaks when the anchor SHA disappears from the current branch's history:

| Operation | Anchor survives? | Why |
|-----------|-----------------|-----|
| Merge commit (`--no-ff`) | Yes | Original SHAs preserved in history |
| Squash merge | No | Feature branch SHAs replaced by single squash commit |
| Rebase + merge | No | All SHAs rewritten during rebase |
| Rebase + fast-forward | No | SHAs rewritten, no merge commit |

**Merge commits are the only strategy that preserves anchor integrity.** Everything else causes a "stale anchor" state where `timbers pending` can't accurately determine what's documented.

### What happens on stale anchor (v0.16.0+)

- `timbers pending`: reports **0 actionable** commits with a warning (no false-positive commit list)
- Pre-commit hook: **does not block** (stale anchor is not actionable pending)
- Stop hook: **does not block** (same)
- `timbers query --range`: falls back to **file-based discovery** via `git diff --name-only` (v0.15.3+)
- Anchor **self-heals** on the next `timbers log` after a real commit on the target branch

Before v0.16.0, stale anchor dumped all reachable commits as "pending," causing agents to re-document already-covered work.

## Git Hook Integration with Beads

Timbers and beads share the pre-commit hook via section-delimited append.

### How it works

Beads owns `core.hooksPath` (pointing to `.beads/hooks/`). Its hooks use markers:

```sh
#!/usr/bin/env sh
# --- BEGIN BEADS INTEGRATION v0.59.0 ---
if command -v bd >/dev/null 2>&1; then
  export BD_GIT_HOOK=1
  bd hooks run pre-commit "$@"
  _bd_exit=$?; if [ $_bd_exit -ne 0 ]; then exit $_bd_exit; fi
fi
# --- END BEADS INTEGRATION v0.59.0 ---
```

Beads preserves content outside its markers across `bd hooks install`. Timbers appends its own delimited section after the beads section:

```sh
# --- timbers section (do not edit) ---
if command -v timbers >/dev/null 2>&1; then
  timbers hook run pre-commit "$@"
  rc=$?
  if [ $rc -ne 0 ]; then exit $rc; fi
fi
# --- end timbers section ---
```

**Key properties:**
- Both sections are idempotent (re-running install is a no-op)
- Both use section delimiters for clean removal
- `bd hooks install --force` preserves timbers section (outside beads markers)
- `timbers hooks install` / `doctor --fix` preserves beads section (appends after)
- Execution order: beads first (blocks on its own checks), timbers second (blocks on pending)

### Tier classification

Timbers classifies the hook environment before installing:

| Tier | Condition | Behavior |
|------|-----------|----------|
| 1 - Uncontested | No `core.hooksPath`, no existing hook | Create hook, append section |
| 2 - Existing | Standard `.git/hooks/`, hook exists | Append section to existing hook |
| 3 - Known Override | `core.hooksPath` = `.beads/hooks` or `.husky` | Append section at managed path |
| 4 - Unknown Override | `core.hooksPath` = unrecognized path | Skip (defer to user config) |

No `--chain` flag needed. The old backup-and-chain approach was replaced with section-delimited append in v0.15.0.

## Recommendations for Plugin Skills

### `repo-init`

When setting up a repo with timbers:

```bash
# Configure merge-friendly git settings
git config merge.ff false          # Always create merge commits
git config pull.rebase false       # Pull merges, not rebases

# Initialize timbers
timbers init                       # Sets up .timbers/, hooks, Claude Code integration

# If beads is present, install hooks into beads-managed directory
timbers hooks install              # Auto-detects core.hooksPath
```

If the repo uses GitHub PRs, recommend setting the repo's default merge strategy to "Create a merge commit" (not "Squash and merge" or "Rebase and merge").

### `repo-health`

Run `timbers doctor` as part of health checks. It already covers:

- **Pending commits**: detects stale anchor, reports actionable count
- **Merge strategy**: warns on `pull.rebase=true` or `merge.ff=only`
- **Git hooks**: tier-aware detection, auto-fix with `--fix`
- **Agent steering**: Claude Code hook presence and staleness
- **Recent entries**: ledger activity check

For programmatic consumption: `timbers doctor --json` returns structured results with `pass`/`warn`/`fail` status per check.

To auto-fix issues: `timbers doctor --fix` installs missing hooks and migrates old formats.

### `worktrees`

This is the biggest pain point for anchor integrity.

**Always use `bd worktree create`** (not `git worktree add`) to get shared beads DB.

**When merging worktree branches back to main:**
```bash
# GOOD: merge commit preserves anchors
git merge --no-ff .worktrees/feature-branch

# BAD: rebase rewrites all SHAs, breaks anchors
git rebase main  # from worktree branch — DON'T
```

The `dm-work:merge` skill should:
1. Check if the branch has `.timbers/` files (`git diff --name-only main..HEAD -- .timbers/`)
2. If yes, enforce `--no-ff` merge and warn against rebase
3. If no timbers entries on branch, any merge strategy is fine

### Steering docs (AGENTS.md / CLAUDE.md)

Add to project steering:

```markdown
## Git Merge Strategy

When merging branches that contain timbers entries, always use merge commits
(`git merge --no-ff`). Squash merges and rebases rewrite commit SHAs, breaking
timbers' anchor-based tracking. The entries survive but `timbers pending`
degrades until the next `timbers log` on the target branch.

For worktree merges: `git merge --no-ff .worktrees/<name>`
```

### Claude Code Stop Hook

The Stop hook (`timbers hook run claude-stop`) blocks session end when undocumented commits exist. On stale anchor (v0.16.0+), it **does not block** — the agent sees a clean exit.

If agents are hitting stale anchor warnings in `timbers pending` output during sessions, the fix is upstream: use merge commits instead of squash/rebase when merging branches with timbers entries.

## Version Requirements

| Feature | Minimum Version |
|---------|----------------|
| Section-delimited hooks | v0.15.0 |
| `--range` squash-merge fallback | v0.15.3 |
| Stale anchor non-blocking | v0.16.0 |
| Doctor merge strategy check | v0.16.0 |

## Quick Reference

```bash
# Check health (includes merge strategy, stale anchor, hooks)
timbers doctor

# Auto-fix hooks and steering
timbers doctor --fix

# Install hooks (beads-aware, section-delimited)
timbers hooks install

# Check hook status with tier info
timbers hooks status

# See what's pending (stale anchor = 0 actionable)
timbers pending
timbers pending --json  # {"count": 0, "status": "stale_anchor", ...}
```
