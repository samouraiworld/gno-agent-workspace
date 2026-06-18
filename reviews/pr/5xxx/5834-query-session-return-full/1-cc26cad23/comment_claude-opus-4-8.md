# Review: PR #5834
Event: APPROVE

## Body
Looks good. Verified on cc26cad23: the signing path is behavior-preserving. `SignTx` now [reads `.BaseSessionAccount.BaseAccount`](https://github.com/gnolang/gno/blob/cc26cad23/gno.land/pkg/gnoclient/client_txs.go#L414-L418) · [↗](../../../../../.worktrees/gno-review-5834/gno.land/pkg/gnoclient/client_txs.go#L414), the exact field the old `QuerySessionAccount` used to return, so the account number and sequence it signs with are unchanged.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5834-query-session-return-full/1-cc26cad23/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)
