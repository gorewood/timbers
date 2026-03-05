+++
title = 'Release Notes'
date = '2026-03-04'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---



## New Features

- **Post-commit reminders keep you on track.** After each commit, timbers now shows a brief reminder to log your work — no more forgetting at session end.
- **Health checks surface problems early.** `timbers prime` now runs a quick health check at session start, flagging missing hooks or integrations before you begin working.
- **Pre-commit hook blocks undocumented commits.** Timbers can now prevent commits when you have pending work to document, with a session-end backstop as a safety net.

## Improvements

- **Smoother onboarding for existing repos.** Setting up timbers in a repo with prior history no longer floods you with hundreds of "pending" commits — only work after setup is tracked.
- **`timbers doctor` is faster in large repos.** Pending commit detection is now significantly more responsive, especially in repositories with thousands of commits.
- **Hook installation works with custom hook paths.** If your Git config uses `core.hooksPath`, timbers now installs hooks to the right location instead of the default `.git/hooks/`.
- **Better guidance when hooks remind you to log.** Stop hook messages now include the full `timbers log` syntax with flags, so you can act on them immediately.
- **Squash merges no longer break `timbers log`.** If a branch was squash-merged, `timbers log` now warns and continues instead of failing with a stale anchor error.

## Bug Fixes

- **Chained pre-commit hooks now properly block commits.** Previously, if timbers was chained with another hook, a failing timbers check would be silently overridden — commits always succeeded.
- **Fixed false "pending commits" warnings after logging.** Timbers' own auto-committed log entries no longer trigger spurious pending-commit alerts.
- **Uninstall now removes all hooks cleanly.** Previously, hooks from older versions could be left behind after uninstalling.
- **Fixed stale LLM commentary in the published changelog.** Cleaned up generated artifacts on the site.
- **Fixed compatibility with macOS default shell.** Example recipes now work correctly with bash 3.x.
- **Security dependency updated.** Addressed a high-severity vulnerability in an upstream library.
