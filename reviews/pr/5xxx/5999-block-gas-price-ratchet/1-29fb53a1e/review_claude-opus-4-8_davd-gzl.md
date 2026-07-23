# PR [#5999](https://github.com/gnolang/gno/pull/5999): fix(tm2/auth): stop the block gas price from ratcheting or panicking

URL: https://github.com/gnolang/gno/pull/5999
Author: davd-gzl | Base: master | Files: 2 | +123 -2
Reviewed by: davd-gzl | Model: claude-opus-4-8 (xhigh, deep) | Commit: 29fb53a1e (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5999 29fb53a1e`

**TL;DR:** The node recalculates the minimum gas price at the end of every block by comparing how much gas the block used against a target. When the chain is configured with no block gas limit, that target came out as zero or negative, which either pushed the price up forever or crashed the node on a division by zero. This adds a guard so a meaningless target leaves the price alone, and stops the price from decaying all the way to zero.

**Verdict: APPROVE** — the arithmetic, the convergence behaviour and the app-hash equivalence all check out; three comment-accuracy nits and one pre-existing panic that this PR does not close (0 Critical, 0 Warning, 3 Nits, 1 Suggestion).

## Summary

`calcBlockGasPrice` derives `targetGas = MaxGas * TargetGasRatio / 100` and divides by it twice. Both "no gas bound" spellings of `Block.MaxGas` break that divisor. `MaxGas == -1` makes the target `-1` (`big.Int.Div` floors), so every `gasUsed >= 0` takes the increase branch, the intermediate quotient goes negative, and the `max(num, 1)` clamp turns it into `+1`: the price rises by 1 on every block including idle ones, with the decrease branch unreachable. `MaxGas == 0` or `1` makes the target `0` and the first non-empty block divides by zero inside `EndBlock`, which halts the node. Both configurations pass [`ValidateConsensusParams`](https://github.com/gnolang/gno/blob/29fb53a1e/tm2/pkg/bft/types/params.go#L69) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/bft/types/params.go#L69), which only rejects `MaxGas < -1`.

The fix returns the price unchanged when the target is non-positive, and raises the decrease floor from `InitialGasPrice` to `max(InitialGasPrice, 1)`.

Both failure shapes are one JSON field away. [`ConsensusParams.Update`](https://github.com/gnolang/gno/blob/29fb53a1e/tm2/pkg/bft/abci/types/params.go#L26) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/bft/abci/types/params.go#L26) copies a non-nil `Block` whole rather than field by field, so a genesis whose `consensus_params.Block` object simply omits `MaxGas` validates cleanly with `MaxGas == 0` and no default backfill. `gnodev -max-gas 0` reaches the same state through [`config.MaxGasPerBlock`](https://github.com/gnolang/gno/blob/29fb53a1e/contribs/gnodev/setup_node.go#L120) · [↗](../../../../../.worktrees/gno-review-5999/contribs/gnodev/setup_node.go#L120), which is copied into the genesis at [`node.go:627`](https://github.com/gnolang/gno/blob/29fb53a1e/contribs/gnodev/pkg/dev/node.go#L627) · [↗](../../../../../.worktrees/gno-review-5999/contribs/gnodev/pkg/dev/node.go#L627) with no `> 0` guard.

```
MaxGas = -1   target = -1*70/100 = -1   (Div floors)
              gasUsed 0 > -1  -> increase branch
              num = (0-(-1))*P/(-1)/10 = -P/10  -> max(-P/10, 1) = 1
              price += 1   every block, forever, decrease unreachable

MaxGas = 0    target = 0
              gasUsed 0  -> Cmp(target, gasUsed) == 0 -> early return, no panic
              gasUsed >0 -> increase branch -> num.Div(num, 0) -> panic
```

## Glossary

- app hash: the per-block commitment to application state, agreed in consensus; two nodes computing different app hashes for the same block halts the chain.
- block gas price: the per-gas minimum fee the node enforces on incoming transactions, recomputed each block from the previous block's gas usage.
- target block gas: `MaxGas * TargetGasRatio / 100`, the gas usage the controller steers towards; usage above it raises the price, below it lowers it.

## Fix

Before, the two divisors were taken on trust: [`keeper.go:429-431`](https://github.com/gnolang/gno/blob/29fb53a1e/tm2/pkg/sdk/auth/keeper.go#L429-L431) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper.go#L429-L431) computed the target and both branches divided by it unchecked. Now [`keeper.go:452-454`](https://github.com/gnolang/gno/blob/29fb53a1e/tm2/pkg/sdk/auth/keeper.go#L452-L454) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper.go#L452-L454) returns `lastGasPrice` whenever the target is non-positive, which covers both the `-1` sentinel and the `MaxGas * ratio < 100` rounding case in one condition. The load-bearing constraint is that the guard must not perturb a chain with a real target: it sits strictly after the target is computed and before any branch is chosen, so a positive target reaches exactly the code master ran. The decrease floor at [`keeper.go:503`](https://github.com/gnolang/gno/blob/29fb53a1e/tm2/pkg/sdk/auth/keeper.go#L503) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper.go#L503) changes only when `InitialGasPrice.Price.Amount` is 0, which [`Params.Validate`](https://github.com/gnolang/gno/blob/29fb53a1e/tm2/pkg/sdk/auth/params.go#L124) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/params.go#L124) accepts.

## Benchmarks / Numbers

Differential of master `d14a03770` against head `29fb53a1e` over 43,632 parameter combinations (`MaxGas` ∈ {-1, 0, 1, 2, 3, 10, 100, 142, 143, 1e3, 1e8, 3e9, 3e10, MaxInt64} × ratio ∈ {1, 7, 50, 70, 99, 100} × compressor ∈ {1, 10, 1000} × initial price ∈ {0, 1, 2, 1000} × last price ∈ {1, 2, 3, 10, 1e3, 1e6} × gasUsed around 0, target±2 and MaxGas):

| Outcome | Count |
|---|---|
| cases compared | 43,632 |
| master panics | 4,464 |
| head panics | 0 |
| divergences explained by the new non-positive-target guard | 6,624 |
| divergences explained by the new `max(init, 1)` floor | 1,080 |
| divergences with any other cause | 0 |

Price trajectory at the shipped shape (`MaxGas` 3e9, ratio 70, compressor 10, initial price 1), 5,000 blocks:

| Gas pattern | Start | End | Shape |
|---|---|---|---|
| every block idle | 1,000,000 | 1 | -10% per block, geometric down to the floor |
| every block exactly at target | 1,000 | 1,000 | fixed point |
| alternating full / empty | 1,000 | 18 | converges down, then oscillates 18↔19 |
| `target+1` every block | 1,000 | 6,000 | +1 per block, linear |
| `target-1` every block | 1,000 | 1 | -1 per block, linear |
| 1 full per 10 blocks | 1,000 | 1 | converges down |
| uniform random `[0, MaxGas]` | 1,000 | 2 | converges down, min 1 |
| 9 full per 10 blocks | 1,000 | panic at block 1,347 | int64 overflow |
| every block full | 1 | panic at block 1,002 | int64 overflow |

ABCI-level A/B, same block sequence driven through `InitChain`/`BeginBlock`/`DeliverTx`/`EndBlock`/`Commit` on both binaries:

| MaxGas | master | head |
|---|---|---|
| 0 | `EndBlock` panics "division by zero" at height 2 | app hash `55a51c2d…` constant across all 3 heights |
| -1 | price 101, 102, 103, 104, 105 on five empty blocks, new app hash each height | price 100 constant, app hash `ca0f5d4c…` constant |
| 1,000,000 | prices 104, 108, 108, 100, 100, 100, 106 | identical prices, identical app hashes at all 7 heights |

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- **[wrong reason recorded for a right change]** `tm2/pkg/sdk/auth/keeper.go:494-502` — the floor comment says a chain configured with a zero `InitialGasPrice` walks the price down to 0, but such a chain never has a non-zero price to walk down from.
  <details><summary>details</summary>

  At genesis the stored block gas price is seeded from `InitialGasPrice` by [`installAuthParams`](https://github.com/gnolang/gno/blob/29fb53a1e/gno.land/pkg/gnoland/app.go#L731) · [↗](../../../../../.worktrees/gno-review-5999/gno.land/pkg/gnoland/app.go#L731), so a zero initial price stores a zero price, and the [`Price.Amount == 0` guard](https://github.com/gnolang/gno/blob/29fb53a1e/tm2/pkg/sdk/auth/keeper.go#L412-L414) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper.go#L412-L414) disables dynamic pricing from block 1. The decay the comment describes needs a non-zero stored price and a zero `InitialGasPrice` at the same time, which only a runtime parameter change produces: `auth` is registered with the params keeper at [`app.go:118`](https://github.com/gnolang/gno/blob/29fb53a1e/gno.land/pkg/gnoland/app.go#L118) · [↗](../../../../../.worktrees/gno-review-5999/gno.land/pkg/gnoland/app.go#L118), and `ParseGasPrice("0ugnot/1000gas")` succeeds because only [`gas.Amount`](https://github.com/gnolang/gno/blob/29fb53a1e/tm2/pkg/std/gasprice.go#L34) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/std/gasprice.go#L34) is required positive, and `Params.Validate` then accepts it. Confirmed behaviourally: `ParseGasPrice("0ugnot/1000gas")` returns amount 0 with a nil error and `Params.Validate` returns nil. The floor is worth keeping either way; only the stated reachability is wrong. Fix: attribute the decay to a post-genesis `auth:p:initial_gasprice` change rather than to genesis configuration.
  </details>

- **[stale contract above the function]** `tm2/pkg/sdk/auth/keeper.go:406-409` — the function doc still promises the old two-branch contract and does not mention either change.
  <details><summary>details</summary>

  It says "when decreasing we floor the result at the initial gas price", which the new [`max(initPrice, 1)`](https://github.com/gnolang/gno/blob/29fb53a1e/tm2/pkg/sdk/auth/keeper.go#L503) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper.go#L503) supersedes, and it describes only the two branches, with no mention that a non-positive target now returns early. A reader who trusts the doc gets both contracts wrong. Fix: fold the floor and the early return into the doc comment.
  </details>

- **[test comment overstates the panic]** `tm2/pkg/sdk/auth/keeper_test.go:439` — "every branch divides by it" is not true for the `gasUsed == 0` case the same subtest exercises first.
  <details><summary>details</summary>

  With `MaxGas` 0 the target is 0, and `gasUsed == 0` matches it exactly, so [the `Cmp == 0` early return](https://github.com/gnolang/gno/blob/29fb53a1e/tm2/pkg/sdk/auth/keeper.go#L457-L460) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper.go#L457-L460) fires before any division. Confirmed behaviourally: on master `maxGas=0, gasUsed=0` returns the price unchanged, and only `gasUsed=1` panics. The subtest's first iteration therefore passes on master too, which the comment implies it should not. Fix: say the division is reached once the block is non-empty.
  </details>

## Missing Tests

None. The upgrade-safety property is already pinned from two sides: [`TestCalcBlockGasPrice/calculated_deltas`](https://github.com/gnolang/gno/blob/29fb53a1e/tm2/pkg/sdk/auth/keeper_test.go#L182-L187) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper_test.go#L182-L187) fixes the arithmetic at a positive target, and [`TestGasPriceUpdate`](https://github.com/gnolang/gno/blob/29fb53a1e/gno.land/pkg/gnoland/app_test.go#L1167) · [↗](../../../../../.worktrees/gno-review-5999/gno.land/pkg/gnoland/app_test.go#L1167) asserts the golden `108ugnot` after a real five-block ABCI run, a value written against master that still holds at head.

## Suggestions

- **[the other panic in the same function]** `tm2/pkg/sdk/auth/keeper.go:506-508` — the int64 overflow panic the title's "or panicking" would also cover is untouched, and sustained congestion still reaches it.
  <details><summary>details</summary>

  At the shipped parameters a completely full block multiplies the price by `1 + 3/7/10 ≈ 1.0429`, so from a price of 1 the `IsInt64` check fires at block 1,002 and `EndBlock` panics on every node at once; a 9-full-per-10 pattern reaches it at block 1,347. Governance can shorten that dramatically without tripping any validation: `TargetGasRatio` 1 and `GasPricesChangeCompressor` 1 both pass `Params.Validate`, and that pair overflows after 9 full blocks. This is not a defect in the diff. It reproduces identically on master, it is pinned as intended behaviour by the pre-existing [`TestCalcBlockGasPrice/int64_overflow`](https://github.com/gnolang/gno/blob/29fb53a1e/tm2/pkg/sdk/auth/keeper_test.go#L198-L202) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper_test.go#L198-L202), and the increase branch already carries an `XXX should we cap it with a max gas price?` note at [`keeper.go:475`](https://github.com/gnolang/gno/blob/29fb53a1e/tm2/pkg/sdk/auth/keeper.go#L475) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper.go#L475). Organic demand cannot pay its way there: each full block costs `price × 3e6` ugnot in fees, so the fee total across the 1,002 blocks is above 1e26 ugnot. Reaching it needs either a proposer share above ~71.5% (the break-even between the +4.29% rise and the -10% idle decay) or the governance parameter change above. Fix: decide whether a price cap belongs in this PR or a follow-up, given the title claims the panic class.
  </details>

## Verified

- Reverting the production hunk to `d14a03770` and keeping the new tests turns both red: in [`TestCalcBlockGasPriceUnboundedMaxGas`](https://github.com/gnolang/gno/blob/29fb53a1e/tm2/pkg/sdk/auth/keeper_test.go#L411) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper_test.go#L411), `MaxGas -1 does not ratchet` fails with 1001 against an expected 1000 and `zero target does not panic` fails with `division by zero` at `maxGas=0, gasUsed=1`. [`TestCalcBlockGasPriceZeroInitialPrice`](https://github.com/gnolang/gno/blob/29fb53a1e/tm2/pkg/sdk/auth/keeper_test.go#L464) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper_test.go#L464) fails with "0 is not greater than or equal to 1". The third subtest, `smallest usable MaxGas still adjusts`, passes on master too, so it is a regression guard rather than a bug proof.
- A rolling upgrade is safe on gno.land's configuration. Driving the same block sequence through the full ABCI cycle on both binaries at `MaxGas` 1,000,000 produced identical app hashes at all seven heights and identical prices 104, 108, 108, 100, 100, 100, 106. At `MaxGas` 0 master panics in `EndBlock` at height 2 while head commits `55a51c2d…` unchanged, and at `MaxGas` -1 master emits a new app hash every block while head holds `ca0f5d4c…`; both of those chains need a coordinated restart, not a rolling one, and a `MaxGas` 0 chain on master is already halted. Harness: [`tests/abci_apphash_ab_test.go`](tests/abci_apphash_ab_test.go).
- No unexplained behaviour change. A 43,632-case differential against master attributes every divergence to one of the two intended causes and finds zero others, so for any chain with a positive target and a non-zero initial price the function is output-identical to master. Harness: [`tests/differential_sim_test.go`](tests/differential_sim_test.go).
- The controller converges in both directions and does not oscillate divergently. Over 5,000 blocks the price falls geometrically at -10% per idle block, rises at +4.29% per full block, holds a fixed point exactly at target, and lands in a ±1 oscillation at the floor under an alternating full/empty pattern. Within the dead band where the proportional term truncates to 0 the move is exactly ±1, symmetric in both branches, so integer truncation carries no directional bias.
- Determinism holds. The function uses only `int64` and `math/big`, with no float, no map iteration and no platform-dependent width; `big.Int.Div` is Euclidean, which is floor division for the positive divisor 100 and identical on every platform.
- Both `MaxGas` failure shapes are reachable from a genesis the tooling produces. `GenesisDoc.ValidateAndComplete` returns nil with `MaxGas` 0 when the `Block` object omits the field, and with 0, -1 and 1 when it sets them explicitly.
- The second divisor is not a hole. `calcBlockGasPrice` also divides by `GasPricesChangeCompressor`, and a value of 0 still panics on head, but every write path validates it: `Params.Validate` rejects `<= 0`, `auth.InitGenesis` goes through `SetParams`, and `WillSetParam` validates. The only unvalidated shape is the all-zero `Params` from an empty params store, where `TargetGasRatio == 0` already returns early.
- [`UpdateGasPrice`](https://github.com/gnolang/gno/blob/29fb53a1e/tm2/pkg/sdk/auth/keeper.go#L374) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper.go#L374) dereferences `ctx.ConsensusParams().Block` without the [nil guard `getMaximumBlockGas` carries](https://github.com/gnolang/gno/blob/29fb53a1e/tm2/pkg/sdk/baseapp.go#L300-L303) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/baseapp.go#L300-L303), and `amino.DeepCopy` on a nil `*ConsensusParams` returns a typed nil, so that would fault. Not reachable for a real node: the handshaker always passes `&csParams` and `ValidateAndComplete` backfills `Block`.
- Freezing the price is the right answer for a non-positive target rather than a way to strand an elevated price. tm2 has no `ConsensusParamUpdates` path and [`LastHeightConsensusParamsChanged`](https://github.com/gnolang/gno/blob/29fb53a1e/tm2/pkg/bft/state/execution.go#L369) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/bft/state/execution.go#L369) is only ever copied forward, so `MaxGas` cannot flip to 0 or -1 mid-chain; on a chain that starts that way the stored price is still `InitialGasPrice`, so the frozen value is the configured one.
- Green at 29fb53a1e: `go test ./tm2/pkg/sdk/auth/`, `go test ./tm2/pkg/sdk/ -run 'Gas|Block|Consensus'`, `go test ./gno.land/pkg/gnoland/ -run TestGasPriceUpdate`, `gofmt -l` clean on both changed files.

## Open questions

- `gnodev -max-gas 0` and `-max-gas -1` reach the genesis states this PR guards, because neither `setup_node.go` nor `node.go` gates the flag on `> 0` the way `gnogenesis generate` does. Not posted: it is a separate tooling fix and this PR makes the node survive it anyway.
- After this change, setting `auth:p:initial_gasprice` to a zero-amount price no longer lets the price decay to 0, so governance loses an accidental way to switch dynamic pricing off. That looks like the intent rather than a regression. Not posted: no decision needed in this PR.
