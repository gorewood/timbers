+++
title = 'Release Notes'
date = '2026-02-28'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---

## New Features

- `timbers log` now auto-commits its entry file — you no longer need a separate `git commit` after logging your work
- You can now run `draft --models` to see which AI providers are available and what API keys to set
- `timbers draft` output can now be piped to Codex and Gemini CLIs in addition to Claude
- The `--color` flag (`never`/`auto`/`always`) gives you explicit control over terminal colors when auto-detection doesn't match your theme
- `timbers doctor` now checks whether your environment is ready for content generation, showing which CLI tools and API keys are available

## Improvements

- Colors automatically adapt to dark terminal backgrounds — dim text is no longer invisible on themes like Solarized Dark
- Stale anchor warnings after squash merges now explain what happened and confirm that the issue self-heals on your next `timbers log`
- `timbers prime` now includes content safety reminders to help keep secrets and personal data out of entries

## Bug Fixes

- Fixed a crash in `timbers prime` when the anchor commit no longer exists in history (e.g., after a squash merge)
- Fixed example recipes failing on macOS with the default system bash (3.x)
- Fixed a security vulnerability in a dependency (CVE-2026-27896)

## Breaking Changes

- The `exec-summary` draft template has been renamed to `standup` — update any scripts that reference the old name
