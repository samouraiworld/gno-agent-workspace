---
name: gno-review
description: Quick adversarial review of one or more Gno PRs. Takes space-separated PR numbers, outputs severity-grouped findings per PR plus a comment.md GitHub review draft, posted after user approval.
argument-hint: <pr-number> [pr-number...]
---

# Gno PR Review

Review one or more PRs from the `gnolang/gno` repository.

Every artifact is written for a human reader: skimmable structure (verdict first, then narrative, then findings), concise prose, clickable references, self-sufficient files.

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

Exclude PRs titled `WIP*` and dependabot PRs (`app/dependabot`) unless the user explicitly includes them. Confirm the final list with the user before reviewing more than one PR, then process via parallel dispatch.

## Parallel dispatch (multi-PR runs)

When `$ARGUMENTS` contains more than one PR, dispatch one Agent per PR in a single message (multiple `Agent` calls in one response). Use `subagent_type: general-purpose` with this prompt per subagent:

> Run the gno PR review workflow at `skills/review.md` on PR `<number>` (URL: `<url>`). Follow every step in that file — fetch, worktree, diff, comments, CI, deep read, write the review file, write `overview.html`, draft `comment.md`. Do not commit, push, regenerate the indexes, or post the review; the parent does all of that at the end. Report back the review file path and a one-paragraph summary of the verdict and headline findings.

Do not sequence the agents. After all return, the parent runs `./scripts/build-indexes.sh` once, then a single `git add reviews/ && git commit && git push` covering all reviews.

Single-PR runs skip the dispatch and execute the steps directly.

## For each PR

### Fetch & understand

- `git -C gno fetch origin master`
- Create a worktree per PR:
  ```bash
  git -C gno worktree add ../.worktrees/gno-review-<number> origin/master
  ```
  This lands at `<workspace-root>/.worktrees/gno-review-<number>`.
- Checkout the PR inside that worktree (cwd = the worktree, chained so a failed cd aborts):
  ```bash
  cd <workspace-root>/.worktrees/gno-review-<number> && gh pr checkout <number> -R gnolang/gno
  ```
  All file reads and test runs for this PR use this worktree as working directory.
- `gh pr view <number> -R gnolang/gno --json title,body,author,baseRefName,headRefName,files,additions,deletions,commits`
- `gh pr diff <number> -R gnolang/gno`
- Read the PR body, all comments (`gh api repos/gnolang/gno/issues/<number>/comments`), and review comments (`gh api repos/gnolang/gno/pulls/<number>/comments`). Note unresolved threads.
- Read past reviews in `reviews/pr/<thousand>xxx/<number>-*/` first (`<thousand>` = leading digit(s): 4 for 4000–4999, 5 for 5000–5999). Focus on what changed since the last reviewed commit.
- Read linked issues.
- Read every changed file in full, not just diff hunks.
- Map callers, dependents, and sibling files for blast radius.

### Re-review rounds (head advanced)

When a prior round exists and the head moved `<old-sha>` → `<new-sha>`, compare patch-ids to tell PR-content changes from base-only moves:

```bash
# workdir: .worktrees/gno-review-<number>
git fetch origin master
git diff $(git merge-base origin/master <old-sha>) <old-sha> | git patch-id --stable
git diff $(git merge-base origin/master <new-sha>) <new-sha> | git patch-id --stable
```

