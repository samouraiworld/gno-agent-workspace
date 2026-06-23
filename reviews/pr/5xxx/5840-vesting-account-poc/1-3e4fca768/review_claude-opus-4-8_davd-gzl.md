# PR #5840: feat: vesting account poc

URL: https://github.com/gnolang/gno/pull/5840
Author: julienrbrt | Base: master | Files: 9 | +1222 -13
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 3e4fca768 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5840 3e4fca768`

**TL;DR:** Genesis accounts can now carry a vesting schedule, so their coins unlock gradually instead of being spendable from day one. Locked coins can pay gas and stay on the chain, but cannot be transferred out until they vest.

**Verdict: NEEDS DISCUSSION** — proof-of-concept; the core logic is correct and enforcement works end to end, but two CI checks are red (regenerate proto, add `t.Parallel()`), the on-chain enforcement and account-upgrade paths have no test, and the scope decisions (spend-vs-send boundary, genesis-only creation, no governance/clawback) want maintainer sign-off before this leaves PoC.

## Summary
Adds three vesting account types to `tm2/pkg/std` (`BaseVestingAccount`, `ContinuousVestingAccount`, `DelayedVestingAccount`) plus a `VestingSchedule`, registers them with amino, and wires genesis to build them from a `Balance.Vesting` field. The bank keeper enforces the lock at its single subtraction chokepoint `SubtractCoins`: a transfer of more than the spendable (vested + non-vesting) amount is rejected with `VestingLockedCoinsError`. Gas and system refunds go through `SendCoinsUnrestricted`, which deliberately bypasses the check, matching the issue's "limit sending, not spending" decision (#5798). Once a schedule fully vests, the account is rewritten to a plain `BaseAccount` so later transfers skip vesting entirely.

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
- restricted denom: denom-based transfer lock via the bank param; orthogonal to vesting's schedule-based lock.
- banker: stdlib coin API; its `RemoveCoin` routes through `SubtractCoins`, so realm-initiated burns/sends are also vesting-checked.

## Fix
Not a bugfix; additive feature. The load-bearing design choice is that enforcement lives only in `SubtractCoins` ([`tm2/pkg/sdk/bank/keeper.go:238-273`](https://github.com/gnolang/gno/blob/3e4fca768/tm2/pkg/sdk/bank/keeper.go#L238-L273) · [↗](../../../../../.worktrees/gno-review-5840/tm2/pkg/sdk/bank/keeper.go#L238)), the one place all restricted transfers subtract, while gas and refunds take the parallel `subtractCoinsUnrestricted` path that skips the check.

## Critical (must fix)
None.

## Warnings (should fix)
- **[the feature's core enforcement has no test]** [`tm2/pkg/sdk/bank/keeper.go:238-273`](https://github.com/gnolang/gno/blob/3e4fca768/tm2/pkg/sdk/bank/keeper.go#L238-L273) · [↗](../../../../../.worktrees/gno-review-5840/tm2/pkg/sdk/bank/keeper.go#L238) — `SubtractCoins` vesting enforcement, the upgrade-to-`BaseAccount` swap, and the `SendCoinsUnrestricted` bypass are exercised by no test in the bank package.
  <details><summary>details</summary>

  All vesting tests live in `tm2/pkg/std` and test the pure schedule math (`GetVestedCoins`, `SpendableCoins`). Nothing tests that the bank keeper actually blocks a locked transfer, allows the unlocked portion, lets gas through while coins are locked, or upgrades the account once vested. These are the security-relevant behaviors of the PR, and they are the integration the std-layer tests cannot cover. I verified all four behaviors hold today with a keeper-level probe (see repro), so this is a coverage gap, not a defect. Fix: add a bank-package test asserting block-when-locked, allow-unlocked-portion, unrestricted-bypass, and upgrade-after-full-vest.

  **Repro:**
  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5840 -R gnolang/gno
  cat > tm2/pkg/sdk/bank/zz_vest_test.go <<'EOF'
  package bank

  import (
  	"testing"
  	"time"

  	bft "github.com/gnolang/gno/tm2/pkg/bft/types"
  	"github.com/gnolang/gno/tm2/pkg/crypto"
  	"github.com/gnolang/gno/tm2/pkg/sdk"
  	"github.com/gnolang/gno/tm2/pkg/std"
  )

  func mkA(b byte) crypto.Address { var a crypto.Address; a[0] = b; return a }
  func at(c sdk.Context, s int64) sdk.Context {
  	return c.WithBlockHeader(&bft.Header{ChainID: "test-chain-id", Time: time.Unix(s, 0)})
  }

  func TestVest(t *testing.T) {
  	env := setupTestEnv()
  	src, dst := mkA(1), mkA(2)
  	base := std.NewBaseAccount(src, std.NewCoins(std.NewCoin("ugnot", 1000)), nil, 0, 0)
  	cva, err := std.NewContinuousVestingAccount(base, std.VestingSchedule{
  		OriginalVesting: std.NewCoins(std.NewCoin("ugnot", 1000)), StartTime: 100, EndTime: 200,
  	})
  	if err != nil { t.Fatal(err) }
  	env.acck.SetAccount(env.ctx, cva)

  	one := std.NewCoins(std.NewCoin("ugnot", 1))
  	t.Logf("before-start send 1 (all locked): %v", env.bankk.SendCoins(at(env.ctx, 50), src, dst, one))
  	t.Logf("halfway send 600 (500 spendable): %v", env.bankk.SendCoins(at(env.ctx, 150), src, dst, std.NewCoins(std.NewCoin("ugnot", 600))))
  	t.Logf("halfway send 400 (ok): %v", env.bankk.SendCoins(at(env.ctx, 150), src, dst, std.NewCoins(std.NewCoin("ugnot", 400))))
  	t.Logf("before-start unrestricted 700: %v", env.bankk.SendCoinsUnrestricted(at(env.ctx, 50), src, dst, std.NewCoins(std.NewCoin("ugnot", 600))))
  	_ = env.bankk.SendCoins(at(env.ctx, 300), src, dst, one)
  	t.Logf("after-end acc type = %T", env.acck.GetAccount(at(env.ctx, 300), src))
  }
  EOF
  go test ./tm2/pkg/sdk/bank/ -run TestVest -v -count=1 2>&1 | grep -E "before|halfway|after|PASS"
  rm tm2/pkg/sdk/bank/zz_vest_test.go
  ```
  ```
  before-start send 1 (all locked): vesting locked coins error
  halfway send 600 (500 spendable): vesting locked coins error
  halfway send 400 (ok): <nil>
  before-start unrestricted 700: <nil>
  after-end acc type = *std.BaseAccount
  --- PASS: TestVest (0.00s)
  ```
  </details>

