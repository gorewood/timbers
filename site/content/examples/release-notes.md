+++
title = 'Release Notes'
date = '2026-06-11'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---

**New Features**
- The commit gate is now provenance-aware: it automatically skips commits authored by someone else or left over from an earlier session, so you're only asked to document work from your current session.
- You can now set how long a session "counts" with a `session-window:` directive in `.timbersignore` (defaults to 24h).
- `timbers pending --explain` now lists every commit and why it is or isn't pending.
- A new `help timbersignore` topic documents the rule types and the canonical bot-exemption recipe.

**Improvements**
- When output is piped or redirected, timbers errors now print as a single readable line instead of a styled box that could be cropped down to a blank line.
- Gate refusal messages now name the tool, cause, fix, and bypass on one line, and tell you when staged work remains in the index (`git diff --cached`).
- `timbers show` and `timbers log --dry-run` now render entries in an aligned, word-wrapped panel that's easy to scan.
- timbers now honors your `.mailmap`, so multi-email setups aren't misread as someone else's work.
- A post-commit note now tells you when your own work was auto-skipped as stale.
- `timbers doctor` now warns when `user.email` is unset, flags malformed `session-window:` values, and catches `[..]`-style globs in `author:`/`msg:` rules.
- `timbers pending` now explains a count of 0 when the anchor is off the first-parent line.
- `timbers log --anchor` now documents a single commit even when nothing is pending, and points you at `--range` when it can't.
- `.timbersignore` exemption rules are now surfaced in `timbers pending` hints and onboarding output.

**Bug Fixes**
- `timbers log --dry-run` no longer drops the Notes field.
- Diffstats now render consistently between `timbers show` and `timbers log --dry-run`.
- `timbers status` now correctly counts commits skipped by a `msg:` rule.

**Breaking Changes**
- `timbers log` now refuses to run on a dirty working tree instead of warning and continuing. Commit your changes first, or use `timbers log --dry-run` to inspect an entry without writing.
- In `timbers prime`, `pending.count` now counts only the in-session commits you should document — foreign-author and stale commits are excluded. If you script against this field, it no longer reflects the raw total.
