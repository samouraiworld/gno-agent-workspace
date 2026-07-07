# Review-all batch — status (started 2026-07-07)

Scope chosen by user: **3 fresh + refresh re-reviews**. Fresh = open non-draft PRs never reviewed. Re-review = already-reviewed open PRs whose head moved with real PR-content change since last review (base-only master merges dropped via patch-id, already-APPROVED-by-davd-gzl dropped).

Model: claude-opus-4-8. Reviewer: davd-gzl. Nothing posted. Normal (non-deep) flow for all.

## Dropped
- 81 head-unchanged since last review.
- 12 already APPROVED by davd-gzl on GitHub: 5048 5068 5134 5361 5431 5559 5645 5757 5766 5775 5825 5896.
- 4 base-only master-merge (patch-id equal): 4908 5020 5123 5754.
- WIP: 5721 5376 5263 5223 4949. Dependabot: 5905 5904 5869.

## Final set — 27 PRs (all worktrees pre-created at head; agents must NOT worktree-add / gh pr checkout)

### Fresh (3) — first review, round 1
| PR | head | worktree | dir |
|----|------|----------|-----|
| 5885 | de0910ee2 | .worktrees/gno-review-5885 | reviews/pr/5xxx/5885-code-submission-policy-param |
| 5907 | fba248c95 | .worktrees/gno-review-5907 | reviews/pr/5xxx/5907-prevent-token-path-overwrite |
| 5908 | 288cdb044 | .worktrees/gno-review-5908 | reviews/pr/5xxx/5908-decouple-token-id-symbol |

### Re-review (24)
| PR | head | last sha | next round | worktree | dir |
|----|------|----------|-----------|----------|-----|
| 5016 | 73942a7ab | 877a57bd8 | r3 | .worktrees/gno-review-5016 | reviews/pr/5xxx/5016-rdocs-additions |
| 5217 | c762e0be5 | 16b633c   | r2 | .worktrees/gno-review-5217 | reviews/pr/5xxx/5217-type-switch-gas-metering |
| 5258 | c8b2138a8 | 554a7546  | r2 | .worktrees/gno-review-5258 | reviews/pr/5xxx/5258-validate-ws-origin |
| 5406 | a3b5a3463 | bf988dd   | r2 | .worktrees/gno-review-5406 | reviews/pr/5xxx/5406-comment-gas-metering |
| 5421 | 13124c534 | 339469041 | r2 | .worktrees/gno-review-5421 | reviews/pr/5xxx/5421-builtin-playground-2 |
| 5531 | 7199a6789 | 3c8f3dbab (GC'd, diff vs merge-base) | r2 | .worktrees/gno-review-5531 | reviews/pr/5xxx/5531-ci-release-build-cache |
| 5585 | f05026d1a | 5ae68a81  | r3 | .worktrees/gno-review-5585 | reviews/pr/5xxx/5585-heading-anchor-clickable |
| 5598 | 747610fee | 0b6b302d2 | r4 | .worktrees/gno-review-5598 | reviews/pr/5xxx/5598-examples-commondao-fixes |
| 5646 | 6076e2f11 | 9a51c19   | r2 | .worktrees/gno-review-5646 | reviews/pr/5xxx/5646-bigint-bigdec-compare-gas |
| 5654 | c4f35e987 | f59deca8  | r2 | .worktrees/gno-review-5654 | reviews/pr/5xxx/5654-validators-v3-allow-list |
| 5676 | 51b992076 | 63c55f963 | r2 | .worktrees/gno-review-5676 | reviews/pr/5xxx/5676-bytes-cut-clone-helpers |
| 5679 | 30af6c37f | 3ac5cda   | r2 | .worktrees/gno-review-5679 | reviews/pr/5xxx/5679-encoding-ascii85-pem |
| 5709 | 752e8c272 | 37db202e (GC'd, diff vs merge-base) | r2 | .worktrees/gno-review-5709 | reviews/pr/5xxx/5709-ledger-stored-pubkey-check |
| 5732 | b6b3e5d42 | d716c5286 | r2 | .worktrees/gno-review-5732 | reviews/pr/5xxx/5732-typedruntimeerror-runtime-errors |
| 5737 | 0019dc436 | ccb6c94ad | r3 | .worktrees/gno-review-5737 | reviews/pr/5xxx/5737-defer-nil-receiver-panic |
| 5741 | 84c1c30dd | a6dc98e3b | r2 | .worktrees/gno-review-5741 | reviews/pr/5xxx/5741-float-const-signed-zero |
| 5749 | 16cf24a2d | 1d2a53f5f | r2 | .worktrees/gno-review-5749 | reviews/pr/5xxx/5749-strings-split-invalid-utf8 |
| 5756 | f9121247a | d940e681b | r2 | .worktrees/gno-review-5756 | reviews/pr/5xxx/5756-add-memberstorage-subpackage |
| 5826 | c1942b74c | 088ce87   | r2 | .worktrees/gno-review-5826 | reviews/pr/5xxx/5826-typecheck-fanout-dos |
| 5840 | 9da3635c4 | 3e4fca768 | r2 | .worktrees/gno-review-5840 | reviews/pr/5xxx/5840-vesting-account-poc |
| 5843 | 03d2585bb | fbeeb60fa | r2 | .worktrees/gno-review-5843 | reviews/pr/5xxx/5843-tmkms-quickstart-secure |
| 5864 | 2c6396d90 | 662cbc5ba | r2 | .worktrees/gno-review-5864-h | reviews/pr/5xxx/5864-fold-negzero-float-args |
| 5867 | 07d4ad373 | 3c7de91d0 | r2 | .worktrees/gno-review-5867 | reviews/pr/5xxx/5867-bigdec-apd-to-rat |
| 5873 | eb829bec3 | 66552ff7a | r2 | .worktrees/gno-review-5873-h | reviews/pr/5xxx/5873-rewrite-gnokey-guide-reference |

## Resume / finalize
1. Dispatch one general-purpose Agent per PR (normal re-review flow; NEW for the 3 fresh). Worktrees exist; agents must NOT worktree-add / gh pr checkout / commit / push / build-indexes / post.
2. After all return: `./scripts/build-indexes.sh`, then one commit `review: review-all batch 2026-07-07` + push.
3. Posting waits for the literal `post` per PR.
