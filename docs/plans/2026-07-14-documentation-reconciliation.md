# Documentation Reconciliation

**Date:** 2026-07-14  
**Owner bead:** `timbers-8x0`  
**Status:** Ready for implementation after the report and Eleventy slices land

## Goal

Make every active documentation surface describe the command and storage model
that actually ships, while preserving dated design records and generated posts
as history. Fix the known wrong examples, document `report`, and add only the
smallest automated checks needed to stop retired capabilities from returning as
present-tense guidance.

This is a reconciliation pass, not a documentation redesign. Prefer deleting
duplicated command detail and linking to canonical help over maintaining the
same flag inventory in several files.

## Documentation Policy

Classify documents before editing them:

1. **Active guidance** describes supported behavior now and must match the CLI.
   This includes `README.md`, `AGENTS.md`, `docs/tutorial.md`,
   `docs/agent-reference.md`, `docs/llm-commands.md`,
   `docs/publishing-artifacts.md`, `docs/agent-dx-guide.md`, and
   `docs/timbers-integration-guide.md`.
2. **Current specification** describes stable data and behavioral contracts.
   `docs/spec.md` remains useful only after implemented and proposed behavior
   are separated. Until then, command help and code win conflicts.
3. **Dated decisions and research** record what was believed at a point in
   time. `docs/plans/`, `docs/design-decisions.md`,
   `docs/agent-trace-integration.md`, and `docs/roadmap.md` must carry a visible
   historical/research banner and link to the current source where one exists.
   Do not rewrite their original reasoning into fake hindsight.
4. **Historical artifacts** under `site/content/posts/` may accurately mention
   Git notes, `catchup`, decision logs, ADR generation, or Hugo when those were
   true. Preserve the prose. The artifact layout must label them as dated
   history; current navigation and adjacent instructional copy must not present
   those features as available now.
5. **Generated output** such as `site/public/` and the future `site/_site/` is
   never edited by hand. Regenerate or remove it through the active site build.

## Canonical Sources

| Subject | Canonical source | Documentation rule |
|---|---|---|
| Command names, args, and flags | Cobra definitions in `cmd/timbers/` and `timbers <command> --help` | Active docs summarize workflows and link to help; do not hand-copy every flag repeatedly. |
| JSON and exit behavior | `internal/output`, command output types, and command tests | Examples must be copied from passing tests or live `--json` output. |
| Entry schema and validation | `internal/ledger/entry.go` and tests | `docs/spec.md` may explain the contract but cannot add unimplemented fields or flags. |
| Storage and SHA behavior | `internal/ledger`, hook implementations, and integration tests | Describe file-per-entry storage, best-effort relinking, stale-anchor degradation, and stored text as the durable fallback. |
| Template syntax and resolution | `internal/draft/render.go`, `template.go`, and `timbers draft --list` | Document literal Timbers tokens, not Go templates. |
| Report profiles | `cmd/timbers/report*.go`, `internal/draft/profile.go`, tests, and the report-profile plan | `report` is the opinionated profile workflow; `draft` remains the explicit low-level renderer. |
| Built-in report semantics | Frontmatter and prompt in each `internal/draft/templates/*.md` | Native ADRs are authoritative; decision digests are retrospective and non-authoritative. |
| Development workflows | `justfile` recipes | Docs name `just` recipes rather than duplicating shell pipelines. |
| Public site and publishing contract | `site/timbermill.json`, `site/package.json`, Eleventy config, and site tests after migration | Hosting providers are downstream; the generated static directory is the publishing boundary. |

The two 2026-07-14 plans are implementation records, not permanent user
documentation. Once their behavior ships, active docs describe the result and
the plans remain dated rationale.

## Concrete Drift Inventory

### Immediate factual corrections