## Nits
- [`tm2/pkg/std/vesting_account.go:31-45`](https://github.com/gnolang/gno/blob/3e4fca768/tm2/pkg/std/vesting_account.go#L31-L45) · [↗](../../../../../.worktrees/gno-review-5840/tm2/pkg/std/vesting_account.go#L31) — `VestingSchedule.Validate` requires `StartTime < EndTime` even for delayed schedules, where `StartTime` is ignored (`GetStartTime` returns 0, vesting is a cliff at `EndTime`). A delayed genesis entry must still supply a start earlier than the end or it is rejected. Harmless, but the constraint is meaningless for the delayed case; confirmed behaviorally: `NewDelayedVestingAccount` with `StartTime:0, EndTime:200` validates fine, and `StartTime:200, EndTime:100` is rejected.
- [`tm2/pkg/sdk/bank/keeper.go:280-304`](https://github.com/gnolang/gno/blob/3e4fca768/tm2/pkg/sdk/bank/keeper.go#L280-L304) · [↗](../../../../../.worktrees/gno-review-5840/tm2/pkg/sdk/bank/keeper.go#L280) — `subtractCoinsUnrestricted` calls `upgradeVestingAccount(ctx, acc)` and discards the return, then `SetCoins` re-fetches the account and writes again. The upgrade preserves coins exactly, so balances are correct, but the account is fetched and `SetAccount`-ed twice in one call. Cosmetic.

## Missing Tests
- **[on-chain integration]** [`gno.land/pkg/gnoland/app.go:653-672`](https://github.com/gnolang/gno/blob/3e4fca768/gno.land/pkg/gnoland/app.go#L653-L672) · [↗](../../../../../.worktrees/gno-review-5840/gno.land/pkg/gnoland/app.go#L653) — no txtar or app-boot test takes a genesis vesting balance through `applyBalance` and then exercises a real transfer being blocked/allowed across block time.
  <details><summary>details</summary>

  `applyBalance` builds the vesting account from `bal.Vesting` and `SetAccount`s it. The parse and verify layers are tested, and the keeper math is tested in isolation, but nothing runs the full path: genesis -> vesting account on chain -> tx send of locked coins rejected -> tx send after vest accepted. For a PoC this is acceptable to defer, but it is the test that proves the feature end to end at the node level. A `gno.land/pkg/integration/testdata/` txtar with a vesting genesis balance and two `gnokey maketx send` calls at different times would cover it.
  </details>

## Suggestions
- [`gno.land/pkg/gnoland/app.go:661-664`](https://github.com/gnolang/gno/blob/3e4fca768/gno.land/pkg/gnoland/app.go#L661-L664) · [↗](../../../../../.worktrees/gno-review-5840/gno.land/pkg/gnoland/app.go#L661) — `applyBalance` panics if the vesting account constructor errors. `Balance.Verify` already validates the schedule, but `applyBalance` does not require `Verify` to have run, so a genesis assembled in code (bypassing `Verify`) with `OriginalVesting > Amount` halts node start with a panic rather than a clean genesis error. Consistent with the surrounding `panic(err)` on `SetCoins`, so leaving it is defensible; noting for whoever hardens genesis ingestion.
  <details><summary>details</summary>

  The constructors re-check `Coins.IsAllGTE(OriginalVesting)` and `schedule.Validate()`, so the panic only fires on an already-invalid genesis, never on valid input. It is a genesis-time, not runtime, failure. The only behavior question is panic vs returned error at the InitChainer boundary, which is a project-wide convention call.
  </details>

## Open questions
- Spend-vs-send boundary: `banker.RemoveCoin` (realm-initiated coin removal, `gno.land/pkg/sdk/vm/builtins.go:56`) routes through `SubtractCoins`, so a vesting account that calls a realm which sends its coins is blocked on locked coins, same as a direct transfer. The issue framed the goal as "limit sending out to exchanges, keep realm usage free." Whether realm-mediated sends should count as "sending" or "spending" is a scope decision; not posted because it is a design question for the author, not a code defect.
- No governance, clawback, or schedule-mutation surface, and vesting accounts can only be created at genesis (no tx/message path). Likely intentional for the finalized-allocation use case in #5798; not posted, deferred-scope.
- `StartTime` defaults to 0 (epoch) when omitted via direct JSON construction, which would vest a continuous schedule from 1970. The CLI parse format requires an explicit start (`vesting=<coins>,<start>,<end>`), so this is unreachable from the genesis sheet; flagging only as a direct-construction footgun, not posted.
