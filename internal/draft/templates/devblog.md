---
name: devblog
description: Developer blog post reflecting on recent work
version: 4
---
Write a short developer blog post from these development log entries.

**Project context** (use only if this is the first batch):
{{project_description}}

**Is this the first batch?**: {{is_first_batch}}
- If "true": This is the genesis post. Weave in a brief intro (1-2 sentences) explaining what the project is and why it exists, using the project context above. Don't make it formal—just enough so a new reader isn't lost.
- If "false": Readers already know the project. Jump straight into the work.

**Style**: Carmack's .plan files meets a developer who actually enjoys their work. Stream-of-consciousness reflection on what got built and why. First person, conversational, one developer talking to peers. Technical but alive.

**Voice**:
- State things plainly—no hedging ("I think," "arguably," "to be fair")
- Use concrete specifics: tool names, what broke, what the fix actually was
- If something was surprising, frustrating, or delightful—*say so*
- Insider dev language is fine—readers are peers
- Short paragraphs. White space is your friend.
- Occasional wit, metaphor, or mild hyperbole keeps it human
- Self-deprecation works; self-flagellation doesn't

**Markdown formatting** (use these!):
- **Bold** for emphasis on key terms or the punchline of a paragraph
- `backticks` for function names, flags, commands, file names
- *Italics* for slight emphasis or inner thoughts
- Occasional > blockquotes for asides or reflections
- Short inline code is better than none

**Numbers and metrics**:
- DO NOT cite raw diff stats like "10 insertions, 3 deletions" or "362 lines changed"
- Instead, convey *scale* through feel: "a tiny tweak", "a modest refactor", "a surprisingly hefty chunk of plumbing"
- If a change touched many files, say "scattered across the codebase" not "modified 12 files"
- Counts of commits, files, lines are *robotic*—paraphrase with texture instead

**Structure**:
- Narrative, not listicle. No bold section headers as scaffolding.
- Start mid-thought. No preamble or "In this post..."
- End when the point is made. No summary, no "what's next" teaser.
- If entries don't support a rich narrative, write *less*—not filler.
- Let the interesting parts breathe. The weird bug, the unexpected win, the "wait that actually worked?" moment gets more space.

**Notes field**: Some entries include a `notes` field with deliberation context — the journey to a decision, alternatives explored, surprises encountered. When present, mine these for the most interesting narrative material. The notes often contain the "wait, that actually worked?" moments that make good blog posts.

**Critical constraints**:
- ONLY write about what's actually in the entries. Do not speculate about future work, roadmaps, or plans.
- Do not invent metrics, performance numbers, or details not present.
- Do not reference technologies, patterns, or concepts not mentioned in the entries.
- If the entries are thin, write a shorter post. Omission over fabrication.

**Output discipline**:
- Output the blog post ONLY. No preamble, commentary, acknowledgment, or meta-discussion.
- Do not begin with "Here is..." or "I'll generate..." or any thinking-out-loud.
- Do not end with "Let me know..." or any sign-off.
- The first line of your response must be part of the blog post itself.

**Length**: 300-600 words. Shorter is better if the entries are sparse.

**Footer**: End with a brief, minimal transparency note (one line, no specific model names).

## Development Log Entries

{{entries_json}}
