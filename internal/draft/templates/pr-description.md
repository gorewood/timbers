---
name: pr-description
description: PR body focused on intent, decisions, and risk areas
version: 5
---
Generate a pull request description from these development log entries.

In an age of agent-assisted code review, reviewers can read diffs themselves.
A good PR description answers the questions that diffs don't: why does this
change exist, what trade-offs were made, where should reviewers focus, and
how was it built.

**Adapt to PR size**:
- Tiny / single-purpose PR (one entry, one obvious fix): output just `## Why` and `## Test Plan` — no need for the full skeleton.
- Standard PR: use all relevant sections from the format below.
- Large or cross-cutting PR: include `## Migration / Rollback` if entries hint at config, schema, feature flag, or staged-rollout work.

**Format**:
```
## Why

[1-3 sentences: the problem or need this PR addresses, and the intent behind
the approach. Not a restatement of the title — explain *why now* and *why
this way*. If the work was reactive (a bug a user hit, a metric that drifted,
a review comment), say so.]

## Design Decisions

[Bullet list of key trade-offs, alternatives considered, and choices made.
Pull from --why and --notes fields. Only include decisions that a reviewer
would find non-obvious or worth knowing. Omit if entries lack design context.]

## How This Was Built

[Optional. Include ONLY when entries explicitly describe meaningful human-agent
collaboration: the agent surfaced an edge case the operator missed, the operator
overrode the agent's first plan, etc. One short paragraph. This calibrates how
the reviewer should engage — agent-implemented code with operator review needs
different attention than operator-implemented code. Skip when entries are silent
on this; never fabricate.]

## Risk & Reviewer Attention

[Where should reviewers look closely? What's subtle, what could break,
what has edge cases? Be specific: name files, functions, or behaviors.
Omit if the change is straightforward.]

## Migration / Rollback

[Optional. Include only when this PR touches: schema/migrations, feature flags,
config defaults, or staged rollouts. Answer: how does a consumer adopt this,
and how do we back out if it goes wrong? One short paragraph.]

## Scope

[1-2 sentences conveying the shape of the change: what areas of the codebase
are touched, whether it's a focused fix or a cross-cutting change. Texture,
not stats — "Touches the CLI layer and storage, leaves templates alone" not
"12 files changed".]

## Test Plan

[How to verify this works. Distinguish honestly between what was *verified*
(tests run, manual checks performed — pull from entries) and what *should be
verified* by reviewers or CI. If entries don't mention testing, write "Reviewers
should verify: [list of behaviors]" — don't invent that tests were run.]
```

**Style**:
- Concise. Reviewers skim.
- Lead with intent, not implementation.
- Use `backticks` for file names, function names, flags, commands.
- Call out breaking changes or behavioral shifts explicitly with **BREAKING:** prefix.
- Active voice.

**Constraints**:
- Only describe changes present in the entries.
- Don't invent test steps or verification claims not implied by the work.
- If entries lack detail for a section, omit that section entirely rather
  than filling it with generic content.
- The "Design Decisions" section is the highest-value section — prioritize it.
- Don't pad small PRs with empty sections.

**Output discipline**:
- Output the PR description ONLY. No preamble, commentary, acknowledgment, or meta-discussion.
- Do not begin with "Here is..." or "I'll generate..." or any thinking-out-loud.
- Do not end with "Let me know..." or any sign-off.
- The first line of your response must be part of the document itself.

## Entries ({{entry_count}}) | Branch: {{branch}}

{{entries_json}}
