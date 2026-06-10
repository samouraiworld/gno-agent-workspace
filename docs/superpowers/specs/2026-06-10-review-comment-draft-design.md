# Review skill: GitHub review draft (`comment.md`)

Date: 2026-06-10
Status: approved

## Goal

After a PR review, the AI drafts the actual GitHub review (verdict + inline comments) into a `comment.md` file. The user prunes it by hand, then explicitly approves the upload. Each inline comment carries proof the AI ran: repro command plus observed output.

## Flow

1. Review file written, index regenerated, committed and pushed (existing pre-authorized flow).
2. AI drafts `comment.md` next to the review file: `reviews/pr/<bucket>/<number>-<slug>/<n>-<sha>/comment.md`.
3. User reads and edits comment.md, deleting non-pertinent sections.
4. On explicit user approval ("post it" / "upload"), the AI runs `scripts/post-pr-review.py`, which parses comment.md and submits one GitHub review via `gh api repos/gnolang/gno/pulls/<n>/reviews`. Posting is never covered by the push pre-authorization.

## comment.md format

Markdown with anchors:

- Header line `Event: APPROVE | REQUEST_CHANGES | COMMENT`. NEEDS DISCUSSION and CLOSE map to COMMENT.
- `## Body` section: TL;DR + verdict sentence (3-5 lines max), link to the pushed review file on GitHub, `*(AI Agent)*` footer. Questions for author and findings without a file:line go here.
- One `## <path>:<line>` (or `## <path>:<start>-<end>`) section per finding with a file:line, all severities. Line numbers reference the PR head commit, side RIGHT.

## Inline comment rules

- 1-3 visible sentences, hard cap. No headers, no priority tags, no bold.
- Repro command + observed output collapsed in `<details><summary>repro</summary>`. Same repro rules as the review file: runnable from a fresh gnolang/gno clone, actually run by the AI, output included.
- Before drafting, attempt a repro for every Critical and Warning. Findings without a run proof are worded as observations, never "I ran X".
- Every comment (Body and each inline) ends with `*(AI Agent)*`.
- Link to the full review only when the details block is not enough.

## scripts/post-pr-review.py

- Parses comment.md (fence-aware: `##` inside code blocks is not a header).
- Pre-validates each anchor against `gh pr diff`: inline comments only attach to lines present in the diff (context or added, RIGHT side). Invalid anchors are reported with a hint to move them into Body; nothing is posted. `--skip-invalid` posts the valid ones only.
- Posts the review with `gh api`. `--repo` defaults to `gnolang/gno`. `--dry-run` prints the JSON payload.
- Starts with the standard NOT AUDITED disclaimer.
