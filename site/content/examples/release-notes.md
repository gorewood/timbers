---
title: 'Release Notes'
date: '2026-07-18'
tags: ['example', 'release-notes']
authors: ["Bob Bergman"]
---

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---

I'll pull the release notes from the entries, filtering to only user-observable changes.

New Features
- You can now run `timbers report` to turn captured decisions into a finished report from a named profile — no manual selection or prompt-wrangling required.
- Timbers now records **who** contributed to each entry automatically, drawn from Git authors and `Co-authored-by` trailers, so credit survives rebases, squashes, and pruning.
- Use `--who "Name <email>"` to set contributors explicitly for pairing, shared work, bots, or corrections — repeatable, and it replaces the automatically captured set.
- New recurring report baselines ship as ready-to-run profiles, including a user-focused project update, so common reports work without setup.
- Generated reports now carry contributor attribution, and the published site shows optional bylines.

Improvements
- The project site has a new **Working Mill** theme with a redesigned hero, refined typography, dark mode, and reduced-motion-safe visuals.
- Reports read as finished, direct artifacts — generated output no longer exposes model selection or drafting notes.

Bug Fixes
- Standup reports now include contributors as promised; earlier versions omitted them from the report input.
- Corrupt or malformed ledger entries no longer vanish silently — they surface in `timbers doctor` and query output, and `timbers draft`/`timbers report` now fail cleanly instead of emitting a partial artifact.
- The site hero install command no longer forces an oversized horizontal scrollbar.

Breaking Changes
- The built-in ADR generator has been replaced by a non-authoritative **decision digest** that cites its source entries. Generated digests no longer carry ADR numbers or lifecycle status — keep your hand-authored ADRs as the authoritative record; they remain unaffected. _Automated migration not detailed in the entries._
