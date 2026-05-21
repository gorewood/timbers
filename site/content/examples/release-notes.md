+++
title = 'Release Notes'
date = '2026-05-21'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---

## New Features

- You can now create a `.timbersignore` file with `author:<glob>` lines to skip commits from specific authors (useful for bots and automated commits)
- New `timbers ack` command lets you mark commits as intentionally skipped with a reason, replacing the need for `--no-verify` workarounds
- Set `TIMBERS_DEBUG=1` to get a detailed trace of why each commit was classified as pending or skipped
- Set `TIMBERS_SKIP_CROSS_AGENT_DEBT=1` as an escape hatch when working alongside parallel agents whose merge commits would otherwise block your workflow
- `timbers prime` now defaults to a compact session-start output; use `--full` or `timbers guide` for the complete coaching guide
- `timbers doctor` now reports when your latest entry's anchor is on a side branch, pointing you at the right escape hatch
- `timbers pending` surfaces a hint when the latest anchor falls off the first-parent line, so you know why detection looks unusual

## Improvements

- `timbers log` now warns when you push a commit before documenting it, catching a race that previously stranded entries locally
- Pending detection no longer blocks you on another agent's undocumented commits when working in parallel — the gate now scopes to your own first-parent line
- Empty merge commits and clean `--allow-empty` commits no longer appear as actionable pending work
- The post-commit hook no longer nudges you to document commits that only touch ignored paths (like `.beads/issues.jsonl`-only commits)
- `--batch` mode now picks a sensible anchor commit on the first-parent line instead of sometimes landing on a side-branch SHA
- Session-start prime output is now significantly smaller, freeing context for your actual work
- Compact prime output preserves full resolvable entry IDs so you can paste them straight into `timbers show`
- Compact prime tells you when a custom `PRIME.md` workflow is active in your repo
- Provider model aliases for Anthropic, OpenAI, and Gemini refreshed to current model IDs

## Bug Fixes

- Fixed post-commit hook nudging on `.beads/`-only commits in repos like vellum where the bug was reported
- Fixed `--json` mode on `timbers prime` reporting the wrong mode in its output
- Fixed `--batch` entry anchors occasionally pointing at side-branch commits, breaking downstream linear-anchor assumptions
- Documented commits that previously fell through the pending filter are now correctly recognized as already-documented
