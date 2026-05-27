+++
title = 'Release Notes'
date = '2026-05-27'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---

## New Features

- You can now run `timbers ack` to record an honest skip-with-reason for a commit you're intentionally not documenting, instead of fabricating an entry or reaching for `--no-verify`.
- `.timbersignore` now accepts `author:<glob>` lines, so you can skip commits by author — handy for excluding bot authors in automated pipelines.
- Set `TIMBERS_DEBUG=1` to see a trace explaining why each commit was counted or skipped during pending detection.
- Set `TIMBERS_SKIP_CROSS_AGENT_DEBT` to bypass the commit gate when another agent's undocumented commits would otherwise block you.

## Improvements

- The commit gate now only considers commits on your first-parent line and ignores clean merge commits, so parallel agents sharing a `.timbers/` directory no longer block each other on each other's work.
- Merge commits with no file changes are dropped from the `timbers pending` display, so you no longer see bare merge SHAs with no obvious next step.
- `timbers log` now warns when you've already pushed the documented commit but not the entry, catching the race that would otherwise strand an entry on your machine.
- `timbers pending` and `timbers doctor` now detect when your latest entry is anchored to a side branch and point you at the right next step instead of showing confusing results.
- Rebase-relink hints in `timbers pending` and `timbers doctor` are now copy-pasteable.
- `timbers prime` compact output now shows full, resolvable entry IDs you can paste straight into `timbers show`, and flags when a custom `PRIME.md` workflow is in effect.
- Provider model shortcuts used by `timbers draft` and `timbers generate` now resolve to current official model IDs.

## Bug Fixes

- `timbers log --batch` no longer anchors an entry to a side-branch commit when a work item spans branches; it picks the commit on your main line.
- `timbers pending` no longer lists commits that are already documented when your latest anchor sits on a side branch — merge-topology cases now clear to zero.
- The post-commit hook no longer nudges you to document commits that `timbers log` would refuse, such as `.beads`-only commits.
- `timbers prime --json` now reports the output mode you actually requested instead of always reporting `full`.
