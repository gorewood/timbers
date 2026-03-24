+++
title = 'Release Notes'
date = '2026-03-23'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---



## New Features

- You can now use `--range` with `timbers query` to filter entries by commit range, matching the behavior already available in `export` and `draft`

## Improvements

- `--range` is smarter about finding entries after squash merges — it no longer loses track of work that was merged via squash or rebase
- `timbers pending` gracefully handles stale anchors instead of dumping hundreds of false-positive commits — you'll see a clear message explaining what happened and what to do
- Hooks no longer block when the anchor is stale, so your workflow keeps moving after squash merges
- `timbers doctor` now checks your merge strategy and warns about configurations that can cause stale anchors
- The devblog template produces richer, more structured essays with clearer takeaways

## Bug Fixes

- Fixed `--range` silently dropping entries when some anchors were stale but others were valid
- Fixed `query --range` returning empty results after a squash merge

## Security

- Upgraded Go to 1.25.8 to address vulnerabilities in IPv6 URL parsing and directory traversal
- Updated go-sdk to v1.4.1 to fix a JSON parsing vulnerability that could allow message field override via duplicate keys
