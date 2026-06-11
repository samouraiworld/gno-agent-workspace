---
name: gno-review
description: Quick adversarial review of one or more Gno PRs. Takes space-separated PR numbers, outputs severity-grouped findings per PR plus a comment.md GitHub review draft, posted after user approval.
argument-hint: <pr-number> [pr-number...]
---

# Gno PR Review

Review one or more PRs from the `gnolang/gno` repository.

**Always optimize for the human reader.** Reviews, repros, test files, commit messages, comments — every artifact this skill produces is read by a person making a decision (merge or block; trust or verify; paste or edit; pay attention or skim). Optimize for their cognitive load, not the LLM's convenience. Concrete consequences: skimmable structure (verdict first, then narrative, then findings); concise prose (strip filler, hedging, articles); clickable references (markdown links over bare paths); self-sufficient artifacts (a file pulled out of context still makes sense); enough explanation to act on, no more.

**Input:** `$ARGUMENTS` — space-separated PR numbers or GitHub URLs. Process each PR independently.

## Review all

When invoked with "review all" (no explicit PR numbers), build the target set:

```bash
ls reviews/pr/2xxx reviews/pr/4xxx reviews/pr/5xxx 2>/dev/null | grep -oE '^[0-9]+' | sort -un > /tmp/reviewed.txt
gh pr list -R gnolang/gno --state open --limit 200 --json number,title,isDraft \
  --jq '.[] | select(.isDraft==false) | "\(.number)\t\(.title)"' > /tmp/open_nondraft.txt
while IFS=$'\t' read -r num title; do
  grep -qx "$num" /tmp/reviewed.txt || printf '%s\t%s\n' "$num" "$title"
done < /tmp/open_nondraft.txt
```

From the result, exclude PRs whose title starts with `WIP` and dependabot PRs (`app/dependabot`) unless the user explicitly asks to include them. Confirm the final list with the user before reviewing more than one PR, then process each via the parallel dispatch below.

## Parallel dispatch (multi-PR runs)

When `$ARGUMENTS` contains more than one PR, dispatch **one Agent per PR** in a single message (multiple `Agent` tool calls in the same response so they run concurrently). Use `subagent_type: general-purpose` and pass each subagent a self-contained prompt of the form:

> Run the gno PR review workflow at `skills/review.md` on PR `<number>` (URL: `<url>`). Follow every step in that file — fetch, worktree, diff, comments, CI, deep read, write the review file, write `overview.html`, draft `comment.md`. Do not commit, push, regenerate the indexes, or post the review; the parent does all of that at the end. Report back the review file path and a one-paragraph summary of the verdict and headline findings.

Do **not** sequence the agents (no waiting for one before launching the next). After all subagents return, the parent runs `./scripts/build-indexes.sh` once, then a single `git add reviews/ && git commit && git push` covering all reviews.

For a single-PR run, skip the dispatch and execute the steps below directly.

## For each PR

### Fetch & understand

- Fetch the latest master in the `gno/` submodule: `git -C gno fetch origin master`
- Create a new git worktree for each PR based on latest master to keep reviews independent:
  ```bash
  git -C gno worktree add ../.worktrees/gno-review-<number> origin/master
  ```
  This creates the worktree at `<workspace-root>/.worktrees/gno-review-<number>` (relative to `gno/`, `../` resolves to the workspace root).
- Checkout the PR **inside that worktree** (use the worktree path as the working directory):
  ```bash
  # workdir: <workspace-root>/.worktrees/gno-review-<number>
  gh pr checkout <number> -R gnolang/gno
  ```
  All subsequent file reads and test runs for this PR must also use this worktree path as the working directory.
