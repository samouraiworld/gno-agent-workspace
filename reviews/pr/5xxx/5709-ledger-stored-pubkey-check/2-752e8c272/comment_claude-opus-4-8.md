# Review: PR [#5709](https://github.com/gnolang/gno/pull/5709)
Event: APPROVE

## Body
CI red is codecov-upload only: the `Go test` step is green in every failing job, and each one fails at `Upload coverage to Codecov`. Not a code problem.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5709-ledger-stored-pubkey-check/2-752e8c272/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## tm2/pkg/crypto/keys/keybase.go:255 [↗](../../../../../.worktrees/gno-review-5709/tm2/pkg/crypto/keys/keybase.go#L255)
The stored-vs-live check here, [`validateKey`](https://github.com/gnolang/gno/blob/752e8c272/tm2/pkg/crypto/ledger/ledger_secp256k1.go#L188) inside [`sign()`](https://github.com/gnolang/gno/blob/752e8c272/tm2/pkg/crypto/ledger/ledger_secp256k1.go#L207), and the post-sign [`VerifyBytes`](https://github.com/gnolang/gno/blob/752e8c272/tm2/pkg/crypto/keys/keybase.go#L273) guard three different windows on the Ledger sign path but read like duplicates. Note what each one protects, so a later refactor doesn't drop one as redundant.

## tm2/pkg/crypto/keys/keybase_ledger_test.go:51-52 [↗](../../../../../.worktrees/gno-review-5709/tm2/pkg/crypto/keys/keybase_ledger_test.go#L51)
The [`Discover`](https://github.com/gnolang/gno/blob/752e8c272/tm2/pkg/crypto/keys/keybase_ledger_test.go#L52) closure captures the outer `device` variable, so reassigning `device` between create and sign makes `Discover()` return a different device. The whole test turns on this, yet no comment says so.
