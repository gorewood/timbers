---
title: 'Project Update'
date: '2026-07-16'
summary: 'Contributor attribution, repeatable report profiles, and a clearer publishing path landed together.'
tags: ['example', 'project-update']
authors: ['Bob Bergman']
---

Generated with `timbers report project-update --since 7d --model opus`.

---

Timbers can now preserve who contributed alongside what changed and why. This
update also makes recurring reports easier to run without hiding the prompt or
locking publication to one platform.

## New

- Timbers records a capture-time contributor snapshot from mailmap-normalized
  Git authors and `Co-authored-by` trailers. Use repeatable `--who` values when
  you need to replace the automatic set explicitly.
- `timbers report` runs frontmatter-defined profiles with useful default scopes.
  Built-in profiles now cover standups, project updates, sprint reports,
  decision digests, and development narratives.
- The in-repository Eleventy site publishes generated reports and native
  Markdown artifacts through the same host-neutral collection contract.

## Improved

- `timbers log` can derive `what` from selected commit subjects, avoiding
  duplicate capture work while preserving durable text across rewritten Git
  history.
- Ledger reads now surface corrupt entry paths and prevent `draft` or `report`
  from silently producing partial artifacts.

## Action required

- The inferred `catchup` command has been removed. Use the first-log baseline,
  batch logging, acknowledgements, and ignore rules when adopting Timbers in an
  existing repository.
- Generated numbered ADRs have been replaced by non-authoritative decision
  digests. Keep project ADRs and design documents as the authoritative record.