- `gh pr view <number> -R gnolang/gno --json title,body,author,baseRefName,headRefName,files,additions,deletions,commits`
- `gh pr diff <number> -R gnolang/gno`
- **Always read the PR body, all comments (`gh api repos/gnolang/gno/issues/<number>/comments`), and review comments (`gh api repos/gnolang/gno/pulls/<number>/comments`).** Note unresolved threads.
- Check `reviews/pr/<thousand>xxx/<number>-*/` for past reviews of this PR (e.g. PR #5405 → `reviews/pr/5xxx/5405-*/`). `<thousand>` is the leading digit(s) of the PR number (4 for 4000–4999, 5 for 5000–5999). Always read previous reviews first — they provide context on known issues, prior discussions, and what changed since. Focus on what changed since the last reviewed commit.
- Read linked issues for motivation.
- Read every changed file in full (not just diff hunks) for surrounding context.
- Identify callers, dependents, and sibling files to understand blast radius.

### Re-review rounds (head advanced)

When a prior round exists and the head moved from `<old-sha>` to `<new-sha>`, first check whether the PR's own content changed, or only its base moved (master merge / rebase). Compare the patch-id of the PR's diff against its merge-base at both heads:

```bash
# workdir: .worktrees/gno-review-<number>
git fetch origin master
git diff $(git merge-base origin/master <old-sha>) <old-sha> | git patch-id --stable
git diff $(git merge-base origin/master <new-sha>) <new-sha> | git patch-id --stable
```

- **Patch-ids equal** (only the base moved): do NOT re-author anything. Run `./scripts/reanchor-round.py <number> <new-sha>` — it re-runs this patch-id gate, copies the latest round's `.md` files into `<n+1>-<new-sha>/`, rewrites every sha reference, and remaps line anchors against the new head, flagging anchors it cannot map. Fix the flagged anchors by reading the worktree, update the round note at the top of the review file (one line: head advanced, PR content unchanged, anchors re-cut, verdict unchanged), regenerate the indexes, commit. Skip the rest of the workflow; `overview.html` is not touched.
- **Patch-ids differ**: full re-review round, focused on what changed since `<old-sha>`.

### Run tests

- Check CI status first: `gh pr checks <number> -R gnolang/gno`. Note any failures.
- `.gno` packages: `gno test -v ./path/to/package`
- `.go` packages: `go test -v -run 'relevant' ./path/to/package/...`
- Record pass/fail per affected package.
- For PRs changing the runtime behavior of a server or tool (`contribs/gnodev`, `gnovm/cmd/gno`, `gnovm/pkg/packages`, `gno.land/pkg/gnoweb`), boot it for real and exercise the changed behavior over actual usage (gnodev from the PR worktree + `curl` for gnoweb changes; a real external gno workspace, e.g. `github.com/samouraiworld/gnodaokit`, for loader/tooling changes). Record what was verified live in the review file. Unit tests alone don't count as verification for these PRs.

### Review the diff

Read every line. Look for:

- Correctness (logic errors, nil checks, type assertions, off-by-one)
- Security (caller validation, access control, coin/banker handling)
- Determinism violations
- Realm state safety (partial updates, re-entrancy)
- Error handling (panics vs errors, swallowed errors)
- Untested code paths
- Breaking changes without migration
- Style inconsistencies with the codebase
- Reuse and simplification: logic duplicating an existing helper, code foldable into something simpler, and anything the next developer would have to re-learn (unclear naming, missing doc comment on an exported symbol, non-obvious invariant). These land as Suggestions/Nits, never blockers.
- Check if the PR touches areas covered by `docs/`. Flag if documentation needs updating.

**Verify every finding against the actual file before including it.** Re-read the source — never write findings from memory or summaries.

### (Optional) Write adversarial tests

When findings suggest fragile or under-tested code, write edge-case tests to validate or break the PR. Run them. Failures are potential bugs — report them. Save all test files to `reviews/pr/<thousand>xxx/<number>-<short-slug>/<n>-<short-commit-hash>/tests/`.

Each adversarial test file MUST start with this two-part header, in this order, BEFORE the `package` line:

1. A single-line audit disclaimer: `// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.`
2. A `/* Run: ... */` multiline block giving the exact commands to reproduce. Use `/* */` (not `//` per line) so the commands paste cleanly into a shell.

**The header MUST be runnable from a gnolang/gno clone alone** — no `.worktrees/`, `$GNO`, or home paths. The `git checkout <hash>` pin is what makes the repro survive a force-push. (Pin test-file headers only; review and comment.md repro blocks never pin — they target the PR head.)

```
/* Run: from a gno checkout:
gh pr checkout <N> -R gnolang/gno && git checkout <short-commit-hash>
curl -fsSL -o gnovm/tests/files/<name>.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/<thousand>xxx/<number>-<short-slug>/<n>-<short-commit-hash>/tests/<name>.gno
go test -v -run 'TestFiles/<name>.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/<name>.gno
*/
```

Same shape for `.txtar` integration tests — `#` comments instead of `/* */`, destination under `gno.land/pkg/integration/testdata/`.

For empirical-claim `**Repro:**` blocks inside the review itself, prefer inline heredocs (`cat > … <<'EOF' … EOF`) over `curl` so the bug shape scans without following a reference.

**Adversarial test/repro headers must stand alone (~20 lines).** Name code paths by actual symbol, not review labels (`Warning 2`, `finding above`). Shape: disclaimer + `Run:` block, one paragraph on the mechanism, 2-3 lines on the observed result, one line on how to flip the assertion.

**Pair the bug with its related baseline invariant in one assertion.** E.g. `"p==q=false q==r=true"`.

**Ship two `stdout` assertions side-by-side — one active (current bug), one commented (post-fix expectation).** Flipping the comment is a one-line edit to assert the fix:

```
stdout 'p==q=false q==r=true'   # IS:     bug — cross-tx pointer-identity break
# stdout 'p==q=true q==r=true'  # SHOULD: parity preserved across persistence
```

### Gno vs Go comparison

When the PR contains `.gno` code, write an equivalent Go test to verify behavior parity. Run both and note any discrepancies. Save test files to the same `reviews/pr/<thousand>xxx/<number>-<short-slug>/<n>-<short-commit-hash>/tests/` directory.

## Output

Reviews are read by humans. Optimize for top-to-bottom skim: verdict first, then narrative, then findings. Maximize signal per line. Strip articles, hedging, filler. No emoji. No ADR framing — refer to file paths only, not "the ADR's argument". One block per PR, in this exact format:

```markdown
# PR #<number>: <title>

URL: https://github.com/gnolang/gno/pull/<number>
Author: <author> | Base: <base> | Files: <count> | +<add> -<del>
Reviewed by: <GitHub username> | Model: <model used> | Commit: <short-sha> (<status>)
Local worktree: `git -C gno worktree add .worktrees/gno-review-<number> <short-sha>`
Overview: [visual overview](https://samouraiworld.github.io/gno-agent-workspace/reviews/pr/<thousand>xxx/<number>-<short-slug>/overview.html) · [↗](../overview.html)

`<status>` is `latest` when `<short-sha>` matches the PR's current head, or `stale — +N commits since` when the PR has advanced. Recomputed by `scripts/convert-review-links.py` on every run.

**TL;DR:** <1-2 plain-language sentences: what this PR is about and what it does, for a reader with zero context. Goal is the reviewer recalling the PR at a glance — no jargon, no findings, no decision. Always include it, even on re-reviews. Distinct from the Summary below, which is denser and carries the bug/feature shape with anchored numbers.>

**Verdict: APPROVE / REQUEST CHANGES / NEEDS DISCUSSION / CLOSE** — <one terse sentence stating decision and the open concerns by name>. Use `CLOSE` only when the PR should not be merged at all — superseded by a merged PR, abandoned for months with no path forward, premise invalidated by a later design decision, or fundamentally wrong direction. Cite the load-bearing reason in the same sentence.

## Summary
<2-4 dense sentences. What the bug/feature is, why it matters (anchor numbers — "20% of MaxTxBytes", "multiple block-production budgets"), the one-sentence shape of the fix. No jargon yet; that comes in Glossary.>

<Optional ASCII diagram when the bug/fix is shape-y (tree, chain, state machine, dataflow). A picture saves a paragraph.>

## Glossary
<Include only if 2+ project-internal terms (e.g. `setSpan`, `SpanFromGo`, `Realm`, `MsgRun`) appear below. One terse line each, in order of first use. Skip section if not needed.>

## Fix
<2-4 sentence prose explanation of what the PR changed: before-state in one sentence, after-state in one sentence, the load-bearing constraint or gate. Avoid code blocks here — the goal is to let the reader understand the change without reading code. Link to `file:Lstart-Lend` inline. Skip the section if the diff is purely additive/trivial.>

## Benchmarks / Numbers
<Table when comparing N values, before/after, or percentages. Strip prose explaining the table — let the columns speak. Always anchor naked numbers to a known budget.>

## Critical (must fix)
- **[<priority tag, plain-English>]** `file:line` — <one-line TL;DR, scannable in 2 seconds>
  <details><summary>details</summary>

  <2-4 sentence prose explanation, then a final sentence starting "Fix:" with the concrete suggestion. Only use labeled sub-bullets (**Shape:** / **Mechanism:** / etc.) when the finding has a concrete, shaped repro that benefits from being parsed structurally — bug + input + output. For conceptual findings (dead code, decay risk, audit gap) use plain prose; labels just slow the reader down.>
  </details>

## Warnings (should fix)
- **[<priority tag, plain-English>]** `file:line` — <one-line TL;DR>
  <details><summary>details</summary>

  <Same approach: prose by default, labeled sub-bullets only when the finding has a tangible Shape/Mechanism/Result/Fix breakdown.>
  </details>

If a finding was already raised by another reviewer, surface it in the TL;DR before the tag, e.g.: `- **[parse-time O(N²)]** [@thehowl](review-url) file:line — TL;DR`. The reader sees one source of truth.

## Nits
- `file:line` — <one-line TL;DR>
  <Omit `<details>` for trivial nits; add only if reasoning is non-obvious.>

## Missing Tests
- **[<priority tag>]** `file:line` — <one-line TL;DR of the missing scenario>
  <details><summary>details</summary>

  <Why the gap matters, what edge case it covers, link to adversarial test if written.>
  </details>

## Suggestions
- `file:line` — <one-line TL;DR>
  <details><summary>details</summary>

  <Rationale.>
  </details>

## Open questions
<Optional. Interesting-but-low-gain thoughts the reviewer should see but that are not posted to the PR: deferred-scope follow-ups, possible extensions, design musings. One terse line each, ending with why it wasn't posted.>

```

Efficiency rules:
- **Verdict at top.** Reader knows decision before scrolling.
- **Summary, not Story.** Plain-English narrative carrying the bug for someone who hasn't opened the diff.
- **Glossary over in-line definitions.** Define each term once at the top, not sprinkled through prose.
- **Diff/diagram up top, not buried.** Shape-y bug = draw it. Small fix = show it inline.
- **Priority tags on findings, in plain English.** Short bracketed label, prefer plain language (`[bug can come back invisibly]`) over jargon (`[invariant decay risk]`). Reader skims tags first; if the tag needs translating, it's failed.
- **Cross-reviewer attribution in the TL;DR**, not buried in `<details>`. Surfaces overlap without duplicating analysis.
- **Prose in `<details>` by default**; use labeled sub-bullets (`**Shape:**` / `**Mechanism:**` / `**Fix:**`) only when the finding has a tangible repro to structure. For conceptual findings, 2-4 sentences flow better than five one-line tags.
- **No emoji.** Status/icons are noise; the bullet structure and tags carry the visual hierarchy.
- **No Test Results section.** Test names are noise to the reader. If a test failure is review-worthy, surface it as a Critical or Warning. Otherwise stay silent — the reviewer ran the tests, that's enough.
- **Anchor numbers.** "13s" alone is meaningless; "13s = multiple block-production budgets" tells the reader why to care.
- **Cite line numbers for every assumption.** When the review claims something is true ("every translated child already has a non-zero Span", "this function is only called from X", "the cap is enforced at parse time") back it up with the `file:line` where the reader can see it for themselves. No hand-waved facts.
- **Every `file:line` reference is a clickable markdown link with two destinations**: a primary GitHub URL (renders correctly when the review is pasted into a PR comment on the web) and a suffix `↗` link to the local worktree (for in-IDE one-click navigation when reading the file directly from this repo). Format: `` [`file:line`](https://github.com/gnolang/gno/blob/<short-sha>/<path>#L<line>) · [↗](../../../../../.worktrees/gno-review-<number>/<path>#L<line>) ``. Use `#L<a>-L<b>` for ranges. `<short-sha>` is the commit hash from the review file's parent directory (`<n>-<sha>/`). Applies to TL;DRs, details, suggestions, nits — every reference. A bare `file:line` in backticks is wrong; readers can't click it. Files and tests referenced by name without a line number (a test file in Missing Tests, a test case cited in prose) get the same dual link, pointed at the file or the test's declaration line — never a bare backticked name. To convert existing reviews from the old local-only format to the dual format, run `./scripts/convert-review-links.py`.
- **Drop GitHub checkboxes (`- [ ]`)** unless reviewer wants the author to tick items — the reviewer chooses, not the template.
- **Never frame findings as "the ADR says X is fine, but actually..."** — refer to the file by path (e.g. `pr5648_spanfromgo_quadratic.md:195`) and critique the argument directly. No "the ADR" / "the audit table" editorializing.
- **Ship a copy-pasteable reproducer for every empirical claim** ("I ran X and saw Y"). Fenced `bash` block, self-contained (restore any reverted files at the end), one clear pass/fail signal. Only pin env vars (`GNOROOT=$PWD`, etc.) when the test actually depends on them — defensive padding adds noise.
- **A repro demonstrates behavior** — a test run, a request, an executed binary — never source inspection. A grep/cat over the code is not a repro. Scale repro effort to severity: heredoc behavioral tests (asserting the correct post-fix state so they fail now and pass once fixed) are for Critical/Warning empirical claims only. For Nits/Suggestions whose fact is visible at the anchor, cite the `file:line` and ship no repro block — a one-line "confirmed behaviorally: X" observation in the details is enough.
- **Every repro `bash` block MUST be followed by the actual output you observed**, in a second fenced block (no language tag — it's terminal output). The pair tells the reader two things at once: how to reproduce, and what success/failure looks like before they paste anything. Without the output, the claim is unverifiable from the review alone. Trim output to the signal-bearing lines (5–20 typically); use `# …` to mark omitted noise.
- **Every bash block in the review MUST start with `gh pr checkout <N> -R gnolang/gno` and contain ZERO references to local paths.** No `/home/...`, no `$HOME`, no `cd .worktrees/...`, no `cd reviews/...`, no `$WT`/`$REVIEW` variables pointing at this workspace. No trailing `git checkout <hash>` pin either — `gh pr checkout` lands on the PR head, that's the contract. Reviews are pasted into public GitHub PR comments — external readers run them from a fresh gno checkout, not from our workspace. If the repro needs a test file, inline it with a heredoc (`cat > path/to/file.gno <<'EOF' ... EOF`) rather than referencing it under `reviews/...` or `.worktrees/...`. Clean up at the end (`rm path/to/file.gno`, `git checkout HEAD -- ...`). Prepend a one-line prelude comment naming the prerequisite — `# from a local clone of gnolang/gno:` — above the `gh pr checkout` line so the reader knows where to be before pasting (spell out "local clone of gnolang/gno" rather than shorthand like "gno checkout", which is ambiguous to readers outside this workspace).

Calibration — finding count and severity:
- **No target count.** Stop when the diff is read in full and blast radius mapped — not when the review looks proportionate. A clean PR has zero warnings; a sprawling one may have ten. The comfortable middle is a tell you stopped early or padded.
- **Severity is binary, not a slider.** Warnings = a maintainer could plausibly block on it (correctness, security, decay, missing invariant). Nits = style, polish, optional. If a finding could go either way, it's a Nit.
- **Map the full call graph before claiming dead / redundant / unused.** Grep every caller, not just the one in the diff. One missed caller flips a real finding to a wrong one.
- **Never flag contribution-policy compliance** (AGENTS.md ADR requirement, commit conventions, AI-disclosure rules) as a finding, in the review file or in comment.md. Findings cover the code only.
- **Gain-gate deferred-scope and extension questions.** When the PR deliberately scopes something out ("for now", a TODO, an explicit non-goal), the thought goes in the review file's Open questions section, where the reviewer sees it. It reaches comment.md only when the gain is real: a concrete risk, or a decision the author must make in this PR. Low-gain housekeeping ("is this tracked anywhere?") is never posted.

Rules:
- Write one file per review: save each PR review to `reviews/pr/<thousand>xxx/<number>-<short-slug>/<n>-<short-commit-hash>/<model>_<reviewer>.md` (e.g. `reviews/pr/5xxx/5405-fix-banker-overflow/1-a1b2c3f/claude-sonnet-4_davd-gzl.md`). `<thousand>xxx` buckets PRs by leading digit(s) (4xxx for 4000–4999, 5xxx for 5000–5999). `<short-slug>` is 3-4 words from the PR title, lowercase, hyphenated. `<n>` is the review round number, incremented per PR in the order reviews are written (check existing directories to determine the next number). `<model>` is the model used (lowercase, hyphenated). `<reviewer>` is the GitHub username (get it via `gh api user --jq '.login'`). Use the HEAD commit hash of the PR branch. Multiple reviews of the same commit share the same directory, making it easy to compare across reviewers and models.
- Every finding has two layers: a one-line TL;DR with priority tag (scannable in 2 seconds, states the problem) and a `<details>` block below with sub-bullet structure: **Shape:** / **What you see:** / **Why it matters:** / **Fix:**. The TL;DR must stand alone — no "see below", no hedging. Trivial nits may omit the `<details>` block.
- The TL;DR (plus the details block's final "Fix:" sentence) is the canonical text for this finding everywhere: comment.md copies it verbatim, never re-words it. Write it so it works as a PR inline comment as-is — self-contained, names the place and the consequence, no parenthetical mechanism chains. If it doesn't read as a PR comment, rewrite it here, then copy.
- Keep bold/emphasis to a minimum. The bullet structure and `file:line` in backticks already provide enough visual separation — don't bold every TL;DR or label. Reserve bold for the rare phrase that genuinely needs to stand out inside a paragraph.
- The review is meant to be pasted into a GitHub PR comment, so the format must render correctly in GitHub-flavored markdown. Key constraints: `<details>` blocks need a blank line after `<summary>` for the inner markdown to render; indent continuation content by 2 spaces under list items; do not nest `<details>` more than one level deep.
- Every finding MUST include `file:line`.
- Empty categories: write "None" — never fabricate issues.
- Priority: correctness > security > determinism > state safety > tests > docs > style.
- Be direct. No filler. State the problem and why it matters.
- Large PRs (>20 files): summarize changes by area first, then deep-dive on critical paths.
- After writing the review file(s), regenerate the indexes: `./scripts/build-indexes.sh`. Then commit and push to this repo (`git@github.com:samouraiworld/gno-agent-workspace.git`) only: `git add reviews/ && git commit -m "review: PR #<number>" && git push`. For a multi-PR parallel run, the **parent** does this once after all subagents return (use a multi-PR commit message like `review: PRs #<a> and #<b>`); subagents must not commit or push.
- **Push is pre-authorized for this skill.** The user has standing approval for commit + push when running the review skill — do not stop to ask. This overrides the global "ask before push" rule, scoped to this skill only.
- Never push to the gnolang/gno repository.
- This skill must be run from the workspace root.
- Once the review is finished (file written, index regenerated, commit done), ask the user before opening the review worktree in VSCode. If they confirm, open it as a new window: `code <workspace-root>/.worktrees/gno-review-<number>`.

## PR overview (`overview.html`)

After writing the review file, write `overview.html` at the PR directory root — `reviews/pr/<thousand>xxx/<number>-<slug>/overview.html`, NOT inside the round directory; it explains the PR, not one commit. It is a single self-contained HTML file (inline CSS/JS, zero external requests — no fonts, CDNs, analytics) that makes the PR itself faster to grasp than prose.

**Scope: explainer only — zero review state.** No verdict, no findings, no reviewed sha, no round references. Review state lives in the review file and comment.md; duplicating it here is forbidden. Include exactly one pointer to it: a `Review files` link to the PR directory on GitHub (`https://github.com/samouraiworld/gno-agent-workspace/tree/main/reviews/pr/<thousand>xxx/<number>-<slug>/` — a tree URL, so it never goes stale).

Content — pick what fits the PR: plain-language explanation of what the PR does and why, request/state/dataflow diagram, decision table, before/after payload or benchmark bars, an interactive simulator mirroring the changed logic. When the PR hinges on domain concepts the reader may not have at hand (header semantics, cache rules, consensus internals), add a short Concepts section defining each in one or two plain sentences. If the page mirrors PR logic in JS, verify the mirror against the PR's own test table before committing and state the result on the page. No emoji. End with an "AI-generated artifact" footer linking the PR and the review directory.

Update `overview.html` only when new commits change the PR's own files (behavior, scope, shape). Head bumps where only the base moved, new findings, verdict changes, and new review rounds never touch it.

After writing or updating any `overview.html`, run `./scripts/build-indexes.sh` to regenerate `reviews/README.md` and the root `index.html` — the central page linking every review and overview, served via GitHub Pages (`https://samouraiworld.github.io/gno-agent-workspace/`). Commit `index.html` together with the review artifacts.

## GitHub review draft (`comment.md`)

After the review file is committed and pushed, draft the GitHub review into `comment.md` in the same directory as the review file. The user prunes it by hand before upload: to drop a comment, prefix its header with `SKIP ` (`## SKIP <path>:<line>`) rather than deleting it — the script never posts SKIP sections, and the marker survives regeneration.

Format:

```markdown
# Review: PR #<number>
Event: APPROVE | REQUEST_CHANGES | COMMENT

## Body
<One-line assessment folding in the repro pin ("verified on the current head (<short-sha>)"). NO per-finding bullets and no "see the inline comments" pointer — GitHub shows the inline comments with the review; any restatement is duplication. Only findings or questions without a file:line anchor get a bullet here. No PR re-description — the author knows their PR.>

Full review: <GitHub URL of the pushed review file in gno-agent-workspace>

*(AI Agent)*

## <path>:<line>
<1-3 sentences: the problem and why it matters>

<details><summary>repro</summary>

<fenced bash repro block + fenced observed-output block>
</details>

*(AI Agent)*
```

Rules:
- `Event:` maps from the verdict: APPROVE → APPROVE, REQUEST CHANGES → REQUEST_CHANGES, NEEDS DISCUSSION and CLOSE → COMMENT.
- One `## <path>:<line>` section per finding that has a file:line, all severities. Range form: `## <path>:<start>-<end>`. Line numbers reference the PR head commit (side RIGHT). Genuine questions and findings without a file:line go at the end of Body.
- Verify every anchor by reading the code at those lines in the worktree before drafting. Review-file line refs may be stale or approximate; the anchor must cover exactly the lines the sentence talks about ("this guard" must point at the guard).
- Append a local IDE link to each anchor header for one-click navigation while pruning: `## <path>:<start>-<end> [↗](../../../../../.worktrees/gno-review-<number>/<path>#L<start>)`. The upload script strips everything after the first space, so the link never reaches GitHub.
- Inline comment visible text = the finding's TL;DR plus its "Fix:" sentence, copied verbatim from the review file (priority tag stripped). comment.md never re-authors a finding; if the text doesn't work as a PR comment, fix it in the review file first, then copy. Hard cap 1-3 visible sentences. No headers, no priority tags, no bold. The repro command and its observed output go in a collapsed `<details><summary>repro</summary>` block. A repro lives in exactly one file: comment.md owns it for any finding anchored there, and the review file's finding states the observed result and links it (`[repro](comment.md)`) instead of duplicating the block; only findings that never reach comment.md keep their repro in the review file.
- State findings as facts: "X hangs forever", not "Did you run X?". No rhetorical questions, no filler openers. A genuine question is one terse line, gain-gated: post only questions whose answer changes the verdict or the author's next action. Deferred-scope and extension thoughts with small gain stay in the review file's Open questions section, not here.
- Every file or test referenced by name in visible text or inside a repro `<details>` gets the same dual link as review files: GitHub blob URL at the reviewed sha, then ` · [↗](<local worktree path>)` — never a bare backticked name. The "Full review:" line gets a relative `↗` to the review file in the same directory. The upload script strips every `[↗](...)` link (headers and bodies) at post time, so local links never reach GitHub.
- Repro blocks follow the same rules as review repros: start with `gh pr checkout`, runnable from a fresh gnolang/gno clone, zero local paths, actually run, output included.
- Repro placement: line-specific repros stay with their inline comment; suite/PR-wide repros (not tied to the anchored lines) go in a `<details>` block in the Body, and the inline comment points to it.
- Zero duplication between Body and inline comments. A finding lives in exactly one place: anchored findings are inline comments only, never restated in the Body (not even as one-liners); suite/PR-wide findings and questions without an anchor live in the Body alone, no inline echo.
- Update comment.md every time the review or its findings change: new PR commits, a new review round, re-run repros, or format/style rule changes. comment.md must always reflect the current state, never lag behind the review file.
- When the PR head has advanced past the reviewed commit: diff `<reviewed-sha>..<head>`, drop findings the new commits fix, re-run the remaining repros on the new head, and re-verify every anchor against the current diff before posting.
- Before regenerating comment.md, read the existing file and preserve every `SKIP` marker whose finding still exists. A user's prune decision is never undone by an update.
- Pin the repro commit with a "Repros run at <short-sha>." line at the end of the Body by default — the PR may advance after the repros ran. Only when the sha still matches the PR head at drafting time, fold it into the opener instead ("reproduced on the current head (<short-sha>)").
- Before drafting, attempt a repro for every Critical and Warning. Word findings without a run proof as observations, never "I ran X". Behavioral repros only — a grep over the source is not a repro; cite the anchor instead and drop the repro block.
- End every comment (Body and each inline) with `*(AI Agent)*`.
- Link to the full review inside an inline comment only when the details block is not enough.
- **Never post without explicit user approval in the current turn** ("post it", "upload"). The push pre-authorization does NOT cover posting the review.
- **APPROVE is a human decision.** Before posting `Event: APPROVE`, state the verdict and wait for the user to confirm the approval itself — a generic "post it" covers REQUEST_CHANGES/COMMENT only. Only then run the script with `--approve` (it refuses APPROVE without the flag).
- Post with `./scripts/post-pr-review.py <number> <path-to-comment.md>`. The script pre-validates anchors against the PR diff (GitHub rejects inline comments on lines outside the diff) and reports invalid ones — move those into Body, or re-run with `--skip-invalid` to post the rest. Use `--dry-run` to print the payload without posting.
- After a successful post, the script writes the GitHub URLs back into comment.md: a `Posted: <review-url>` line under the title and a `[posted](<comment-url>)` link on each anchor header. Commit and push the updated comment.md.
