+++
title = 'Release Notes'
date = '2026-03-23'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---



## New Features

- You can now filter `timbers query` results by commit range using the `--range` flag, matching the flexibility already available in `export` and `draft`
- `timbers hooks status` shows you exactly how your git hooks are configured and whether timbers is active
- `timbers doctor` now detects stale anchors and merge strategy mismatches, with actionable guidance

## Improvements

- Hook installation plays nicely with other tools — timbers no longer takes over your hook files, instead appending its section alongside existing hooks
- `timbers init` adapts to your environment, detecting whether hooks are uncontested, shared with other tools, or managed by an external `core.hooksPath` configuration
- Hooks are now installed to wherever `core.hooksPath` points, so setups using custom hook directories work correctly out of the box
- The devblog template produces richer, more structured essays with clearer takeaways

## Bug Fixes

- Squash-merged branches no longer cause `timbers pending` to list every commit in history — it now gracefully reports zero pending commits with guidance
- Squash merges no longer break `timbers query --range` — entries are found even when original branch commits are absent from history
- Stale anchors after squash merges no longer block hooks or produce confusing agent errors
- Fixed a security vulnerability in JSON parsing by upgrading a dependency that could allow crafted Unicode characters to override message fields
- Updated Go runtime to address standard library security fixes for URL parsing and directory traversal
