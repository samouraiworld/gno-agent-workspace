# Review-all batch — status (started 2026-07-10)

Scope: 7 fresh reviews, no re-reviews. User confirmed "All 7". Model claude-opus-4-8, reviewer davd-gzl. Normal flow (not deep). Nothing posted.

## Set split (14 candidates → keep 7, drop 7)

DROP — dependabot (`app/dependabot`):
5869 5905 5914

DROP — `WIP`-titled:
4949 5223 5263 5922

KEEP (7, all fresh — absent from `reviews/pr/`):
5721 5916 5920 5921 5924 5928 5933

No re-reviews in this batch, so the head-unchanged, patch-id-equal base-only move, and already-APPROVED drop reasons do not apply.

5721 was dropped as `WIP`-titled in the 2026-07-09 batch; its title no longer carries the prefix, so it is in scope now.

## Final set

Every worktree was created fresh from `origin/master` and checked out at the PR head.

| PR | Author | Size | Head sha | Worktree | Review dir |
|----|--------|------|----------|----------|------------|
| [5933](https://github.com/gnolang/gno/pull/5933) | aeddi | +300/-69, 25f | `61f762f9b` | `.worktrees/gno-review-5933` | `reviews/pr/5xxx/5933-honor-explicit-zero-config/1-61f762f9b/` |
| [5928](https://github.com/gnolang/gno/pull/5928) | aeddi | +92/-20, 4f | `cf7fb56bd` | `.worktrees/gno-review-5928` | `reviews/pr/5xxx/5928-fork-valoper-seed-payable/1-cf7fb56bd/` |
| [5924](https://github.com/gnolang/gno/pull/5924) | aeddi | +83/-0, 3f | `f79f1ae62` | `.worktrees/gno-review-5924` | `reviews/pr/5xxx/5924-surface-genesis-replay-failures/1-f79f1ae62/` |
| [5921](https://github.com/gnolang/gno/pull/5921) | ltzmaxwell | +168/-0, 4f | `6bcdde7e3` | `.worktrees/gno-review-5921` | `reviews/pr/5xxx/5921-reject-generics-syntax/1-6bcdde7e3/` |
| [5920](https://github.com/gnolang/gno/pull/5920) | ltzmaxwell | +21/-1, 2f | `3732be8d3` | `.worktrees/gno-review-5920` | `reviews/pr/5xxx/5920-typecheck-blank-func-decls/1-3732be8d3/` |
| [5916](https://github.com/gnolang/gno/pull/5916) | moul | +69/-15, 2f | `7e6955997` | `.worktrees/gno-review-5916` | `reviews/pr/5xxx/5916-symmetric-gas-price-decay/1-7e6955997/` |
| [5721](https://github.com/gnolang/gno/pull/5721) | ltzmaxwell | +612/-218, 12f | `16d5227b9` | `.worktrees/gno-review-5721` | `reviews/pr/5xxx/5721-shallowest-match-embedded-lookup/1-16d5227b9/` |

## Dispatch

One `general-purpose` agent per PR, all in one message, per `skills/review.md` *Parallel dispatch*. The parent created every worktree and ran `gh pr checkout`; subagents never do. Subagents write `review_claude-opus-4-8_davd-gzl.md` and `comment_claude-opus-4-8.md`, and do not commit, push, regenerate indexes, or post.

## Results

| PR | Verdict | Inline comments | Headline |
|----|---------|-----------------|----------|
| 5933 | pending | | |
| 5928 | pending | | |
| 5924 | pending | | |
| 5921 | pending | | |
| 5920 | pending | | |
| 5916 | pending | | |
| 5721 | pending | | |

## Finalize (parent)

1. Verify each PR wrote both a review file and a comment draft.
2. Sweep stray scratch test files from the worktrees.
3. `./scripts/build-indexes.sh` once.
4. `git add reviews/ docs/glossary.md && git commit -m "review: review-all batch 2026-07-10" && git push`
5. Hand back a link to each PR's `comment_claude-opus-4-8.md`. Posting waits for the literal `post`.

Nothing has been posted to GitHub.

## Resume

If the session dies mid-batch: check which review dirs above hold both `review_*.md` and `comment_*.md`, and re-dispatch only the incomplete ones. The worktrees already exist at the shas in the table; do not re-create them.
