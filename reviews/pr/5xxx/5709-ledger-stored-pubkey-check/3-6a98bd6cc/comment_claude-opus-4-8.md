# Review: PR [#5709](https://github.com/gnolang/gno/pull/5709)
Event: APPROVE
Status: not posted. Round re-anchored to 6a98bd6cc. On `post as an AI` the Body leads with `[AI review, opus 4.8] (not manually verified)`, then `Status: APPROVE`.

## Body
Looks good.

## tm2/pkg/crypto/keys/keybase.go:255 [gh](https://github.com/gnolang/gno/blob/6a98bd6cc/tm2/pkg/crypto/keys/keybase.go#L255) · [↗](../../../../../.worktrees/gno-review-5709/tm2/pkg/crypto/keys/keybase.go#L255)
The stored-vs-live check here, [`validateKey`](https://github.com/gnolang/gno/blob/6a98bd6cc/tm2/pkg/crypto/ledger/ledger_secp256k1.go#L192) inside [`sign()`](https://github.com/gnolang/gno/blob/6a98bd6cc/tm2/pkg/crypto/ledger/ledger_secp256k1.go#L211), and the post-sign [`VerifyBytes`](https://github.com/gnolang/gno/blob/6a98bd6cc/tm2/pkg/crypto/keys/keybase.go#L273) guard three different windows on the Ledger sign path but read like duplicates. Note what each one protects, so a later refactor doesn't drop one as redundant.

## tm2/pkg/crypto/keys/keybase_ledger_test.go:51-52 [gh](https://github.com/gnolang/gno/blob/6a98bd6cc/tm2/pkg/crypto/keys/keybase_ledger_test.go#L51-L52) · [↗](../../../../../.worktrees/gno-review-5709/tm2/pkg/crypto/keys/keybase_ledger_test.go#L51)
The [`Discover`](https://github.com/gnolang/gno/blob/6a98bd6cc/tm2/pkg/crypto/keys/keybase_ledger_test.go#L52) closure captures the outer `device` variable, so reassigning `device` between create and sign makes `Discover()` return a different device. The whole test turns on this, yet no comment says so.
