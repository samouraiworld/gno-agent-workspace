---
name: pr-body
description: Write the title and body of a pull request. Use whenever a fix is being proposed, before opening the PR. Produces pr-body.md beside the review, and loops on it until it passes the checks below.
---

# PR body

The body is read by someone with no context who has to decide whether to merge. Everything that does not help that decision is noise, and noise is the failure mode to design against.

Model: [gnolang/gno#5999](https://github.com/gnolang/gno/pull/5999) and [#5996](https://github.com/gnolang/gno/pull/5996). Read one before drafting.

## Shape

Prose, broken small. Both halves matter: unstructured walls of text and header-and-table scaffolding are the same failure seen from opposite sides.

- **Short paragraphs, one idea each.** Two to four sentences. A paragraph that runs past five is two paragraphs.
- **A one-line paragraph carries a turn in the argument.** "The reliability rating rests on one issue, so it clears on its own." Let it stand alone; that line is what a skimmer reads.
- **No headers.** Not "Purpose", not "What changed", not "Testing". A body short enough to need no navigation is the target.
- **No tables, no bullet lists, no bold, no emoji.** If the content is genuinely parallel and long, that is a signal the PR is too big, not that it needs a table.
- **A diagram wherever shape beats sentences.** See below.
- **No code block** unless it is real observed output or a diagram, and then only the signal-bearing lines.
- Symbols in backticks. Link a file, PR, or issue the first time it is named; an in-repo symbol needs no link.

Paragraph order:

1. **The symptom, first sentence, in the reader's terms.** What breaks, under what condition, concretely. Then the mechanism that causes it, named by symbol. Never open with what the change does.
2. **The fix**, in a clause, as a property of the new code rather than a narration of the edit.
3. **Anything riding along**, each with its own why. One paragraph, or one per change if they are unrelated.
4. **What was verified**, last, in one or two sentences. What fails on the base branch and passes here. Never a table of jobs, never "all tests pass".

Say what was deliberately not fixed, and why, when a reader would otherwise wonder. One short paragraph, not a section.

## Title

Lowercase after the scope, no trailing period, names the outcome not the edit. Match the target repo's convention: check its recent merged titles and its `.gitlint` before choosing. "stop the block gas price from climbing forever or panicking" beats "fix gas price bug".

## Diagrams

A diagram is not decoration. Draw one whenever the reader has to hold a shape in their head that a sentence makes them assemble themselves: which of N checks is the failing one, a request crossing a trust boundary, a before and after structure, an ordering that changed.

ASCII in a fenced block by default, since it renders everywhere and diffs. Mermaid only past six nodes or when edges cross.

Rules that keep it useful:

- Label nodes with real symbols and real numbers, never placeholders.
- Mark the thing the PR changes with an arrow and three words. The reader should find it without reading the prose.
- One diagram per idea. Two small ones beat one that tries to carry the whole PR.
- It replaces the sentences it makes redundant. A diagram plus the paragraph that restates it is worse than either alone.
- Keep it under about twelve lines. Past that it is a diagram of the wrong thing.

```
commit 8cbcad76 on main
├── meet Workflow ........ 12/12 green   ← all `gh run list` shows
├── CodeQL ............... green
└── SonarCloud ........... FAIL
    ├── reliability    D → needs A    1 issue     ← this PR
    └── security       C → needs A    89 issues   84 are policy
```

## Loop

Do not ship the first draft. Re-read it against these, revise, and repeat until a pass changes nothing. Record the rounds in `plan.md`.

1. **Would someone with no context understand the first sentence?** If it needs a symbol they have not met, rewrite it in observable terms.
2. **Cut every sentence that does not change the merge decision.** Restating what the diff shows, narrating process, thanking reviewers, "this PR" openers.
3. **Every adjective replaced by a number or deleted.** "much faster" becomes the measurement or goes.
4. **Every claim traceable.** A statement about behaviour has a run behind it, and a statement about a limit names the limit.
5. **Read it aloud in one pass.** Anywhere you stop to re-parse, split the sentence.
6. **Skim it in ten seconds, reading only first lines and diagrams.** If that does not give the decision, the turns in the argument are buried inside paragraphs and need lifting out.
7. **Check it against the diff one last time.** A body describing a change that is not there is worse than no body.

An exit criterion that matters: the body should be shorter after each round for the first two rounds. If it grows, the loop is adding rather than sharpening.

## Visual evidence

A change with a user-visible surface ships a screenshot. A change to an interaction, a flow, or anything with motion ships a short video or GIF. Prose describing a UI is the weakest form of evidence.

- Before and after, side by side, same viewport and same data. A single after-shot proves nothing about what changed.
- Crop to the surface under discussion. A full-page screenshot buries the change.
- Attach by dragging into the PR body on GitHub, which uploads and inserts the markdown. Files under `reviews/pr/<thousand>xxx/<number>-<slug>/<n>-<sha>/media/` in this workspace, so the draft is reproducible.
- Backend-only, tooling, and lint changes ship none. Do not manufacture a screenshot to look thorough.

Capture with Playwright, driving the app booted from the PR worktree:

```bash
npm i -D playwright && npx playwright install chromium
npx playwright screenshot --viewport-size=1280,800 http://localhost:3000/<route> after.png
```

For motion, `page.video` in a Playwright script, or record the viewport and convert with `ffmpeg -i in.webm -vf fps=12 out.gif`.

Neither Playwright nor a browser is installed in this environment by default, and installing chromium pulls roughly 150 MB. Ask before installing it. When it cannot be installed, say the screenshot is missing and why; never describe what a screenshot would have shown as though it were evidence.
