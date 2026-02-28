+++
title = 'Example Artifacts'
+++

Timbers generates documents from your development ledger using the `draft` command. Each template produces a different artifact type from the same structured data.

These examples were generated from Timbers' own development ledger, including entries with `--notes` capturing deliberation context.

Browse the examples below, or generate your own:

```bash
timbers draft --list                                  # See all templates
timbers draft standup --since 1d | claude -p          # Daily standup
timbers draft decision-log --last 20 | claude -p      # Architectural decisions
timbers draft changelog --since 7d | claude -p        # Weekly changelog
```
