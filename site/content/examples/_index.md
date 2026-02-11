+++
title = 'Example Artifacts'
+++

Timbers generates documents from your development ledger using the `draft` command. Each template produces a different artifact type from the same structured data.

These examples were generated from Timbers' own development ledger using `timbers draft <template> --last 30 --model opus`.

> **A note on quality:** These entries were largely backfilled using `timbers catchup`, which infers what/why/how from commit messages and diffs. Projects that use `timbers log` from day one will produce significantly richer output — especially in the decision-log, where the *why* field matters most.

Browse the examples below, or generate your own:

```bash
timbers draft --list              # See all templates
timbers draft changelog --last 20 --model opus
```
