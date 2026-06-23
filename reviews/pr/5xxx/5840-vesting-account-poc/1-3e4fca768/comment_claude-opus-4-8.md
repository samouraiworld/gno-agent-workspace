# Review: PR #5840
Event: COMMENT

## Body
Proof of concept, so direction-check rather than merge-gate. The core is sound. Verified at the keeper level on 3e4fca768: a continuous vesting account is blocked from sending locked coins, can send the vested portion, lets gas through `SendCoinsUnrestricted` while coins are still locked, and is rewritten to a plain `BaseAccount` once fully vested.

Two scope questions, neither posted inline:

- `banker.RemoveCoin` routes through `SubtractCoins`, so a vesting account calling a realm that moves its coins is blocked just like a direct transfer. The issue framed the goal as limit sending out, keep realm usage free, so whether realm-mediated sends count as sending or spending is worth pinning.
- Vesting accounts can only be created at genesis, with no governance, clawback, or schedule-mutation surface. Likely intentional for the finalized-allocation case, flagging to confirm.

The two red checks are not code problems: `genproto2` wants `make -C misc/genproto2` committed, and lint wants `t.Parallel()` on two subtests.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5840-vesting-account-poc/1-3e4fca768/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## tm2/pkg/sdk/bank/keeper.go:238-273 [↗](../../../../../.worktrees/gno-review-5840/tm2/pkg/sdk/bank/keeper.go#L238)
Optional for a PoC: the enforcement itself has no test. The vesting tests cover the schedule math in `tm2/pkg/std`, but nothing in the bank package asserts the security-relevant behaviors: a locked transfer is blocked, the vested portion goes through, gas bypasses the lock, the account upgrades to `BaseAccount` once fully vested. All hold today.

<details><summary>repro</summary>

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
	t.Logf("before-start unrestricted 600: %v", env.bankk.SendCoinsUnrestricted(at(env.ctx, 50), src, dst, std.NewCoins(std.NewCoin("ugnot", 600))))
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
before-start unrestricted 600: <nil>
after-end acc type = *std.BaseAccount
--- PASS: TestVest (0.00s)
```
</details>

## gno.land/pkg/gnoland/app.go:653-672 [↗](../../../../../.worktrees/gno-review-5840/gno.land/pkg/gnoland/app.go#L653)
Optional for a PoC: no integration test takes a genesis vesting balance through `applyBalance` onto a running node, then sends a real transaction that the lock blocks before vesting and allows after. The parse, verify, and keeper-math layers are tested in isolation, but the full genesis-to-transfer path is not. A txtar with a vesting genesis balance and two sends at different block times would close it.

## tm2/pkg/std/vesting_account.go:31-45 [↗](../../../../../.worktrees/gno-review-5840/tm2/pkg/std/vesting_account.go#L31)
Nit: `Validate` requires `StartTime < EndTime` for delayed schedules too, where the start is ignored and coins cliff at `EndTime`. A delayed genesis entry must still carry a start earlier than its end or it is rejected, which is a meaningless constraint for that type.

## tm2/pkg/sdk/bank/keeper.go:280-304 [↗](../../../../../.worktrees/gno-review-5840/tm2/pkg/sdk/bank/keeper.go#L280)
Nit: `subtractCoinsUnrestricted` discards the `upgradeVestingAccount` return, then `SetCoins` fetches the account and writes it again, so the upgrade reads and stores the account twice in one call. Balances are correct since the upgrade preserves coins.
