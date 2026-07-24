# Review: PR [#5996](https://github.com/gnolang/gno/pull/5996)
Event: COMMENT

## Body
The race is real on the production wiring, not theoretical. The query connection runs on its own `queryMtx` ([`proxy/client.go:26`](https://github.com/gnolang/gno/blob/4b521105b/tm2/pkg/bft/proxy/client.go#L26)), and a query that omits its height reads `LastCommitID().Version` through `app.LastBlockHeight()` ([`baseapp.go:498`](https://github.com/gnolang/gno/blob/4b521105b/tm2/pkg/sdk/baseapp.go#L498), [`:525`](https://github.com/gnolang/gno/blob/4b521105b/tm2/pkg/sdk/baseapp.go#L525)) concurrent with `Commit` writing it, with nothing serialising the two. The guard is complete: grepping the field leaves only the two accessor bodies as raw accesses. `lastCommitIDMu` is a leaf lock, so no inversion against the store's other mutexes or bptree's `pruneMu`. Reverting the accessors to raw access makes `TestLastCommitIDConcurrentWithCommit` fail under `-race` and pass without it, which is the right shape for a data-race guard.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5996-guard-lastcommitid-query/1-4b521105b/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## tm2/pkg/store/rootmulti/store.go:270
Suggestion, not a blocker: this guard sits on `Commit`, which runs once per block, and the PR asserts the added locking is free without a number. A write lock around a struct assignment is nanoseconds and cannot show up against a block commit, so this is almost certainly true, but a one-line benchmark would make it a measured claim rather than an asserted one.
