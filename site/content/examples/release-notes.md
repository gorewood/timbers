+++
title = 'Release Notes'
date = '2026-03-22'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---



## New Features

- You can now filter `timbers query` results by commit range using the `--range` flag, matching the filtering already available in `export` and `draft`
- `timbers hooks status` shows you which hooks are installed and how timbers coexists with other tools in your repo
- `timbers init` now detects your environment and chooses smart defaults — if another tool already manages git hooks, timbers adapts instead of overwriting

## Improvements

- Hook installation now respects `core.hooksPath`, so timbers installs hooks where git actually reads them — even when other tools redirect the hooks directory
- The stop hook now provides the full `timbers log` syntax with `--why` and `--how` flags, so agents can act on it immediately without guessing the command format
- Devblog drafts now use a richer three-voice essay structure, producing more engaging and structured posts from your development entries
- Multi-tool environments are handled gracefully — timbers classifies your setup and cooperates with existing hooks rather than taking ownership of them

## Bug Fixes

- Fixed chained pre-commit hooks silently ignoring timbers' exit code — previously, commits would succeed even when timbers flagged undocumented work
- Fixed pending commit detection producing false positives in certain repository states
