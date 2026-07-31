---
name: pr-body
description: Write the title and body of a pull request. Use whenever a fix is being proposed, before opening the PR. Produces pr-body.md in the fix directory, and loops on it until it passes the checks below.
---

# PR body

The body is as important as the change. A correct diff with a bad body gets read wrong, argued over, or ignored, so treat the body as part of the deliverable and make it perfect before opening the PR.

It is read by someone with no context who has to decide whether to merge. Cut everything that does not help that decision. Prose follows `skills/writing-style.md`; the rules below are the PR-body deltas.

Two shapes, chosen by how many independent changes the PR carries. Read the matching model before drafting.

**One concern** — [gno#5999](https://github.com/gnolang/gno/pull/5999), [#5996](https://github.com/gnolang/gno/pull/5996). Four short paragraphs, about 200 words, no headers. That is the target length, not a floor.

**Several independent changes** — [gno#6006](https://github.com/gnolang/gno/pull/6006). One `### <symbol>: <one-line diagnosis>` section per change, separated by `---`, each self-contained and readable alone. Two framing paragraphs before the first section, then a one-line bridge counting what follows ("There are three distinct bugs, plus one budget that was too tight"). A final `### How each was proved` section carries all the verification, so no section argues its own case. Budget about 150 words per section; the whole may run long, because a reader takes one section at a time.

Never mix the two. A multi-change PR written as flat prose forces the reader to hold four unrelated things at once, which is the failure this shape exists to prevent.

## File

`pr-body.md` sits in the PR review directory. It opens with a header block — `Target:` (the opened PR URL, else the `compare/...?expand=1` URL), `Head:` and `Base:` with shas, `Status:` when there is something to say — then `## Title`, `## Body`, and `## Visual evidence` only when there is something to attach. The header is metadata for the user; only Title and Body get pasted into GitHub.

## Shape

Prose, broken small.

- **Short paragraphs, one idea each.** Two to four sentences. A paragraph past five is two paragraphs.
- **A one-line paragraph carries a turn in the argument.** "The reliability rating rests on one issue, so it clears on its own." That line is what a skimmer reads.
- **No process headers.** Not "Purpose", not "What changed", not "Testing". The only headers allowed are the per-change `###` sections of the multi-change shape, named for the symbol they diagnose.
- **No tables, no bullet lists, no bold, no emoji.** Content parallel and long enough to want a table wants the multi-change shape instead.
- **A diagram wherever shape beats sentences.** See below.
- **No code block** unless it is real observed output or a diagram, and then only the signal-bearing lines.
- Symbols in backticks. Delta from the link rule: an in-repo symbol needs no link.
- **A commit sha goes bare.** No backticks, no markdown link. GitHub auto-links a bare sha in its own repo and gives it the native hovercard; wrapping it in either suppresses that.
- **An empty section is deleted, not written as "None".** No `## Visual evidence` saying there is none, no heading with nothing under it.

Paragraph order, for the one-concern shape and inside each `###` section of the multi-change shape:

1. **The symptom, first sentence, in the reader's terms.** What breaks, under what condition, concretely. Then the mechanism that causes it, named by symbol. Never open with what the change does.
2. **The fix**, in a clause, as a property of the new code rather than a narration of the edit.
3. **Anything riding along**, each with its own why. One paragraph, or one per change if they are unrelated.
4. **What was verified**, last, in one or two sentences: what fails on the base branch and passes here. Never a table of jobs, never "all tests pass".

**Do not over-explain.** The reader has the diff and will read it. Give the defect, the consequence, and the context they cannot get from the code: why it matters now, what else it touches, what a number means. Never walk through a mechanism the diff shows plainly.

**Hyperlink everything.** Every symbol, file, job, endpoint, standard-library call, external API and setting named in the body gets a link on first mention, to the blob at the reviewed sha or to upstream documentation. A named thing with no link is a defect. The exception is a bare commit sha, above.

**State what the change does not achieve, up front.** A PR that does not fully solve the problem it opens with says so in the framing paragraphs, not only at the end. "This does not turn the check green. It clears one condition of three." A reader who discovers the gap in the last section has been misled by the first.

Say what was deliberately not fixed, and why, when a reader would otherwise wonder. One short paragraph, or its own section in the multi-change shape.

## Title

Lowercase after the scope, no trailing period, names the outcome not the edit. Match the target repo's convention: check its recent merged titles and its `.gitlint` before choosing. "stop the block gas price from climbing forever or panicking" beats "fix gas price bug".

## Diagrams

Draw one whenever the reader has to hold a shape in their head that a sentence makes them assemble themselves: which of N checks is the failing one, a request crossing a trust boundary, a before and after structure, an ordering that changed.

ASCII in a fenced block by default, since it renders everywhere and diffs. Mermaid only past six nodes or when edges cross.

- Label nodes with real symbols and real numbers, never placeholders.
- Mark the thing the PR changes with an arrow and three words.
- One diagram per idea. Two small ones beat one that carries the whole PR.
- It replaces the sentences it makes redundant; never keep both.
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

Do not ship the first draft. Re-read it against these, revise, and repeat until a pass changes nothing. Three rounds is normal and worth the time; stopping at one is the failure. Record the rounds in `plan.md`.

1. **Would someone with no context understand the first sentence?** If it needs a symbol they have not met, rewrite it in observable terms.
2. **Cut every sentence that does not change the merge decision.** Restating what the diff shows, narrating process, thanking reviewers, "this PR" openers.
3. **Every adjective replaced by a number or deleted.** "much faster" becomes the measurement or goes.
4. **Every claim traceable.** A statement about behaviour has a run behind it, and a statement about a limit names the limit.
5. **Read it aloud in one pass.** Anywhere you stop to re-parse, split the sentence.
6. **Skim it in ten seconds, reading only first lines and diagrams.** If that does not give the decision, lift the turns of the argument out of the paragraphs.
7. **Check it against the diff one last time.** A body describing a change that is not there is worse than no body.

The body should be shorter after each of the first two rounds. If it grows, the loop is adding rather than sharpening.

Length is the check that catches the rest, measured against the shape's own budget: about 200 words for one concern, about 150 per section for several. Past it, cut rather than restructure; the detail belongs in the review file and the plan, which the body can point at. A one-concern body at twice the model length has failed check 2, whatever it scores on the others.

## Visual evidence

A change with a user-visible surface ships a screenshot. A change to an interaction, a flow, or anything with motion ships a short video or GIF.

- Before and after, side by side, same viewport and same data. A single after-shot proves nothing about what changed.
- Crop to the surface under discussion.
- Attach by dragging into the PR body on GitHub, which uploads and inserts the markdown. Files under `reviews/pr/<thousand>xxx/<number>-<slug>/<n>-<sha>/media/` in this workspace, so the draft is reproducible. The draft marks each attachment point with the `media/` path; the real URL exists only after the drag, so never fabricate a `user-images` URL.
- Backend-only, tooling, and lint changes ship none. Do not manufacture a screenshot to look thorough.

Capture with Playwright, driving the app booted from the PR worktree:

```bash
npm i -D playwright && npx playwright install chromium
npx playwright screenshot --viewport-size=1280,800 http://localhost:3000/<route> after.png
```

For motion, `page.video` in a Playwright script, or record the viewport and convert with `ffmpeg -i in.webm -vf fps=12 out.gif`.

Playwright and chromium are not installed by default; chromium pulls roughly 150 MB, so ask before installing. When it cannot be installed, say the screenshot is missing and why; never describe what a screenshot would have shown as though it were evidence.
