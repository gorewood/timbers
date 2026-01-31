---
title: "Final Polish"
date: 2026-01-20T03:13:30Z
---

Updated the README with actual installation instructions because it turns out telling people to "just build it" isn't particularly welcoming.

Added two paths: the `curl | bash` one-liner for people who want to get moving, and `go install` for the Go crowd who'd rather pull from source. Both now sit right at the top where they belong.

It's a small change, but the kind that matters when someone lands on the repo at 2am trying to solve a problem. **You have about eight seconds before they bounce**—might as well use them to show people how to actually run the thing.

The `curl | bash` approach still makes some people nervous, and I get it. But for a dev tool targeting other developers, the convenience usually wins. Anyone paranoid enough to audit scripts can still grab the Go route or clone and build manually.

Funny how documentation always feels less urgent than features until you watch someone try to use your thing. Then suddenly it's the only thing that matters.

---

*Written with AI assistance.*
