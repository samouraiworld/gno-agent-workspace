# PR [#5840](https://github.com/gnolang/gno/pull/5840): feat: vesting account poc

URL: https://github.com/gnolang/gno/pull/5840
Author: julienrbrt | Base: master | Files: 13 | +1764 -13
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 9da3635c4 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5840 9da3635c4`

Round 2. Head advanced 3e4fca768 → 9da3635c4 (+5 PR commits plus two master merges): a feedback commit wired `ErrInvalidVestingSchedule` into `Validate`, deleted the unused `Proto*` constructors, switched `GetVestedCoins` from `overflow.Mul` (panic on int64 overflow) to `big.Int`, and fixed the empty `baseAcc.Coins` / `AccountNumber == 0` genesis bugs by setting both and pulling the number from a new `GetNextAccountNumber` interface method; proto was regenerated (clears the old red genproto2 check); `t.Parallel()` was added to the subtests (clears the old red lint check); a gas-restriction commit was added then reverted, so gas is paid from the full balance via `SendCoinsUnrestricted` again; and a new `TestInitChainer_VestingAccount` covers genesis account creation. All four of my round-1 findings carry (none were resolved by these commits); the round-1 on-chain missing-test narrows to transfer enforcement, since genesis creation is now covered. One new Warning: a genesis vesting account listed in the unrestricted whitelist panics node start.

**TL;DR:** Genesis accounts can now carry a vesting schedule, so their coins unlock gradually instead of being spendable from day one. Locked coins can pay gas and stay on the chain, but cannot be transferred out until they vest.

**Verdict: NEEDS DISCUSSION** — proof-of-concept; the core schedule math and keeper enforcement are correct and the round-1 maintainer feedback is addressed, but a genesis that whitelists a vesting account panics node start on an unchecked `*GnoAccount` assertion, the bank-level enforcement still has no test, and the scope decisions (spend-vs-send boundary, whether vesting accounts should be whitelist-eligible, genesis-only creation with no clawback) want maintainer sign-off before this leaves PoC.

## Summary
Adds three vesting account types to `tm2/pkg/std` (`BaseVestingAccount`, `ContinuousVestingAccount`, `DelayedVestingAccount`) plus a `VestingSchedule`, registers them with amino, and wires genesis to build them from a `Balance.Vesting` field. The bank keeper enforces the lock at its single subtraction chokepoint `SubtractCoins`: a transfer of more than the spendable (vested + non-vesting) amount is rejected with `VestingLockedCoinsError`. Gas and system refunds go through `SendCoinsUnrestricted`, which deliberately bypasses the check, matching the issue's "limit sending, not spending" decision ([#5798](https://github.com/gnolang/gno/issues/5798)); the round-1 experiment to restrict gas to spendable coins was reverted for that reason. Once a schedule fully vests, the account is rewritten to a plain `BaseAccount` so later transfers skip vesting entirely.

```
SubtractCoins (restricted)        SendCoinsUnrestricted (gas, refunds)
  ├─ upgradeVestingAccount          ├─ subtractCoinsUnrestricted
  │    fully vested? -> BaseAccount  │    upgrades if fully vested
  └─ spendable >= amt ? ok : ERR    └─ no spendable check  -> always ok
        ▲                                   ▲
  bank.SendCoins, InputOutputCoins,   ante gas (auth/ante.go),
  banker.RemoveCoin (vm/builtins)     storage deposit refund (vm/keeper)
