+++
title = 'Release Notes'
date = '2026-06-01'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---

**New Features**

- You can now set a per-repo staleness window in `.timbersignore` with a `session-window:` directive, so long-running sessions aren't treated as stale.
- `timbers pending --explain` classifies every commit and shows you why each one is or isn't pending.
- A new `help timbersignore` topic (and onboarding blurb) documents the three `.timbersignore` rule types and the canonical bot-exemption recipe.

**Improvements**

- The commit gate now skips commits authored by someone else and commits older than the session window (default 24h), so you're only nudged to document your own in-session work.
- timbers now honors your `.mailmap`, so a multi-email setup is no longer mistaken for another author's work.
- When your own work is auto-skipped because the session ran past the window, a post-commit note tells you what happened instead of skipping silently.
- The `timbers prime` pending count now reflects only the in-session commits you actually need to document.
- When the pre-commit gate aborts a commit, it now tells you your staged changes are still in the index and how to inspect them (`git diff --cached`).
- `timbers pending` now explains when the anchor is off the first-parent line, instead of just reporting "No pending commits".
- `timbers log --anchor` now documents a single commit even when nothing is detected as pending, and the refusal message points you to `--range`.
- `timbers doctor` now warns when `git user.email` is unset, flags literal-looking character classes in `author:`/`msg:` globs (e.g. `author:dependabot[bot]`), and warns on malformed `session-window:` values.

**Bug Fixes**

- `timbers status` no longer undercounts auto-skipped commits when a `msg:` rule matches.

**Breaking Changes**

- `timbers log` now refuses to run on a dirty working tree instead of warning and proceeding (this prevents phantom ledger entries after an aborted commit). Commit your changes first, or use `--dry-run` to inspect an entry without writing one.
