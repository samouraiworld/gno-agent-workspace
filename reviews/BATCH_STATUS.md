# Review-all batch — status (started 2026-06-27)

Scope: `review all` + re-review every open PR whose head moved with **real PR-content change** since last review (base-only master merges excluded via patch-id). Confirmed scope: 1 new + 32 re-reviews = 33. Deep mode on 9 complex PRs.

All 33 worktrees were pre-created at the current PR head under `.worktrees/gno-review-<num>` (detached). **Re-verify heads before resuming** — PRs may have moved again. Nothing here is committed/posted unless noted.

## DONE this run (drafts written, not committed yet at time of writing, not posted)
| PR | mode | round dir | verdict |
|----|------|-----------|---------|
| 5653 | deep | reviews/pr/5xxx/5653-test-13-hardfork-rc/2-f45cc5c88/ | NEEDS DISCUSSION |
| 5598 | deep | reviews/pr/5xxx/5598-examples-commondao-fixes/3-0b6b302d2/ | REQUEST CHANGES |
| 5069 | normal | reviews/pr/5xxx/5069-grc20reg-pagination/2-0a87d3d9d/ | APPROVE |
| 5117 | normal | reviews/pr/5xxx/5117-multisig-docs-default-sorting/2-bff5572db/ | APPROVE |
| 5504 | normal | reviews/pr/5xxx/5504-royalty-bps-eip2981/2-6d6ab81a5/ | APPROVE |

## PENDING (28) — dispatch one agent per PR per skills/review.md (re-review flow; new=first review)
Cols: PR · mode · review dir · next round · last-reviewed sha · current head (verify!).

### Deep (7)
- 5835 · DEEP · reviews/pr/5xxx/5835-audit-pattern-harness · r3 · 34ac1e7cd · e8281bcbe  ← USER TOP PRIORITY
- 5728 · DEEP · reviews/pr/5xxx/5728-grc721-private-ledger-teller · r2 · c463023cb · eac94f444
- 5737 · DEEP · reviews/pr/5xxx/5737-defer-nil-receiver-panic · r2 · 4c57c37e4 · ccb6c94ad
- 5431 · DEEP · reviews/pr/5xxx/5431-simulate-immutable-snapshot · r4 · b84ee8bdb · 4b8fadb23
- 5576 · DEEP · reviews/pr/5xxx/5576-deterministic-testing-b · r2 · ca2efcf92 · 79c02d050
- 5739 · DEEP · reviews/pr/5xxx/5739-preserve-embedded-alias-name · r2 · 155f1a7 · 216e8aee3
- 5813 · DEEP · reviews/pr/5xxx/5813-recycle-blocks-machine-pool · r2 · 697316b4c · becc5fa87

### Normal re-review (20)
- 4885 · reviews/pr/4xxx/4885-correctly-reuse-count-string · r2 · e9199dc9e · ff05ec11f
- 5016 · reviews/pr/5xxx/5016-rdocs-additions · r3 · 877a57bd8 · 12a7129f3 (docs)
- 5134 · reviews/pr/5xxx/5134-gnokey-list-multisig-pubkey · r2 · 250e46eeb · bd65207ac
- 5406 · reviews/pr/5xxx/5406-comment-gas-metering · r2 · bf988dd · a3b5a3463
- 5522 · reviews/pr/5xxx/5522-render-oldest-newest · r2 · 4f7ef8e · 79e35f08d
- 5559 · reviews/pr/5xxx/5559-private-realm-unit-test · r2 · 2b7114f · 54b89f90d
- 5585 · reviews/pr/5xxx/5585-heading-anchor-clickable · r3 · 5ae68a81 · 19d7a8245
- 5645 · reviews/pr/5xxx/5645-namereg-realm-fixes · r2 · 6e569d9 · b8dff8e6a
- 5646 · reviews/pr/5xxx/5646-bigint-bigdec-compare-gas · r2 · 9a51c19 · 6076e2f11
- 5654 · reviews/pr/5xxx/5654-validators-v3-allow-list · r2 · f59deca8 · c4f35e987
- 5676 · reviews/pr/5xxx/5676-bytes-cut-clone-helpers · r2 · 63c55f963 · 1c5f15202
- 5692 · reviews/pr/5xxx/5692-gnodev-default-test-account · r2 · 6aecc94 · 6381a1295
- 5709 · reviews/pr/5xxx/5709-ledger-stored-pubkey-check · r2 · 37db202e (old sha GC'd, may be unfetchable; diff vs merge-base) · 752e8c272
- 5722 · reviews/pr/5xxx/5722-typed-nil-func-preprocess-qfuncs · r2 · 378a51ec · a6592fc23
- 5732 · reviews/pr/5xxx/5732-typedruntimeerror-runtime-errors · r2 · d716c5286 · 6a00ff7f4
- 5763 · reviews/pr/5xxx/5763-unsealed-declaredtype-mutual-recursion · r2 · 61fc396e4 · 093c32be0
- 5766 · reviews/pr/5xxx/5766-type-switch-sole-nil · r2 · 65e2441 · d69d7b33f
- 5830 · reviews/pr/5xxx/5830-claude-md-local-agents-md · r2 · 127513b · cf66e0298
- 5756 · reviews/pr/5xxx/5756-add-memberstorage-subpackage · r2 · d940e681b · 8b3329332
- 5840 · reviews/pr/5xxx/5840-vesting-account-poc · r2 · 3e4fca768 · f9b7547f6

### New (1)
- 5861 · NEW (first review, round 1) · reviews/pr/5xxx/5861-<slug> · head 18955cb84 · feat(tm2): implement address book

## Excluded as base-only (master-merge only, patch-id equal) — do NOT re-review
4908, 5020, 5123, 5641, 5754, 5790, 5821, 5822

## Resume procedure
1. `git -C gno fetch origin master`. For each pending PR, re-check current head vs the "current head" above; if it moved again, re-run that PR's worktree to the new head (`.worktrees/gno-review-<num>`).
2. Dispatch one `general-purpose` Agent per PR with the per-PR prompt (re-review flow; deep PRs get the deep-mode block). Worktrees already exist — agents must NOT run `git worktree add` / `gh pr checkout`.
3. After all return: `./scripts/build-indexes.sh`, then one commit `review: review-all batch (cont.)` + push.
4. Posting still waits for the literal `post` per PR.

## Per-PR agent prompt template (re-review)
> Run skills/review.md on PR <num> (<url>). RE-REVIEW, head advanced. Worktree .worktrees/gno-review-<num> already at head <new> — do NOT worktree add / gh pr checkout. Prior rounds under <dir>, last sha <old>; patch-ids confirm content changed, do a full round focused on <old>..HEAD. Carry valid findings verbatim, drop fixed, add delta findings. Write round <next> at <dir>/<next>-<new>/ : review_claude-opus-4-8_davd-gzl.md + comment_claude-opus-4-8.md. Model claude-opus-4-8, reviewer davd-gzl. Do NOT commit/push/build-indexes/post. Report path + verdict + one-paragraph summary.
> DEEP add: follow the Deep mode section; apply red-team/blue-team/correctness lenses as distinct in-context passes, plus critic + claim-verification gate; do not spawn sub-agents.
