# Publishing Timbers Artifacts

This document outlines strategies for publishing artifacts generated from your development ledger.

---

## Artifact Types

| Artifact | Audience | Template | Update Frequency |
|----------|----------|----------|------------------|
| CHANGELOG.md | Contributors, users | `changelog` | Per release |
| Release Notes | End users | `release-notes` | Per release |
| Decision Digest | Technical team | `decision-digest` | Weekly/milestone |
| Standup | Internal stakeholders | `standup` | Daily/weekly |
| Sprint Report | Team | `sprint-report` | Per sprint |
| Dev Blog | External community | `devblog` | Monthly/milestone |

---

## Strategy 1: Repo Artifacts (Checked In)

Best for: CHANGELOG.md, release notes that should be versioned with code.

Decision digests are retrospective reports, not ADRs. If a project maintains
native ADRs or design documents, publish those as the authoritative record and
use `decision-digest` only to review explicit decisions captured in Timbers.

For a repeatable report, prefer a report profile. It supplies its default entry
scope and compact input while keeping stdout as the publishing boundary:

```bash
timbers report decision-digest --model opus > decision-digest.md
```

Use `draft` when the caller intentionally owns the template and scope, such as
a release range.

### Manual Generation

```bash
# Before release
timbers draft changelog --range <last-release-tag>..HEAD | claude -p --model haiku > CHANGELOG-draft.md
# Review, edit, commit
```

### Just Task

Add to `justfile`:

```just
# Generate changelog draft for review
changelog-draft:
    timbers draft changelog --range $(git describe --tags --abbrev=0 2>/dev/null || echo "HEAD~50")..HEAD \
        | claude --model haiku --print > CHANGELOG-draft.md
    @echo "Review CHANGELOG-draft.md, then rename to CHANGELOG.md"

# Generate release notes
release-notes tag:
    timbers draft release-notes --range {{tag}}..HEAD \
        | claude --model haiku --print
```

---

## Strategy 2: GitHub Releases

Best for: User-facing release notes attached to tags.

### GitHub Action

```yaml
# .github/workflows/release-notes.yml
name: Generate Release Notes
on:
  release:
    types: [created]

jobs:
  generate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0  # Full history for release ranges

      - name: Install timbers
        run: go install github.com/gorewood/timbers/cmd/timbers@latest

      - name: Generate release notes
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
        run: |
          PREV_TAG=$(git describe --tags --abbrev=0 HEAD^ 2>/dev/null || echo "")
          if [ -n "$PREV_TAG" ]; then
            RANGE="--range $PREV_TAG..HEAD"
          else
            RANGE="--last 20"
          fi
          timbers draft release-notes $RANGE | claude --model haiku --print > release-notes.md

      - name: Update release
        uses: softprops/action-gh-release@v1
        with:
          body_path: release-notes.md
```

---

## Strategy 3: GitHub Pages

Best for: Dev blogs, public project updates, documentation.

### Directory Structure

```
docs/
├── index.md           # Project overview
├── tutorial.md        # Setup guide
├── changelog.md       # Generated from ledger
└── blog/
    ├── index.md       # Blog index
    ├── 2026-01-week3.md   # Weekly update
    └── 2026-01-release-1.0.md  # Milestone post
```

### GitHub Action for Weekly Blog

```yaml
# .github/workflows/weekly-blog.yml
name: Weekly Dev Blog
on:
  schedule:
    - cron: '0 9 * * 1'  # Monday 9am
  workflow_dispatch:

jobs:
  generate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Install timbers
        run: go install github.com/gorewood/timbers/cmd/timbers@latest

      - name: Generate blog post
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
        run: |
          WEEK=$(date +%Y-week%V)
          timbers draft devblog --since 7d \
            --append "This is the weekly update for $WEEK." \
            | claude --model haiku --print > docs/blog/$WEEK.md

      - name: Commit and push
        run: |
          git config user.name "github-actions[bot]"
          git config user.email "github-actions[bot]@users.noreply.github.com"
          git add docs/blog/
          git commit -m "docs: weekly blog $(date +%Y-week%V)" || exit 0
          git push
```

### Static Site and Timbermill Integration

Timbers writes report content to stdout; the publishing layer owns artifact
paths, site frontmatter, navigation, and deployment. This keeps the same report
usable with GitHub Pages, another static host, or a local build.

The in-repository Timbermill demo consumes configured Markdown collections and
produces a static directory. Generated reports and native documents can share a
site, but retain distinct semantics:

- Native ADRs and design documents remain authoritative and preserve their
  checked-in paths and frontmatter.
- Generated decision digests are retrospective reports, never numbered or
  routed as ADRs.
- The caller may wrap report output with the YAML frontmatter its collection
  expects. Timbers template frontmatter configures the prompt/profile; it is
  not emitted as artifact frontmatter.

Timbermill is optional. Any publisher that accepts reviewed Markdown can use
the same stdout contract.

For this repository, the canonical publishing interface is:

```bash
just site-test
just site-build   # writes site/_site/
just site-serve
```

`site/_site/` is host-neutral static output. GitHub Pages is one deployment
target, not part of the collection or rendering contract.

---

## Strategy 4: Internal Reports

Best for: Stakeholder updates, sprint reviews.

### Slack/Email via CLI

```bash
# Weekly standup to clipboard (macOS)
timbers draft standup --since 7d | claude --model haiku --print | pbcopy

# Or save to shared location
timbers draft sprint-report --since 14d | claude --model haiku --print > /shared/reports/sprint-$(date +%Y%m%d).md

# Or use the profile's default scope
timbers report decision-digest --model opus > /shared/reports/decision-digest.md
```

### Cron Job

```bash
# crontab -e
0 9 * * 1 cd /path/to/repo && timbers draft standup --since 7d | claude --model haiku --print | mail -s "Weekly Dev Summary" team@example.com
```

---

## Model Selection

| Use Case | Recommended Model | Reasoning |
|----------|-------------------|-----------|
| Changelog | haiku | Structured, factual |
| Release notes | haiku | Structured, factual |
| Exec summary | haiku | Brief, bullet points |
| Sprint report | sonnet | More context needed |
| Dev blog | sonnet | Creative, engaging |
| Technical deep-dive | opus | Complex analysis |

```bash
# Examples
timbers draft changelog --since 30d | claude --model haiku --print
timbers draft devblog --last 20 | claude --model sonnet --print
```

---

## Recommended Setup for New Projects

1. **CHANGELOG.md** - Generate before each release, commit to repo
2. **GitHub Releases** - Auto-generate via action, attach to tags
3. **Weekly internal summary** - Cron or manual, share via Slack/email
4. **Monthly dev blog** (optional) - GitHub Pages for public projects

---

## Tips

- Use `--model haiku` for most reports (fast, cheap, sufficient quality)
- Use `--model sonnet` for public-facing content that needs polish
- Always review generated content before publishing
- Keep templates minimal - let the LLM do the formatting
- Tag entries well to enable filtered reports (`timbers query --tag security`)