- **Patch-ids equal** (base-only move): do NOT re-author. Run `./scripts/reanchor-round.py <number> <new-sha>` (re-runs this gate, copies the latest round's `.md` files into `<n+1>-<new-sha>/`, rewrites sha references, remaps line anchors, flags unmappable ones). Fix flagged anchors from the worktree, add a one-line round note at the top of the review file (head advanced, PR content unchanged, anchors re-cut, verdict unchanged), regenerate indexes, commit. Skip the rest of the workflow; `overview.html` untouched.
- **Patch-ids differ**: full re-review round, focused on what changed since `<old-sha>`.

### Run tests

- `gh pr checks <number> -R gnolang/gno` first. Note failures.
- `.gno` packages: `gno test -v ./path/to/package`
- `.go` packages: `go test -v -run 'relevant' ./path/to/package/...`
- Record pass/fail per affected package.
- PRs changing runtime behavior of a server or tool (`contribs/gnodev`, `gnovm/cmd/gno`, `gnovm/pkg/packages`, `gno.land/pkg/gnoweb`): boot it from the PR worktree and exercise the changed behavior live (gnodev + `curl` for gnoweb; a real external gno workspace, e.g. `github.com/samouraiworld/gnodaokit`, for loader/tooling changes). Record what was verified live in the review file. Unit tests alone are not sufficient verification for these PRs.

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
- Reuse and simplification: duplicated helpers, foldable code, unclear naming, missing doc comments on exported symbols, non-obvious invariants. These land as Suggestions/Nits, never blockers.
- Docs impact: flag if `docs/` needs updating.

Verify every finding against the actual file before including it — never from memory or summaries.

### (Optional) Write adversarial tests

When findings suggest fragile or under-tested code, write edge-case tests, run them, report failures. Save to `reviews/pr/<thousand>xxx/<number>-<short-slug>/<n>-<short-commit-hash>/tests/`.

Each test file starts with, before the `package` line:

1. `// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.`
2. A `/* Run: ... */` block with exact repro commands (use `/* */`, not `//` per line).

The header must be runnable from a gnolang/gno clone alone — no `.worktrees/`, `$GNO`, or home paths. Pin `git checkout <hash>` in test-file headers only; review and comment.md repro blocks never pin.

```
/* Run: from a gno checkout:
gh pr checkout <N> -R gnolang/gno && git checkout <short-commit-hash>
curl -fsSL -o gnovm/tests/files/<name>.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/<thousand>xxx/<number>-<short-slug>/<n>-<short-commit-hash>/tests/<name>.gno
go test -v -run 'TestFiles/<name>.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/<name>.gno
*/
```

Same shape for `.txtar` tests — `#` comments, destination `gno.land/pkg/integration/testdata/`.

For `**Repro:**` blocks inside the review, prefer inline heredocs (`cat > … <<'EOF' … EOF`) over `curl`.

Headers stand alone (~20 lines): disclaimer + `Run:` block, one paragraph on the mechanism, 2-3 lines on the observed result, one line on how to flip the assertion. Name code paths by actual symbol, not review labels.

Pair the bug with its related baseline invariant in one assertion (e.g. `"p==q=false q==r=true"`).

Ship two `stdout` assertions side-by-side — active (current bug) and commented (post-fix expectation):

```
stdout 'p==q=false q==r=true'   # IS:     bug — cross-tx pointer-identity break
# stdout 'p==q=true q==r=true'  # SHOULD: parity preserved across persistence
```

### Gno vs Go comparison

When the PR contains `.gno` code, write an equivalent Go test to verify behavior parity. Run both, note discrepancies. Save to the same `tests/` directory.

## Output

Verdict first, then narrative, then findings. Maximize signal per line. Strip articles, hedging, filler. No emoji. One block per PR, in this exact format:

```markdown
# PR #<number>: <title>

URL: https://github.com/gnolang/gno/pull/<number>
Author: <author> | Base: <base> | Files: <count> | +<add> -<del>
Reviewed by: <GitHub username> | Model: <model used> | Commit: <short-sha> (<status>)
Local worktree: `git -C gno worktree add .worktrees/gno-review-<number> <short-sha>`
Overview: [visual overview](https://samouraiworld.github.io/gno-agent-workspace/reviews/pr/<thousand>xxx/<number>-<short-slug>/overview.html) · [↗](../overview.html)

`<status>` is `latest` when `<short-sha>` matches the PR's current head, or `stale — +N commits since`. Recomputed by `scripts/convert-review-links.py` on every run.

**TL;DR:** <1-2 plain-language sentences: what this PR is about and does, for a reader with zero context. No jargon, no findings, no decision. Always include, even on re-reviews.>

**Verdict: APPROVE / REQUEST CHANGES / NEEDS DISCUSSION / CLOSE** — <one terse sentence: decision plus open concerns by name>. `CLOSE` only when the PR should not be merged at all (superseded, abandoned with no path forward, premise invalidated, wrong direction); cite the load-bearing reason in the same sentence.

## Summary
<2-4 dense sentences: the bug/feature, why it matters (anchor numbers — "20% of MaxTxBytes"), one-sentence shape of the fix. Jargon goes to Glossary.>

<Optional ASCII diagram when the bug/fix is shape-y.>

## Glossary
<Only if 2+ project-internal terms appear below. One terse line each, in order of first use. Skip otherwise.>

## Fix
<2-4 sentences prose: before-state, after-state, the load-bearing constraint. No code blocks. Link `file:Lstart-Lend` inline. Skip if the diff is purely additive/trivial.>

## Benchmarks / Numbers
<Table for N values / before-after / percentages. No prose explaining the table. Anchor naked numbers to a known budget.>

## Critical (must fix)
- **[<priority tag, plain-English>]** `file:line` — <one-line TL;DR>
  <details><summary>details</summary>

  <2-4 sentences prose, then a final sentence starting "Fix:". Labeled sub-bullets (**Shape:** / **Mechanism:** / etc.) only when the finding has a concrete repro to structure; plain prose otherwise.>
  </details>

## Warnings (should fix)
- **[<priority tag, plain-English>]** `file:line` — <one-line TL;DR>
  <details><summary>details</summary>

  <Same approach.>
  </details>

If another reviewer already raised a finding, attribute in the TL;DR before the tag: `- **[parse-time O(N²)]** [@thehowl](review-url) file:line — TL;DR`.

## Nits
- `file:line` — <one-line TL;DR>
  <Omit `<details>` for trivial nits.>

## Missing Tests
- **[<priority tag>]** `file:line` — <one-line TL;DR of the missing scenario>
  <details><summary>details</summary>

  <Why the gap matters, the edge case, link to adversarial test if written.>
  </details>

## Suggestions
- `file:line` — <one-line TL;DR>
  <details><summary>details</summary>

  <Rationale.>
  </details>

## Open questions
<Optional. Thoughts the reviewer should see but that are not posted to the PR: deferred-scope follow-ups, extensions, design musings. One terse line each, ending with why it wasn't posted.>

```

Format rules:
- Priority tags in plain English (`[bug can come back invisibly]`, not `[invariant decay risk]`).
- Prose in `<details>` by default; labeled sub-bullets only for findings with a tangible repro.
- No Test Results section. A review-worthy test failure becomes a Critical or Warning; otherwise silence.
- Anchor numbers to budgets ("13s = multiple block-production budgets").
- Cite `file:line` for every claim the review asserts ("this function is only called from X").
- Every `file:line` reference is a dual link: `` [`file:line`](https://github.com/gnolang/gno/blob/<short-sha>/<path>#L<line>) · [↗](../../../../../.worktrees/gno-review-<number>/<path>#L<line>) `` — GitHub blob URL at the reviewed sha plus local worktree `↗`. Ranges: `#L<a>-L<b>`. `<short-sha>` comes from the round directory name (`<n>-<sha>/`). Applies to every reference, including files/tests cited by name (link the file or its declaration line). Never a bare backticked `file:line` or filename. Converter for old reviews: `./scripts/convert-review-links.py`.
- No GitHub checkboxes (`- [ ]`) unless the author must tick items.
- Every empirical claim ("I ran X and saw Y") ships a copy-pasteable repro: fenced `bash` block, self-contained, one clear pass/fail signal, restoring any modified files at the end. Pin env vars only when the test depends on them.
- A repro demonstrates behavior (test run, request, executed binary) — never source inspection; a grep is not a repro. Heredoc behavioral tests (asserting the correct post-fix state: fail now, pass when fixed) are for Critical/Warning claims only. Nits/Suggestions cite the anchor, no repro block; a one-line "confirmed behaviorally: X" note in the details is enough.
- Every repro `bash` block is followed by the observed output in a second fenced block (no language tag), trimmed to the signal-bearing 5–20 lines, `# …` for omissions.
- Every bash block starts with a `# from a local clone of gnolang/gno:` prelude line, then `gh pr checkout <N> -R gnolang/gno`, and contains zero local paths — no `/home/...`, `$HOME`, `cd .worktrees/...`, `cd reviews/...`, `$WT`/`$REVIEW` variables, and no trailing `git checkout <hash>` pin. Needed test files are inlined with heredocs, never referenced under `reviews/` or `.worktrees/`. Clean up at the end (`rm`, `git checkout HEAD -- ...`).

Calibration:
- No target finding count. Stop when the diff is read in full and blast radius mapped.
- Severity is binary. Warnings = a maintainer could plausibly block (correctness, security, decay, missing invariant). Nits = style, polish, optional. Borderline → Nit.
- Map the full call graph before claiming dead / redundant / unused. Grep every caller.
- Never flag contribution-policy compliance (AGENTS.md ADR requirement, commit conventions, AI-disclosure rules). Findings cover the code only.
- Never flag or critique the ADR — its wording, symbols it names, claims it makes. Don't reference "the ADR" or editorialize "as the ADR claims" anywhere; state behavior facts directly against the code by path. If the underlying code is wrong, the finding is about the code.
- Gain-gate deferred-scope and extension questions. Deliberately scoped-out items go in Open questions; they reach comment.md only when there's a concrete risk or a decision the author must make in this PR.

Rules:
- One file per review: `reviews/pr/<thousand>xxx/<number>-<short-slug>/<n>-<short-commit-hash>/<model>_<reviewer>.md` (e.g. `reviews/pr/5xxx/5405-fix-banker-overflow/1-a1b2c3f/claude-sonnet-4_davd-gzl.md`). `<short-slug>`: 3-4 words from the PR title, lowercase, hyphenated. `<n>`: review round number (check existing directories). `<model>`: lowercase, hyphenated. `<reviewer>`: `gh api user --jq '.login'`. Hash = PR branch HEAD. Reviews of the same commit share the directory.
- Every finding: one-line TL;DR with priority tag, plus a `<details>` block (prose by default, per the format above). The TL;DR stands alone — no "see below", no hedging. Trivial nits may omit `<details>`.
- The TL;DR plus the details' final "Fix:" sentence is the canonical finding text: comment.md copies it verbatim. Write it to work as a PR inline comment as-is; if it doesn't, rewrite it here first, then copy.
- Minimal bold. Reserve it for the rare phrase that must stand out.
- Must render in GitHub-flavored markdown: blank line after `<summary>`, continuation content indented 2 spaces under list items, `<details>` nested at most one level.
- Every finding has `file:line`.
- Empty categories: "None". Never fabricate.
- Priority: correctness > security > determinism > state safety > tests > docs > style.
- Large PRs (>20 files): summarize by area first, then deep-dive critical paths.
- After writing review file(s): `./scripts/build-indexes.sh`, then `git add reviews/ && git commit -m "review: PR #<number>" && git push` — to this repo (`git@github.com:samouraiworld/gno-agent-workspace.git`) only. Multi-PR runs: the parent commits once (`review: PRs #<a> and #<b>`); subagents never commit or push.
- Push is pre-authorized for this skill — do not stop to ask. Overrides the global ask-before-push rule, scoped to this skill only.
- Never push to gnolang/gno.
- Run from the workspace root.
- After the review is finished, ask the user before opening the worktree in VSCode (`code <workspace-root>/.worktrees/gno-review-<number>`).

## PR overview (`overview.html`)

After the review file, write `overview.html` at the PR directory root — `reviews/pr/<thousand>xxx/<number>-<slug>/overview.html`, NOT inside the round directory (it explains the PR, not one commit). Single self-contained HTML file: inline CSS/JS, zero external requests. Light theme only: white/light background, dark text. Generating model in the page title — both the `<title>` tag and the visible subtitle.

Scope: explainer only — zero review state. No verdict, no findings, no reviewed sha, no round references. Exactly one pointer to the review: a `Review files` link to the PR directory tree on GitHub (`https://github.com/samouraiworld/gno-agent-workspace/tree/main/reviews/pr/<thousand>xxx/<number>-<slug>/`).

Back-to-index link to the root `index.html` at both the top and in the footer, relative path `../../../../index.html`.

Content — pick what fits: plain-language explanation, request/state/dataflow diagram, decision table, before/after payload or benchmark bars, an interactive simulator mirroring the changed logic. Add a short Concepts section when the PR hinges on domain concepts the reader may not have (one-two plain sentences each). If the page mirrors PR logic in JS, verify the mirror against the PR's own tests before committing and state the result on the page. No emoji. End with an "AI-generated artifact" footer linking the PR and the review directory.

Update `overview.html` only when new commits change the PR's own files. Base-only head bumps, new findings, verdict changes, and new review rounds never touch it.

After writing or updating any `overview.html`, run `./scripts/build-indexes.sh` (regenerates `reviews/README.md` and the root `index.html`, served at `https://samouraiworld.github.io/gno-agent-workspace/`). Commit `index.html` with the review artifacts.

## GitHub review draft (`comment.md`)

After the review file is committed and pushed, draft `comment.md` in the same directory. The user prunes by hand before upload: dropping a comment = prefixing its header with `SKIP ` (`## SKIP <path>:<line>`), never deleting — the script skips SKIP sections and the marker survives regeneration.

Format:

```markdown
# Review: PR #<number>
Event: APPROVE | REQUEST_CHANGES | COMMENT

## Body
<One-line assessment folding in the repro pin ("verified on the current head (<short-sha>)"). No per-finding bullets, no "see the inline comments" pointer, no PR re-description, no list of what the PR does or what passed. Only findings or questions without a file:line anchor get a bullet here. When clean: "Looks good. Verified on the current head (<short-sha>): <what ran> passes." and nothing else.>

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
- One `## <path>:<line>` section per finding with a file:line, all severities. Ranges: `## <path>:<start>-<end>`. Line numbers reference the PR head commit (side RIGHT). Unanchored findings and questions go at the end of Body.
- Verify every anchor by reading those lines in the worktree before drafting; the anchor must cover exactly the lines the sentence talks about.
- Append a local IDE link to each anchor header: `## <path>:<start>-<end> [↗](../../../../../.worktrees/gno-review-<number>/<path>#L<start>)`. The upload script strips everything after the first space.
- Inline comment visible text = the finding's TL;DR plus its "Fix:" sentence, verbatim from the review file, priority tag stripped. Hard cap 1-3 visible sentences. No headers, no priority tags, no bold. Repro command + observed output go in a collapsed `<details><summary>repro</summary>` block. A repro lives in exactly one file: comment.md owns it for findings anchored there; the review file states the observed result and links it (`[repro](comment.md)`); only findings that never reach comment.md keep their repro in the review file.
- State findings as facts ("X hangs forever"), not questions. A genuine question is one terse line, posted only if the answer changes the verdict or the author's next action.
- Every file or test referenced by name (visible text or repro `<details>`) gets the dual link: GitHub blob URL at the reviewed sha + ` · [↗](<local worktree path>)`. The "Full review:" line gets a relative `↗`. The upload script strips every `[↗](...)` link at post time.
- Repro blocks: same rules as review repros — start with `gh pr checkout`, runnable from a fresh gnolang/gno clone, zero local paths, actually run, output included.
- Repro placement: line-specific repros stay with their inline comment; suite/PR-wide repros go in a Body `<details>` block, inline comments point to it.
- Zero duplication between Body and inline comments. Anchored findings are inline only; unanchored findings/questions are Body only.
- Update comment.md whenever the review or findings change (new PR commits, new round, re-run repros, format changes). It never lags the review file.
- When the PR head advanced past the reviewed commit: diff `<reviewed-sha>..<head>`, drop fixed findings, re-run remaining repros on the new head, re-verify every anchor against the current diff before posting.
- Before regenerating comment.md, read the existing file and preserve every `SKIP` marker whose finding still exists.
- Pin repros with a "Repros run at <short-sha>." line at the end of the Body. When the sha still matches the PR head at drafting time, fold it into the opener instead ("reproduced on the current head (<short-sha>)").
- Attempt a repro for every Critical and Warning before drafting. Findings without a run proof are worded as observations, never "I ran X". Behavioral repros only — for source-visible facts, cite the anchor and drop the repro block.
- End every comment (Body and each inline) with `*(AI Agent)*`.
- Link to the full review inside an inline comment only when the details block is not enough.
- Never post without explicit user approval in the current turn ("post it", "upload"). Push pre-authorization does NOT cover posting.
- APPROVE is a human decision: state the verdict and wait for the user to confirm the approval itself — a generic "post it" covers REQUEST_CHANGES/COMMENT only. Then run the script with `--approve` (it refuses APPROVE without the flag).
- Post with `./scripts/post-pr-review.py <number> <path-to-comment.md>`. It pre-validates anchors against the PR diff and reports invalid ones — move those into Body, or re-run with `--skip-invalid`. `--dry-run` prints the payload without posting.
- After a successful post, the script writes GitHub URLs back into comment.md (`Posted: <review-url>` under the title, `[posted](<comment-url>)` on each anchor header). Commit and push the updated comment.md.
