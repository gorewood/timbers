# Agent Instructions

This project uses **bd** (beads) for issue tracking. Run `bd onboard` to get started.

## Session Orientation

Before starting any work, verify your context:

1. **Branch:** `git branch --show-current` — confirm you're on the expected branch
2. **Worktree:** `git worktree list` — are you in a worktree or the main repo?
3. **Confirm with user:** "I'm on branch X in [worktree/main]. Is this where you want me working?"
4. **Check beads:** `bd ready` — what work is available?

**NEVER skip orientation.** Working on the wrong branch wastes entire sessions silently.

---

## Settled Decisions

Do NOT revisit items marked SETTLED without explicit user request.

<!-- Add decisions as they're made:
| Decision | Date | Rationale | Status |
|----------|------|-----------|--------|
| Example: Auth uses JWT | 2025-01-15 | See docs/plans/auth.md | SETTLED |
-->

---

## Quality Gates

Before every commit, run the appropriate quality gate:

```bash
just check          # If justfile exists
npm run check       # If package.json with check script
```

**Do not commit if checks fail.**

---

## Worktree Guardrails

**All worktrees go under `.worktrees/` in the repo root.** Before creating any worktree, ensure `.worktrees/` is in `.gitignore`:

```bash
git check-ignore -q .worktrees/ || echo '.worktrees/' >> .gitignore
```

If you added the line, commit it before proceeding. See `dm-work:worktrees` for full workflow details.

When using git worktrees for feature development:

1. Create worktree: `bd worktree create .worktrees/<name>`
2. Run quality gates
3. **STOP AND GET USER SIGN-OFF** before merging
4. Only after explicit approval: merge to main, sync beads, push, cleanup worktree

---

## Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --status in_progress  # Claim work
bd close <id>         # Complete work
bd sync               # Sync with git
```

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd sync
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds

