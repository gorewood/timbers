+++
title = 'Release Notes'
date = '2026-03-22'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---



## New Features

- You can now filter entries by commit range with `timbers query --range`, bringing it in line with `export` and `draft`
- `timbers hooks status` shows you how your hooks are configured and whether timbers is active
- `timbers init` now handles multi-tool environments gracefully — if another tool manages your git hooks, timbers appends its section instead of taking over the hook file

## Improvements

- Hook installation now respects `core.hooksPath`, so timbers works correctly alongside tools like beads that redirect hooks to a custom directory
- Devblog template rewritten with a structured three-voice essay format for richer, more engaging generated posts
- `timbers doctor` gives clearer guidance when hooks are managed by another tool, with specific next steps instead of generic warnings

## Bug Fixes

- `timbers query --range` now finds entries after a squash merge — previously, squash-merged branch entries were invisible because their commit SHAs no longer appeared in main's history
- Chained pre-commit hooks now correctly propagate exit codes — previously, a failing timbers check was silently overridden by the next hook in the chain, allowing commits to slip through
- Fixed a JSON parsing vulnerability by upgrading a dependency that mishandled null Unicode characters
- Updated Go runtime to patch two standard library security issues affecting URL parsing and directory traversal
