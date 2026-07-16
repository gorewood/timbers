# Neutral Reporting Baseline Migration

**Date:** 2026-07-16
**Owner bead:** `timbers-y54`
**Status:** Approved for implementation

## Goal

Improve Timbers' built-in prompts and demo artifacts using lessons from a
private multi-project reporting implementation without copying its product
language, audiences, domains, examples, renderer grammar, or organizational
assumptions.

This work prepares reliable public report inputs for a later white-label
Timbermill harness. It does not build that harness now.

## Clean-Room Rule

The private implementation is research material, not a source tree to vendor.
Only general reporting requirements may cross the boundary:

- Each artifact names a concrete reader and decision horizon.
- Higher-level summaries should prefer synthesis over activity logs.
- Contributor attribution is descriptive and optional, never a leaderboard.
- External updates exclude internal work unless it changes the reader's world.
- Generated decision reports never claim ADR authority, numbering, or status.
- Sparse input produces a short artifact rather than fabricated completeness.

No private names, project identifiers, organizations, people, audience labels,
example incidents, publishing paths, visual terminology, or custom Markdown
extensions may appear in public Timbers files. A repository scan for known
private identifiers is a required gate before commit.

## Existing Parts To Keep

- `draft` remains the explicit template-and-scope primitive.
- `report` remains a thin profile layer over the same selection and rendering
  path.
- Project and global templates continue to override built-ins.
- Changelog and release-note generation remain release-scoped drafts because a
  timeless default range would be misleading.
- `decision-digest` remains retrospective and non-authoritative.
- Standard Markdown remains the portable output contract. Renderer-specific
  containers, diagrams, frontmatter, and publication metadata belong to the
  publishing layer.

## Migration Slice

### Contributor-aware projections

Include the entry's optional `contributors` snapshot in both compact report
projections. Preserve name, email, and sources exactly as captured. Absence
means unknown; prompts must not reconstruct identity from Git or infer whether
a contributor is human, automated, or an AI agent.

### First-class profiles

Add safe defaults to built-ins that have stable time semantics:

| Profile | Default scope | Projection | Purpose |
|---------|---------------|------------|---------|
| `standup` | `since: 1d` | narrative | Immediate team signal |
| `sprint-report` | `since: 14d` | narrative | Cycle outcomes and friction |
| `devblog` | `last: 20` | narrative | Narrative technical retrospective |
| `project-update` | `since: 7d` | narrative | Recurring user-facing update |

Do not add default profiles to `changelog`, `release-notes`, or
`pr-description`; their correct scope depends on a release or branch boundary.

### Prompt improvements

- `standup`: permit a short collaboration note only when captured attribution
  or notes support it.
- `sprint-report`: optionally summarize who worked across which themes, using
  prose only and never counts, rankings, or productivity judgments.
- `devblog`: use captured contributor names when collaboration changes the
  story; never infer roles or feelings.
- `project-update`: report only changes users can experience, use plain
  language, surface known limitations stated in entries, and omit empty
  sections.
- `decision-digest`: retain the current source-cited, non-ADR contract.

### Timbers site

- Render optional artifact authors in report headers and list rows.
- Publish a clean `project-update` example alongside the existing artifacts.
- Remove model deliberation and meta-commentary from current examples.
- Update example commands to use `timbers report` for profile-backed artifacts.
- Keep the Timbers brand site distinct from the future generic harness.

## Deferred To Timbermill

- Multiple repository acquisition and refresh.
- Artifact rollups that consume previously published lower tiers.
- Audience-channel navigation, filtering, unread state, and people pages.
- Native ADR and design-document ingestion.
- Renderer-specific callouts, diagrams, syntax extensions, feeds, and search.
- A theme/plugin contract or independently versioned companion repository.

These require at least a second real source and belong to the next design pass.

## Required Checks

- Projection tests prove contributor inclusion and continued omission of Git
  bookkeeping.
- Built-in loading tests prove each promoted profile's scope and projection.
- Existing custom templates still parse and render unchanged.
- Site tests and output checks pass with artifacts both with and without
  authors.
- Authored prompts contain no private identifiers or renderer-specific syntax.
- `just check` passes.
- Desktop and mobile captures cover the homepage, artifact list, and one
  attributed report.
