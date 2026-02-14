+++
title = 'Release Notes'
date = '2026-02-14'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 25 | claude -p --model opus`

---

## Release Notes — v0.7.0 through v0.10.0

### New Features

- **Capture your decision-making process with `--notes`.** You can now add a `--notes` flag to `timbers log` to record the thinking behind a decision — alternatives you considered, surprises you hit, trade-offs you weighed. The `--why` flag stays focused on the verdict; `--notes` tells the story of how you got there.
- **Control terminal colors with `--color`.** If you use a color scheme like Solarized Dark where some text was hard to read, you can now pass `--color never`, `--color auto`, or `--color always` to override color detection.
- **Entries are auto-committed when you log them.** `timbers log` now commits the entry file to git automatically, so you no longer need a separate `git commit` step after logging your work.
- **MCP server for tool integrations.** Timbers now ships with an MCP server (stdio transport) exposing 6 tools, so editor extensions and other MCP-compatible clients can read and write ledger entries directly.
- **Content safety coaching built into `timbers prime`.** The session workflow now reminds agents to keep secrets, personal data, and internal URLs out of entries — since entries are git-committed and potentially public.

### Improvements

- **Smarter coaching for better entries.** The guidance that `timbers prime` provides has been rewritten with clearer explanations, concrete examples, and a five-point checklist for when to include notes. The result: agents write more useful `--why` fields and know when `--notes` adds value.
- **Ready for more agent environments.** `timbers init`, `timbers doctor`, and `timbers uninstall` now use a pluggable architecture internally, making it straightforward to support additional agent environments beyond Claude Code in the future. The `--no-claude` flag has been renamed to `--no-agent` (the old flag still works).
- **Post-commit reminder hook fixed.** The hook that reminds you to run `timbers log` after a git commit was silently broken — it now works correctly. If you have an older installation, `timbers init` will detect and upgrade the stale hook automatically.
- **Landing page and documentation refresh.** Timbers has a proper homepage now, and the tutorial, README, and reference docs have all been updated to cover `--notes` and reflect current best practices.
- **Daily dev blog generation.** The automated dev blog now publishes daily instead of weekly, better matching the pace of development.

### Bug Fixes

- **Fixed invisible text on Solarized Dark and similar themes** — dim/hint text that used color 8 (bright black) is now readable regardless of your terminal's color scheme.
- **Fixed site publishing under the wrong URL** after an organization migration.
- **Fixed a CI issue** where automated blog posts weren't triggering site rebuilds.
