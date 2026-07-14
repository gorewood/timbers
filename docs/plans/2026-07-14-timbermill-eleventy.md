# Timbermill: Eleventy Publishing Slice

**Date:** 2026-07-14  
**Owner bead:** `timbers-67f`  
**Status:** Reviewed; ready for implementation

## Goal

Turn `site/` into the first working Timbermill slice: configured local Markdown
collections in, a static directory out. Replace Hugo with Eleventy, preserve the
current public URLs, support authoritative native artifacts alongside generated
reports, and materially improve the demo's presentation and accessibility.

This remains a Timbers subproject until a second consumer requires independent
installation, versioning, or release cadence.

## Boundaries

Timbermill owns:

- discovering configured Markdown files in the current repository;
- preserving source content, frontmatter, and relative-path identity;
- rendering collection indexes and artifact pages;
- producing a self-contained static output directory.

Timbermill does not own report generation, LLM execution, schedules, Git
operations, authentication, remote-repository transport, or deployment. GitHub
Pages remains one downstream deployment of the static output, not part of the
publishing contract.

## Collection Contract

Add `site/timbermill.json`:

```json
{
  "site": {
    "title": "Timbers",
    "description": "Git knows what changed. Timbers captures why.",
    "path_prefix": "/timbers/"
  },
  "collections": [
    {
      "id": "examples",
      "label": "Examples",
      "kind": "generated",
      "root": "site/content/examples",
      "include": "[!_]*.md",
      "route": "examples"
    },
    {
      "id": "reports",
      "label": "Development reports",
      "kind": "generated",
      "root": "site/content/posts",
      "include": "*.md",
      "route": "posts"
    }
  ]
}
```

Rules:

- Paths are repository-root-relative. Absolute paths and roots outside the
  repository are rejected.
- `id` and `route` are unique; routes cannot overlap renderer-owned paths.
- `include` uses Node's stable `fs.globSync` syntax. Pin Node 22.17 or newer so
  no glob dependency is needed.
- Symlinked directories are not followed in the first slice.
- Each match must be a Markdown file inside its declared root. Duplicate output
  identities fail the build.
- The source-relative path is the artifact's stable identity. The public path is
  `/<route>/<relative-path-without-extension>/`.
- The materializer copies source bytes. It does not normalize prose or rewrite
  frontmatter.
- YAML frontmatter is the supported public contract. Convert the demo's simple
  TOML frontmatter during migration rather than adding a TOML parser.

Recognized optional frontmatter is `title`, `date`, `status`, `summary`, `tags`,
and `source`. Unknown fields survive and remain available to templates. Missing
fields receive presentation defaults; they do not make an artifact invalid.

`kind: native` marks artifacts authored outside Timbers, such as repository
ADRs or design documents. Native artifacts remain authoritative. A generated
decision digest is a report and must never be labeled, numbered, or routed as a
native ADR.

## Materialization

Add one Node script, `site/scripts/materialize.mjs`, using only `node:fs` and
`node:path`:

1. Read and validate `timbermill.json`.
2. Remove and recreate ignored `site/.generated/` so deleted sources cannot
   remain published.
3. Expand each root/include pair without following directory symlinks.
4. Validate containment and route uniqueness.
5. Copy matching Markdown under `.generated/<route>/<relative-path>`.
6. Emit minimal directory data that supplies collection metadata, the shared
   artifact layout, and a permalink with the `.generated` prefix removed.

Eleventy reads the staged tree plus renderer templates and writes `site/_site/`.
The build entry point is:

```text
node scripts/materialize.mjs && eleventy
```

Add one built-in Node test covering path preservation, nested paths, cleanup
after source deletion, root escape rejection, and duplicate-route rejection.

## Route And Content Migration

Preserve these routes exactly:

- `/`
- `/examples/`
- `/examples/<slug>/`
- `/posts/`
- `/posts/<date-slug>/`

Keep historical reports as dated historical material even when they describe
retired behavior. Current navigation and instructional copy must only describe
current capabilities.

Before deployment, audit all public demo Markdown and generalize references to
non-public projects, repositories, people, domains, and operational details.
This is a content correction, not a reusable redaction engine. Add an automated
deny-list only if the same leak recurs.

## Eleventy Structure

Add only the renderer pieces the current site needs:

```text
site/
  eleventy.config.js
  package.json
  package-lock.json
  timbermill.json
  _data/site.json
  _includes/base.njk
  _includes/home.njk
  _includes/collection.njk
  _includes/artifact.njk
  assets/site.css
  assets/site.js
  scripts/materialize.mjs
  scripts/materialize.test.mjs
```

Use the stable Eleventy 3 release recorded in the lockfile. Do not add a theme,
CSS framework, animation library, font service, component framework, search
plugin, or image pipeline.

After route parity is verified, remove `site/hugo.toml`, `site/go.mod`,
`site/go.sum`, and `site/layouts/index.html`.

## Style And Experience

Replace the long, effect-driven marketing page with a concise product and
publishing demo:

- Keep `Timbers` and its literal value proposition prominent in the first
  viewport.
