---
name: devblog
description: Narrative developer blog post reflecting on recent work
version: 5
---
Write a developer blog post from these development log entries. The post should read as an engaging technical essay — the story of what happened and what it meant, not a log of what changed.

**Project context** (use only if this is the first batch):
{{project_description}}

**Is this the first batch?**: {{is_first_batch}}
- If "true": This is the genesis post. Weave in a brief intro (1-2 sentences) explaining what the project is and why it exists. Don't make it formal — just enough so a new reader isn't lost.
- If "false": Readers already know the project. Jump straight into the work.

**Voice**: Three influences, blended naturally:
- *Storyteller* (Atwood/Coding Horror): Lead with curiosity or frustration, not conclusions. Narrate failure honestly. Use self-deprecating humor to make hard stuff approachable. The human experience of building software matters as much as the technical substance.
- *Conceptualist* (Fowler): Elevate specific decisions into general principles. Name the pattern or concept at play. Zoom out from the specific to the general without losing the thread.
- *Practitioner* (Orosz/Pragmatic Engineer): Show real tradeoffs and constraints. Good engineering means choosing between imperfect options — acknowledge that. Ground abstract ideas in real stakes.

**Tone**:
- First person, conversational — one developer talking to peers and stakeholders at once
- State things plainly — no hedging ("I think," "arguably," "to be fair")
- Use concrete specifics: tool names, what broke, what the fix actually was
- If something was surprising, frustrating, or delightful — *say so*
- Short paragraphs. White space is your friend.
- Occasional wit, metaphor, or mild hyperbole keeps it human
- Self-deprecation works; self-flagellation doesn't
- Own your decisions — acknowledge tradeoffs without apologizing for them

Tone calibration — aim for the right column:

| Instead of... | Try... |
|---|---|
| "I implemented a spatial hash to improve collision detection performance." | "The collision system worked fine until we hit a certain object density. Then it just... didn't." |
| "I tried three approaches before settling on the current solution." | "The first approach was elegant on paper and embarrassing in profiling." |
| "Refactored the module to reduce coupling." | "The module had its fingers in everything. We spent the morning teaching it some boundaries." |

**Structure**: Essay, not listicle. No section headers — use these as invisible scaffolding only:
1. **Hook**: Open with the problem, the question, or the moment something broke. Do NOT start with the solution. Do NOT start with "Today I worked on..." Start mid-thought if possible.
2. **The work**: The bulk of the post. Structure it as scenes, not steps. Each scene has: a challenge or question → what you tried → what you learned. Write in prose, not bullets. Let the interesting parts breathe — the weird bug, the unexpected win, the "wait that actually worked?" moment gets more space. Aim for 3-5 paragraphs.
3. **The insight**: Name the thing you actually learned. One clear takeaway the reader will remember. It can be a design principle, a surprising system behavior, or a hard-won tradeoff. Acknowledge what the solution *doesn't* solve.
4. **The landing**: Where does this leave the work? Not a roadmap, not "next steps include..." Just: does it feel like solid ground? End when the point is made.

Do NOT use `#`, `##`, or `###` headers in the output. The post flows as continuous prose with paragraph breaks — no visible scaffolding.

**Working with entries**: Treat entries as raw material, not structure.
- Don't narrate commit-by-commit. The commits are receipts; the blog is the story. Group related entries into a single narrative beat.
- Look across all entries for the arc: What was the initial assumption? Where did it break down? What did the breakthrough look like?
- Drop the timestamps. "At 2pm I..." is a log entry. A dev blog exists outside the timeline of the session.
- Not every entry needs to make it into the post. Edit aggressively — the goal is the essential arc, not completeness.

**Notes field**: Some entries include a `notes` field with deliberation context — alternatives explored, surprises, reasoning chains. When present, mine these aggressively. The notes often contain the best narrative material: the journey to a decision, the dead ends that taught something, the "wait, that actually worked?" moments.

**Numbers and metrics**:
- DO NOT cite raw diff stats like "10 insertions, 3 deletions" or "362 lines changed"
- Convey scale through feel: "a tiny tweak", "a modest refactor", "a surprisingly hefty chunk of plumbing"
- If a change touched many files, say "scattered across the codebase" not "modified 12 files"
- Counts of commits, files, lines are *robotic* — paraphrase with texture instead

**Markdown formatting** (use these!):
- **Bold** for emphasis on key terms or the punchline of a paragraph
- `backticks` for function names, flags, commands, file names
- *Italics* for slight emphasis or inner thoughts
- Occasional > blockquotes for asides or reflections
- Short inline code is better than none

**Anti-patterns to avoid**:
- Changelog voice: "Added X. Fixed Y. Refactored Z." — that's documentation, not a story
- Tutorial voice: step-by-step instruction assumes the reader wants to reproduce the work; a dev blog assumes they want to *understand* it
- Apology voice: "This is probably not the best approach, but..." — own your decisions
- Metrics dump: "Changed 12 files, 362 insertions" — convey scale through feel, not numbers
- Cliffhanger non-ending: "I'll write more about this later" is not a landing — commit to a closing thought
- Padding: if entries are thin, write *less*, not filler. Omission over fabrication.

**Critical constraints**:
- ONLY write about what's actually in the entries. Do not speculate about future work, roadmaps, or plans.
- Do not invent metrics, performance numbers, or details not present.
- Do not reference technologies, patterns, or concepts not mentioned in the entries.
- If the entries are thin, write a shorter post. A tight 200-word post beats a padded 600-word one.

**Output discipline**:
- Output the blog post ONLY. No preamble, commentary, acknowledgment, or meta-discussion.
- Do not begin with "Here is..." or "I'll generate..." or any thinking-out-loud.
- Do not end with "Let me know..." or any sign-off.
- The first line of your response must be part of the blog post itself.

**Length**: 300-800 words. Shorter is better if the entries are sparse.

**Footer**: End with a brief, minimal transparency note (one line, no specific model names).

## Development Log Entries

{{entries_json}}
