# Review: [#5382](https://github.com/gnolang/gno/pull/5382)
Posted: https://github.com/gnolang/gno/pull/5382#pullrequestreview-5061296626
Event: COMMENT
Status: posted as an AI on 2026-08-30, forced to COMMENT with the verdict on the Body Status line. Round 2 at aef4cd9b5. Two of round 1's findings are fixed at this head and were dropped; the rest were re-tested and are still open. On `post as an AI` the Body leads with `[AI review, opus 5] (not manually verified)`, then `Status: REQUEST_CHANGES`.

## Body
[AI review, opus 5] (not manually verified)
Status: REQUEST_CHANGES

- A sponsored tx's compute lands on the block gas meter like any other, [`ConsumeGas` on the block meter runs for every DeliverTx](https://github.com/gnolang/gno/blob/aef4cd9b5/tm2/pkg/sdk/baseapp.go#L920-L923), and the next block's price is derived from [what the block meter consumed](https://github.com/gnolang/gno/blob/aef4cd9b5/tm2/pkg/sdk/auth/keeper.go#L363). So 0-fee traffic raises `LastGasPrice`, the floor every normal fee-payer must clear, while contributing no fee itself, and the [credit window may be sized up to a whole block](https://github.com/gnolang/gno/blob/aef4cd9b5/tm2/pkg/bft/types/params.go#L90-L92). Deterministic, so not a fork; it gates setting `MaxGasCreditPerTx` above 0 rather than merging. Excluding sponsored gas from the price signal is the other reading, and the design document names neither.

## gno.land/pkg/gnoland/app.go:298 [gh](https://github.com/gnolang/gno/blob/aef4cd9b5/gno.land/pkg/gnoland/app.go#L298) · [↗](../../../../../.worktrees/gno-review-5382/gno.land/pkg/gnoland/app.go#L298) [posted](https://github.com/gnolang/gno/pull/5382#discussion_r3889878988)
This pays every realm's freed-storage refund to the PayStorage sponsor, while [the branch just below](https://github.com/gnolang/gno/blob/aef4cd9b5/gno.land/pkg/gnoland/app.go#L311) routes the same refund to `ctx.TxCaller()`, so a realm that sponsors one byte of growth also collects the deposit released by any storage the tx frees, including storage funded in an earlier transaction by someone else. Routing refunds to the tx caller in both branches keeps the recipient the same whether or not a sponsor is present.

## tm2/pkg/std/tx.go:55 [gh](https://github.com/gnolang/gno/blob/aef4cd9b5/tm2/pkg/std/tx.go#L55) · [↗](../../../../../.worktrees/gno-review-5382/tm2/pkg/std/tx.go#L55) [posted](https://github.com/gnolang/gno/pull/5382#discussion_r3889878989)
Missing test: nothing in [`tm2/pkg/std`](https://github.com/gnolang/gno/tree/aef4cd9b5/tm2/pkg/std) calls `Tx.ValidateBasic`, so the relaxed fee check that decides which transactions enter the mempool has no test pinning either side of it, and a later tightening back to `!IsZero() && !IsValid()` reddens nothing.

<details><summary>test cases</summary>

A table over the canonical zero coin, a valid coin, a bad denom carrying a zero amount, an empty denom with a non-zero amount, and a negative amount. Passes at aef4cd9b5: [tx_validatebasic_fee_test.go](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5382-realm-transaction-sponsorship/2-aef4cd9b5/tests/tx_validatebasic_fee_test.go).
</details>

## tm2/pkg/bft/types/params.go:80-92 [gh](https://github.com/gnolang/gno/blob/aef4cd9b5/tm2/pkg/bft/types/params.go#L80-L92) · [↗](../../../../../.worktrees/gno-review-5382/tm2/pkg/bft/types/params.go#L80) [posted](https://github.com/gnolang/gno/pull/5382#discussion_r3889878991)
Missing test: [`makeParams`](https://github.com/gnolang/gno/blob/aef4cd9b5/tm2/pkg/bft/types/params_test.go#L51-L67) never sets `MaxGasCreditPerTx`, so both new rejections run in no test and a genesis carrying a credit window larger than a block would be accepted if either branch were dropped.

<details><summary>test cases</summary>

A table over credit 0, below `MaxGas`, equal to it, above it, negative, and both cases under `MaxGas == -1`. Passes at aef4cd9b5: [params_maxgascredit_test.go](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5382-realm-transaction-sponsorship/2-aef4cd9b5/tests/params_maxgascredit_test.go).
</details>

## gno.land/pkg/sdk/vm/keeper.go:1861-1870 [gh](https://github.com/gnolang/gno/blob/aef4cd9b5/gno.land/pkg/sdk/vm/keeper.go#L1861-L1870) · [↗](../../../../../.worktrees/gno-review-5382/gno.land/pkg/sdk/vm/keeper.go#L1861) [posted](https://github.com/gnolang/gno/pull/5382#discussion_r3889878993)
Nit: a blank line separates this comment from the `func` it describes, so `go doc` shows the newly-exported `ProcessStorageDeposit` undocumented; the text also still says the caller is charged and refunded, which under sponsorship is the PayStorage realm, and for a restricted denom the refund goes to `StorageFeeCollector`.

## gno.land/pkg/sdk/vm/keeper.go:2050-2052 [gh](https://github.com/gnolang/gno/blob/aef4cd9b5/gno.land/pkg/sdk/vm/keeper.go#L2050-L2052) · [↗](../../../../../.worktrees/gno-review-5382/gno.land/pkg/sdk/vm/keeper.go#L2050) [posted](https://github.com/gnolang/gno/pull/5382#discussion_r3889878995)
Nit: the defensive fallback raises the spending cap it is defending, since reaching it swaps the sponsor's committed budget for `DefaultDeposit` and charges up to that instead; returning an error keeps the invariant the comment above states without the failure mode being a larger charge.