- Show real ledger/report output as the primary visual proof and expose reports
  and examples as first-class destinations.
- Let the next section remain visible in the first viewport.
- Use a restrained neutral palette with amber as an accent and a second
  semantic color; remove decorative gradients, blurred orbs, glow effects, and
  nested cards.
- Use system type and authored CSS. Keep reading width, heading scale, and
  metadata density appropriate for long technical reports.
- Give collection pages compact, scannable lists rather than decorative cards.
- Label native artifacts and generated reports clearly. Artifact pages show
  available date, status, source, and provenance without inventing missing
  metadata.

The release badge should not be hard-coded into homepage HTML. Prefer removing
the version number; release state already has an authoritative home and should
not create presentation-only commits.

## Accessibility And Responsive Requirements

- Semantic `header`, `nav`, `main`, `article`, and `footer`, plus a skip link.
- Visible keyboard focus and no interaction available only on hover.
- Mobile navigation uses a native disclosure where practical; otherwise expose
  `aria-expanded` and `aria-controls` and support Escape.
- No content starts hidden. JavaScript is optional progressive enhancement.
- Copy feedback uses an accessible live status.
- Decorative icons are hidden from assistive technology; icon buttons have
  accessible names.
- Meet WCAG AA contrast for body text and metadata.
- Respect reduced motion; no scroll-triggered animation is required.
- Code and tables scroll within their container without widening the page.
- At 320px, install commands wrap or scroll without clipping and no text or
  control overlaps.
- Long titles and unbroken identifiers wrap without shifting navigation or
  collection layouts.

## Build And Deployment Changes

- `.mise.toml`: replace Hugo with a pinned Node version at least 22.17.
- `justfile`: add canonical `site-build`, `site-test`, and `site-serve` recipes;
  update blog/example generators to emit YAML frontmatter; replace
  `blog-serve`; remove the release-time HTML version `sed`.
- `just check`: include the deterministic site test and build once dependencies
  are installed.
- `.gitignore`: allowlist the new package, config, scripts, templates, data, and
  assets; ignore `site/node_modules/`, `site/.generated/`, and `site/_site/`.
- `.github/workflows/pages.yml`: use `actions/setup-node`, `npm ci`, and
  `npm run build`; upload `site/_site`. Retain Pages configure/upload/deploy
  actions and the `site/**` trigger.
- Keep `path_prefix` in site data and use Eleventy's URL/base-path handling so
  the same build works at a subpath or `/` without editing templates.

## Verification Matrix

| Surface | Desktop | Mobile | Additional check |
|---|---:|---:|---|
| Home | 1440x900 | 390x844, 320x700 | Product and next section visible |
| Collection indexes | 1440x900 | 390x844 | Sorting, labels, long titles |
| Long changelog | 1280x800 | 390x844 | Tables/code do not widen page |
| Typical report | 1280x800 | 390x844 | Metadata and prose hierarchy |

For each surface:

- capture browser screenshots and inspect layout and overflow;
- verify all internal links under both `/timbers/` and `/` path prefixes;
- compare the pre/post-migration route inventory and artifact counts;
- test keyboard navigation, mobile disclosure, no-JavaScript rendering, and
  reduced-motion mode;
- check rendered page landmarks, heading order, accessible names, and contrast;
- inspect browser console and network failures;
- run a static link crawl against `site/_site/`.

Do not remove Hugo until this matrix passes.

## Implementation Order

1. Add Node/Eleventy tooling, config validation, materializer, and its test.
2. Add shared layouts and preserve all existing routes with unchanged content.
3. Replace the homepage and shared styling; complete accessibility and mobile
   behavior before visual polish.
4. Convert frontmatter, scrub public content, and update current instructional
   copy.
5. Update `just`, mise, ignore rules, and Pages.
6. Run the verification matrix, then remove Hugo and its module files.

## Explicit Deferrals

- Publishing or fetching content across repositories.
- Git submodules, package-based content modules, webhooks, and API ingestion.
- Deployment-provider adapters or a Timbermill deployment CLI.
- Report generation, LLM providers, schedules, and artifact writeback.
- Full-text search, feeds, pagination, comments, analytics, and theme plugins.
- Automatic redaction or a generalized content policy engine.
- Symlink traversal and artifact formats other than Markdown with YAML
  frontmatter.
- Extracting Timbermill into a separate repository or independently versioned
  package.

Add these only after the in-repository demo works and a second real consumer
demonstrates the missing requirement.

## Acceptance Criteria

- The materializer validates configured roots/globs and deterministically
  stages generated and native Markdown collections.
- Native artifacts retain authority, relative paths, content, and frontmatter.
- Eleventy produces a host-neutral `site/_site/` with every existing route.
- The redesigned site is usable without JavaScript and passes the browser,
  responsive, keyboard, reduced-motion, and overflow checks above.
- Public content contains no non-public project or operational references.
- GitHub Pages deploys the Eleventy output without platform logic entering the
  Timbermill build.
- Hugo, PaperMod, Tailwind Play CDN, Google Fonts, and GSAP are absent.
- The only runtime build dependency added is Eleventy.
