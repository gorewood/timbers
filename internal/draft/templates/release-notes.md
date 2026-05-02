---
name: release-notes
description: User-facing release notes
version: 4
---
Generate user-facing release notes from these development log entries.

**Audience**: End users of the product. Not developers, not contributors, not packagers — the people who *use* this thing.

**Format** (include only sections with content):
- **New Features** — capabilities the user can now invoke
- **Improvements** — existing capabilities that work better, faster, or more reliably
- **Bug Fixes** — broken behavior the user may have hit
- **Breaking Changes** — anything that requires the user to change what they do

**Strict exclusion criteria**: anything the user cannot directly observe does NOT belong in release notes:
- Internal refactors with no behavior change
- Test or coverage changes
- Documentation rewrites (unless docs ARE the product)
- Build, CI, or tooling tweaks
- Dependency bumps unless they fix something users hit
- Code-quality or developer-experience improvements

If applying these exclusions empties a section, omit it. If they empty the entire release, write a single sentence ("This release contains internal improvements only — no user-facing changes.") rather than padding.

**Breaking changes — every bullet must answer "what should I do?"**:
- "Removed `--legacy-mode` flag (use `--mode=legacy` instead)" — good
- "Removed `--legacy-mode` flag" — incomplete; reviewer should reject
- For schema/config breaks: name the migration step or link to a migration note in the entry's notes field

**Style**:
- Benefit-oriented language ("You can now..." not "Added support for...")
- Stay in second person ("you", "your") for active capabilities
- Drop to neutral system voice for behind-the-scenes improvements ("Searches now return results in under 100ms")
- Avoid technical jargon where possible — but use `backticks` for commands, flags, file names users will type
- One line per item — multi-sentence bullets are usually a sign the change should be split or trimmed
- Warm but not gushing — users appreciate clarity over excitement
- Match the project's existing tone: a B2B dev tool reads plainer than a consumer app. If unsure, lean conservative.

**Numbers and metrics**:
- DO NOT cite developer metrics (lines changed, files modified, test counts)
- Cite user-observable performance numbers ONLY when entries explicitly state them — never invent ("3× faster import" requires the entries to actually say so)
- "Faster", "more responsive", "uses less memory" without a number is fine

**Constraints**:
- Only include what's in the entries
- Don't invent user-facing benefits not implied by the changes
- Don't translate internal achievements into fake user benefits ("Refactored auth" → "Improved security and reliability" is fabrication unless the entries say so)
- Skip sections with no relevant entries

**Output discipline**:
- Output the release notes ONLY. No preamble, commentary, acknowledgment, or meta-discussion.
- Do not begin with "Here is..." or "I'll generate..." or any thinking-out-loud.
- Do not end with "Let me know..." or any sign-off.
- The first line of your response must be part of the document itself.

## Entries ({{entry_count}}) | {{date_range}}

{{entries_json}}
