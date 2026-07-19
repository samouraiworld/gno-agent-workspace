# PR [#5975](https://github.com/gnolang/gno/pull/5975): feat(examples): add parimutuel payout math library

URL: https://github.com/gnolang/gno/pull/5975
Author: zardozmonopoly | Base: master | Files: 3 | +188 -0
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: e3b7a7934 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5975 e3b7a7934`

**TL;DR:** Adds a small pure-package library that computes how much a winning bet is owed out of a shared betting pool, with an optional house commission and a market-implied probability helper. All amounts are plain `int64` and all failures are panics.

**Verdict: REQUEST CHANGES** — the shared `mulDiv` helper multiplies before dividing with no overflow guard, so payouts silently wrap to negative at pools above roughly 3037 GNOT, and the rake wrapper hard-panics whenever the winning outcome holds most of the pool (2 Critical, 1 Warning, 3 Nits, 1 Missing test, 1 Suggestion).

## Summary

Three exported functions over one private helper. [`CalculatePayout`](https://github.com/gnolang/gno/blob/e3b7a7934/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L9-L22) · [↗](../../../../../.worktrees/gno-review-5975/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L9-L22) returns `userBet * totalPool / winningPool`, [`CalculatePayoutWithRake`](https://github.com/gnolang/gno/blob/e3b7a7934/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L32-L41) · [↗](../../../../../.worktrees/gno-review-5975/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L32-L41) shaves a basis-point commission off the total first, and [`ImpliedProbabilityBps`](https://github.com/gnolang/gno/blob/e3b7a7934/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L51-L60) · [↗](../../../../../.worktrees/gno-review-5975/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L51-L60) returns an outcome's pool share in basis points. Every path funnels through [`mulDiv`](https://github.com/gnolang/gno/blob/e3b7a7934/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L72-L77) · [↗](../../../../../.worktrees/gno-review-5975/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L72-L77), which evaluates `a*b` in `int64` before dividing, so the intermediate product is the real magnitude limit, not the individual amounts. Two defects follow: the product wraps with no signal at pools a testnet market could plausibly reach, and the rake wrapper hands its post-commission `netPool` to `CalculatePayout`, which then compares the pre-commission `winningPool` against it and panics on the unit mismatch. Packages under `examples/` are [pre-deployed to gno.land testnets](https://github.com/gnolang/gno/blob/e3b7a7934/examples/README.md?plain=1#L7-L8) · [↗](../../../../../.worktrees/gno-review-5975/examples/README.md#L7-L8), so both land on chain as an importable library.

## Examples

Values observed at e3b7a7934; the want column is what the doc comments describe.

| Call | Returns | Want |
|---|---|---|
| `CalculatePayout(3037000499, 3037000499, 3037000499)` | `3037000499` | `3037000499` |
| `CalculatePayout(3037000500, 3037000500, 3037000500)` | `-3037000499` | `3037000500` |
| `CalculatePayout(5e9, 1e10, 5e9)` | `-1068046444` | `10000000000` |
| `ImpliedProbabilityBps(922337203685478, 922337203685478)` | `-9999` | `10000` |
| `CalculatePayoutWithRake(950, 1000, 950, 500)` | `950` | `950` |
| `CalculatePayoutWithRake(951, 1000, 951, 500)` | panic | `950` |
| `CalculatePayoutWithRake(1000, 1000, 1000, 100)` | panic | `990` |
| `CalculatePayoutWithRake(100, -1000, 400, 0)` | `0` | panic |

## Critical (must fix)

- **[payouts wrap to negative with no signal]** [`examples/gno.land/p/demo/parimutuel/parimutuel.gno:72-77`](https://github.com/gnolang/gno/blob/e3b7a7934/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L72-L77) · [↗](../../../../../.worktrees/gno-review-5975/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L72-L77) — `mulDiv` computes `a*b` before dividing with no overflow check, so a winner-takes-all payout returns a negative number once the pool reaches 3,037,000,500 ugnot.
  <details><summary>details</summary>

  The limit is on the product `userBet * totalPool`, not on either amount, so the guard the doc comment describes measures the wrong quantity: it reasons about whether pools "fit comfortably within int64" while the value that overflows is their product. For the winner-takes-all shape the wrap starts at pool 3,037,000,500 ugnot, about 3037 GNOT, and `CalculatePayout(4e9, 4e9, 4e9)` returns `-611686018` instead of `4000000000`. `ImpliedProbabilityBps` has the same defect through `mulDiv(outcomePool, 10000, totalPool)` and wraps at outcomePool 922,337,203,685,478 ugnot, returning `-9999` where 10000 is correct. Nothing panics and nothing is logged, so a realm distributing prizes writes the wrapped value straight into a banker send. The exported doc comments state no magnitude limit at all, and the advice to "pre-scale inputs or use a bignum-backed alternative" sits on an unexported helper a caller never reads. Fix: route the multiplications through [`overflow.Mul64p`](https://github.com/gnolang/gno/blob/e3b7a7934/gnovm/stdlibs/math/overflow/overflow_generated.gno#L582-L589) · [↗](../../../../../.worktrees/gno-review-5975/gnovm/stdlibs/math/overflow/overflow_generated.gno#L582-L589) so an out-of-range product panics rather than wrapping, the way [`p/demo/tokens/grc20`](https://github.com/gnolang/gno/blob/e3b7a7934/examples/gno.land/p/demo/tokens/grc20/token.gno#L193-L194) · [↗](../../../../../.worktrees/gno-review-5975/examples/gno.land/p/demo/tokens/grc20/token.gno#L193-L194) already does. [repro](comment_claude-opus-4-8.md)
  </details>

- **[a dominant winning outcome cannot be paid out]** [`examples/gno.land/p/demo/parimutuel/parimutuel.gno:40`](https://github.com/gnolang/gno/blob/e3b7a7934/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L40) · [↗](../../../../../.worktrees/gno-review-5975/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L40) — `CalculatePayoutWithRake` passes the post-rake `netPool` as `totalPool`, so `CalculatePayout` compares the pre-rake `winningPool` against it and panics for any outcome holding more than `10000-rakeBps` of the pool.
  <details><summary>details</summary>

  `winningPool` is a gross quantity: the stakes actually placed on the winning outcome, drawn from the full pool. `netPool` is that pool after the commission. Comparing them mixes units, so the [`winningPool cannot exceed totalPool`](https://github.com/gnolang/gno/blob/e3b7a7934/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L15-L17) · [↗](../../../../../.worktrees/gno-review-5975/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L15-L17) guard fires on legitimate markets. With a 5% rake on a pool of 1000, a winning pool of 950 returns 950 and 951 panics. Winner-takes-all, which the package's own [first test](https://github.com/gnolang/gno/blob/e3b7a7934/examples/gno.land/p/demo/parimutuel/parimutuel_test.gno#L5-L10) · [↗](../../../../../.worktrees/gno-review-5975/examples/gno.land/p/demo/parimutuel/parimutuel_test.gno#L5-L10) treats as a normal case unraked, panics under any nonzero rake. In a realm this aborts the settlement transaction, so the market cannot be resolved and the stakes stay locked. Fix: scale the payout by the net pool without re-validating the gross `winningPool` against it. [repro](comment_claude-opus-4-8.md)
  </details>

## Warnings (should fix)

- **[invalid input silently returns zero]** [`examples/gno.land/p/demo/parimutuel/parimutuel.gno:36-39`](https://github.com/gnolang/gno/blob/e3b7a7934/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L36-L39) · [↗](../../../../../.worktrees/gno-review-5975/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L36-L39) — `CalculatePayoutWithRake` computes `netPool` before any input validation, so a negative `totalPool` short-circuits to 0 instead of panicking the way `CalculatePayout` does.
  <details><summary>details</summary>

  [`assertNonNegative(totalPool, ...)`](https://github.com/gnolang/gno/blob/e3b7a7934/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L11) · [↗](../../../../../.worktrees/gno-review-5975/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L11) only runs inside `CalculatePayout`, which the rake wrapper never reaches when `netPool <= 0`. `CalculatePayoutWithRake(100, -1000, 400, 0)` returns 0. The two sibling functions therefore disagree on the same input: one rejects it, one reports "nothing to distribute". A realm that derives `totalPool` from a subtraction and gets the sign wrong sees zero payouts everywhere instead of a panic pointing at the bad input. Fix: validate `userBet` and `totalPool` before computing `netPool`, so the exhausted-pool return covers only a genuinely exhausted pool. [repro](comment_claude-opus-4-8.md)
  </details>

## Nits

- **[dead branch reads as a live guard]** [`examples/gno.land/p/demo/parimutuel/parimutuel.gno:73-75`](https://github.com/gnolang/gno/blob/e3b7a7934/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L73-L75) · [↗](../../../../../.worktrees/gno-review-5975/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L73-L75) — the `c == 0` panic in `mulDiv` is unreachable: all three call sites fix the divisor first.
  <details><summary>details</summary>

  [`CalculatePayout`](https://github.com/gnolang/gno/blob/e3b7a7934/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L12-L14) · [↗](../../../../../.worktrees/gno-review-5975/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L12-L14) rejects `winningPool <= 0` before calling, [`ImpliedProbabilityBps`](https://github.com/gnolang/gno/blob/e3b7a7934/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L52-L54) · [↗](../../../../../.worktrees/gno-review-5975/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L52-L54) rejects `totalPool <= 0`, and [the rake path](https://github.com/gnolang/gno/blob/e3b7a7934/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L36) · [↗](../../../../../.worktrees/gno-review-5975/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L36) passes the constant 10000. The branch reads as protection the helper does not need, which invites a future caller to skip its own divisor check. Fix: drop it, or keep it and say in the comment that it is a backstop for future call sites.
  </details>

- **[test name asserts a property the function lacks]** [`examples/gno.land/p/demo/parimutuel/parimutuel_test.gno:84-91`](https://github.com/gnolang/gno/blob/e3b7a7934/examples/gno.land/p/demo/parimutuel/parimutuel_test.gno#L84-L91) · [↗](../../../../../.worktrees/gno-review-5975/examples/gno.land/p/demo/parimutuel/parimutuel_test.gno#L84-L91) — `TestImpliedProbabilityBpsSumsToTotal` only picks pool splits that divide exactly, so it reads as a general invariant that truncation breaks.
  <details><summary>details</summary>

  `ImpliedProbabilityBps` rounds down, so three outcomes of 1000 in a 3000 pool each return 3333 and sum to 9999, confirmed behaviorally against both the gno and Go runs. A realm author reading this test will assume implied probabilities always sum to 10000 and may build a settlement check on it. Fix: name the test after the exact-split case it covers, and add a truncating split so the residual is documented.
  </details>

- **[no package doc]** [`examples/gno.land/p/demo/parimutuel/parimutuel.gno:1`](https://github.com/gnolang/gno/blob/e3b7a7934/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L1) · [↗](../../../../../.worktrees/gno-review-5975/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L1) — the package has no `// Package parimutuel ...` comment, so the generated docs page opens on nothing; sibling [`p/demo/svg`](https://github.com/gnolang/gno/blob/e3b7a7934/examples/gno.land/p/demo/svg/doc.gno#L2) · [↗](../../../../../.worktrees/gno-review-5975/examples/gno.land/p/demo/svg/doc.gno#L2) ships one in a `doc.gno`.

