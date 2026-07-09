# Review-all batch — status (started 2026-07-09)

Scope: 7 fresh reviews, no re-reviews. User confirmed "All 7". Model claude-opus-4-8, reviewer davd-gzl. Normal flow (not deep). Nothing posted.

## Set split (14 candidates → keep 7, drop 7)

DROP — dependabot (`app/dependabot`):
5869 5905 5914

DROP — `WIP`-titled:
4949 5223 5263 5721

KEEP (7, all fresh — absent from `reviews/pr/`):
5888 5890 5891 5892 5893 5911 5917

No re-reviews in this batch, so the head-unchanged, patch-id-equal base-only move, and already-APPROVED drop reasons do not apply.

## Final set

Base `origin/master` = `be3a5ade9`. Every worktree was created fresh and checked out at the PR head; each local head sha matches the GitHub `headRefOid`.

| PR | Author | Size | Head sha | Base | Worktree | Review dir |
|----|--------|------|----------|------|----------|------------|
| [5911](https://github.com/gnolang/gno/pull/5911) | notJoon | +15/-3, 2f | `6c20a2f7b` | master | `.worktrees/gno-review-5911` | `reviews/pr/5xxx/5911-enforce-max-slug-length/1-6c20a2f7b/` |
| [5917](https://github.com/gnolang/gno/pull/5917) | aeddi | +311/-0, 4f | `c8a5a2f2a` | master | `.worktrees/gno-review-5917` | `reviews/pr/5xxx/5917-govdao-test13-scripts/1-c8a5a2f2a/` |
| [5888](https://github.com/gnolang/gno/pull/5888) | moul | +940/-2, 12f | `b8818eb8e` | master | `.worktrees/gno-review-5888` | `reviews/pr/5xxx/5888-inert-package-storage-oracle/1-b8818eb8e/` |
| [5891](https://github.com/gnolang/gno/pull/5891) | jaekwon | +507/-22, 11f | `057894796` | master | `.worktrees/gno-review-5891` | `reviews/pr/5xxx/5891-split-mempackage-prod-test/1-057894796/` |
| [5892](https://github.com/gnolang/gno/pull/5892) | jaekwon | +218/-58, 31f | `d2f3d1337` | `pr1-mempackage-split` (PR 5891) | `.worktrees/gno-review-5892` | `reviews/pr/5xxx/5892-meter-preprocess-gas/1-d2f3d1337/` |
| [5893](https://github.com/gnolang/gno/pull/5893) | jaekwon | +829/-132, 43f | `131c5fccb` | master | `.worktrees/gno-review-5893` | `reviews/pr/5xxx/5893-deterministic-typecheck-verdict/1-131c5fccb/` |
| [5890](https://github.com/gnolang/gno/pull/5890) | jaekwon | +2632/-232, 50f | `8a115c8ca` | master | `.worktrees/gno-review-5890` | `reviews/pr/5xxx/5890-realm-sub-subrealm-identities/1-8a115c8ca/` |

Stacking: 5892 targets 5891's branch, so its `gh pr diff` carries only pr2's own changes. 5888, 5890, 5891, and 5893 each diff against master.

## Dispatch

One `general-purpose` agent per PR, all in one message, per `skills/review.md` *Parallel dispatch*. The parent created every worktree and ran `gh pr checkout`; subagents never do. Subagents write `review_claude-opus-4-8_davd-gzl.md` and `comment_claude-opus-4-8.md`, and do not commit, push, regenerate indexes, or post.

## Results — all 7 returned

| PR | Verdict | Inline comments | Headline |
|----|---------|-----------------|----------|
| 5911 | APPROVE | 1 | `maxSlugLen = 128` has no unit comment though the limit is applied in bytes |
| 5917 | APPROVE | 2 + Body question | stale comment, missing `lock-transfer` command; sibling singular scripts emit a stale 3-arg builder call |
| 5888 | REQUEST_CHANGES | 7 | `EnablePackage` skips storage deposit, skips the gnomod gates, and runs init as the approver |
| 5891 | APPROVE | 0 | mempackage split verified lossless and deterministic |
| 5892 | APPROVE | 0 | gas charge revert-proofed at 1250 × 84 bytes |
| 5893 | APPROVE | 0 | `GoVersion` pin revert-proofed; error text off the hashed path |
| 5890 | APPROVE | 0 | banker caller-drain fix revert-proofed; sub-minting unforgeable |

5888's three Warnings were independently re-verified by the parent against `gno.land/pkg/sdk/vm/keeper.go` at `b8818eb8e`: `processStorageDeposit` is called at `:783`, `:1014`, `:1229` but never inside `EnablePackage` (`:803-858`); the release path panics at `:1917` when `rlm.Storage < released` and divides by `rlm.Storage` at `:1935`; `EnablePackage` runs no `gnomod.ParseMemPackage` gate; `OriginCaller` is set to `msg.Approver.Bech32()` at `:831`.

All seven worktrees are clean, with no scratch files left behind.

## Finalize (parent)

1. ~~Verify each PR wrote both a review file and a comment draft.~~ Done, all 14 files present.
2. ~~Sweep stray scratch test files from the worktrees.~~ Done, all clean.
3. `./scripts/build-indexes.sh` once.
4. `git add reviews/ docs/glossary.md && git commit -m "review: review-all batch 2026-07-09" && git push`
5. Hand back a link to each PR's `comment_claude-opus-4-8.md`. Posting waits for the literal `post`.

Nothing has been posted to GitHub.

## Resume

If the session dies mid-batch: check which review dirs above hold both `review_*.md` and `comment_*.md`, and re-dispatch only the incomplete ones. The worktrees already exist at the shas in the table; do not re-create them.
