# Review: PR [#5840](https://github.com/gnolang/gno/pull/5840)
Event: COMMENT

## Body
Proof of concept, so direction-check rather than merge-gate. The round-1 feedback is addressed. Verified on 9da3635c4: `amino.MarshalAny`/`UnmarshalAny` on a continuous vesting account, the path account storage uses, round-trips back to the same type with account number and coins intact; and `GetVestedCoins` for 100,000 gnot over a 3-year schedule now computes near the end without a panic, where the old int64 `overflow.Mul` overflowed at that scale.

One design question: vesting accounts don't implement `AccountUnrestricter` and carry no token-lock-whitelist bit, so during a global GNOT token-lock they can't transfer even their vested restricted-denom coins, and there is no way to whitelist them to change that. If launch allocations need both schedule-vesting and whitelist-eligibility, that shape is worth deciding now.

Two scope questions still open from round 1. `banker.RemoveCoin` routes through `SubtractCoins`, so a vesting account calling a realm that moves its coins is blocked just like a direct transfer; whether realm-mediated sends count as sending or spending is worth pinning against the issue's limit-sending framing. And vesting accounts can only be created at genesis, with no governance, clawback, or schedule-mutation surface; confirming that is intentional for the finalized-allocation case.

The one red check is Merge Requirements, the approvals gate, not a code problem.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5840-vesting-account-poc/2-9da3635c4/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gno.land/pkg/gnoland/app.go:685-694 [↗](../../../../../.worktrees/gno-review-5840/gno.land/pkg/gnoland/app.go#L691)
`applyUnrestrictedAddrs` force-asserts every whitelisted address to `*GnoAccount`, but a vesting account is `*std.ContinuousVestingAccount`, so a genesis that lists a vesting address in the unrestricted whitelist panics node start on an interface conversion. The sibling runtime check `canSendCoins` handles the same account with a checked `AccountUnrestricter` assertion. Handle the non-GnoAccount case here too, or reject it as a clean genesis error.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5840 -R gnolang/gno
cat > gno.land/pkg/gnoland/zz_vest_unrestricted_test.go <<'EOF'
package gnoland

import (
	"testing"
	"time"

	abci "github.com/gnolang/gno/tm2/pkg/bft/abci/types"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/db/memdb"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/stretchr/testify/require"
)

func TestVestUnrestricted(t *testing.T) {
	key := getDummyKey(t)
	addr := key.PubKey().Address()
	app, err := NewAppWithOptions(TestAppOptions(memdb.NewMemDB()))
	require.NoError(t, err)

	state := DefaultGenState()
	state.Balances = []Balance{{
		Address: addr,
		Amount:  std.NewCoins(std.NewCoin("ugnot", 1_000_000)),
		Vesting: &std.VestingSchedule{
			OriginalVesting: std.NewCoins(std.NewCoin("ugnot", 500_000)),
			StartTime:       100, EndTime: 200,
		},
	}}
	// Whitelist the vesting account's own address.
	state.Auth.Params.UnrestrictedAddrs = []crypto.Address{addr}

	app.InitChain(abci.RequestInitChain{
		ChainID: "test",
		Time:    time.Unix(150, 0),
		ConsensusParams: &abci.ConsensusParams{
			Block:     defaultBlockParams(),
			Validator: &abci.ValidatorParams{PubKeyTypeURLs: []string{}},
		},
		AppState: state,
	})
}
EOF
go test ./gno.land/pkg/gnoland/ -run TestVestUnrestricted -count=1 2>&1 | grep -iE "conversion|^panic|--- FAIL|^FAIL" | head
rm gno.land/pkg/gnoland/zz_vest_unrestricted_test.go
```
```
--- FAIL: TestVestUnrestricted (2.74s)
panic: interface conversion: std.Account is *std.ContinuousVestingAccount, not *gnoland.GnoAccount [recovered, repanicked]
FAIL	github.com/gnolang/gno/gno.land/pkg/gnoland
```
</details>

## tm2/pkg/sdk/bank/keeper.go:238-273 [↗](../../../../../.worktrees/gno-review-5840/tm2/pkg/sdk/bank/keeper.go#L238)
Missing test: the bank keeper's vesting enforcement is untested. Nothing asserts that a locked transfer is blocked, the vested portion goes through, gas bypasses the lock via `SendCoinsUnrestricted`, and the account upgrades to `BaseAccount` once fully vested. The schedule math is tested in `tm2/pkg/std` and genesis creation in `TestInitChainer_VestingAccount`, though all four behaviors hold today.

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

## gno.land/pkg/gnoland/app.go:653-679 [↗](../../../../../.worktrees/gno-review-5840/gno.land/pkg/gnoland/app.go#L653)
Missing test: no txtar or app-boot test sends a real transaction that the vesting lock blocks before the schedule vests and allows after, across block time. `TestInitChainer_VestingAccount` covers genesis account creation but no transfer. A txtar with a vesting genesis balance and two `gnokey maketx send` calls at different block times would cover it.

## tm2/pkg/std/vesting_account.go:31-45 [↗](../../../../../.worktrees/gno-review-5840/tm2/pkg/std/vesting_account.go#L31)
`Validate` rejects a delayed schedule unless `StartTime < EndTime`, yet a delayed account ignores the start and cliffs all coins at `EndTime`. Should delayed schedules skip that ordering check, or is the shared constraint intentional?

## tm2/pkg/sdk/bank/keeper.go:280-304 [↗](../../../../../.worktrees/gno-review-5840/tm2/pkg/sdk/bank/keeper.go#L280)
`subtractCoinsUnrestricted` discards the `upgradeVestingAccount` return, then `SetCoins` fetches the account and writes it again, so the upgrade reads and stores the account twice in one call. Balances are correct since the upgrade preserves coins.
