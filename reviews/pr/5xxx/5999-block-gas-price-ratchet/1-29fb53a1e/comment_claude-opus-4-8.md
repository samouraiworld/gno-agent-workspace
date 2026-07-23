# Review: PR [#5999](https://github.com/gnolang/gno/pull/5999)
Event: COMMENT

## Body
The guard and the floor both hold up. Verified on 29fb53a1e: the same block sequence driven through the full ABCI cycle on this branch and on master, at a block gas limit of 1000000, produces identical app hashes at every height. A 43632-case sweep against master attributes every output difference to the non-positive-target return or the new floor, and finds no other. Deleting the three lines that return early when the target is not positive brings both failures straight back, the +1 per idle block at a limit of -1 and the division by zero at a limit of 0.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5999-block-gas-price-ratchet/1-29fb53a1e/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## tm2/pkg/sdk/auth/keeper.go:494-502 [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper.go#L494-L502)
Nit: a chain configured with a zero `InitialGasPrice` never has a non-zero price to walk down from. [`installAuthParams`](https://github.com/gnolang/gno/blob/29fb53a1e/gno.land/pkg/gnoland/app.go#L731) seeds the stored price from `InitialGasPrice`, so the [`Price.Amount == 0` return](https://github.com/gnolang/gno/blob/29fb53a1e/tm2/pkg/sdk/auth/keeper.go#L412-L414) disables pricing from block 1. The decay described here needs a post-genesis change to `auth:p:initial_gasprice`, which is reachable because [only the gas amount](https://github.com/gnolang/gno/blob/29fb53a1e/tm2/pkg/std/gasprice.go#L34) has to be positive.

## tm2/pkg/sdk/auth/keeper.go:406-409 [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper.go#L406-L409)
Nit: this still promises a decrease floor at the initial gas price and describes only the two branches. Both contracts moved in this diff, to [`max(initial, 1)`](https://github.com/gnolang/gno/blob/29fb53a1e/tm2/pkg/sdk/auth/keeper.go#L503) and to an [early return on a non-positive target](https://github.com/gnolang/gno/blob/29fb53a1e/tm2/pkg/sdk/auth/keeper.go#L452-L454).

## tm2/pkg/sdk/auth/keeper_test.go:439 [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper_test.go#L439)
Nit: with `MaxGas` 0 and `gasUsed` 0 the target matches exactly, so the [`Cmp == 0` return](https://github.com/gnolang/gno/blob/29fb53a1e/tm2/pkg/sdk/auth/keeper.go#L457-L460) fires before any division. That combination returns the price unchanged on master, so the first iteration of the subtest is not covering the panic.

## tm2/pkg/sdk/auth/keeper.go:506-508 [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper.go#L506-L508)
Suggestion: the overflow panic in this function is untouched and sustained congestion still reaches it. At the shipped parameters a run of completely full blocks raises the price by 4.29% each time and trips this check at block 1002, and a `TargetGasRatio` of 1 with a `GasPricesChangeCompressor` of 1 trips it after 9, both of which [`Params.Validate`](https://github.com/gnolang/gno/blob/29fb53a1e/tm2/pkg/sdk/auth/params.go#L112-L117) accepts. Worth deciding whether the [cap the increase branch already asks for](https://github.com/gnolang/gno/blob/29fb53a1e/tm2/pkg/sdk/auth/keeper.go#L475) belongs in this PR or a follow-up.
