# PR [#5875](https://github.com/gnolang/gno/pull/5875): chore(docs): Run make embed_markdown

URL: https://github.com/gnolang/gno/pull/5875
Author: jefft0 | Base: master | Files: 10 | +10 -10
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 90d02a8 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5875 90d02a8`

**TL;DR:** Runs the `make -C examples embed_markdown` tooling target over the examples tree. The only effect is adding a missing trailing newline to 10 markdown files (mostly READMEs). No embedded code, no prose, no `.gno` code changes.

**Verdict: APPROVE** — mechanical newline normalization, faithful output of a documented make target, no open concerns.

## Summary
Ten markdown files under `examples/` lacked a trailing newline. Running the repo's own `embed_markdown` target (`embedmd -w` over every `*.md`) normalizes them by appending a single `\n`. Each hunk is content-identical to master except for the removed `\ No newline at end of file` marker. No embedded code blocks were rewritten, so nothing but end-of-file whitespace moves.

## Fix
Each of the 10 files gains one trailing newline at end of file; nothing else changes. Verified byte-identical content via `git diff --word-diff` (only newline markers differ), and confirmed completeness by re-running the target on the branch and observing a clean tree.

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
None.

## Missing Tests
None. Tooling-generated whitespace normalization; no behavior to test.

## Suggestions
None.

## Open questions
None.

---

Verified on 90d02a8: re-running `make -C examples embed_markdown` on the PR branch leaves `git status` clean, so the PR is the complete and idempotent output of the documented target and embedmd rewrote no embedded code. Content is byte-identical to master except the added trailing newline (`git diff --word-diff` shows only newline markers changing). CI green.
