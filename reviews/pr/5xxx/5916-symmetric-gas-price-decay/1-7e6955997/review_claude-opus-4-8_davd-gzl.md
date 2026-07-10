# PR [#5916](https://github.com/gnolang/gno/pull/5916): fix(tm2/auth): make block gas price decay symmetric (ratchet fix for #5906)

URL: https://github.com/gnolang/gno/pull/5916
Author: moul | Base: master | Files: 2 | +69 -15
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 7e6955997 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5916 7e6955997`

**TL;DR:** gno.land raises the block gas price when blocks fill up and is supposed to lower it again when they empty out. A rounding bug meant the price could only ever go up, so after the GnoSwap competition on `test13` it stayed at 10x the floor even once traffic dropped. This makes the price able to fall again.

**Verdict: NEEDS DISCUSSION** — the decay logic is correct and matches [#5906](https://github.com/gnolang/gno/issues/5906), but the change alters EndBlock state transitions and the AppHash on empty blocks, so it is consensus-breaking and needs a coordinated rollout decision (the author flags `test14`); the only code gap is that the empty-block behavior of `UpdateGasPrice` ships untested.

## Summary
Correct fix. The dynamic gas price adjusts each block by `lastPrice ± lastPrice·|gasUsed − target| / target / compressor` in integer arithmetic. The increase branch floored its step at `+1`; the decrease branch did not, so once the price fell to `≤ GasPricesChangeCompressor` (the default is 10) every under-target block computed a `0` decrease and the price stuck. On `test13` it stuck at `10ugnot/1000gas` (10x the floor) while blocks ran at ~14% of the 70% target. The fix floors the decrease at `1`, symmetric to the increase, still clamped at `InitialGasPrice`, and removes the two empty-block guards so idle periods also decay the price.

## Examples
| block gasUsed (target = 2.1B) | old next price (from 10) | new next price (from 10) |
|---|---|---|
| 14% full (420M) | 10 (stuck) | 9 |
| empty (0) | 10 (skipped) | 9 |
| 100% full (3B) | 11 | 11 |
| exactly on target | 10 | 10 |

## Glossary
- **AppHash** — hash of application state committed each block; every validator must compute the same one or the chain halts.
- **EndBlock** — the ABCI step after a block's txs where the next block's gas price is recomputed.

## Fix
Before, the decrease branch set `newPrice = lastPrice − (target−gasUsed)·lastPrice/target/compressor` with integer division rounding any sub-unit step to `0`, and both [`UpdateGasPrice`](https://github.com/gnolang/gno/blob/7e6955997/tm2/pkg/sdk/auth/keeper.go#L361) · [↗](../../../../../.worktrees/gno-review-5916/tm2/pkg/sdk/auth/keeper.go#L361) and `calcBlockGasPrice` skipped `gasUsed == 0`. After, the decrease step is floored at `1` via [`diff := maxBig(num, bigOne)`](https://github.com/gnolang/gno/blob/7e6955997/tm2/pkg/sdk/auth/keeper.go#L453) · [↗](../../../../../.worktrees/gno-review-5916/tm2/pkg/sdk/auth/keeper.go#L453), still clamped at `InitialGasPrice` on the next line, and the empty-block guards are removed so `gasUsed == 0` routes through the decrease branch. The load-bearing constraint is that the decay is deterministic: all arithmetic is `math/big`, `gasUsed` is the `int64` meter reading, no floats reach the state transition.

## Critical (must fix)
None.

## Warnings (should fix)
None. The consensus-break is intended and already flagged in the PR body; the merge-path decision is the reason for the NEEDS DISCUSSION verdict, not a code defect.

## Nits
- **[defensive branch can't trigger]** [`keeper.go:370`](https://github.com/gnolang/gno/blob/7e6955997/tm2/pkg/sdk/auth/keeper.go#L370) · [↗](../../../../../.worktrees/gno-review-5916/tm2/pkg/sdk/auth/keeper.go#L370) — the new `if gasUsed < 0` guard is unreachable: `basicGasMeter` and `infiniteGasMeter` both clamp `consumed` to `≥ 0` (`RefundGas` resets a negative to 0), so `GasConsumed()` never returns a negative. Harmless belt-and-suspenders; the comment reads as if a negative reading is a real state. Not posted (no action needed).

## Missing Tests
- **[consensus change ships untested]** [`keeper.go:370`](https://github.com/gnolang/gno/blob/7e6955997/tm2/pkg/sdk/auth/keeper.go#L370) · [↗](../../../../../.worktrees/gno-review-5916/tm2/pkg/sdk/auth/keeper.go#L370) — no test drives `UpdateGasPrice` on an empty block, so the guard change here has no direct coverage.
  <details><summary>details</summary>

  The committed regression test [`TestCalcBlockGasPriceRatchet`](https://github.com/gnolang/gno/blob/7e6955997/tm2/pkg/sdk/auth/keeper_test.go#L189) · [↗](../../../../../.worktrees/gno-review-5916/tm2/pkg/sdk/auth/keeper_test.go#L189) exercises `calcBlockGasPrice` directly and never enters `UpdateGasPrice`. The line that actually changes consensus behavior is the guard relaxation from `gasUsed <= 0` to `gasUsed < 0`: that is what makes an empty block reach `calcBlockGasPrice` and persist a decayed price instead of returning early. A test that seeds a price above the floor, runs `UpdateGasPrice` with a zero-gas block meter, and asserts the stored price dropped locks that behavior in. Verified: on the fix it decays 10 → 9; restoring the old `gasUsed <= 0` guard leaves it stuck at 10 ([repro](comment_claude-opus-4-8.md)). Fix: add the `UpdateGasPrice` empty-block test in the details of the comment draft.
  </details>

## Suggestions
None.

## Verified
- Revert-proof, CI-invisible: driving `UpdateGasPrice` end-to-end on an empty block (zero-gas block meter) with a price seeded at 10 decays the stored price to 9; restoring the pre-fix guard `gasUsed <= 0` at [`keeper.go:370`](https://github.com/gnolang/gno/blob/7e6955997/tm2/pkg/sdk/auth/keeper.go#L370) · [↗](../../../../../.worktrees/gno-review-5916/tm2/pkg/sdk/auth/keeper.go#L370) leaves it stuck at 10. No committed test covers this path.
- Determinism: the state transition uses only `math/big` and the `int64` `GasConsumed()` reading; the `0.14` float in `TestCalcBlockGasPriceRatchet` is test-only and never reaches the keeper.
- `math/big` aliasing in the reused `num`/`diff` operands is safe: `num.Sub(lastPriceInt, diff)` and `num.Add(lastPriceInt, diff)` produce the intended result even when `diff` aliases `num`.
- Tests green at 7e6955997: `TestCalcBlockGasPrice`, `TestCalcBlockGasPriceRatchet`, `TestMax`, and the full `tm2/pkg/sdk/auth` suite; CI is green on all checks.

## Open questions
- Increase branch is still uncapped ([`keeper.go:438`](https://github.com/gnolang/gno/blob/7e6955997/tm2/pkg/sdk/auth/keeper.go#L438) · [↗](../../../../../.worktrees/gno-review-5916/tm2/pkg/sdk/auth/keeper.go#L438) `// XXX should we cap it`): under sustained maximal congestion the price grows unbounded until the `IsInt64()` guard panics the state transition. Author explicitly left capping out of scope; not posted.
- Empty-block decay now reaches `calcBlockGasPrice` for chains with `Block.MaxGas <= 0` too. `target = maxGas*ratio/100` floors to `≤ 0` via `big.Int.Div`; for `maxGas == -1` (unlimited) an empty block computes `target = -1` and takes the increase branch, nudging the price up instead of down. Degenerate config only (unlimited block gas with an active dynamic fee market), no panic. Not posted.
