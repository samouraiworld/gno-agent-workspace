# PR [#5999](https://github.com/gnolang/gno/pull/5999): fix(tm2/auth): stop the block gas price from climbing forever or panicking

URL: https://github.com/gnolang/gno/pull/5999
Author: davd-gzl | Base: master | Files: 2 | +135 -9
Reviewed by: davd-gzl | Model: claude-opus-5 (deep) | Commit: 28dc8f9fd (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-5999 28dc8f9fd`

Round 2. Head advanced 29fb53a1e → 28dc8f9fd: three commits plus a clean merge of master (`git show 28dc8f9fd --cc` prints no conflict hunks). One production change, the int64 overflow `panic` is now a clamp at `math.MaxInt64`; the rest is comment compression and two new tests. Round 1's Suggestion asked for that decision and it is now made. Round 1's three nits are resolved, one of them partly re-opened by the compression. Every finding below except the floor Warning is already fixed on the branch; see *Applied on the branch*.

**TL;DR:** The node recalculates the minimum gas price at the end of every block from how much gas the block used against a target. This PR stops two ways that math breaks on a chain with no usable gas limit, and stops the third way it breaks under sustained congestion: instead of crashing the node when the price no longer fits in a 64-bit integer, the price now stops at the largest value that fits.

**Verdict: APPROVE** — the clamp is output-identical to master everywhere master does not crash, it is not an absorbing state, and no panic moves downstream; the two Warnings are a missing operator signal on the new silent path and a floor bug that predates this diff, neither of them a regression against master (0 Critical, 2 Warnings, 4 Nits, 1 Missing test).

## Applied on the branch

The reviewer authored this PR, so the findings went in as commits rather than as a posted review. Head is now 9d04904e9.

| Commit | Closes |
|---|---|
| 29a80685c `docs(tm2/auth)` | the three comment nits: the guard comment now names the target rather than the limit and gives `-1` its own harm, the `XXX` says which cap it still asks for, and the doc reads "and never below 1" |
| 0814069ba `fix(tm2/auth)` | the silent-ceiling Warning: `UpdateGasPrice` logs at `ERROR` on any block whose new price is `math.MaxInt64`, with a test asserting it fires there and nowhere below |
| 9d04904e9 `test(tm2/auth)` | the missing test: the descent now runs from the cap to the floor under a 1,000-block bound, and a new subtest sweeps the decrease branch across last and initial prices up to `math.MaxInt64` |

Left for a decision: the floor comparing amounts rather than ratios. It predates the branch, and fixing it changes the price an idle chain settles at, so it is a behavior change on a consensus path rather than a cleanup. Green after each commit: `go test ./tm2/pkg/sdk/...`, `go test ./gno.land/pkg/gnoland/ -run 'TestGasPriceUpdate|TestInitChainer'`, `gofmt -l` and `go vet` on the changed package. `golangci-lint` is not installed here and was not run.

## Verify first

- [`keeper.go:481-485`](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/sdk/auth/keeper.go#L481-L485) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper.go#L481-L485) — confirm the clamp is unreachable from the decrease branch, since it hardcodes `math.MaxInt64` rather than a bound derived from the branch. Run `TestCalcBlockGasPriceFloorAboveOne` with `InitialGasPrice` set to `math.MaxInt64`: every decrease result lands in `[1, max(lastPrice, initialPrice)]`, either through `lastPrice - diff` floored at `max(init, 1)` or through the early return that hands back `InitialGasPrice` whole, and both ends of that interval are representable by construction.
- [`keeper.go:437-439`](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/sdk/auth/keeper.go#L437-L439) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper.go#L437-L439) — confirm the guard still freezes only chains that have no target. Set `TargetGasRatio` to 1 and check which `MaxGas` values reach it: every value from 1 to 99 freezes, which is correct but wider than the comment beside it says.

## Summary

`calcBlockGasPrice` derives `targetGas = MaxGas * TargetGasRatio / 100`, divides by it twice, and multiplies the price by a ratio that grows without bound. Round 1 fixed the divisor: a non-positive target now returns the price unchanged, and the decrease floor is `max(InitialGasPrice, 1)`. Round 2 fixes the growth: at the shipped parameters (`MaxGas` 3e9 from [`MaxBlockMaxGas`](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/bft/types/params.go#L27) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/bft/types/params.go#L27), ratio 70, compressor 10) a run of completely full blocks raises the price 4.29% per block and leaves int64 range after 1,002 blocks, where master panicked in `EndBlock` on every node at once. Head returns `math.MaxInt64` instead and keeps producing blocks.

That end state is not a halt, but it is not usable either. `EnsureSufficientMempoolFees` compares the fee against the stored price through [`GasPrice.IsGTE`](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/std/gasprice.go#L61-L81) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/std/gasprice.go#L61-L81), and at a price of `MaxInt64/1000gas` nothing above 1,000 gas wanted can be paid for at all, because the fee that would clear stops being representable in the int64 `Coin.Amount`. Every mempool empties, blocks go idle, the decrease branch runs, and the price walks back to the floor in 407 idle blocks. The cap is a bounded outage where master had a coordinated crash.

## Diagram

```
price
MaxInt64 ─┐                        clamp: keeper.go:481-485
          │        ┌───────────────●─────────┐
          │       /                          \        master: panic here,
          │      /   1002 full blocks         \       every node, same height
          │     /                              \
          │    /                       407 idle \ blocks
        1 ─┴───────────────────────────────────────●──── floor = max(init, 1)
                                                        keeper.go:478
```

## Fix

Before, the same test panicked with `The min gas price is out of int64 range` at [`keeper.go:474`](https://github.com/gnolang/gno/blob/d1a33f574/tm2/pkg/sdk/auth/keeper.go#L474) on the merge base when the big-int result no longer fit; now [`keeper.go:481-485`](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/sdk/auth/keeper.go#L481-L485) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper.go#L481-L485) writes `math.MaxInt64` into the price and returns. The load-bearing constraint is that the saturation direction is always up: only the increase branch can produce a non-int64 value, because every decrease result lands in `[1, max(lastPrice, initialPrice)]` and both ends of that interval are already int64. A 91,584-case sweep against the merge base finds the clamp firing in the decrease branch zero times.

## Benchmarks / Numbers

Differential of merge base `d1a33f574` against head `28dc8f9fd`, one line per case, diffed:

| Outcome | Count |
|---|---|
| cases compared | 91,584 |
| identical | 71,999 |
| merge base panics | 14,976 |
| head panics | 0 |
| divergence: overflow, base panics and head caps at `MaxInt64` | 7,040 |
| divergence: target 0, base panics and head freezes the price | 7,936 |
| divergence: target below 0, base ratchets and head freezes the price | 3,360 |
| divergence: decrease floor `max(init, 1)`, base 0 and head 1 | 1,249 |
| divergence with any other cause | 0 |

Grid: `MaxGas` ∈ {-1, 0, 1, 2, 3, 10, 100, 142, 143, 1e3, 1e8, 3e9, 3e10, MaxInt64} × ratio ∈ {0, 1, 7, 50, 70, 99, 100} × compressor ∈ {1, 2, 10, 1000} × initial price ∈ {0, 1, 2, 1000} × last price ∈ {0, 1, 2, 3, 10, 1e3, 1e6, MaxInt64-1, MaxInt64} × `gasUsed` around 0, target±1 and `MaxGas`. Harness: [`tests/sweep_test.go`](tests/sweep_test.go).

Trajectory at the shipped shape, from a price of 1:

| Pattern | Blocks | End |
|---|---|---|
| every block full | 1,002 | `math.MaxInt64`, master panics at the same block |
| then every block idle | 407 | back at the floor of 1 |

ABCI-level A/B at head, same block sequence through `InitChain`/`BeginBlock`/`DeliverTx`/`EndBlock`/`Commit` on both binaries:

| MaxGas | merge base `d1a33f574` | head `28dc8f9fd` |
|---|---|---|
| 1,000,000 | prices 104, 108, 108, 100, 100, 100, 106 | identical prices, identical app hashes at all 7 heights |
| 0 | `EndBlock` panics "division by zero" at height 2 | app hash `55a51c2d…` constant across 3 heights |
| -1 | price 101 to 105, new app hash each height | price 100 constant, app hash `ca0f5d4c…` constant |

## Critical (must fix)

None.

## Warnings (should fix)

- **[nothing reports the ceiling]** `tm2/pkg/sdk/auth/keeper.go:481-485` — a chain that reaches the cap rejects every transaction it is offered and emits no log line, no error, and no telemetry at all.
  <details><summary>details</summary>

  Master's panic was brutal but loud. The clamp writes the price and returns, and the telemetry that would carry it is switched off by the state it creates: at the cap `calcBlockGasPrice` returns a value equal to its input, so [`UpdateGasPrice`'s skip](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/sdk/auth/keeper.go#L389-L391) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper.go#L389-L391) fires on every subsequent block and returns before both `SetGasPrice` and `logTelemetry`. The one hook that does fire, on the read path, records the wrong field: [`logTelemetry`](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/sdk/auth/keeper.go#L537-L541) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper.go#L537-L541) writes `gp.Gas`, the constant 1,000 denominator, into the [`BlockGasPriceAmount`](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/telemetry/metrics/metrics.go#L90-L91) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/telemetry/metrics/metrics.go#L90-L91) histogram, and the amount reaches only an unaggregatable string attribute. That metric defect predates this PR; what the diff changes is the cost of it, moving the failure mode from a crash every operator sees to a silent state in which the mempool rejects everything for hundreds of blocks. Fix: log at the clamp; the `sdk.Context` is one frame up in [`UpdateGasPrice`](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/sdk/auth/keeper.go#L362) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper.go#L362).
  </details>

- **[the floor compares amounts, not prices]** `tm2/pkg/sdk/auth/keeper.go:478` — the decrease floor compares raw `Price.Amount` values while a `GasPrice` is a ratio, so a floor configured with a different `Gas` denominator lands at the wrong price. Predates the diff; this diff edits the line and adds the one test that would have caught it.
  <details><summary>details</summary>

  Both [`keeper.go:463-464`](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/sdk/auth/keeper.go#L463-L464) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper.go#L463-L464) and [`keeper.go:478`](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/sdk/auth/keeper.go#L478) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper.go#L478) take `params.InitialGasPrice.Price.Amount` and compare it against the stored amount, ignoring both `Gas` fields, while [`GasPrice.IsGTE`](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/std/gasprice.go#L61-L81) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/std/gasprice.go#L61-L81) and [`ParseGasPrice`](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/std/gasprice.go#L17-L42) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/std/gasprice.go#L17-L42) treat the pair as a ratio, which is what the ratio form is for. With `auth:p:initial_gasprice` set to `100ugnot/2000gas`, a floor of 0.05 per gas, an idle chain settles at `100ugnot/1000gas`, which is 0.10 per gas, twice the configured floor. The other direction exists too: [`keeper.go:465`](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/sdk/auth/keeper.go#L465) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper.go#L465) returns `params.InitialGasPrice` whole, rewriting the stored denominator mid-chain. Both reproduce identically on the merge base, so neither is caused here. What puts it in scope is that the diff edits the floor line and adds [`TestCalcBlockGasPriceFloorAboveOne`](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/sdk/auth/keeper_test.go#L507) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper_test.go#L507), the file's only test of a floor above 1, with `Gas: 1000` on both sides, so it pins the amount contract rather than the price one. Fix: decide whether the floor is a ratio comparison, here or in a follow-up.
  </details>

## Nits

- **[comment names the one shape it does not cover]** `tm2/pkg/sdk/auth/keeper.go:435-436` — the guard comment blames an unbounded `MaxGas`, but a bounded `MaxGas` of 1 hits the guard too, and the sentinel it does name is frozen for a different reason than the one it gives.
  <details><summary>details</summary>

  The condition is on the target, not on the limit: [`targetGasInt.Sign() <= 0`](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/sdk/auth/keeper.go#L437) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper.go#L437) fires for `MaxGas` 0 and 1 at the default ratio of 70, and for every `MaxGas` from 1 to 99 at a `TargetGasRatio` of 1, which [`Params.Validate`](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/sdk/auth/params.go#L115-L117) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/params.go#L115-L117) accepts. "Both branches below divide by it" is also the wrong harm for the sentinel: at `MaxGas -1` the target is -1, dividing by it is legal, and what breaks is the +1-per-block ratchet. The function names that ratchet twice, at [`keeper.go:407-408`](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/sdk/auth/keeper.go#L407-L408) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper.go#L407-L408) and [`keeper.go:471-474`](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/sdk/auth/keeper.go#L471-L474) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper.go#L471-L474), but never in the comment on the guard that exists to prevent it. Fix: name the target rather than the limit, and give the sentinel its own harm.
  </details>

- **[two contracts on one function]** `tm2/pkg/sdk/auth/keeper.go:460` — the doc comment now says the increase branch caps, while the `XXX` inside that branch still asks whether to cap at all.
  <details><summary>details</summary>

  [`keeper.go:408-409`](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/sdk/auth/keeper.go#L408-L409) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper.go#L408-L409) reads "when increasing we cap at the largest int64 price", and [`keeper.go:460`](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/sdk/auth/keeper.go#L460) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper.go#L460) still reads `XXX should we cap it with a max gas price?`. Both are defensible on their own: the open question is about a policy ceiling a chain would choose, the doc is about the representational one the type imposes. Read together they cancel out. Fix: say which cap the `XXX` is still asking for.
  </details>

- **[floor reads as a choice between two values]** `tm2/pkg/sdk/auth/keeper.go:409` — "we floor the result at the initial gas price, or at 1" describes an alternative, and the code is a maximum of the two.
  <details><summary>details</summary>

  [`keeper.go:478`](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/sdk/auth/keeper.go#L478) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper.go#L478) is `maxBig(num, maxBig(initPriceInt, bigOne))`, so an initial price of 1,000 floors at 1,000 and an initial price of 0 floors at 1. The current wording invites "at 1 when there is no initial price". Fix: "at the initial gas price, never below 1".
  </details>

- **[the exception went missing again]** `tm2/pkg/sdk/auth/keeper_test.go:450` — the comment says both branches divide by the zero target, which two of the subtest's six rows never reach.
  <details><summary>details</summary>

  Round 1 raised this and commit `391bc87f5` fixed it by adding the exception, then `4119cddff` deleted that sentence again. The subtest runs `MaxGas` 0 and 1 against `gasUsed` 0, 1 and 1,000,000. In the two `gasUsed` 0 rows the usage equals the target, so on the merge base they return at [the `Cmp == 0` check](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/sdk/auth/keeper.go#L443-L445) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper.go#L443-L445) before either division and pass there too; at head they stop one check earlier, at the new guard. The comment's claim holds for the other four rows only. Fix: restore the one clause naming the empty-block case.
  </details>

## Missing Tests

- **[the descent is the property that makes the cap safe]** `tm2/pkg/sdk/auth/keeper_test.go:212` — the "cap is not absorbing" assertion proves one decrement, not that the price returns to the floor.
  <details><summary>details</summary>

  `require.Less(..., int64(math.MaxInt64))` fails only if the very first idle block leaves the price at the cap. What makes the clamp acceptable rather than a trap is the full descent: 407 idle blocks from `math.MaxInt64` back to the floor at the shipped parameters, and the first 10 of those already take the price below 3.3e18. The decay is multiplicative in the gap to target, so it is fast only while blocks are near-empty: held at exactly `target-1` gas the price falls by about 4.4e8 per block, and 100,000 such blocks move it from 9223372036854775807 to 9223328116140174858. A future change to the floor, the compressor, or the min-1 decrement could leave the price stuck near the cap for a practically unbounded number of blocks and this assertion would still pass. The same test file also assumes, and nowhere asserts, that the decrease branch cannot reach the clamp, which matters because [the clamp](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/sdk/auth/keeper.go#L481-L485) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper.go#L481-L485) writes a hardcoded `math.MaxInt64` for both. Both properties are covered by [`tests/clamp_test.go`](tests/clamp_test.go), green at head, where its first test panics with `The min gas price is out of int64 range` against the merge base.
  </details>

## Suggestions

None.

## Verified

- The clamp is reached, keeps the rest of the price intact, and is not absorbing. From a price of 1 at the shipped parameters, 1,002 consecutive full blocks reach `math.MaxInt64` with `Gas` still 1,000 and the denom still `ugnot`, and 407 consecutive idle blocks bring it back to the floor of 1. Against the merge base the same harness panics with `The min gas price is out of int64 range` on its first case. Harness: [`tests/clamp_test.go`](tests/clamp_test.go).
- The decrease branch cannot reach the clamp. Across last prices from 1 to `math.MaxInt64` and initial prices from 0 to `math.MaxInt64`, every decrease result stays within `[1, max(last, initial)]`, and the 91,584-case sweep records zero clamp hits in that branch.
- No unexplained behaviour change against the merge base. Every one of the 19,585 divergent cases falls into one of the four intended causes, and the merge base panics in all 14,976 cases where head does not.
- A rolling upgrade is safe on gno.land's configuration, and only there. The same block sequence driven through the full ABCI cycle on both binaries at `MaxGas` 1,000,000 gives identical app hashes at all seven heights. The two divergent configurations differ from each other: a `MaxGas` 0 chain is already halted on master, so there is no live network to fork, while a `MaxGas` -1 chain keeps producing blocks with a climbing price, so mixed binaries there would fork and it needs a coordinated restart. No config in the tree sets -1; [`DefaultBlockParams`](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/bft/types/params.go#L44-L51) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/bft/types/params.go#L44-L51) is 3e9. Harness: [`tests/abci_apphash_ab_test.go`](tests/abci_apphash_ab_test.go).
- A capped chain accepts nothing worth sending. `IsGTE` against a stored `MaxInt64/1000gas` reduces to `fee ≥ gasWanted × MaxInt64 / 1000`, so 1,000 gas wanted needs the entire int64 range as its fee, roughly 361,700 times the whole devnet genesis supply of 25,500,000,000,000 ugnot, and anything above 1,000 gas wanted has no representable clearing fee at all.
- Green at 28dc8f9fd: `go test ./tm2/pkg/sdk/auth/`, `go test ./gno.land/pkg/gnoland/ -run TestGasPriceUpdate`, `gofmt -l` clean on both changed files. CI at head: 82 checks pass, 2 skipped, none failing.

## Open questions

- `gnokey --simulate` computes its fee with [`overflow.Mulp`](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/crypto/keys/client/broadcast.go#L227-L231) · [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/crypto/keys/client/broadcast.go#L227-L231) on the price it fetches, which panics client-side well before the price reaches the cap. It reproduces on master, where the chain then halts a few hundred blocks later; head keeps serving the price, so the client-side panic lasts longer. Not posted: a separate client fix, and nothing in this diff makes it worse for a chain that is not already at an unpayable price.
- Round 1's Verified section claimed the function is output-identical to master for any chain with a positive target and a non-zero initial price. That now holds only where master does not panic; the clamp is a third intended divergence. Not posted: it corrects an unposted draft, not the PR.