```

## Examples
Genesis balance entry forms (`gno.land/pkg/gnoland/balance.go` `Parse`):

| Entry | Effect |
|-------|--------|
| `g1…=100ugnot` | plain account, all spendable |
| `g1…=100ugnot;vesting=50ugnot,100,200` | continuous: 50 vest linearly over `[100,200]`, other 50 spendable now |
| `g1…=100ugnot;vesting=50ugnot,0,200;type=delayed` | delayed: 50 locked until time 200, then all unlock at once |

## Glossary
- vesting account: `std.Account` whose coins unlock on a continuous or delayed (cliff) schedule; created only at genesis.
- spendable coins: total balance minus currently locked coins; the cap `SubtractCoins` enforces.
- restricted denom: denom-based transfer lock via the bank param, gated by the account's token-lock-whitelist bit; orthogonal to vesting's schedule-based lock.
- banker: stdlib coin API; its `RemoveCoin` routes through `SubtractCoins`, so realm-initiated burns/sends are also vesting-checked.

## Fix
Not a bugfix; additive feature. Enforcement lives only in `SubtractCoins` ([`tm2/pkg/sdk/bank/keeper.go:238-273`](https://github.com/gnolang/gno/blob/9da3635c4/tm2/pkg/sdk/bank/keeper.go#L238-L273) · [↗](../../../../../.worktrees/gno-review-5840/tm2/pkg/sdk/bank/keeper.go#L238)), the one place all restricted transfers subtract, while gas and refunds take the parallel `subtractCoinsUnrestricted` path that skips the check.

## Critical (must fix)
None.

## Warnings (should fix)
- **[genesis can crash node start with a cryptic panic]** [`gno.land/pkg/gnoland/app.go:685-694`](https://github.com/gnolang/gno/blob/9da3635c4/gno.land/pkg/gnoland/app.go#L685-L694) · [↗](../../../../../.worktrees/gno-review-5840/gno.land/pkg/gnoland/app.go#L691) — `applyUnrestrictedAddrs` does an unchecked `acc.(*GnoAccount)`; a genesis vesting account listed in `Auth.Params.UnrestrictedAddrs` makes `InitChain` panic with `interface conversion: std.Account is *std.ContinuousVestingAccount, not *gnoland.GnoAccount`.
  <details><summary>details</summary>

  This PR introduces the first genesis account type that is not a `*GnoAccount` (vesting accounts are `*std.ContinuousVestingAccount` / `*std.DelayedVestingAccount`, embedding `std.BaseAccount`). `applyUnrestrictedAddrs` runs after `applyBalance` and force-asserts every whitelisted address to `*GnoAccount` at [line 691](https://github.com/gnolang/gno/blob/9da3635c4/gno.land/pkg/gnoland/app.go#L691), so a vesting address on the whitelist panics node start with no clean genesis error. The sibling runtime check `canSendCoins` handles the same shape gracefully via a checked interface assertion `acc.(std.AccountUnrestricter)` ([`tm2/pkg/sdk/bank/keeper.go:136-140`](https://github.com/gnolang/gno/blob/9da3635c4/tm2/pkg/sdk/bank/keeper.go#L136-L140) · [↗](../../../../../.worktrees/gno-review-5840/tm2/pkg/sdk/bank/keeper.go#L136)), treating a vesting account as not-whitelisted. The config is realistic: vesting accounts hold the exact launch allocations a genesis author would want on the unrestricted whitelist so they can move their vested portion during the global GNOT token-lock. Reproduced on 9da3635c4, see [repro](comment_claude-opus-4-8.md). Fix: mirror the checked `AccountUnrestricter` assertion (skip or return a clean genesis error), or make vesting accounts carry the whitelist bit.

  Note the deeper design point behind this: since vesting accounts don't implement `AccountUnrestricter`, `canSendCoins` never treats them as whitelisted, so during a global GNOT lock a vesting account cannot transfer even its vested restricted-denom coins and cannot be whitelisted to do so without hitting this panic. That is a design decision for the [#5798](https://github.com/gnolang/gno/issues/5798) launch-allocation use case, tracked in Open questions.
  </details>

- **[the feature's core enforcement has no test]** [`tm2/pkg/sdk/bank/keeper.go:238-273`](https://github.com/gnolang/gno/blob/9da3635c4/tm2/pkg/sdk/bank/keeper.go#L238-L273) · [↗](../../../../../.worktrees/gno-review-5840/tm2/pkg/sdk/bank/keeper.go#L238) — `SubtractCoins` vesting enforcement, the upgrade-to-`BaseAccount` swap, and the `SendCoinsUnrestricted` bypass are exercised by no test in the bank package.
  <details><summary>details</summary>

  All vesting tests live in `tm2/pkg/std` and test the pure schedule math (`GetVestedCoins`, `SpendableCoins`); the new `TestInitChainer_VestingAccount` covers genesis account creation but no transfer. Nothing in the bank package tests that the keeper actually blocks a locked transfer, allows the unlocked portion, lets gas through `SendCoinsUnrestricted` while coins are locked, or upgrades the account once vested. These are the security-relevant behaviors of the PR, and they are the integration the std-layer tests cannot cover. I verified all four hold on 9da3635c4 with a keeper-level probe (see [repro](comment_claude-opus-4-8.md)), so this is a coverage gap, not a defect. Fix: add a bank-package test asserting block-when-locked, allow-unlocked-portion, unrestricted-bypass, and upgrade-after-full-vest.
  </details>

## Nits
- [`tm2/pkg/std/vesting_account.go:31-45`](https://github.com/gnolang/gno/blob/9da3635c4/tm2/pkg/std/vesting_account.go#L31-L45) · [↗](../../../../../.worktrees/gno-review-5840/tm2/pkg/std/vesting_account.go#L31) — `VestingSchedule.Validate` requires `StartTime < EndTime` even for delayed schedules, where `StartTime` is ignored (`GetStartTime` returns 0, vesting is a cliff at `EndTime`). A delayed genesis entry must still supply a start earlier than the end or it is rejected. Harmless, but the constraint is meaningless for the delayed case; confirmed behaviorally at 9da3635c4: `NewDelayedVestingAccount` with `StartTime:0, EndTime:200` validates fine, and `StartTime:200, EndTime:100` is rejected.
- [`tm2/pkg/sdk/bank/keeper.go:280-304`](https://github.com/gnolang/gno/blob/9da3635c4/tm2/pkg/sdk/bank/keeper.go#L280-L304) · [↗](../../../../../.worktrees/gno-review-5840/tm2/pkg/sdk/bank/keeper.go#L280) — `subtractCoinsUnrestricted` calls `upgradeVestingAccount(ctx, acc)` and discards the return, then `SetCoins` re-fetches the account and writes again. The upgrade preserves coins exactly, so balances are correct, but the account is fetched and `SetAccount`-ed twice in one call. Cosmetic.

## Missing Tests
- **[on-chain enforcement]** [`gno.land/pkg/gnoland/app.go:653-679`](https://github.com/gnolang/gno/blob/9da3635c4/gno.land/pkg/gnoland/app.go#L653-L679) · [↗](../../../../../.worktrees/gno-review-5840/gno.land/pkg/gnoland/app.go#L653) — no txtar or app-boot test drives a real transfer being blocked before vesting and allowed after, across block time, on a node built from a genesis vesting balance.
  <details><summary>details</summary>

  `TestInitChainer_VestingAccount` now proves `applyBalance` builds the vesting account and it survives commit/query, closing the round-1 genesis-creation gap. Still untested end to end: genesis vesting balance -> `gnokey maketx send` of locked coins rejected -> send after vest accepted. For a PoC this is acceptable to defer, but it is the test that proves enforcement at the node level. A `gno.land/pkg/integration/testdata/` txtar with a vesting genesis balance and two `gnokey maketx send` calls at different block times would cover it.
  </details>

## Suggestions
- [`gno.land/pkg/gnoland/app.go:668-670`](https://github.com/gnolang/gno/blob/9da3635c4/gno.land/pkg/gnoland/app.go#L668-L670) · [↗](../../../../../.worktrees/gno-review-5840/gno.land/pkg/gnoland/app.go#L668) — `applyBalance` panics if the vesting account constructor errors. `Balance.Verify` already validates the schedule, but `applyBalance` does not require `Verify` to have run, so a genesis assembled in code (bypassing `Verify`) with `OriginalVesting > Amount` halts node start with a panic rather than a clean genesis error. Consistent with the surrounding `panic(err)` on `SetCoins`, so leaving it is defensible; noting for whoever hardens genesis ingestion.
  <details><summary>details</summary>

  The constructors re-check `Coins.IsAllGTE(OriginalVesting)` and `schedule.Validate()`, so the panic only fires on an already-invalid genesis, never on valid input. It is a genesis-time, not runtime, failure. The only behavior question is panic vs returned error at the InitChainer boundary, which is a project-wide convention call.
  </details>

## Verified
- Amino storage round-trip: `amino.MarshalAny` then `amino.UnmarshalAny` on a `ContinuousVestingAccount` (the path `AccountKeeper.SetAccount` uses) decodes back to `*std.ContinuousVestingAccount` with account number and coins preserved and still satisfying `VestingAccount`. Not covered by the existing `Balance` JSON round-trip test, which round-trips the genesis entry, not the stored account. Confirmed on 9da3635c4.
- big.Int overflow fix: `GetVestedCoins` for 100,000 gnot (`1e11` ugnot) over a 3-year schedule near end computes `99999998943ugnot` with no panic; the old `overflow.Mul(amount, elapsed)` path multiplied `~1e11 * ~9.4e7 ≈ 9.4e18`, past the int64 max `~9.22e18`, and panicked. The switch to `big.Int` removes a reachable genesis-amount panic. Confirmed on 9da3635c4.
- Account number: the vesting path now assigns `AccountNumber` from `GetNextAccountNumber(ctx)` (read-and-increment, same source as the non-vesting `NewAccountWithAddress`), so vesting and regular genesis accounts get distinct sequential numbers; the upgrade-to-`BaseAccount` swap preserves it.
- Genesis panic reproduced: a genesis vesting balance whose address is also in `UnrestrictedAddrs` panics `InitChain` with `interface conversion: std.Account is *std.ContinuousVestingAccount, not *gnoland.GnoAccount` (Warning 1).
- Green at 9da3635c4: `go test ./tm2/pkg/std/` (vesting schedule math), `./tm2/pkg/sdk/bank/`, `./tm2/pkg/sdk/auth/`, and `./gno.land/pkg/gnoland/` (`TestInitChainer_VestingAccount`, `TestBalance_Vesting*`, `TestBalances_VestingFromEntries`).

## Open questions
- Spend-vs-send boundary: `banker.RemoveCoin` (realm-initiated coin removal, `gno.land/pkg/sdk/vm/builtins.go`) routes through `SubtractCoins`, so a vesting account that calls a realm which sends its coins is blocked on locked coins, same as a direct transfer. The issue framed the goal as "limit sending out to exchanges, keep realm usage free." Whether realm-mediated sends should count as "sending" or "spending" is a scope decision; not posted as a defect.
- Vesting accounts are not whitelist-eligible: they don't implement `AccountUnrestricter`, so during a global GNOT token-lock they can't transfer even their vested restricted-denom coins, and can't be added to the unrestricted whitelist without the genesis panic in Warning 1. Whether launch allocations need both schedule-vesting and whitelist-eligibility is a design question for #5798; surfaced in the Body.
- Gas-spends-locked-coins is now a decided design point: 74979df restricted gas to spendable coins, then 9da3635c4 reverted it per #5798 (send-limiting, not spend-limiting), so a fully-locked vesting account can still pay fees and realm calls, and gas can draw down principal below the still-locked amount. No action; recording that the round-1 open question is resolved by design.
- No governance, clawback, or schedule-mutation surface, and vesting accounts can only be created at genesis (no tx/message path). Likely intentional for the finalized-allocation use case in #5798; not posted, deferred-scope.
- `StartTime` defaults to 0 (epoch) when omitted via direct JSON construction, which would vest a continuous schedule from 1970. The CLI parse format requires an explicit start (`vesting=<coins>,<start>,<end>`), so this is unreachable from the genesis sheet; flagging only as a direct-construction footgun, not posted.
