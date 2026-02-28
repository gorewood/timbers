+++
title = 'Standup'
date = '2026-02-28'
tags = ['example', 'standup']
+++

Generated with `timbers draft standup --since 2026-02-13 --until 2026-02-13 | claude -p --model opus`

---

- Fixed a silent bug in the PostToolUse hook — it read `$TOOL_INPUT` (always empty) instead of stdin, meaning the post-commit reminder was a **no-op since it was created**
- Three CI fixes stacked up: stale `git-notes` fetch referencing storage timbers no longer uses, Hugo `baseURL` pointing at old org, and `GITHUB_TOKEN` pushes not triggering downstream deploys. Chained devblog→pages deploy to close the loop.
- Two pre-existing test failures unblocked: `TestRepoRoot` broke in worktree environments, `TestCommitFiles` used live HEAD which returns empty on merge commits. Both now own their test state.
- Released **v0.8.0** and **v0.9.0** — manual release flow required because `claude -p` can't nest inside Claude Code sessions
- Rewrote coaching system informed by Opus 4.6 prompt guide: motivated rules (WHY behind each rule), concrete 5-point notes trigger checklist, XML tag structure. Council confirmed no model-specific variants needed — clear coaching IS Opus-optimized coaching.
- Added `AgentEnv` interface with registry pattern so future agent environments (beyond Claude) are self-contained via `init()` registration. Refactored `init`, `doctor`, `setup`, `uninstall` to use it.
- Shipped marketing landing page for the Hugo site — competitive positioning matters with Entire.io in the landscape. Follow-up polish pass on terminal blocks.
- `--notes` flag was shipped in v0.8.0 but undocumented everywhere except prime coaching — backfilled across README, tutorial, spec, and agent-reference
- Regenerated example artifacts using 55 real entries instead of 30 backfilled ones — decision-log and sprint-report output quality jumped noticeably with richer `why` and `notes` data
