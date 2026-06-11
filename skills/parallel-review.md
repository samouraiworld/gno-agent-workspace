---
name: gno-parallel-review
description: Deeper multi-angle review of a single Gno PR. Dispatches red-team / blue-team / correctness agents in parallel, then a single parallel critic pass with a hard severity bar. Use when a PR warrants more scrutiny than the single-reviewer `skills/review.md` (large diffs, security-sensitive, consensus-relevant, prior review felt thin).
argument-hint: <pr-number>
---

# Gno PR Parallel Review

Multi-angle adversarial review of ONE PR. Different from `skills/review.md`:

- `skills/review.md` runs ONE reviewer per PR and parallelizes ACROSS PRs.
- This skill runs MANY reviewers on ONE PR (different lenses), then a single parallel critic pass.

**Always optimize for the human reader.** Same artifact discipline as `skills/review.md` — verdict first, concrete consequences, clickable references, self-sufficient files.

**Input:** `$ARGUMENTS` — single PR number or URL.

## Workflow

### 1. Fetch & set up worktree

Follow the same worktree procedure as `skills/review.md` (fetch master, create `.worktrees/gno-review-<number>`, `gh pr checkout` inside it). Then collect:

- PR metadata (`gh pr view ... --json title,body,author,baseRefName,headRefName,files,additions,deletions,commits`)
- Full diff (`gh pr diff <number> -R gnolang/gno`)
- All comments and review threads
- CI status (`gh pr checks`)
- Prior reviews under `reviews/pr/<thousand>xxx/<number>-*/` if any

### 2. Dispatch parallel reviewer agents

ONE message, multiple `Agent` tool calls (so they run concurrently). Default three angles; add more for large PRs.

Each agent prompt is self-contained. Each agent gets:
- worktree path, PR number, diff path, prior-review paths
- a narrow lens (see below)
- format: severity-grouped findings (Blocking / Major / Minor / Nit) with file:line citations

**Red team — adversarial.** Find bugs, broken invariants, security holes, edge cases, gas / determinism issues, missing input validation, footguns for downstream consumers.

**Blue team — defensive.** Find missing tests, undocumented invariants, hardening gaps, ergonomics that invite misuse, migration / rollback risks.

**Correctness — spec vs. implementation.** Does the code do what the PR description and linked issue claim? Find scope drift, silent behavior changes, contract mismatches.

Optional add-ons for big PRs: **perf**, **docs**, **consensus impact**, **API surface**.

### 3. Synthesize

Read all agent reports. Dedupe overlaps. Re-rank by severity. Verify each finding against the actual file before including — never trust a finding from memory or an agent summary alone.

Write the draft review to:
`reviews/pr/<thousand>xxx/<number>-<slug>/<n>-<short-commit>/claude-opus-4-7_davd-gzl.md`

(Same path convention as `skills/review.md`.)

### 4. Critic pass (parallel, single round)

Dispatch **2-3 critic agents in parallel** (one message, multiple `Agent` calls) with the synthesized draft + diff + worktree path. Sequential critic loops regress past 1-2 iterations on judgment tasks ([Huang 2024](https://arxiv.org/abs/2310.01798)); production AI-review systems converged on parallel specialists + single merge with no looping ([Cloudflare](https://blog.cloudflare.com/ai-code-review/)). One round is enough — don't loop.

Each critic gets a distinct lens. Suggested set:

- **Verdict-check** — does the verdict match the findings? Is anything Blocking misclassified as Warning, or vice versa?
- **Missing-blocking** — what Blocking/Major issue is absent from the draft?
- **Severity-calibration** — which findings are over- or under-graded by one band?

Each critic prompt MUST set a hard bar: return ONLY findings that (a) flip the verdict, (b) raise an existing finding's severity by ≥1 band, or (c) add a Blocking/Major issue absent from the draft. Everything sub-bar is dropped at the source, not surfaced. If nothing qualifies, return exactly `NO_MATERIAL_FINDINGS`. Avoid open-ended "what's wrong?" / "what's missing?" framings — they are the canonical inflation trigger.

After all critics return:

1. Dedupe across critics.
2. Re-read each cited `file:line` in the worktree. Drop any that don't hold.
3. Revise the draft with what survives. No second pass — at this severity bar, a clean parallel pass is the convergence signal.

### 5. Output and index

- Final review at the path above.
- Update the reviews index: `./scripts/build-indexes.sh`.
- Commit to THIS repo only:
  ```
  git add reviews/ && git commit -m "review: PR #<number> (parallel)"
  ```
- Push only with explicit user approval.

## Constraints

- One PR per invocation. For multi-PR runs, use `skills/review.md`.
- Never push to gnolang/gno.
- Never modify the gno submodule in-place — all work in `.worktrees/gno-review-<number>`.
- One critic round only — do not loop, even if findings still surface. The hard bar in §4 is the convergence signal.
- Verify every finding against source. Re-read files; do not trust summaries.
