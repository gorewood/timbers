# Publishing Timbers Artifacts

This document outlines strategies for publishing artifacts generated from your development ledger.

---

## Artifact Types

| Artifact | Audience | Template | Update Frequency |
|----------|----------|----------|------------------|
| CHANGELOG.md | Contributors, users | `changelog` | Per release |
| Release Notes | End users | `release-notes` | Per release |
| Standup | Internal stakeholders | `standup` | Daily/weekly |
| Sprint Report | Team | `sprint-report` | Per sprint |
| Dev Blog | External community | `devblog` | Monthly/milestone |

---

## Strategy 1: Repo Artifacts (Checked In)

Best for: CHANGELOG.md, release notes that should be versioned with code.

### Manual Generation

```bash
# Before release
timbers draft changelog --since <last-release-tag> | claude --model haiku > CHANGELOG-draft.md
# Review, edit, commit
```

### Just Task

Add to `justfile`:

```just
# Generate changelog draft for review
changelog-draft:
    timbers draft changelog --since $(git describe --tags --abbrev=0 2>/dev/null || echo "HEAD~50") \
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
          fetch-depth: 0  # Full history for notes

      - name: Install timbers
        run: go install github.com/gorewood/timbers/cmd/timbers@latest

      - name: Generate release notes
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
        run: |
          PREV_TAG=$(git describe --tags --abbrev=0 HEAD^ 2>/dev/null || echo "")
          if [ -n "$PREV_TAG" ]; then
            RANGE="$PREV_TAG..HEAD"
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

### Jekyll/Hugo Integration

For static site generators, add frontmatter to templates:

```bash
# Custom template with frontmatter
cat > .timbers/templates/jekyll-post.md << 'EOF'
---
layout: post
title: "Weekly Update: {{ .Week }}"
date: {{ .Date }}
---

Write a developer blog post from these entries.
Use casual, engaging tone. Include code snippets where relevant.

## Entries
EOF
```

---

## Strategy 4: Internal Reports

Best for: Stakeholder updates, sprint reviews.

### Slack/Email via CLI

```bash
# Weekly standup to clipboard (macOS)
timbers draft standup --since 7d | claude --model haiku --print | pbcopy

# Or save to shared location
timbers draft sprint-report --since 14d | claude --model haiku --print > /shared/reports/sprint-$(date +%Y%m%d).md
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
- Tag entries well to enable filtered reports (`timbers query --tags security`)
