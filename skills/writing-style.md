---
name: writing-style
description: Use when writing or editing gno docs, code comments, or PR review comments.
---

# Writing style

- Lead with the conclusion, so a human or an agent gets the point from the first line. In a doc that is the rule; in a review it is the verdict.
- Pitch to the audience. A user-facing doc states what the reader observes in a sentence or two, then links to the deeper doc; the example and the why live in the deeper doc, not here. Keep VM internals out of user-facing docs.
- Keep it small everywhere. The deeper doc has three parts and no more: the rule, one short example of a single case, and the why in a sentence. A subtle or rare topic does not earn a second example, a footnote, or a table of cases. Deeper mechanism goes in code comments or the source, linked.
- Define a term before first use: introduce the thing, then name it. Never use
  a term and bolt the definition on a sentence later.
- Precise term over hedge: "unspecified", not "may be true or false". A technical term needed for comprehension beats a full explanation of it; use the word. Spell out only opaque abbreviations: copy-on-write, not COW.
- Cut filler: drop any clause the reader already infers.
- Never sign-post the document from inside it. No "see below", no "the last section says which", no "as mentioned above". The reader arrives there anyway, and the pointer costs a clause to say nothing.
- No em-dashes, no parentheticals. Short sentences, one idea each. No "This page" openers.
- Wrap around 80 columns, no trailing whitespace.
- Don't vouch for code with a bare adjective ("sound", "correct", "safe", "fine") or a bare absence ("no auth defect found", "nothing broken"). Both are unverifiable reassurance. State the specific checks run and what each showed, or locate the findings ("every finding is in the docs, not the auth path"). A bare absence-claim with no named check behind it is filler; cut it or name the check.
- When more than one thing is verified, prefer one plain claim that covers them all ("ran the realm and both guards; each rejects the attacker case it claims to"). List separately, one per line, only when synthesis would drop something load-bearing. Never a prose run-on of several packed, jargon-dense checks.
- State a verification only when it's a runtime check the test suite doesn't and can't cover (revert-repro, cross-language parity, an e2e path the harness can't assert). Static-analysis reasoning (call-site reads, idempotency arguments) and anything a unit test already asserts add nothing: tests carry that proof, so drop them. When the only proof is the tests, name what they cover in one line and stop; don't narrate the trace.
- Plain words over named jargon in visible text: "a middle realm can't pass the admin check", not "no confused-deputy path". Use the jargon term only when it saves real length and the reader surely knows it.
- State the problem and stop. Drop the why-it-matters chain (the reader infers it) and the fix (they work it out). Keep a fix only when the remedy is non-obvious, and then name the outcome, not the steps.
- A commit sha in prose GitHub renders goes bare: no backticks, no markdown link. Both suppress the native hovercard that a bare sha in its own repo gets.
- Never write a section to say it is empty. Delete the heading.
- Always link every named thing: a file, symbol, PR, issue, package, or external project gets a link the first time it appears, no exceptions. Anchor the link on the words already in the prose. A reference with no link is a defect.
- A claim about someone else's platform carries the link that proves it: the vendor's own documentation, the specification, the release note. Never assert what a browser, an operating system or a runtime does from memory.
- When the source already states the reason, link the line and stop. Do not restate in prose what a reader reaches in one click: the paraphrase doubles the length, and it drifts from the source the moment the source changes.
- Never fold a live observation and a source read into one setup line. A deployed instance runs an unknown revision unless it publishes one, so writing that it was checked against a sha claims a match nobody verified. Name each source and what it proves, separately.
- Name the setup only when the claim rests on it: the instance, the browser, the device, the command. A claim read straight out of the source names nothing, because the permalink already says which sha it came from. A missing device is worth naming, since it turns a verdict into a question.
- A link into code carries the line: `#L37` for one, `#L35-L42` for a range, on a `blob` URL pinned to a sha. A symbol links to the line the sentence is about: the call site when the claim is what the code does there, the definition when the claim is what the symbol is. A guard named for its effect points at the branch it guards, not at its own one-line body. Read the range back before shipping it: one that starts a line early or spills into the next symbol sends the reader to the wrong code, which is worse than no range at all. Never link a bare file, and never link a directory: a reader who lands at the top of a 700-line file has to hunt for the thing the sentence named. When the claim is about a directory, link the one file and line inside it that shows the claim.
- In code comments, keep the symbols a contributor needs, drop other-language jargon, link to the canonical source instead of restating it.
- Scannable, without losing anything. The reader must get the whole picture in one pass and reach the detail only if they want it. Lead with the state in one line. Put anything with repeating structure (findings left out, jobs run, commits) in a table, one row each, the consequence in the last column, never as prose paragraphs. Keep the reasoning behind a `<details>` block rather than cutting it: completeness lives there, speed lives above it. A reader who stops after the tables still knows what happened, what is fixed, and what is not.
- In a review, lead with the verdict only where no separate field already states it. The review file Summary opens with a short phrase like "Looks good." or "Correct fix."; the comment draft's `Event:` line carries the verdict, so its body never restates it and goes straight to substance. Everything else, including what the PR does, comes after. One finding per block headed by its file:line. State the problem directly; if it is written, it is meant to be read. Do not soften a finding with an opener like "Optional" or "non-blocking"; the severity-band prefix `skills/review.md` requires on non-Warning inline comments (`Critical:` / `Missing test:` / `Nit:` / `Suggestion:`) is the only label, and it stays. State the problem, never the fix; a human or an agent works the fix out. Keep CI and merge noise out of the findings.

## Short form

A one-line comment, a question, a chat reply. The rules above still hold, but the register is clipped.

- One idea. Stop when it lands.
- Imperative. "put all fixes in one branch", not "could you put".
- Open with `And` or `But` when adding to a previous point; don't smooth it into a transition.
- No greeting, no thanks, no apology, no "just", no "I think", no "feel free".
- A question is one line and ends there.

Sample, hand-typed: "You should rephrase that introduction and move it in #profiling-a-transaction".

## Provenance warning

Most `davd-gzl` PR bodies and inline review comments on GitHub were produced by this workflow, not typed by hand; the `Critical:` and repro-block ones are this skill's own output. Do not mine them as style samples, or the voice drifts further from the real one each round. Re-derive from a fresh hand-typed sample when one appears, preferring a short comment on someone else's PR.
