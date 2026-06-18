---
name: writing-style
description: Use when writing or editing gno docs, code comments, or PR review comments.
---

# Writing style

- Lead with the conclusion, so a human or an agent gets the point from the first line. In a doc that is the rule; in a review it is the verdict.
- Pitch to the audience. A user-facing doc states what the reader observes in a sentence or two, then links to the deeper doc; the example and the why live in the deeper doc, not here. Keep VM internals out of user-facing docs.
- Keep it small everywhere. The deeper doc has three parts and no more: the rule, one short example of a single case, and the why in a sentence. A subtle or rare topic does not earn a second example, a footnote, or a table of cases. Deeper mechanism goes in code comments or the source, linked.
- Precise term over hedge: "unspecified", not "may be true or false". A technical term needed for comprehension beats a full explanation of it; use the word. Spell out only opaque abbreviations: copy-on-write, not COW.
- Cut filler: drop any clause the reader already infers.
- No em-dashes, no parentheticals. Short sentences, one idea each. No "This page" openers.
- Wrap around 80 columns, no trailing whitespace.
- In code comments, keep the symbols a contributor needs, drop other-language jargon, link to the canonical source instead of restating it.
- In a review, the body's first sentence is the verdict and nothing else: a short phrase like "Looks good.", "Correct fix.", or "Blocking." Everything else, including what the PR does, comes after. One finding per block headed by its file:line, severity at the first word: Nit, Optional, or blocking. State the problem, never the fix; a human or an agent works the fix out. Keep CI and merge noise out of the findings.
