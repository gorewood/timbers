# First-Class Report Profiles

**Date:** 2026-07-14  
**Owner bead:** `timbers-zy5`  
**Status:** Ready for implementation

## Goal

Add `timbers report <profile>` as a thin, opinionated layer over the existing
template, entry-selection, rendering, and LLM paths. A profile supplies useful
defaults and compact report input; `timbers draft` remains the lower-level,
fully composable prompt renderer.

The first slice proves one real workflow: `decision-digest`. It does not add a
second config file, write artifacts, publish content, or manage provider
credentials.

## Existing Parts To Reuse

- Template resolution: project `.timbers/templates/` -> global -> built-in.
- YAML frontmatter parsing and template variables in `internal/draft`.
- `--last`, `--since`, `--until`, and `--range` entry selection from `draft`.
- `llm.New` and the current provider/model flags.
- `generationMetadata`, output sanitization, `output.Printer`, and exit codes.

`report` should call shared functions extracted from `draft` only where needed;
it should not duplicate a second rendering or LLM pipeline.

## Profile Format

A report profile is an ordinary Timbers Markdown template with one optional
`report` block:

```yaml
---
name: decision-digest
description: Retrospective digest of explicit design decisions
version: 2
report:
  scope:
    last: 20
  projection: decision
  format: markdown
  quiet_output: _No explicit design decisions in this range._
---
```

Add these fields to `draft.Template`:

```go
Report *ReportProfile `yaml:"report,omitempty"`

type ReportProfile struct {
    Scope       ReportScope `yaml:"scope"`
    Projection  string      `yaml:"projection"`
    Format      string      `yaml:"format"`
    QuietOutput string      `yaml:"quiet_output,omitempty"`
}

type ReportScope struct {
    Last  string `yaml:"last,omitempty"`
    Since string `yaml:"since,omitempty"`
}
```

Rules for the first slice:

- `scope` contains exactly one of `last` or `since`. Static `range` and `until`
  defaults are excluded because refs and end dates are invocation-specific.
- Supported projections are `narrative` and `decision`.
- The only supported format is `markdown`. Reject unknown values rather than
  silently pretending to support them.
- `quiet_output`, when present, is compared exactly after trimming and existing
  LLM-output sanitization.
- A template without `report` remains valid for `draft`, but `report` rejects it
  with: `template <name> is not a report profile; use timbers draft <name> or add report frontmatter`.

Do not put a model, provider, credentials, destination, schedule, or deployment
setting in portable template frontmatter.

## Command Behavior

```text
timbers report decision-digest
timbers report decision-digest --model opus
timbers report decision-digest --since 30d --model opus
timbers report decision-digest --range main..HEAD --model opus
```

`report` accepts the existing selection, `--model`, `--provider`, `--append`,
`--var`, and `--with-frontmatter` flags.

- With no selection flags, use the profile's default scope.
- An explicit `--last`, `--since`, or `--range` replaces the complete default
  scope. `--until` may refine an explicit or default `since` scope.
- Reuse `draft`'s validation for conflicting or malformed selection flags.
- Without `--model`, render the resolved prompt for piping, matching `draft`.
- With `--model`, use the existing LLM client and two-minute timeout.
- Never write an artifact file. stdout remains the composition boundary.

`report` is not a write command, so it does not add `--dry-run`. Invoking it
without `--model` is the cost-free preview of the exact prompt and input. This
avoids two names for the same behavior.

## Compact Input

Do not change `{{entries_json}}` for `draft`. For `report`, populate that same
token with a compact projection so existing prompt bodies continue to work.

`narrative` entries contain:

```json
{
  "id": "...",
  "created_at": "...",
  "what": "...",
  "why": "...",
  "how": "...",
  "notes": "...",
  "tags": [],
  "work_items": [],
  "git_subjects": []
}
```

`decision` entries omit `how` and contain `id`, `created_at`, `what`, `why`,
`notes`, `tags`, `work_items`, and `git_subjects`. Empty optional values are
omitted. Both projections deliberately omit schema bookkeeping, `updated_at`,
range strings, full worksets, and diffstats.

The fixed projections are preferable to a dot-path field language. Add another
named projection only when a real built-in report cannot use these two.

No byte-budget truncation ships in this slice. Compact projection addresses the
known duplication without introducing lossy selection policy. Add a budget only
after a real report exceeds a provider limit.

## Git Enrichment And SHA Rewrites

