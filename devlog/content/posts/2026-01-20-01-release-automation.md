---
title: "Release Automation"
date: 2026-01-20T03:12:00Z
---

I was staring at the empty prompt line, wondering why every time I needed a quick LLM answer I had to pipe it through some external tool like a medieval messenger. Then it hit me: *why not bake it right into the command?* That little spark turned into the new `--model` and `--provider` flags on the `prompt` command. I added them with a few lines of code, hooked the flag parsing into the existing `llm` package, and suddenly you can fire off a model locally or point it at a cloud endpoint without breaking your flow. It felt like slipping a new lens into an old telescope—still the same view, but suddenly clearer.

Later, I thought about the catchup command. It was fine, but the batch size was fixed, and the grouping was hard‑coded. Users kept asking, “Can I limit how many entries I process at once?” and “Can I group by day instead of work‑item?” So I tossed in `--limit`/`-l` and `--group-by`/`-g`. Nothing massive—just a tiny tweak that caps the batch and lets you pick the grouping strategy. The diff was modest, but the flexibility felt like a modest refactor that paid off every time I ran a catchup loop.

And then there’s the release story. I’ve been pushing binaries manually for years, and each time I’d think, “There’s got to be a smoother way.” Enter `goreleaser`. I dropped a `.goreleaser.yaml` into the repo, spun up a GitHub Actions workflow that fires on tag pushes, and added an `install.sh` that does checksum verification before handing you the binary. The `just install-local` helper lets me test locally before the release hits the world. It turned a clunky manual process into a hefty chunk of automation that feels almost invisible once it’s running.

*These changes are tiny in the grand scheme, but they’re the kind of incremental polish that makes the whole toolchain feel alive.*  

*Transparency: this post was generated from the recent development log entries.*
