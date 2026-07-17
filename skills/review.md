---
name: gno-review
description: Adversarial review of one or more gnolang/gno PRs. Takes space-separated PR numbers or URLs; writes a severity-grouped review file plus a comment_<model>.md GitHub draft, posted only after user approval. Supports "review all" batch runs, multi-PR parallel dispatch, and a deep multi-angle mode for a single PR.
argument-hint: <pr-number> [pr-number...]
---

# Gno PR Review

Review one or more PRs from the `gnolang/gno` repository.

**Input:** `$ARGUMENTS` — space-separated PR numbers or GitHub URLs. Process each PR independently.

Every artifact is written for a human reader: verdict first, then narrative, then findings; concise prose, clickable references, self-sufficient files. Maximize signal per line: strip articles, hedging, filler; no emoji. Cut anything that doesn't change what the reader does next. Plain English everywhere, including test comments — real words, no shorthand like "decls". Lean on `docs/glossary.md`; name a concept rather than re-explain its mechanics.

## Workflow

Single-PR run, in order (multi-PR and batch runs wrap it via *Parallel dispatch*):

1. *Fetch & understand* — worktree, PR data, prior reviews.
2. *Re-review rounds* gate, when a prior round exists.
3. *Run tests*.
4. *Review the diff*, including the *Invariant catalog* walk.
5. *Write tests* for test-shaped findings; *Gno vs Go comparison* when `.gno` code changed.
6. Write the review file (*Output*).
7. Draft `comment_<model>.md` (*GitHub review draft*), run its *Final check* and QA agents.
8. `./scripts/build-indexes.sh`, then one commit and push covering everything (pre-authorized; see *Rules*).
9. Hand over: link each PR's `comment_<model>.md` draft, not only the review file. Post only on the literal `post`.

Run from the workspace root. After the review is finished, ask the user before opening the worktree in VSCode (`code <workspace-root>/.worktrees/gno-review-<number>`).

## Modes

### Review all

When invoked with "review all" (no explicit PR numbers), build the target set:

```bash
ls reviews/pr/2xxx reviews/pr/4xxx reviews/pr/5xxx 2>/dev/null | grep -oE '^[0-9]+' | sort -un > /tmp/reviewed.txt
gh pr list -R gnolang/gno --state open --limit 200 --json number,title,isDraft \
  --jq '.[] | select(.isDraft==false) | "\(.number)\t\(.title)"' > /tmp/open_nondraft.txt
while IFS=$'\t' read -r num title; do
  grep -qx "$num" /tmp/reviewed.txt || printf '%s\t%s\n' "$num" "$title"
done < /tmp/open_nondraft.txt
```

- Exclude PRs titled `WIP*` and dependabot PRs (`app/dependabot`) unless the user explicitly includes them. Confirm the final list with the user before reviewing more than one PR, then process via *Parallel dispatch*.
- When the run also covers already-reviewed PRs whose head advanced: keep only heads whose change since the last reviewed sha is real PR content — compare patch-ids per the *Re-review rounds* gate; drop patch-id-equal base-only moves. Also drop any PR the reviewer already APPROVED on GitHub (`gh api repos/gnolang/gno/pulls/<num>/reviews --jq '[.[]|select(.user.login=="<reviewer>")]|last|.state'` = `APPROVED`) — don't re-review approved work.
- Write `reviews/BATCH_STATUS.md` before dispatch and update it as agents return: the user-confirmed scope; dropped PRs grouped by reason (head-unchanged, already APPROVED, base-only move, WIP, dependabot); the final set as a table (PR, head sha, last reviewed sha and next round for re-reviews, worktree path, review dir); the resume/finalize steps. Commit it with the batch.

### Parallel dispatch (multi-PR)

When `$ARGUMENTS` contains more than one PR, the parent first creates each PR's worktree and checks out the PR (per *Fetch & understand*); subagents never run `worktree add` or `gh pr checkout`. Then dispatch one Agent per PR in a single message (`subagent_type: general-purpose`), this prompt per subagent:

> Run the gno PR review workflow at `skills/review.md` on PR `<number>` (URL: `<url>`). The worktree already exists at `<worktree-path>` with the PR checked out — never `worktree add` or `gh pr checkout`. Follow every other step in that file — diff, comments, CI, deep read, write the review file, draft `comment_<model>.md`. Do not commit, push, regenerate the indexes, or post the review; the parent does all of that at the end. Report back the review file path and a one-paragraph summary of the verdict and headline findings.

Do not sequence the agents. After all return, the parent runs `./scripts/build-indexes.sh` once, then a single commit (`review: PRs <a> and <b>`) and push covering all reviews; subagents never commit or push.

### Deep mode (multi-angle, single PR)

Trigger: the user asks for a **parallel**, **red-team / blue-team**, or **deeper** review of one PR, or "review and loop until perfect". Deep mode runs many lenses on one PR; *Parallel dispatch* runs one reviewer across many PRs. Everything else — worktree, output format, comment.md, indexes, push rules — is identical to the normal flow.

1. **Set up.** Run *Fetch & understand* and *Run tests*. Collect the diff, comments, CI status, and prior reviews once; hand the same paths to every agent.
2. **Dispatch lens agents.** One message, multiple `Agent` calls (`subagent_type: general-purpose`) so they run concurrently. Default three lenses; add more for large PRs (perf, docs, consensus impact, API surface). Each prompt is self-contained — worktree path, PR number, diff path, prior-review paths, one narrow lens — tells the agent to load `skills/invariant-catalog.md` and `docs/glossary.md` and check the catalog classes in its lens, and returns findings in this skill's severity model (Critical / Warning / Nit / Suggestion) with `file:line` citations.
   - **Red team** — bugs, broken invariants, security holes, edge cases, determinism / gas issues, missing input validation, downstream footguns.
   - **Blue team** — missing tests, undocumented invariants, hardening gaps, misuse-inviting ergonomics, migration / rollback risk.
   - **Correctness** — does the code match the PR description and linked issue? Scope drift, silent behavior changes, contract mismatches.