## Missing Tests

- **[the failing shapes are the untested ones]** [`examples/gno.land/p/demo/parimutuel/parimutuel_test.gno:5-109`](https://github.com/gnolang/gno/blob/e3b7a7934/examples/gno.land/p/demo/parimutuel/parimutuel_test.gno#L5-L109) · [↗](../../../../../.worktrees/gno-review-5975/examples/gno.land/p/demo/parimutuel/parimutuel_test.gno#L5-L109) — every case uses a pool of at most 1000, and no raked case puts more than 40% of the total on the winner, which is exactly the region where both Critical findings are invisible.
  <details><summary>details</summary>

  The suite's largest pool is 1000, six orders of magnitude below the overflow threshold, and its largest `winningPool` relative to `totalPool` under a rake is 40%, well under the panic threshold of `10000-rakeBps`. Two ready-to-add files close the gap and both fail red at e3b7a7934: [`tests/overflow_test.gno`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5975-parimutuel-payout-math/1-e3b7a7934/tests/overflow_test.gno) · [↗](tests/overflow_test.gno) asserts that a 4000 GNOT winner-takes-all pool and a whole-pool implied probability return their true values, and [`tests/rake_test.gno`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5975-parimutuel-payout-math/1-e3b7a7934/tests/rake_test.gno) · [↗](tests/rake_test.gno) asserts that a raked payout on a dominant winning pool returns the reduced amount. A third case belongs with the Nit above: a truncating three-way split, asserting the 9999 residual rather than 10000.
  </details>

## Suggestions

- **[callers still hand-roll the commission]** [`examples/gno.land/p/demo/parimutuel/parimutuel.gno:32`](https://github.com/gnolang/gno/blob/e3b7a7934/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L32) · [↗](../../../../../.worktrees/gno-review-5975/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L32) — nothing exported returns the rake amount, so a realm routing the house cut recomputes `totalPool * rakeBps / 10000` inline.
  <details><summary>details</summary>

  The package's stated purpose is to stop realms rewriting this formula inline, but the commission itself is only available as the difference the wrapper keeps internally at [line 36](https://github.com/gnolang/gno/blob/e3b7a7934/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L36) · [↗](../../../../../.worktrees/gno-review-5975/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L36). Every caller that pays a house address needs that number, so each one re-derives it, unguarded, with its own rounding. Exporting a `RakeAmount(totalPool, rakeBps int64) int64` would keep the two figures consistent and give the overflow guard one place to live.
  </details>

## Verified

- Gno and Go agree byte for byte on the wrapped values: a Go port of the three functions returns the same `-611686018`, `-1068046444` and `-9999` the gno run produces, so the overflow is deterministic across validators rather than a consensus split. Port and observed output: [`tests/go_parity.go`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5975-parimutuel-payout-math/1-e3b7a7934/tests/go_parity.go) · [↗](tests/go_parity.go).
- Payouts never over-distribute inside the safe range: sweeping every three-way split of a winning pool for all totals up to 120 found zero splits whose summed payouts exceed `totalPool`, with at most 2 units of dust left behind. Same file.
- Overflow thresholds are exact, not estimates: `CalculatePayout(p, p, p)` returns `p` at `p = 3037000499` and `-3037000499` at `p = 3037000500`; `ImpliedProbabilityBps(o, o)` returns 10000 at `o = 922337203685477` and `-9999` at `o = 922337203685478`.
- The rake panic threshold is `winningPool > netPool`, not only the winner-takes-all case: with a 5% rake on a pool of 1000, 950 returns 950 and 951 panics.
- `go run ./gnovm/cmd/gno lint ./gno.land/p/demo/parimutuel` is clean, and rejects an injected undefined symbol, so it is typechecking rather than passing vacuously. The 13 shipped tests pass at e3b7a7934.

## Open questions

- `examples/README.md` still [recommends `p/demo` for generic components](https://github.com/gnolang/gno/blob/e3b7a7934/examples/README.md?plain=1#L36-L38) · [↗](../../../../../.worktrees/gno-review-5975/examples/README.md#L36-L38), but no general-purpose library has landed there since September 2025: new contributions go to a personal namespace (`p/jeronimoalbi/murmur3`, `p/jeronimoalbi/bitset`) or to `p/nt/<name>/v0`. Not posted: the documented guidance is on the author's side, and the placement question is a maintainer call, not a defect.
- The package caps magnitudes at `int64` by design. If the maintainers want a pool-math library usable for aggregate treasury figures rather than single markets, the same three functions over `p/nt/uint256` would remove the ceiling entirely. Not posted: out of scope for a first cut, and the overflow guard is the fix this PR needs.
