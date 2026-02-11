+++
title = 'Release Notes'
date = '2026-02-10'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 30 --model opus`

---

# Release Notes

## New Features

- You can now use `timbers draft` to generate documents like changelogs, release notes, and decision logs directly from your ledger entries — with built-in LLM support via `--model` and `--provider` flags so you don't need to pipe to external tools.
- You can now generate ADR-style decision logs with the new `decision-log` template, which extracts the "why" behind your work into a structured architectural decision record. Use `timbers draft decision-log` to try it.
- You can now modify existing ledger entries with `timbers amend` — fix typos, add missing context, or update summary fields using `--what`, `--why`, `--how`, and `--tag` flags. Use `--dry-run` to preview changes before saving.
- You can now filter entries by tag in both `timbers query` and `timbers export` using the `--tag` flag. Multiple tags use OR logic, so `--tag feature --tag bugfix` returns entries matching either tag.
- You can now control catchup batch size and grouping with `--limit` and `--group-by` flags on the `catchup` command.
- You can now store API keys in a `.env.local` file in your config directory instead of setting environment variables — helpful if your environment conflicts with other tools.
- `timbers doctor` now checks your configuration directory, environment files, API keys, templates, and whether you're running the latest version.
- You can now cleanly remove timbers from a repository with the `uninstall` command.

## Improvements

- `timbers init` now works seamlessly — you no longer need an extra step before `timbers prime` is ready to use.
- Configuration now works correctly on Windows, macOS, and Linux, respecting `TIMBERS_CONFIG_HOME`, `XDG_CONFIG_HOME`, and platform-appropriate defaults.
- Git hooks are now opt-in (use `--hooks` to enable them), so timbers no longer conflicts with other tools that use pre-commit hooks.
- Claude integration now defaults to project-level setup instead of global, which is a better fit since timbers is configured per-repository.
- When piping output to other commands, errors and warnings now go to stderr so they don't corrupt your piped data.
- CLI output is more readable with improved styling, tables, and formatting that adapts to your terminal.
- Timbers now gracefully handles repositories that have git notes from other tools, skipping entries it doesn't recognize instead of erroring out.
- Comprehensive tutorial and documentation now available covering setup, daily workflow, agent integration, querying, and troubleshooting.

## Breaking Changes

- The `timbers prompt` command has been renamed to `timbers draft` — update any scripts or workflows that reference the old name.
- Git hooks are now opt-in rather than installed by default. If you rely on automatic hooks, re-run setup with the `--hooks` flag.
- Claude hook setup now targets the current project by default. Use `--global` if you need the previous global behavior.