3. **Synthesize.** Read all reports, dedupe overlaps, re-rank by the severity ladder. Verify each finding against the actual file before keeping it — never trust an agent summary alone. Confirm every invariant-catalog class was covered by at least one lens; walk any uncovered class against the diff before finalizing.
4. **Critic pass (one round, parallel).** Dispatch 2-3 critics in one message, each a distinct lens — verdict-check, missing-blocking, severity-calibration — over the synthesized draft plus diff and worktree. Each critic returns ONLY findings that (a) flip the verdict, (b) raise a finding by ≥1 severity band, or (c) add a Critical/Warning absent from the draft; everything sub-bar is dropped at the source; nothing qualifying → exactly `NO_MATERIAL_FINDINGS`. No open-ended "what's wrong / what's missing" prompts. After critics return: dedupe, re-read each cited `file:line`, drop what doesn't hold, revise. One critic round only — never loop.
5. **Claim-verification gate (parallel).** Before drafting comment.md, dispatch one agent over the synthesized review plus worktree: extract every falsifiable claim — behavioral ("FormatFloat prints X"), structural ("only caller is keeper.go:678"), numeric ("bits = 0x7FF8…") — and for each run a check in the worktree designed to falsify it. It returns only claims that fail or can't be verified. Re-read those against the code, drop or fix, then finalize. Facts only — never severity, verdict, or design judgment; that's the critic pass.
6. **Output.** Continue with the normal flow. The metadata line appends intensity and mode to the model name: `Model: <model> (<intensity>, deep)`, e.g. `(xhigh, deep)`; ask the user if the intensity is not known. Deep mode over a commit an earlier round already reviewed opens a new round directory `<n+1>-<same-sha>`; its round note names the mode and the prior verdict it confirms or overturns. The commit message may suffix `(deep)`.

### Non-gno repositories

A PR outside `gnolang/gno` goes under `reviews/<repo>/`, not `reviews/pr/` (gno-only).

- First review for a repo: create `reviews/<repo>/README.md` with the repo's GitHub link and a one-line description.
- Write `reviews/<repo>/<number>-<short-slug>/review_<model>.md` and `comment_<model>.md`, same formats as below.
- Skip gno-only steps: submodule worktree, glossary, invariant catalog, `build-indexes.sh`, gno blob/`↗` dual links. Cite plain `file:line` from the repo's own checkout.
- Post via `gh` against the target repo (no `post-pr-review.py`), after the literal `post`.

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
  All file reads and test runs for this PR use this worktree as working directory. A reused worktree may carry unrelated uncommitted edits — leave them; never stash, clean, or revert them.