Stored entry text is authoritative report input. For each distinct stored commit
SHA, make a best-effort lookup of its current subject:

- Add a resolved subject to `git_subjects` only when it adds information not
  already represented by the stored `what` value.
- If a rebase hook updated the SHA, enrichment follows the updated SHA.
- If a squash or destructive rewrite left the SHA unreachable, omit the subject
  and continue with stored `what`, `why`, `how`, and `notes`.
- A lookup failure never drops an entry or fails a report.
- Do not use patch IDs; they do not map many commits to one squash commit.

The first implementation may perform one `git show -s` per distinct SHA in the
selected entries. The default profile selects 20 entries, so the simple path is
adequate; batch it only if measured report latency warrants the extra parser.

Record resolved and unresolved SHA counts in provenance. This makes degraded
lineage visible without turning common squash history into operational noise.

## Quiet Results

There are two successful quiet paths:

1. Selection returns zero entries. Do not render or call an LLM.
2. Generated, sanitized output exactly matches `quiet_output`.

Both exit `0`, emit no artifact content on stdout, and print no human diagnostic
when piped. JSON output is deterministic:

```json
{
  "status": "quiet",
  "profile": "decision-digest",
  "reason": "no_entries",
  "entry_count": 0,
  "provenance": {}
}
```

The second reason is `no_reportable_content`. In a TTY, a short status may be
printed to stderr. A provider failure, malformed profile, corrupt ledger read,
or invalid selection is an error, not a quiet result.

## Output And Provenance

Non-JSON output preserves current `draft` behavior: rendered prompt without a
model, sanitized Markdown with a model, and optional existing TOML frontmatter
when `--with-frontmatter` is requested.

JSON output adds a stable `status` of `rendered`, `generated`, or `quiet` and
includes the existing prompt/response fields as applicable. Extend existing
generation metadata rather than creating a parallel type:

- profile/template name, source, and version
- resolved selection
- projection and format
- entry IDs and count
- model and generation timestamp when generated
- resolved and unresolved Git-subject counts

Provenance contains no destination or publishing-platform fields. Timbermill or
another caller may wrap the stdout artifact with its own publication metadata.

All errors use `output.Printer`: user errors exit `1`, Git/I/O/provider failures
exit `2`, and JSON errors retain the existing `{error, code}` contract and a
recovery hint.

## Draft Compatibility

- `timbers draft` ignores `report` metadata and retains its required explicit
  selection, full `entries_json`, output shape, flags, and template precedence.
- Existing custom templates parse unchanged because `report` is optional.
- `timbers draft decision-digest --last 20` remains valid.
- The first implementation adds `report` metadata only to the built-in
  `decision-digest`; other built-ins move after the command is proven useful.

## Implementation Sequence

1. Parse and validate the optional report frontmatter; add focused parser tests.
2. Add the two compact projection render contexts and best-effort subject lookup.
3. Extract the minimum shared draft execution functions and add `report` to the
   Agent command group.
4. Add `report` metadata to `decision-digest` and update its version.
5. Update command reference and examples after behavior is passing.

## Required Tests

- Frontmatter accepts the example and rejects conflicting scope, unknown
  projection, and non-Markdown format.
- Project/global/built-in resolution remains identical to `draft`.
- Default scope applies; each explicit primary selection replaces it; `until`
  refines a default `since` scope.
- `decision` projection omits `how`, workset, and diffstat; `narrative` keeps
  authored `how` but omits Git bookkeeping.
- Reachable SHAs add non-duplicate subjects; missing or rewritten SHAs preserve
  the entry and increment unresolved provenance.
- Zero entries and the exact quiet sentinel return exit `0` without content or
  an LLM call.
- Human rendered/generated output, JSON statuses, metadata, and structured error
  output follow the command contract.
- A mock LLM verifies model execution without a network call.
- Regression: `draft` with the same template still receives full entries and
  requires an explicit selection.

One integration test in a temporary Git repository should cover default scope,
one reachable and one stale SHA, prompt rendering, and JSON provenance. The
remaining cases are table-driven unit or command tests.

## Deferred Until There Is A Consumer

- A separate `.timbers/config.yaml` report registry.
- Default providers/models and model-specific token budgets.
- Artifact file writing, schedules, publishing, or deployment adapters.
- Arbitrary projection field expressions.
- Automatic compaction or high-water marks for discrete publication workflows.
- Native ADR/document ingestion; that belongs to the publishing layer, not this
  ledger-to-report command.
