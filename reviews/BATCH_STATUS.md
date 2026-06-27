# Review-all batch — status (started 2026-06-27)

Scope: `review all` + re-review every open PR whose head moved with **real PR-content change** since last review (base-only master merges excluded via patch-id). **Exclude any PR davd-gzl already APPROVED on GitHub** (no point re-reviewing approved work).

All worktrees pre-created at the current PR head under `.worktrees/gno-review-<num>` (detached). **Re-verify heads before resuming** — PRs may have moved again. Nothing posted.

## DONE this run (drafts written + pushed, not posted)
| PR | verdict | round dir |
|----|---------|-----------|
| 5653 | NEEDS DISCUSSION | reviews/pr/5xxx/5653-test-13-hardfork-rc/2-f45cc5c88/ |
| 5598 | REQUEST CHANGES | reviews/pr/5xxx/5598-examples-commondao-fixes/3-0b6b302d2/ |
| 5069 | APPROVE | reviews/pr/5xxx/5069-grc20reg-pagination/2-0a87d3d9d/ |
| 5117 | APPROVE (redundant — already approved on GitHub) | reviews/pr/5xxx/5117-multisig-docs-default-sorting/2-bff5572db/ |
| 5504 | APPROVE (redundant — already approved on GitHub) | reviews/pr/5xxx/5504-royalty-bps-eip2981/2-6d6ab81a5/ |

## EXCLUDED — davd-gzl already APPROVED on GitHub (do NOT re-review)
5117, 5134, 5431, 5504, 5522, 5559, 5645, 5692, 5722, 5766, 5830

## EXCLUDED — base-only master-merge (patch-id equal, no content change)
4908, 5020, 5123, 5641, 5754, 5790, 5821, 5822

## PENDING (19) — important first. One agent per PR per skills/review.md (re-review; new=first review)
Cols: PR · mode · review dir · next round · last-reviewed sha · current head (verify!) · note

### Deep / important (6)
- 5835 · DEEP · reviews/pr/5xxx/5835-audit-pattern-harness · r3 · 34ac1e7cd · e8281bcbe · **USER TOP PRIORITY**
- 5737 · DEEP · reviews/pr/5xxx/5737-defer-nil-receiver-panic · r2 · 4c57c37e4 · ccb6c94ad · davd-gzl had CHANGES_REQUESTED
- 5739 · DEEP · reviews/pr/5xxx/5739-preserve-embedded-alias-name · r2 · 155f1a7 · 216e8aee3 · CHANGES_REQUESTED
- 5728 · DEEP · reviews/pr/5xxx/5728-grc721-private-ledger-teller · r2 · c463023cb · eac94f444
- 5576 · DEEP · reviews/pr/5xxx/5576-deterministic-testing-b · r2 · ca2efcf92 · 79c02d050
- 5813 · DEEP · reviews/pr/5xxx/5813-recycle-blocks-machine-pool · r2 · 697316b4c · becc5fa87

### Normal re-review (12)
- 5763 · reviews/pr/5xxx/5763-unsealed-declaredtype-mutual-recursion · r2 · 61fc396e4 · 093c32be0 · CHANGES_REQUESTED
- 4885 · reviews/pr/4xxx/4885-correctly-reuse-count-string · r2 · e9199dc9e · ff05ec11f
- 5016 · reviews/pr/5xxx/5016-rdocs-additions · r3 · 877a57bd8 · 12a7129f3 · docs
- 5406 · reviews/pr/5xxx/5406-comment-gas-metering · r2 · bf988dd · a3b5a3463
- 5585 · reviews/pr/5xxx/5585-heading-anchor-clickable · r3 · 5ae68a81 · 19d7a8245
- 5646 · reviews/pr/5xxx/5646-bigint-bigdec-compare-gas · r2 · 9a51c19 · 6076e2f11
- 5654 · reviews/pr/5xxx/5654-validators-v3-allow-list · r2 · f59deca8 · c4f35e987
- 5676 · reviews/pr/5xxx/5676-bytes-cut-clone-helpers · r2 · 63c55f963 · 1c5f15202
- 5709 · reviews/pr/5xxx/5709-ledger-stored-pubkey-check · r2 · 37db202e (old sha may be GC'd; diff vs merge-base) · 752e8c272
- 5732 · reviews/pr/5xxx/5732-typedruntimeerror-runtime-errors · r2 · d716c5286 · 6a00ff7f4
- 5756 · reviews/pr/5xxx/5756-add-memberstorage-subpackage · r2 · d940e681b · 8b3329332
- 5840 · reviews/pr/5xxx/5840-vesting-account-poc · r2 · 3e4fca768 · f9b7547f6

### New (1)
- 5861 · NEW (first review, round 1) · reviews/pr/5xxx/5861-<slug> · head 18955cb84 · feat(tm2): implement address book

## Resume procedure
1. `git -C gno fetch origin master`. Per pending PR, re-check current head vs above; if moved, re-cut its worktree to the new head.
2. Dispatch one `general-purpose` Agent per PR (re-review flow; deep block for the 6 deep). Worktrees exist — agents must NOT `git worktree add` / `gh pr checkout`.
3. After all return: `./scripts/build-indexes.sh`, then one commit `review: review-all batch (cont.)` + push.
4. Posting waits for the literal `post` per PR.

## Per-PR agent prompt template (re-review)
> Run skills/review.md on PR <num> (<url>). RE-REVIEW, head advanced. Worktree .worktrees/gno-review-<num> already at head <new> — do NOT worktree add / gh pr checkout. Prior rounds under <dir>, last sha <old>; patch-ids confirm content changed, full round focused on <old>..HEAD. Carry valid findings verbatim, drop fixed, add delta findings. Write round <next> at <dir>/<next>-<new>/ : review_claude-opus-4-8_davd-gzl.md + comment_claude-opus-4-8.md. Model claude-opus-4-8, reviewer davd-gzl. Do NOT commit/push/build-indexes/post. Report path + verdict + one-paragraph summary.
> DEEP add: follow the Deep mode section; red-team/blue-team/correctness lenses as distinct in-context passes, plus critic + claim-verification gate; do not spawn sub-agents.
