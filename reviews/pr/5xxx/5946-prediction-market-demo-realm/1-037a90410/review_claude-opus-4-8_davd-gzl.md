# PR [#5946](https://github.com/gnolang/gno/pull/5946): feat(examples): add prediction market demo realm

URL: https://github.com/gnolang/gno/pull/5946
Author: zardozmonopoly | Base: master | Files: 3 | +236 -0
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 037a90410 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5946 037a90410`

**TL;DR:** Adds a demo realm where an admin opens a sports-style market, users bet ugnot on Home, Away or Draw, and after the admin declares the winner each winner claims a share of the whole pot proportional to their stake. Bets are held by the realm itself and paid back out with the banker.

**Verdict: REQUEST CHANGES** — the package does not typecheck against the current `avl` API so nothing in it builds, and once it does build the payout multiplication wraps `int64` and the realm has no path to return coins when the winning outcome drew no bets (3 Critical, 1 Warning, 3 Nits, 1 Missing test, 3 Suggestions).

## Summary

One realm file, one test file. [`CreateMarket`](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L49-L63) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L49) and [`ResolveMarket`](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L99-L117) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L99) are admin-only, [`PlaceBet`](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L65-L97) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L65) reads the attached coins and records a bet, and [`ClaimWinnings`](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L119-L153) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L119) pays out and zeroes the claimed bets. Escrow is implicit: the VM keeper moves a `MsgCall` send into the realm's package address [before the call runs](https://github.com/gnolang/gno/blob/037a90410/gno.land/pkg/sdk/vm/keeper.go#L850) · [↗](../../../../../.worktrees/gno-review-5946/gno.land/pkg/sdk/vm/keeper.go#L850), so `PlaceBet` only has to record the amount, and `ClaimWinnings` is the sole exit. That single exit is where the money findings concentrate: it wraps on multiplication and it refuses to run at all when the declared winner has an empty pool. The package also does not compile, so CI's `ci / examples` job fails as soon as it is allowed to run; the checks currently visible on the PR are the bot's, and the test jobs are [pending initial approval](https://github.com/gnolang/gno/pull/5946#issuecomment-4953814027). Realms under `examples/` are [pre-deployed to gno.land testnets](https://github.com/gnolang/gno/blob/037a90410/examples/README.md?plain=1#L7-L9) · [↗](../../../../../.worktrees/gno-review-5946/examples/README.md#L7), so this holds real testnet ugnot once merged.

## Examples

Payout for a sole Home bettor when Home and Away each hold the same stake. Observed at 037a90410 with the build fixed; `want` is the stake plus the losing pool.

| Stake per side | Payout returned | Want |
|---|---|---|
| 1,000,000 ugnot | `2000000` | `2000000` |
| 3,000,000,000 ugnot | `-148914691` | `6000000000` |
| 5,000,000,000 ugnot | `-1068046444`, claim panics | `10000000000` |

## Glossary

- crossing / `cross`: a call into `func F(cur realm, ...)`; the callee reads its caller through `cur.Previous()`.
- banker: `chain/banker`, the stdlib API for sending coins out of a realm.
- realm: a stateful package under `r/` whose objects persist across transactions.
- unsafe: `chain/runtime/unsafe`, the quarantined stack-walking origin primitives.

## Critical (must fix)

- **[package does not build]** [`examples/gno.land/r/demo/predictionmarket/predictionmarket.gno:74`](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L74) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L74) — `avl.Tree.Get` returns one value, so all five two-value assignments fail to typecheck and `gno test` reports zero passing tests.
  <details><summary>details</summary>

  [`Get`](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/p/nt/avl/v0/tree.gno#L58) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/p/nt/avl/v0/tree.gno#L58) has returned a single `any` since [#5314](https://github.com/gnolang/gno/pull/5314) collapsed the found flag into a nil result; the two-value form the code uses is the older signature. The sites are lines [74](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L74) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L74), [107](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L107) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L107) and [121](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L121) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L121) in the realm, plus [14](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/r/demo/predictionmarket/predictionmarket_test.gno#L14) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket_test.gno#L14) and [31](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/r/demo/predictionmarket/predictionmarket_test.gno#L31) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket_test.gno#L31) in the test. Both `gno test` and `gno lint` stop at the same three realm errors, so the four tests the PR body describes have never run. The `ci / examples` job [runs on any `examples/**` change](https://github.com/gnolang/gno/blob/037a90410/.github/workflows/ci-dir-examples.yml#L7-L10) · [↗](../../../../../.worktrees/gno-review-5946/.github/workflows/ci-dir-examples.yml#L7) and will fail once a maintainer releases it. Fix: drop the `exists` result and test the returned value against nil. [repro](comment_claude-opus-4-8.md)
  </details>

- **[winnings become unclaimable above roughly 3000 GNOT]** [`examples/gno.land/r/demo/predictionmarket/predictionmarket.gno:143`](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L143) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L143) — `b.amount * totalPool` wraps `int64` before the division, so the payout goes negative and the banker send panics, leaving the whole pot stuck in the realm.
  <details><summary>details</summary>

  The magnitude limit is on the product, not on either amount, so a market only needs its stake and its pot to reach the low billions of ugnot together. With 5,000,000,000 ugnot on Home and the same on Away, the sole Home bettor's payout evaluates to `-1068046444` and [`SendCoins`](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L152) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L152) aborts with `non-positive coin amount: -1068046444`. Since `ClaimWinnings` is the only path that moves coins out, every bettor in that market is locked out permanently. The same expression evaluated in Go returns the identical wrapped values, so this is plain `int64` overflow rather than a VM divergence; see [`payout_go_parity_test.go`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5946-prediction-market-demo-realm/1-037a90410/tests/payout_go_parity_test.go) · [↗](tests/payout_go_parity_test.go). Fix: route the multiplication through [`math/overflow`](https://github.com/gnolang/gno/blob/037a90410/gnovm/stdlibs/math/overflow/overflow_generated.gno#L583-L589) · [↗](../../../../../.worktrees/gno-review-5946/gnovm/stdlibs/math/overflow/overflow_generated.gno#L583) so an out-of-range product panics instead of wrapping, the way [`p/demo/tokens/grc20`](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/p/demo/tokens/grc20/token.gno#L193-L194) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/p/demo/tokens/grc20/token.gno#L193) already does, or keep the intermediate in range by dividing first. [repro](comment_claude-opus-4-8.md)
  </details>

- **[a winner with no backers strands every bet]** [`examples/gno.land/r/demo/predictionmarket/predictionmarket.gno:130-133`](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L130-L133) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L130) — when the declared outcome drew no bets the claim panics `no winning pool`, and no other entry point can move the escrow, so the coins stay in the realm forever.
  <details><summary>details</summary>

  [`ResolveMarket`](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L104-L106) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L104) accepts any outcome in 0..2 with no check that anyone backed it, and a real match ending in a draw when both bettors split Home and Away reaches this state on the first try. The single [`SendCoins`](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L152) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L152) call in the package sits after that panic, so there is no refund, no admin sweep and no cancel. A market the admin simply never resolves has the same outcome, since [`ClaimWinnings`](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L126-L128) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L126) rejects unresolved markets. Confirmed behaviorally: with the build fixed, a market resolved to `OutcomeDraw` while only Home and Away hold stakes panics `no winning pool` on every claim. Fix: give bettors their stake back when the winning pool is empty. [repro](comment_claude-opus-4-8.md)
  </details>

## Warnings (should fix)

- **[nothing states that the admin can take the pot]** [`examples/gno.land/r/demo/predictionmarket/predictionmarket.gno:99-103`](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L99-L103) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L99) — the admin declares the winner with no deadline and may also bet, so the admin can bet, then resolve their own outcome and claim the pot; the package carries no doc comment saying so.
  <details><summary>details</summary>

  There is no betting cutoff, no oracle and no dispute window anywhere in the file: `ResolveMarket` is callable at any height and [`PlaceBet`](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L65-L72) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L65) admits any externally owned account including the [admin set at deploy time](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L44-L47) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L44). A reader copying this realm as a starting point inherits that property silently. Fix: state the trust assumption in a package doc comment. [repro](comment_claude-opus-4-8.md)
  </details>

## Nits

- **[the two functions that move money print nothing]** [`examples/gno.land/r/demo/predictionmarket/predictionmarket.gno:155`](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L155) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L155) — `Render` ignores its `path` argument, so a market page, a bettor's open positions and a claimable balance are all unreachable from the web view.

- **[market list appears in the wrong order]** [`examples/gno.land/r/demo/predictionmarket/predictionmarket.gno:61`](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L61) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L61) — the tree key is the decimal id, and [`Iterate`](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/p/nt/avl/v0/tree.gno#L89) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/p/nt/avl/v0/tree.gno#L89) walks keys as strings, so `Render` lists market 10 before market 2.
  <details><summary>details</summary>

  Deterministic, so not a consensus concern, just a display order that inverts once the realm passes nine markets. Fix: zero-pad the key.
  </details>

- **[no doc comments on any exported symbol]** [`examples/gno.land/r/demo/predictionmarket/predictionmarket.gno:15-21`](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L15-L21) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L15) — the outcome constants, all four exported functions and the package itself carry no comment, which matters more than usual for a realm meant to be read as an example. Not posted to the PR: no enabled linter checks this, and it is subsumed by the Warning asking for a package doc comment.

## Missing Tests

- **[the whole money path is untested]** [`examples/gno.land/r/demo/predictionmarket/predictionmarket_test.gno:53-62`](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/r/demo/predictionmarket/predictionmarket_test.gno#L53-L62) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket_test.gno#L53) — no test places a bet or completes a claim, so both overflow and the empty-winning-pool lock reach the on-chain realm unchallenged.
  <details><summary>details</summary>

  The four existing tests cover creation, resolution and two rejection paths. `PlaceBet` never appears, and `ClaimWinnings` appears only in the case where it is expected to panic, so nothing asserts that a winner is paid the right amount. The admin gate is also vacuous in the harness: `unsafe.OriginCaller()` and `cur.Previous().Address()` are both the empty address under `gno test`, so `admin == caller` holds by accident and a rejection test for a non-admin caller would not be exercising the gate as written. `PlaceBet` cannot be driven from an in-package test because [`AssertOriginCall`](https://github.com/gnolang/gno/blob/037a90410/gnovm/stdlibs/chain/runtime/native.go#L9-L27) · [↗](../../../../../.worktrees/gno-review-5946/gnovm/stdlibs/chain/runtime/native.go#L9) requires the first frame to be a message call, so the payout assertions seed the market directly and a txtar integration test is the right home for the bet path itself. Ready-to-add payout test: [`payout_overflow_test.gno`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5946-prediction-market-demo-realm/1-037a90410/tests/payout_overflow_test.gno) · [↗](tests/payout_overflow_test.gno), red at 037a90410 and green once the multiplication stays in range.
  </details>

## Suggestions

- **[a lost admin key ends the realm]** [`examples/gno.land/r/demo/predictionmarket/predictionmarket.gno:44-47`](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L44-L47) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L44) — `admin` is fixed at deploy and has no transfer path, so no market can ever be created or resolved again if that key is lost.
  <details><summary>details</summary>

  Combined with the Critical on unresolved markets, this also means every open market's escrow becomes permanently unreachable. An ownable helper such as [`p/nt/ownable/v0`](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/p/nt/ownable/v0/ownable.gno#L1) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/p/nt/ownable/v0/ownable.gno#L1) covers this and is the usual shape in `examples/`.
  </details>

- **[two-way markets still accept draw bets]** [`examples/gno.land/r/demo/predictionmarket/predictionmarket.gno:70-72`](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L70-L72) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L70) — the accepted outcomes are hardcoded here, so a market has no way to declare itself binary and a bettor can stake on Draw in a contest that cannot draw.
  <details><summary>details</summary>

  Those stakes are then guaranteed losers rather than rejected. A per-market outcome count on `Market` would let the range check reject them at bet time.
  </details>

- **[rounding dust accumulates with no sweep]** [`examples/gno.land/r/demo/predictionmarket/predictionmarket.gno:143`](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L143) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L143) — integer division truncates each payout, so a few ugnot per market stay in the realm with nothing able to reclaim them.
  <details><summary>details</summary>

  The magnitude is under one ugnot per winner, so this is an accounting note rather than a loss. Worth deciding explicitly whether the residue is a house edge or should be swept.
  </details>

## Verified

- The escrow is real rather than notional: the VM keeper transfers a `MsgCall` send into the realm's package address [before the call body runs](https://github.com/gnolang/gno/blob/037a90410/gno.land/pkg/sdk/vm/keeper.go#L850) · [↗](../../../../../.worktrees/gno-review-5946/gno.land/pkg/sdk/vm/keeper.go#L850), so `PlaceBet` recording the amount without an explicit transfer is correct.
- Applying only the `avl.Get` arity fix makes the package build and all four shipped tests pass, so the compile break is the sole build-level defect.
- The overflow reproduces end to end, not just in the arithmetic: with the build fixed, a claim at 5,000,000,000 ugnot per side panics inside `bankerSendCoins` with `non-positive coin amount: -1068046444`, and the same claim passes once the multiplication is reordered to divide first.
- Evaluating the payout expression in Go at the same inputs returns the identical wrapped values, so the wrap is `int64` semantics rather than a GnoVM divergence.
- `unsafe.OriginSend()` is paired with both [`runtime.AssertOriginCall()`](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L66) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L66) and [`cur.Previous().IsUserCall()`](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L67) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L67), exactly the pairing [OriginSend's doc comment requires](https://github.com/gnolang/gno/blob/037a90410/gnovm/stdlibs/chain/runtime/unsafe/unsafe.gno#L57-L63) · [↗](../../../../../.worktrees/gno-review-5946/gnovm/stdlibs/chain/runtime/unsafe/unsafe.gno#L57); admin authentication reads the threaded `cur`, not the stack-walking primitives. No caller-authentication defect found.
- `ClaimWinnings` zeroes each claimed bet [before](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L144) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L144) it calls the banker, and the banker send invokes no realm code, so there is no re-entrancy window on the payout path.
- Duplicate `ugnot` entries cannot reach the [bet-amount loop](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L85-L89) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L85): `MsgCall.ValidateBasic` rejects a send that [is not a valid `Coins`](https://github.com/gnolang/gno/blob/037a90410/gno.land/pkg/sdk/vm/msgs.go#L152-L154) · [↗](../../../../../.worktrees/gno-review-5946/gno.land/pkg/sdk/vm/msgs.go#L152), and validity requires sorted, positive, duplicate-free denoms.

## Open questions

- `ClaimWinnings` scans every bet in the market on each claim. Measured with the build fixed: 950k gas at 10 bets, 1.89M at 100, 28.9M at 1000, against a 3B default block gas limit, so it does not reach a denial-of-service at any plausible demo scale. Not posted; the measurement kills the concern.
- Cleared bets stay in `m.bets` forever, so a market's storage never shrinks after payout. Not posted; the realm has no pruning story either way and this is a design choice, not a defect.
- The same `amount * total / winning` overflow appears in the author's separate parimutuel library PR [#5975](https://github.com/gnolang/gno/pull/5975). Not posted here; raising it once, on the PR that owns the helper, is enough.
