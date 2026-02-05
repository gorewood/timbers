+++
title = 'Modernizing CLI Output'
date = '2026-01-18'
tags = ['cli', 'ux', 'lipgloss']
+++

# Modernizing CLI Output: A Journey Toward Better Readability

We've just shipped a significant quality-of-life improvement for anyone using our CLI tools: a comprehensive styling system that makes output more readable, scannable, and visually organized. This post walks through what we built, why we built it this way, and where we could use your help.

## The Problem We Solved

If you've spent time with our CLI lately, you've probably noticed that output can feel... utilitarian. Long walls of text, inconsistent formatting, and information that all blends together. This isn't just an aesthetic concern—when you're juggling multiple tools and switching between them frequently, visual clarity directly impacts productivity.

We also faced a practical challenge: different environments have different capabilities. Some users pipe output to files, others use it in CI systems without TTY support, and many work in modern terminals that support rich formatting. Our styling solution needed to handle all these cases gracefully.

## What We Built

Over this sprint, we introduced a new `Printer` abstraction with four core helpers:

**Table** renders structured data with aligned columns and optional headers. Perfect for listing resources or showing comparison data.

**Box** wraps content in a visually distinct container—great for highlighting important information or warnings without it getting lost in output noise.

**Section** organizes logical groupings of information with clear headings and indentation, making complex nested data much easier to scan.

**KeyValue** displays structured data as readable pairs, with smart alignment and optional grouping.

Each helper is **TTY-aware**: when output isn't going to a terminal (detected via standard isatty checks), we automatically degrade to plain text. This means:

- Piping to a file works perfectly without escape codes
- CI logs remain clean and readable
- Terminal output gets the visual polish it deserves

The implementation uses [lipgloss](https://github.com/charmbracelet/lipgloss) under the hood, a lightweight styling library that's become a solid standard in the Go CLI ecosystem. We chose it because it's well-maintained, has minimal dependencies, and the API is intuitive enough that new contributors can pick it up quickly.

## Architectural Decisions and Tradeoffs

**Why a new abstraction layer?** We could have sprinkled lipgloss calls throughout our command implementations, but that would've created inconsistency and made future style changes painful. By centralizing styling logic in `Printer` helpers, we get:

- Consistent visual language across the entire tool
- A single place to update styles if we rebrand or refactor
- Easier testing (we can test styling separately from command logic)

**The cost?** Adding an abstraction adds a small amount of indirection. Commands need to understand which Printer helper fits their data best, and there's a slightly steeper learning curve for new contributors. We think this tradeoff is worth it, but we're watching for feedback.

**TTY detection strategy:** We're using Go's standard library approach. This works reliably in most environments, but if you're hitting edge cases (unusual terminal emulators, specific CI systems), please [open an issue](https://github.com/timbers-dev/timbers/issues)—we want to know.

## How to Help

**Want to contribute?** Here are some ways to get involved:

1. **Test in your environment.** Our TTY detection is solid in common cases, but uncommon setups are exactly where bugs hide. Try piping output, redirecting to files, and testing in your terminal emulator of choice. [Report any formatting weirdness.](https://github.com/timbers-dev/timbers/issues/new)

2. **Suggest styling improvements.** If a particular command's output feels cluttered or hard to parse, let us know. Show us what you'd want to see instead. These insights directly shape what we build next.

3. **Help us document patterns.** New contributors often wonder: "Which Printer helper should I use here?" If you're adding a new command, consider documenting your choice in a code comment or PR description. This helps the next person (and helps us build better guidelines).

4. **Review and improve the lipgloss integration.** Our Printer helpers are straightforward, but there's room for sophistication: theme support, dark/light mode detection, accessibility considerations. If you have thoughts on any of these, we'd love to hear them.

5. **Port more commands.** Not every command uses the new helpers yet. This is intentional—we wanted to test the approach on a few commands before rolling it out everywhere. If you see a command that hasn't been updated, that's a great starting point for a contribution.

## Technical Details

The changes span 12 files with 317 insertions and 189 deletions. The largest additions are the Printer implementations (~150 lines) and command updates (~100 lines). Most of the deletions are older formatting code we've now replaced.

If you want to dig in, the anchor commit is `0386e77`, and the full range is `abf716e..0386e77`. The implementation is straightforward enough that it's a good reference for anyone looking to understand how the project handles cross-cutting concerns.

## What's Next

We're planning to:

- Expand Printer helpers to cover more use cases (progress bars, tree structures)
- Add built-in theme support so users can customize colors
- Explore accessibility features like reducing reliance on color alone

But we'll do this iteratively based on what you actually need. If you have a use case in mind, jump into the discussion.

## Thanks

This sprint was a great example of the project's collaborative spirit. Thanks to everyone who's filed output-related issues, those who tested early versions, and the contributors who reviewed the implementation. This is how we build tools that work well for everyone.

What would *you* want to see in the CLI output next? Drop a comment, open an issue, or swing by our discussions. We're listening.