| Surface | Current claim | Required correction |
|---|---|---|
| `docs/llm-commands.md:110-137` | Custom templates use Go-template expressions such as `{{.RepoName}}`, `{{range .Entries}}`, and field access. | Replace with supported literal tokens from `internal/draft/render.go`: `{{repo_name}}`, `{{branch}}`, `{{entry_count}}`, `{{entries_json}}`, `{{entries_summary}}`, `{{date_range}}`, `{{project_description}}`, `{{is_first_batch}}`, `{{total_entries}}`, and declared `{{vars.key}}`. State that iteration and conditionals belong in the LLM prompt, not a template engine; `--append` adds an Additional Instructions section after rendering. |
| `docs/design-decisions.md:7-15` | A package-global `jsonFlag` is an accepted current pattern. | Mark the document as a dated review record and add a current note: commands now call `isJSONMode(cmd)`; shared mutable JSON state was removed. |
| `AGENTS.md:270-281` | The output example reads global `jsonFlag`. | Change the project reference example to `isJSONMode(cmd)` or describe behavior without copying implementation. |
| `docs/agent-trace-integration.md:40` | Timbers storage is Git notes. | Preserve the research document but add a historical correction: current storage is JSON files under `.timbers/YYYY/MM/DD/` committed through ordinary Git. |
| `docs/roadmap.md:65-91` | The roadmap presents `refs/notes/timbers` as a future/current implementation epic. | Label the roadmap historical and completed/superseded. Link storage readers to the current spec and the dated flat-file migration post rather than updating every old checkbox. |
| `README.md:167` | Separate entry commits “survive rebases and squash merges cleanly.” | Replace with the bounded guarantee: entry text travels through ordinary Git; local one-to-one rewrites are relinked when possible; squash merges can stale SHAs; queries use file-based range discovery where possible; reports retain stored text and omit unavailable Git enrichment. |
| `docs/design-decisions.md:89` and `cmd/timbers/log.go` help | Separate commits/self-healing documentation survive rebases or squash merges without qualification. | Use the same bounded SHA language. Command help should not promise more than hooks and stale-anchor handling deliver. |
| `docs/spec.md:218` | Positional `<what>` is required except for auto/batch. | Document current manual behavior: explicit `<what>` wins; when absent, Timbers snapshots non-empty selected commit subjects; failure to derive requires explicit input. `why` and `how` remain required outside minor/auto/batch modes. |
| `README.md`, `AGENTS.md`, `docs/tutorial.md`, `docs/agent-reference.md` | Examples imply positional `what` is always required. | Keep explicit `what` examples because they are clearest, but add one concise optional-derivation example and explain capture-time snapshot semantics. Do not encourage SHA lookup as the primary report input. |

### Retired and changed capabilities

| Surface | Finding | Required correction |
|---|---|---|
| Active docs and help | `catchup` has been removed. README contains one clearly historical reference; dated plans/posts contain historical references. | Keep the README sentence only because it explicitly says “historically” and “now-retired.” Remove any active command examples, migration instructions, or future intent for `catchup`. Do not scrub dated posts or plans; label their container as historical. |
| Historical plans | `docs/plans/2026-06-01-cross-agent-debt-classification.md` discusses a possible future catchup flow. | Leave the dated plan unchanged after the plans index/banner makes its status clear. It is evidence of a rejected direction, not current intent. |
| Active template docs | `decision-log` has become `decision-digest`; the digest must not mint ADR numbers, status, or supersession. | Ensure README, tutorial, agent reference, LLM guide, publishing guide, example index, and built-in list consistently use `decision-digest` and call it retrospective/non-authoritative. |
| Historical site posts | Several posts describe generated numbered ADRs and the former `decision-log` workflow. | Preserve as dated release history. Remove them from current feature examples or add historical artifact context in the Eleventy layout. |
| README and command summaries | `report` is absent even though it is now an Agent command. | Add `report <profile>` to the core table and generation overview. Show default-scope `timbers report decision-digest`, explicit scope override, prompt preview without a model, quiet success, and `draft` compatibility. |
| `docs/llm-commands.md` | Claims there are three LLM integration commands and only explains `draft`. | Add `report` as the opinionated profile command and retain `draft` as the composable renderer. Avoid duplicating every shared flag; document differences. |
| Agent guidance | Key-command tables and onboarding snippets omit `report`. | Mention it as a consumption workflow, not as required session-close ceremony. Capture friction must not increase. |

### Publishing and site transition

| Surface | Finding | Required correction |
|---|---|---|
| `.mise.toml`, `justfile`, `.github/workflows/pages.yml` | Hugo is still the active build and Pages deployment. | Let the Timbermill/Eleventy bead replace these first. Documentation reconciliation must describe only the verified final recipes and output directory. |
| `docs/publishing-artifacts.md:155` | The generic Jekyll/Hugo example uses unsupported `{{ .Week }}` and `{{ .Date }}` tokens. | Replace with Timbermill/static-site guidance and supported Timbers variables. Project-specific dates belong in declared `vars` or artifact frontmatter supplied by the caller. |
| `docs/publishing-artifacts.md` | Publishing examples hand-roll draft, model, frontmatter, and deployment pipelines. | Lead with `timbers report` for configured report generation, then describe stdout/Markdown as the platform-neutral boundary. Link to the in-repo Timbermill demo without making Timbermill mandatory. |
| `README.md` site links and examples | Current URLs must remain, but implementation is Hugo-specific behind the scenes. | After route-parity verification, update development instructions to `just site-build`, `just site-test`, and `just site-serve`; no user-facing copy needs to advertise the renderer. |
| Historical site output | Hugo/PaperMod footers and old Git-notes claims exist in generated `site/public/`. | Do not patch generated HTML. Eleventy replaces the generated tree. Historical Markdown remains dated content; the new layout supplies current navigation and historical context. |
| Timbermill terminology | The public publishing slice is new and may later become a companion repository. | Describe Timbermill as the in-repo, platform-agnostic static publishing slice. Do not promise a separate package, remote ingestion, schedules, or multi-repository hosting until a second consumer proves that need. |

