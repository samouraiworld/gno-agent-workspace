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
- Don't vouch for code with a bare adjective ("sound", "correct", "safe", "fine") or a bare absence ("no auth defect found", "nothing broken"). Both are unverifiable reassurance. State the specific checks run and what each showed, or locate the findings ("every finding is in the docs, not the auth path"). A bare absence-claim with no named check behind it is filler; cut it or name the check.
- When more than one thing is verified, prefer one plain claim that covers them all ("ran the realm and both guards; each rejects the attacker case it claims to"). List separately, one per line, only when synthesis would drop something load-bearing. Never a prose run-on of several packed, jargon-dense checks.
- State a verification only when it's a runtime check the test suite doesn't and can't cover (revert-repro, cross-language parity, an e2e path the harness can't assert). Static-analysis reasoning (call-site reads, idempotency arguments) and anything a unit test already asserts add nothing: tests carry that proof, so drop them. When the only proof is the tests, name what they cover in one line and stop; don't narrate the trace.
- Plain words over named jargon in visible text: "a middle realm can't pass the admin check", not "no confused-deputy path". Use the jargon term only when it saves real length and the reader surely knows it.
- State the problem and stop. Drop the why-it-matters chain (the reader infers it) and the fix (they work it out). Keep a fix only when the remedy is non-obvious, and then name the outcome, not the steps.
- Always link every named thing: a file, symbol, PR, issue, package, or external project gets a link the first time it appears, no exceptions. Anchor the link on the words already in the prose. A reference with no link is a defect.
- In code comments, keep the symbols a contributor needs, drop other-language jargon, link to the canonical source instead of restating it.
- In a review, lead with the verdict only where no separate field already states it. The review file Summary opens with a short phrase like "Looks good." or "Correct fix."; the comment draft's `Event:` line carries the verdict, so its body never restates it and goes straight to substance. Everything else, including what the PR does, comes after. One finding per block headed by its file:line. No severity tag at the first word: not Nit, not Optional, not blocking. State the problem directly; if it is written, it is meant to be read. State the problem, never the fix; a human or an agent works the fix out. Keep CI and merge noise out of the findings.