- `gh pr view <number> -R gnolang/gno --json title,body,author,baseRefName,headRefName,files,additions,deletions,commits`
- `gh pr diff <number> -R gnolang/gno`
- Read the PR body, all comments (`gh api repos/gnolang/gno/issues/<number>/comments`), and review comments (`gh api repos/gnolang/gno/pulls/<number>/comments`). Note unresolved threads.
- Read past reviews in `reviews/pr/<thousand>xxx/<number>-*/` first (`<thousand>` = leading digit(s): 4 for 4000–4999, 5 for 5000–5999). Focus on what changed since the last reviewed commit.
- Read linked issues.
- Read `docs/glossary.md` so its terms are in context while drafting findings; add any term you use but can't find there.
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
- **New head is a merge of master**: patch-ids differ whenever master touched a file the PR touches, so the move is never base-only. Run `git show <new-sha> --cc` — any hunk it prints is conflict-resolution content authored in the merge, is PR content, and is reviewed like any other diff. Master commits the merge pulls in may also add tests that the PR's own code now fails; run the affected suite on the new head.
- **`<old-sha>` unreachable (GC'd)**: skip the gate, run a full round, and read the change as the diff against the merge-base with master; note the fallback in the round note.

Every full re-review round opens with a round-note paragraph between the metadata block and the TL;DR: `Round <n>.` — head movement and shape (rebase, +N commits), what changed in the PR, prior-round findings resolved or carried.

### Run tests

- `gh pr checks <number> -R gnolang/gno` first. Note failures.
- `.gno` packages: `gno test -v ./path/to/package`
- `.go` packages: `go test -v -run 'relevant' ./path/to/package/...`
- `-run` splits its pattern on `/`, one regex per subtest level. A filetest under a subdirectory is `-run 'TestFiles/types/foo.gno$'`; an alternation may never span a `/`, or it silently matches nothing. Run one `-run` per test when comparing results.
- Baseline every failing test against the PR's merge-base before attributing it to the diff: add a worktree at that commit and run the same test there. A local Go newer than CI's drifts `go/types` message text and reddens filetests unrelated to the PR.
- Example-package tests on a branch that also modifies a stdlib: run `gno test` with `GNOROOT=<worktree-root>`, else new stdlib symbols fail preprocessing (`name X not declared`).
- Record pass/fail per affected package.
- PRs changing runtime behavior of a server or tool (`contribs/gnodev`, `gnovm/cmd/gno`, `gnovm/pkg/packages`, `gno.land/pkg/gnoweb`): boot it from the PR worktree and exercise the changed behavior live (gnodev + `curl` for gnoweb; a real external gno workspace, e.g. `github.com/samouraiworld/gnodaokit`, for loader/tooling changes). Record what was verified live in the review file's Verified section. Unit tests alone are not sufficient verification for these PRs. PRs not touching those dirs (tests/docs-only) skip the live boot.

### Review the diff

Read every line, then look for:

- Correctness (logic errors, nil checks, type assertions, off-by-one)
- Untested code paths
- Breaking changes without migration
- Style inconsistencies with the codebase
- Reuse and simplification: duplicated helpers, foldable code, unclear naming, missing doc comments on exported symbols, undocumented invariants worth a comment. These land as Suggestions/Nits, never blockers.
- Docs impact: flag if `docs/` needs updating.

Verification discipline — every finding passes these before it enters the review:

- Verify against the actual file, never from memory or summaries.
- Every behavioral claim (what a function prints or returns, what the VM produces, what a stdlib call does) ships with an actual run behind it, at every severity including Nits and Suggestions. Never assert stdlib or runtime behavior from memory.
- Any claim that the diff *causes* a behavior runs on the merge-base too: a repro that reproduces on master describes a pre-existing property, not a finding. Attribute only the delta, and state both numbers when one exists. A green repro proves the behavior, never its authorship.
- When a baseline or test kills a finding, drop it; never retrofit a new rationale onto the same conclusion. The pull to keep a finding alive under a fresh justification is the tell that the conclusion came first: re-derive from the evidence or cut it.
- A "bound" or "leak" claim is quantitative and gets measured, not asserted: name the quantity said to be bounded, vary it against the quantity said to bound it, and confirm they actually track.
- Run greps and lint in the PR worktree (`.worktrees/gno-review-<number>`), never in the `gno/` submodule (stale detached HEAD).
- Confirm symbol existence with `gno lint` run from the worktree source (`go run ../gnovm/cmd/gno lint ./path`), not IDE/language-server diagnostics; sanity-check that lint typechecks by feeding it a bogus symbol.

### Invariant catalog (mandatory)

For a PR that touches gno code (the GnoVM, stdlibs, or `.gno` packages and realms), load `skills/invariant-catalog.md`, walk every class against the diff, and confirm coverage before writing the Output. Skip for docs- or tooling-only PRs with no gno-code change.

### Write tests for test-shaped findings

When findings suggest fragile or under-tested code, write edge-case tests, run them, report failures. Save to `reviews/pr/<thousand>xxx/<number>-<short-slug>/<n>-<short-commit-hash>/tests/`.

When a finding's fix is a test the author should add, ship the test, not a description. Write it under `tests/`, assert the post-fix state, run it (red→green when it also proves a bug), and embed it in the comment.md finding to paste in. Fill a filetest golden by seeding the `// Realm:` directive with a placeholder line, then running `go test -run 'TestFiles/<name>.gno$' -update-golden-tests .` from `gnovm/pkg/gnolang/`; an empty `// Realm:` is stripped, not populated. A test-shaped finding that never reaches comment.md may stay prose.

Each test file starts with, before the `package` line, a `/* Run: ... */` block with exact repro commands (use `/* */`, not `//` per line). The header must be runnable from a gnolang/gno clone alone — no `.worktrees/`, `$GNO`, or home paths. Pin `git checkout <hash>` in test-file headers only; review and comment.md repro blocks never pin.

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

Headers stand alone: the `Run:` block, then 2-3 comment lines max covering the mechanism, the observed result at the pinned hash, and what changes when fixed. No flip-check instructions, no restating the finding. Name code paths by actual symbol, not review labels. Keep in-test comments to one line per non-obvious step.

Assert the desired post-fix state, never the bug's current output. Gno filetests: `// Output:` with the correct values, not `// Error:` matching the panic; `// Error:` only when rejection is the correct outcome (e.g. an illegal recursive type).

Pair the bug with its related baseline invariant in one assertion (e.g. `"p==q=false q==r=true"`).

Ship two `stdout` assertions side-by-side — active (current bug) and commented (post-fix expectation):

```
stdout 'p==q=false q==r=true'   # IS:     bug — cross-tx pointer-identity break
# stdout 'p==q=true q==r=true'  # SHOULD: parity preserved across persistence
```

### Gno vs Go comparison

When the PR contains `.gno` code, write an equivalent Go test to verify behavior parity. Run both, note discrepancies. Save to the same `tests/` directory.

## Links & citations

Shared by the review file and comment.md.

- Every `file:line` reference is a dual link: `` [`file:line`](https://github.com/gnolang/gno/blob/<short-sha>/<path>#L<line>) · [↗](../../../../../.worktrees/gno-review-<number>/<path>#L<line>) `` — GitHub blob URL at the reviewed sha plus local worktree `↗`. Ranges: `#L<a>-L<b>`. `<short-sha>` comes from the round directory name (`<n>-<sha>/`). Applies to every reference, including files/tests cited by name (link the file or its declaration line). Never a bare backticked `file:line` or filename. Converter for old reviews: `./scripts/convert-review-links.py`.
- Every behavioral claim links the line that proves it ("this function is only called from X"), not just claims that name a symbol.
- A blob link into a rendered file (`.md`, anything GitHub renders rather than shows as source) puts `?plain=1` before the `#L` anchor, else the line anchor is lost in the rendered page: `.../gno-security-guide.md?plain=1#L366`. Code files (`.go`, `.gno`, `.yml`, `.txtar`) already show as source, so no `?plain=1`. Worktree `↗` links never take it.
- Anchor a supporting link on a coherent word or phrase already in the prose (`a [pointer-to-array](...) hits no case`), never a standalone sentence whose only job is to host the link.
- A named doc subsection (`§5.9`, `the Render section`) is a reference too: link it to the section header line, same blob form, never bare. `[§5.9](https://github.com/gnolang/gno/blob/<short-sha>/<path>#L<header-line>)`.
- A link proves the clause it is anchored on. Read the cited lines and confirm the exact number, symbol, or behavior appears inside the range before shipping; a range that merely sits near the proof is a wrong citation. One claim per anchor: a sentence asserting two numbers carries two links, each on its own number. Holds for external sources too; for a pinned tag (`.../blob/go1.25.0/...`) fetch the file at that tag rather than reading a local checkout at another version.
- Attribute a behavior to what actually guarantees it. A toolchain implementation detail (`go/constant`'s `prec = 512`) is cited to that source and named as such, never to a language spec that does not require it; when the spec floor is weaker than the observed behavior, say so in the review before a maintainer does.

## Repro rules

Shared by `**Repro:**` blocks in the review file and repro blocks in comment.md.

- Every empirical claim ("I ran X and saw Y") ships a copy-pasteable repro: fenced `bash` block, self-contained, one clear pass/fail signal, restoring any modified files at the end. Pin env vars only when the test depends on them.
- Every block starts with a `# from a local clone of gnolang/gno:` prelude line, then `gh pr checkout <N> -R gnolang/gno`, and contains zero local paths — no `/home/...`, `$HOME`, `cd .worktrees/...`, `cd reviews/...`, `$WT`/`$REVIEW` variables, and no trailing `git checkout <hash>` pin. Inline needed test files with heredocs (`cat > … <<'EOF' … EOF`), never `curl` or references into `reviews/` or `.worktrees/`. Clean up at the end (`rm`, `git checkout HEAD -- ...`).
- Follow every repro `bash` block with the observed output in a second fenced block (no language tag), trimmed to the signal-bearing 5–20 lines, `# …` for omissions.
- A repro demonstrates behavior (test run, request, executed binary) — never source inspection; a grep is not a repro. A repro whose only output is a passing run (`--- PASS`) shows nothing CI doesn't; drop it.
- Heredoc behavioral tests (asserting the correct post-fix state: fail now, pass when fixed) are for Critical/Warning claims only. Nits/Suggestions cite the anchor, no repro block; a one-line "confirmed behaviorally: X" note in the details is enough.

## Output

One block per PR, in this exact format:

```markdown
# PR [#<number>](https://github.com/gnolang/gno/pull/<number>): <title>

URL: https://github.com/gnolang/gno/pull/<number>
Author: <author> | Base: <base> | Files: <count> | +<add> -<del>
Reviewed by: <GitHub username> | Model: <model used> | Commit: <short-sha> (<status>)
Local worktree: `git -C gno worktree add .worktrees/gno-review-<number> <short-sha>`
Overview: [visual overview](https://samouraiworld.github.io/gno-agent-workspace/reviews/pr/<thousand>xxx/<number>-<short-slug>/overview.html) · [↗](../overview.html) <— include this line only when the PR directory has an overview.html>

<Round note — re-review and same-commit deep rounds only. Re-review: "Round <n>. Head advanced <old-sha> → <new-sha> (<shape>): <what changed>; <prior findings resolved / carried>." Same-commit deep round: "Round <n> (deep — same commit <sha> round <n-1> reviewed): <prior verdict confirmed / overturned>.">

**TL;DR:** <1-2 plain-language sentences: what this PR is about and does, for a reader with zero context. No jargon, no findings, no decision. Concrete examples go in the Examples section, never piled into the TL;DR. Always include, even on re-reviews.>

**Verdict: APPROVE / REQUEST CHANGES / NEEDS DISCUSSION / CLOSE** — <one terse sentence: decision plus open concerns by name>. `CLOSE` only when the PR should not be merged at all (superseded, abandoned with no path forward, premise invalidated, wrong direction); cite the load-bearing reason in the same sentence.

## Summary
<2-4 dense sentences: the bug/feature, why it matters (anchor numbers — "20% of MaxTxBytes"), one-sentence shape of the fix. Jargon goes to Glossary.>

<Optional ASCII diagram when the bug/fix is shape-y.>

## Examples
<Optional. Concrete written-form to outcome rows (a short list or small table) that make a semantics change tangible: the input as a user would write it, and what it now does. No findings, no `file:line`, no decision; the divergent/buggy cases stay in Summary and the findings sections. Include only when examples land the behavior faster than prose: language/VM/type-system/API-shape PRs. Skip for refactors, plumbing, and bugfixes with no user-visible surface.>

## Glossary
<List the `docs/glossary.md` terms that appear below, one terse line each, in order of first use; include only if 2+ appear.>

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
- **[<priority tag, plain-English>]** `file:line` — <one-line TL;DR>
  <Omit the tag and `<details>` for a trivial nit carrying no distinct risk.>

## Missing Tests
- **[<priority tag>]** `file:line` — <one-line TL;DR of the missing scenario>
  <details><summary>details</summary>

  <Why the gap matters, the edge case, then the ready-to-add test that closes it; write and link the `tests/` artifact.>
  </details>

## Suggestions
- **[<priority tag, plain-English>]** `file:line` — <one-line TL;DR>
  <details><summary>details</summary>

  <Rationale.>
  </details>

## Verified
<Optional; standard in deep mode and for live-boot PRs. One bullet per runtime check CI does not show — the claim, then the evidence, dual-linked: revert-proofs, cross-language parity, byte-identical encodings, live-boot behavior. Never "tests pass". A final bullet may list the tests run green at the reviewed sha. The comment.md Body pin draws its at-most-three checks from here.>

## Open questions
<Optional. Thoughts the reviewer should see but that are not posted to the PR: deferred-scope follow-ups, extensions, design musings. One terse line each, ending with why it wasn't posted.>

```

### Format rules

- `<status>` is `latest` when `<short-sha>` matches the PR's current head, or `stale — +N commits since`; recomputed by `scripts/convert-review-links.py` on every run.
- Every finding line carries a plain-English priority tag, in every severity section, so the tag naming stays uniform across a review (`[bug can come back invisibly]`, not `[invariant decay risk]`). Only a trivial nit with no distinct risk drops it.
- No bare `#<number>` in any text GitHub renders inside this repo (review/comment H1, commit subject): it autolinks to `samouraiworld/gno-agent-workspace#<number>`, the wrong repo. Link it (`[#<number>](pr-url)`) or drop the `#` (commit subjects).
- Prose in `<details>` by default; labeled sub-bullets only for findings with a tangible repro.
- No Test Results section. A review-worthy test failure becomes a Critical or Warning; otherwise silence.
- Anchor numbers to budgets ("13s = multiple block-production budgets").
- Never cite an absolute value for a constant that master recalibrates (gas fixtures, byte-count goldens): it goes stale between the review and the read. Quote the merge base, or say the author re-derives it after merging.
- Cite `file:line` for every claim the review asserts, dual-linked per *Links & citations*.
- No GitHub checkboxes (`- [ ]`) unless the author must tick items.

### Calibration

- No target finding count. Stop when the diff is read in full and blast radius mapped.
- The review's verification claims (the Verified section, the Summary) follow the same rule as comment.md: only what CI does not show. Revert-proofs, behavior/Go parity, exercised edge cases, a new code path CI skips. Never "`go test ...` passes", "lint clean", "build green".
- Severity is binary. Warnings = a maintainer could plausibly block (correctness, security, decay, missing invariant). Nits = style, polish, optional. Borderline → Nit.
- A cosmetic nit that no enabled linter enforces and that changes no meaning (doc-comment periods, comment wrapping, import order) stays in the review file and never reaches comment.md. Check the repo's linter config before flagging a style convention; `.github/golangci.yml` runs `default: none` with an explicit enable list. Record it in the review file with the config link and "not posted, no change needed".
- Map the full call graph before claiming dead / redundant / unused. Grep every caller.
- Never flag contribution-policy compliance (AGENTS.md ADR requirement, commit conventions). Findings cover the code only.
- Never flag or critique the ADR — its wording, symbols it names, claims it makes. Don't reference "the ADR" or editorialize "as the ADR claims" anywhere; state behavior facts directly against the code by path. If the underlying code is wrong, the finding is about the code. When a code or test comment repeats a stale claim, anchor the finding on that comment.
- Gain-gate deferred-scope and extension questions. Deliberately scoped-out items go in Open questions; they reach comment.md only when there's a concrete risk or a decision the author must make in this PR.

### Rules

- One file per review: `reviews/pr/<thousand>xxx/<number>-<short-slug>/<n>-<short-commit-hash>/review_<model>_<reviewer>.md` (e.g. `reviews/pr/5xxx/5405-fix-banker-overflow/1-a1b2c3f/review_claude-sonnet-4_davd-gzl.md`). `<short-slug>`: 3-4 words from the PR title, lowercase, hyphenated. `<n>`: review round number (check existing directories). `<model>`: lowercase, hyphenated. `<reviewer>`: `gh api user --jq '.login'`. Hash = PR branch HEAD. Reviews of the same commit in the same mode share the directory; a deep round over an already-reviewed commit gets a new `<n+1>-<same-sha>` directory. Pre-existing rounds may lack the `review_` prefix.
- Every finding: one-line TL;DR with priority tag, plus a `<details>` block (prose by default, per the format above). The TL;DR stands alone — no "see below", no hedging. Trivial nits may omit `<details>`.
- The TL;DR plus the details' final "Fix:" sentence is the canonical finding text: comment.md copies it verbatim. Write it to work as a PR inline comment as-is; if it doesn't, rewrite it here first, then copy.
- Minimal bold. Reserve it for the rare phrase that must stand out.
- Must render in GitHub-flavored markdown: blank line after `<summary>`, continuation content indented 2 spaces under list items, `<details>` nested at most one level.
- Every finding has `file:line`.
- Empty categories: "None". Never fabricate.
- Priority: correctness > security > determinism > state safety > tests > docs > style.
- Large PRs (>20 files): summarize by area first, then deep-dive critical paths.
- Draft `comment_<model>.md` (see *GitHub review draft*) before committing, then do a single final push at the end covering the review file and comment: run `./scripts/build-indexes.sh`, then `git add reviews/ docs/glossary.md index.html && git commit -m "review: PR <number>" && git push`, to this repo (`git@github.com:samouraiworld/gno-agent-workspace.git`) only. `build-indexes.sh` rewrites the root `index.html`, so it must be staged too.
- Push is pre-authorized for this skill — do not stop to ask. Overrides the global ask-before-push rule, scoped to this skill only.
- New findings surfaced after the initial draft (a follow-up question, a deeper dig) are folded into the review file and `comment_<model>.md`, verified with a real run, and committed/pushed in the same turn automatically — never ask whether to add them. Posting still waits for the literal `post`.
- Never push to gnolang/gno.

## PR overview (`overview.html`)

Opt-in only: write or update `overview.html` ONLY when the user explicitly asks for it in the current turn (e.g. "with overview", "create the overview"). Never create it as part of the default review flow.

When requested, write `overview.html` at the PR directory root — `reviews/pr/<thousand>xxx/<number>-<slug>/overview.html`, NOT inside the round directory (it explains the PR, not one commit). Single self-contained HTML file: inline CSS/JS, zero external requests. Light theme only: white/light background, dark text. Generating model in the page title — both the `<title>` tag and the visible subtitle.

Scope: explainer only — zero review state. No verdict, no findings, no reviewed sha, no round references. Exactly one pointer to the review: a `Review files` link to the PR directory tree on GitHub (`https://github.com/samouraiworld/gno-agent-workspace/tree/main/reviews/pr/<thousand>xxx/<number>-<slug>/`).

Back-to-index link to the root `index.html` at both the top and in the footer, relative path `../../../../index.html`.

Content — pick what fits: plain-language explanation, request/state/dataflow diagram, decision table, before/after payload or benchmark bars, an interactive simulator mirroring the changed logic. Add a short Concepts section when the PR hinges on domain concepts the reader may not have (one-two plain sentences each). If the page mirrors PR logic in JS, verify the mirror against the PR's own tests before committing and state the result on the page. No emoji.

Update `overview.html` only when new commits change the PR's own files. Base-only head bumps, new findings, verdict changes, and new review rounds never touch it.

After writing or updating any `overview.html`, run `./scripts/build-indexes.sh` (regenerates `reviews/README.md` and the root `index.html`, served at `https://samouraiworld.github.io/gno-agent-workspace/`). Commit `index.html` with the review artifacts.

## GitHub review draft (`comment_<model>.md`)

Draft `comment_<model>.md` in the same directory, same `<model>` as the review file (e.g. `comment_claude-opus-4-8.md`); "comment.md" below means this file. Pre-existing rounds may still use bare `comment.md`. The user prunes by hand before upload: dropping a comment = prefixing its header with `SKIP ` (`## SKIP <path>:<line>`), never deleting — the script skips SKIP sections and the marker survives regeneration.

Auto-SKIP duplicates: when another reviewer already raised a finding (PR review or thread comment), prefix its header `SKIP` while drafting without waiting for the user, attribute them in the review file (see the attribution rule), and make `Already raised: <comment-url>` the first body line so the reaction step can find it. Split a section that bundles a raised finding with a novel one so the novel part still posts (keep `&funcName` live while SKIPping the composite-literal cases a reviewer already flagged).

Format:

```markdown
# Review: PR [#<number>](https://github.com/gnolang/gno/pull/<number>)
Event: APPROVE | REQUEST_CHANGES | COMMENT

## Body
<One-line assessment folding in the verification pin ("verified on <short-sha>"), then one-sentence bullets for unanchored findings and questions only — per the Body rules below. When clean: "Looks good. Verified on <short-sha>: <CI-invisible check>." and nothing else.>

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/<review-file-path> [↗](review_<model>_<reviewer>.md)

## <path>:<line>
<1-3 sentences: the problem and why it matters>

<details><summary>repro</summary>

<fenced bash repro block + fenced observed-output block>
</details>
```

### Body rules

- The Body has three jobs: cross-cutting synthesis the per-line comments can't carry (shared root cause, why the chosen layer is wrong), unanchored findings and questions (one sentence each: gap, then fix), and the CI-invisible verification pin. Anything else is cut.
- Anchored findings never appear in any form: no bullets, no prose recap, no "(inline)" pointer, no count ("four doc nits inline"). Zero duplication with the inline comments; unanchored findings and questions live in the Body only.
- No PR re-description, no list of what the PR does or what passed, no review-process narration ("re-review", "cross-check round"), no restating thread state the author already knows (maintainer holds, prior verdicts, merge-order notes).
- The Body is stateless, same as every inline comment: never name a review round or frame the current code as a fix relative to a prior draft ("the round-1 blocker is fixed", "now passes"). The author never saw unposted rounds, so a change already in the diff is just the diff, not a resolution. State the code's current property, not its history.
- A CI-invisible check is a runtime check actually run that the test suite does not and cannot cover: reverting the fix reproduces the bug, output matches Go across the boundary table, a behavior-preserving move returns identical data, an e2e path the harness can't assert. Static-analysis reasoning (call-site reads, idempotency arguments) and anything a unit test already asserts never appear, even folded into a "verified" line: tests carry that proof. Nothing runtime-only checked → no verification line at all.
- Name at most three checks, the strongest ones, each as a claim, not its test matrix — no parenthetical lists of tested values or shapes; the full check inventory stays in the review file's Verified section. Prefer one plain claim covering all the checks ("ran the realm and both guards; each rejects the attacker case it claims to") over a jargon-dense enumeration; list one per line only when synthesis drops something load-bearing, never a prose run-on.
- State each check as an action and its result ("verdicts match the Go compiler", "reverting the fix reproduces the bug"), never as a characterization of the change ("a real correctness gain", "not just error wording"). Never vouch for the code with a bare adjective ("the auth content is sound", "the fix is correct") or a bare absence ("no auth defect found"); state the checks or locate the findings ("every finding is in the docs, not the auth path").
- Name the revert as the concrete edit a reader can picture ("removing the line that sets `ATTR_IFACE_CMP`"), never a noun-phrase shorthand ("reverting the `ATTR_IFACE_CMP` set"). Replace vague labels ("the boundary", "the case the code could have broken") with the actual code element, and tie cause to effect in one chain ("it reads the operand types before `checkOrConvertType` rewrites them, so X still panics") so the claim lands in one pass.
- A Body check that asserts a runtime property a committed test could assert becomes that test (ship it per the missing-test rule), not Body prose. The Body keeps only checks no committed test can carry, e.g. a revert-proof: the negative direction of a shipped golden.
- Pin repros with a "Repros run at <short-sha>." line at the end of the Body. When the sha still matches the PR head at drafting time, fold it into the opener instead ("reproduced on <short-sha>").

### General rules

- Visible prose (Body and every inline comment) follows `skills/writing-style.md`: short sentences one idea each, no em-dashes, no parentheticals, no bold; state the problem directly; state the problem, not the fix.
- `Event:` maps from the verdict: APPROVE → APPROVE, REQUEST CHANGES → REQUEST_CHANGES, NEEDS DISCUSSION and CLOSE → COMMENT. The `Event:` line is the verdict; Body never restates it (no "Changes needed." opener) and goes straight to substance.
- Post only comments that change what the author does: fix, decide, or answer. A finding whose details end "no change needed" / "flagging for whoever touches this next" stays in the review file and never reaches comment.md. Severity never gates this: a Nit or Suggestion that asks for a concrete modification (a wording fix, a corrected value, a dropped line) gets its own anchored comment.md section like any Warning. The discriminator is "should the author change something," not the severity band.
- Never explain routine fixes the author would do anyway (merge master, regenerate assets, re-run a flaky job). A red CI check with a routine cause gets one short Body line ("not a code problem"), no instructions, no repro; detail it only when the cause is non-obvious or changes what the author must do.

### Building each inline comment

Walk these steps, in order, for every finding:

1. **Anchor.** One `## <path>:<line>` section per finding, all severities; ranges `## <path>:<start>-<end>`. Line numbers reference the PR head commit (side RIGHT). Read those exact lines in the worktree first; the anchor must cover exactly the lines the sentence talks about. Append the local IDE link: `## <path>:<start>-<end> [↗](../../../../../.worktrees/gno-review-<number>/<path>#L<start>)`. The upload script strips everything after the first space.
2. **Opener.** `Critical:` / `Nit:` / `Suggestion:` prefix matching the review file's severity band, then the TL;DR. A Warning gets NO prefix: it opens directly with the TL;DR. A missing-test finding opens `Missing test:` plus the uncovered scenario in one clause. The bracketed plain-English priority tag is dropped everywhere.
3. **Sentences.** Hard cap 1-3 visible sentences (code blocks and `<details>` don't count; no headers, no bold). Order: the gap and its stake first, evidence second, fix sentence last. Count the sentences before moving on; over 3 → cut evidence, not the gap.
4. **Fix sentence.** Default to none: state the problem and stop. Add one only when the remedy is genuinely non-obvious and changes what the author would do; name the desired outcome, never the implementation path or an internal symbol ("reject those too", not "call `evalStaticTypeOf` and branch on the `Func` field"). Never a fix sentence whose remedy the problem statement already implies ("the doc comment describes the wrong function" needs no "rewrite it").
5. **Links.** Every file or test referenced by name, and every behavioral claim, gets the dual link per *Links & citations* (comment.md deltas below).
6. **Repro.** Critical and Warning findings get a collapsed repro block when the claim is behavioral (comment.md deltas below).

Example (a Nit; a Warning would start directly at "this says"):

```markdown
## docs/resources/gno-example.md:42 [↗](../../../../../.worktrees/gno-review-1234/docs/resources/gno-example.md#L42)
Nit: this says `Tree.Size` is a field, but [`Size` is a method](https://github.com/gnolang/gno/blob/ab12cd34e/examples/gno.land/p/nt/avl/v0/tree.gno#L37) · [↗](../../../../../.worktrees/gno-review-1234/examples/gno.land/p/nt/avl/v0/tree.gno#L37). Write `Size()`.
```

### Visible-text style

- Plain English, essentials only: the problem and why it matters — short sentences, no stacked technical clauses, no symbol-chain walkthroughs; the reader must get it in one pass. Cut scenario-painting: keep the fact and the stake.
- Don't re-prove the claim in the visible text: mechanism detail, secondary evidence, and source enumerations belong in the repro block or the full review, not inline. If a repro or the review carries the proof, the visible text asserts the claim in one clause. Before: "It's fine here because the parent dir is already 700, but a half-sentence saying the parent dir is the real guard would stop a reader who relocates the socket from relying on a perm that can silently not apply." After: "The real guard is the 700 parent dir; say so, or a reader who relocates the socket loses the protection."
- Lead with the specific gap (the shape that slips past, the line that breaks); never open by explaining what the author's own code does ("the guard measures how far each type can expand"). Assume the author knows their own mechanism. Never restate what the PR does or claims, inline included — the author wrote it; state the gap directly, never "the property the PR is about" / "what the PR claims". Same plain register in any prose comments inside a repro block (txtar header comments included): state the shape and the gap, not a tutorial.
- A latent-risk finding (correct today, breaks for a future caller) states the current safety in one clause ("no current caller passes filetests, so it is latent") and stops.
- Lowercase a source document's emphasis caps (WRONG/RIGHT) when quoted in prose; caps survive only inside code spans.
- State findings as facts ("X hangs forever"), not questions. A genuine question is one terse line, posted only if the answer changes the verdict or the author's next action.
- A design or layering question is two sentences at most: the alternative in one clause, then whether the current choice was deliberate. State the alternative, never re-explain the author's mechanism. Example: `DeleteForKey has the machine and could mark the removed key itself instead of returning it. Deliberate split to keep it a pure container op?`
- Link to the full review inside an inline comment only when the details block is not enough.

### Links (comment.md deltas)

- All *Links & citations* rules apply. The "Full review:" line gets a relative `↗` (`[↗](review_<model>_<reviewer>.md)`). The upload script strips every `[↗](...)` link at post time.
- Write every reviewed-commit sha in comment.md prose (the `Verified on <sha>` / `reproduced on <sha>` pin, `Repros run at <sha>`) as a bare sha, no backticks and no markdown link. GitHub auto-links a bare commit sha in a gnolang/gno comment and gives it the native commit hovercard; backticks or a `[...](commit-url)` wrapper suppress the hovercard. The review file keeps its own shas as-is (rendered in our repo, where a bare gno sha wouldn't resolve; its file-line links are already clickable).

### Repros (comment.md deltas)

- Attempt a repro for every Critical and Warning before drafting. Findings without a run proof are worded as observations, never "I ran X". Behavioral repros only, per *Repro rules* — for source-visible facts, cite the anchor and drop the repro block.
- Repro command + observed output go in a collapsed `<details><summary>repro</summary>` block. A repro lives in exactly one file: comment.md owns it for findings anchored there; the review file states the observed result and links it (`[repro](comment_<model>.md)`); only findings that never reach comment.md keep their repro in the review file.
- Placement: line-specific repros stay with their inline comment; suite/PR-wide repros go in a Body `<details>` block, inline comments point to it.
- A missing-test finding carries the ready-to-add cases in a collapsed `<details><summary>test cases</summary>` block in the file's own test style: the full filetest or table rows when short, or the source plus a dual link to the large `tests/` golden. Paste-ready as-is.

### Rounds & regeneration

- Update comment.md whenever the review or findings change (new PR commits, new round, re-run repros, format changes). It never lags the review file.
- Port carried findings to a new round verbatim: only shas, repro URLs, and anchors that no longer point at the right lines change. No round-relative phrasing ("again", "still"): unposted drafts were never seen by the author.
- A finding the user SKIPped in a prior round stays SKIPped when ported forward, as long as it still applies: carry the `## SKIP` marker into the new round's comment.md with a one-line note (`Skipped in round <n>; keeping it skipped.`), never silently re-promote it to a posted comment. The user un-SKIPs it explicitly or it stays skipped.
- When the PR head advanced past the reviewed commit: diff `<reviewed-sha>..<head>`, drop fixed findings, re-run remaining repros on the new head, re-verify every anchor against the current diff before posting.
- Before regenerating comment.md, read the existing file and preserve every `SKIP` marker whose finding still exists.

### Posting

- Never post without explicit user approval in the current turn: the literal word "post" (or "upload"). "push" authorizes git push only and never covers posting.
- Same gate for mutating posted content (editing or deleting a posted comment, re-posting): update the local draft first, show the user the exact new text, and touch GitHub only after they approve it in the current turn — even when the change itself was requested.
- APPROVE is a human decision: state the verdict and wait for the user to confirm the approval itself — a generic "post it" covers REQUEST_CHANGES/COMMENT only. Then run the script with `--approve` (it refuses APPROVE without the flag).
- Post with `./scripts/post-pr-review.py <number> <path-to-comment.md>`. It pre-validates anchors against the PR diff and reports invalid ones — move those into Body, or re-run with `--skip-invalid`. `--dry-run` prints the payload without posting.
- Thumbs-up acknowledged duplicates: as part of the same `post`, react 👍 to each comment a section SKIPs as already-raised, reading the URL from its `Already raised:` line. The `<id>` is the trailing number of the URL.
  - Inline thread comment (`#discussion_r<id>`): `gh api -X POST repos/gnolang/gno/pulls/comments/<id>/reactions -f content=+1`.
  - Top-level issue/PR comment (`#issuecomment-<id>`): `gh api -X POST repos/gnolang/gno/issues/comments/<id>/reactions -f content=+1`.
  - Review body (`#pullrequestreview-<id>`): not in the REST reactions API but reactable via GraphQL. Resolve the node id (`gh api repos/gnolang/gno/pulls/<pr>/reviews --jq '.[] | select(.id==<id>) | .node_id'`), then `gh api graphql -f query='mutation($id:ID!){addReaction(input:{subjectId:$id,content:THUMBS_UP}){reaction{content}}}' -f id=<node-id>`.
  - Skip targets where `viewerHasReacted` is already true for THUMBS_UP (`reactionGroups` in GraphQL, or the REST reactions list).
- Post every verdict as a PR review with the mapped Event, never a plain issue comment. A findings-free one-liner still goes as a review: `gh api repos/gnolang/gno/pulls/<number>/reviews -f event=<EVENT> -f body='...'`.
- After a successful post, the script writes GitHub URLs back into comment.md (`Posted: <review-url>` under the title, `[posted](<comment-url>)` on each anchor header). Commit and push the updated comment.md.
- Re-running the script on a draft carrying a `Posted:` line rewrites the posted review in place (body and `[posted]`-linked inline comments) instead of duplicating it. Anchors without a `[posted]` link abort: comments can't be added to an existing review. No `--approve` needed, the event doesn't change.
- If the author already has a pending (unsubmitted) review on the PR, the script folds the draft's comments into it and submits in place instead of creating a second review.

### Final check

Verify each line of the draft before handing it over:

1. Every `## <path>:<line>` header ends with its worktree `[↗](...)` link.
2. The Full review line is a `blob/` (not `tree/`) URL ending with `[↗](review_<model>_<reviewer>.md)`.
3. Body names at most three checks, each a runtime check the tests don't/can't cover (no static-analysis reasoning, no test-covered claim), none CI-visible, and neither recaps nor counts anchored findings.
4. No repro block whose output is only a passing run.
5. Every non-Warning inline comment opens with its severity band; Warnings open with the TL;DR directly. Every inline comment is at most 3 sentences, asks for a fix, a decision, or an answer, and carries no fix sentence whose remedy its problem statement already implies.
6. The whole draft conforms to `skills/writing-style.md`: Body goes straight to substance with no verdict restating the `Event:` line, no em-dashes, no parentheticals, no bold, no imported emphasis caps, problem-not-fix, and every named file, symbol, PR, issue, or package carries a link. Fix any deviation before handing over.
7. Open every link and read the lines it lands on. Each one contains the number, symbol, or behavior its anchor text claims, and external links resolve at the pinned ref.

Then two QA agents, re-run on every regeneration of comment.md:

- **Concision recheck.** One `Agent` (`subagent_type: general-purpose`): hand it the comment.md path, the worktree path, and the *Visible-text style* rules; ask only whether any Body line or inline comment can be shorter or clearer without dropping fact, stake, or fix, returning a per-section verdict with the proposed rewrite. Apply the rewrites that hold against the cited lines; discard the rest.
- **Citation audit.** One `Agent` (`subagent_type: general-purpose`): hand it the review file and comment.md paths and the worktree path; for every link in both, it fetches the target, reads the cited lines, and returns only the anchors whose lines do not contain the claim, plus any external link that does not resolve. The `Full review:` self-link 404s until the round is pushed; tell the agent to skip it. Fix each finding before handing over.