### Broader active-document cleanup

- `docs/spec.md` mixes implemented behavior with early Git-notes, `--replace`,
  auto-extraction, and roadmap-era requirements. Split or annotate sections;
  never treat the whole file as executable truth.
- `docs/agent-reference.md` should add `ack`, `report`, corruption behavior, and
  the actual optional-`what` contract while removing duplicated low-value flag
  prose already available from `--help`.
- `docs/tutorial.md` should teach one smooth capture path, one `report` path,
  and the SHA degradation model. Keep explicit `what` as the recommended path
  when commit subjects are weak.
- `docs/agent-dx-guide.md` is partly a reusable design guide. Timbers-specific
  examples must use current names, but generic `mytool` examples should not be
  mechanically rewritten unless their pattern is wrong.
- `docs/timbers-integration-guide.md` correctly warns that squash/rebase rewrite
  anchors, but should also explain stored-text fallback and report enrichment
  degradation so “breaks anchors” is not read as “loses rationale.”
- `AGENTS.md` project reference must list `report` and optional `what`, but the
  injected Beads and Timbers ownership blocks should be regenerated by their
  owning commands rather than hand-maintained in parallel.

## Smallest Implementation Sequence

1. **Land behavior first.** Finish and commit report profiles and the
   Timbermill/Eleventy migration. Record the final help, JSON examples, `just`
   recipes, and site output path from passing builds.
2. **Correct trust-boundary claims.** Fix optional `what`, SHA rewrite language,
   file storage, global `jsonFlag`, corrupt-ledger behavior, and the invalid
   template examples. These are factual defects, not editorial preferences.
3. **Document the report split once.** Update README, LLM guide, tutorial,
   agent reference, publishing guide, and AGENTS command summary with one
   consistent distinction: `report` supplies profile defaults and compact
   input; `draft` remains explicit and fully composable.
4. **Reconcile decision artifacts.** Make every active surface call generated
   decision output a non-authoritative digest and native ADRs authoritative.
   Preserve old decision-log posts as historical artifacts.
5. **Complete the publishing transition.** Replace active Hugo instructions
   with verified Eleventy/Timbermill recipes, regenerate the site, and confirm
   current navigation never presents old capabilities as current.
6. **Label historical material.** Add short status banners to roadmap,
   design-decisions, and research docs; add a `docs/plans/README.md` explaining
   that dated plans are implementation history. Avoid line-by-line rewrites.
7. **Delete repetition.** Where active docs copy full flag tables, retain only
   workflow-relevant flags and direct readers to `timbers <command> --help`.
8. **Add narrow drift checks, run `just check`, and regenerate examples.** Do
   not add a general documentation framework.

## Minimal Automated Drift Checks

Add one focused Go test or small stdlib-only checker invoked by `just check`.
It should fail with file and line, and cover only contracts that have already
drifted:

1. Scan active guidance files for present-tense retired syntax:
   `timbers catchup`, `timbers draft decision-log`, Go-template field/range
   expressions, current-storage claims containing `Git notes`, and global
   `jsonFlag`. Exclude dated plans and historical site posts explicitly.
2. Assert the active command summary contains `report` and that every command
   shown in its command table exists on `newRootCmd()`. Do not attempt to
   execute arbitrary shell snippets from Markdown.
3. Validate the custom-template example against the supported token names from
   `internal/draft/render.go`; a small explicit token set is preferable to a
   documentation parser.
4. After Eleventy lands, assert current build/config files do not contain Hugo
   commands and run the existing deterministic site build/link checks. Do not
   ban the word “Hugo” from historical Markdown.

Do not generate a full command-reference site, add a Markdown AST dependency,
or snapshot all Cobra help. Those approaches create a second generated-doc
workflow to maintain. The narrow checks above target demonstrated failures and
leave prose free to evolve.

## Verification

- Run every active quick-start and report example in a temporary repository,
  using prompt-render mode where a model call is unnecessary.
- Confirm documented JSON examples parse and retain the command's actual top
  level shape.
- Run `timbers draft --list` and `timbers report decision-digest` against the
  built binary; compare template/profile names in active docs.
- Run `just site-test` and `just site-build`, then crawl internal links under
  both `/timbers/` and `/` path prefixes.
- Search active guidance for the retired names and stale storage/template
  claims listed above. Review historical matches to ensure their dated status
  is visible rather than deleting them.
- Run `just check` after all documentation and generated examples settle.

## Explicit Non-Goals

- Rewriting historical posts to match current architecture.
- Building a documentation generator or adopting a documentation framework.
- Documenting deferred report registries, model policy, remote artifact
  transport, schedules, or multi-repository Timbermill hosting.
- Making Timbermill a separate repository before another consumer requires an
  independent release lifecycle.
