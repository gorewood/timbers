+++
title = 'Release Notes'
date = '2026-02-28'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---

## New Features

- You can now control terminal colors with `--color never|auto|always` — helpful if your theme makes text hard to read
- `timbers log` now auto-commits the entry file for you, so there's no extra `git add` and `git commit` step after recording work
- `timbers draft --models` shows which LLM providers are available and whether your API keys are configured
- `timbers doctor` now checks whether you have content generation set up (CLI tools or API keys) and tells you exactly what to configure
- `timbers prime` now includes content safety reminders — entries are git-committed and potentially public, so the workflow coaching helps you avoid accidentally including secrets or personal data
- You can now pipe `timbers draft` output to Codex and Gemini CLIs in addition to Claude — documented with verified syntax for each

## Improvements

- Terminal colors now adapt automatically to dark and light backgrounds, so dim text is readable on both
- The `exec-summary` template has been renamed to `standup` for easier discovery — use `timbers draft standup`
- The `pr-description` template now focuses on intent and decisions rather than repeating the diff
- Example recipes work on macOS default bash (3.2) without needing bash 4+
- Piping workflow guidance in `timbers prime` updated with clearer instructions

## Bug Fixes

- Fixed text being invisible on dark terminal themes like Solarized Dark
- Fixed `timbers prime` crashing on stale anchors after squash merges — you now get a clear warning with guidance instead
- Fixed a security vulnerability in a dependency (CVE-2026-27896)
- Cleaned up generated changelog content that occasionally included LLM processing artifacts

## Breaking Changes

- `timbers draft exec-summary` is now `timbers draft standup` — the old name no longer works
