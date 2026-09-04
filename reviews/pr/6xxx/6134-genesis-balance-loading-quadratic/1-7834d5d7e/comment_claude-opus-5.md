# Review: [#6134](https://github.com/gnolang/gno/pull/6134)
Event: COMMENT

## Body
An address the loader has not written holds no split-tier key on any path, so the account probe answers the question the drain would.

<details><summary>checks that held</summary>

- `setSplitBalance` is reached from `SetCoins`, `InitCoins` and `writeSplitCoins`, each preceded by `SetAccount` in the loader; from `AddCoins` through `ensureAccount`; and from `SubtractCoins`, which cannot debit an address never credited. `bank.InitGenesis` writes params only, `loadStdlibs` writes VM store objects, and `acck.InitGenesis`, `applyUnrestrictedAddrs` and the genesis transactions all run after the balance loop, in both `applyInMemoryAppState` and `applyStreamingAppState`.
- Neither `SetCoins` nor `InitCoins` is on [`vm.BankKeeperI`](https://github.com/gnolang/gno/blob/7834d5d7e/gno.land/pkg/sdk/vm/types.go#L18-L34) or [`auth.BankKeeperI`](https://github.com/gnolang/gno/blob/7834d5d7e/tm2/pkg/sdk/auth/types.go#L25-L33), and the full interface is held only by [`InitChainerConfig.bankk`](https://github.com/gnolang/gno/blob/7834d5d7e/gno.land/pkg/gnoland/app.go#L396), so no realm and no ante handler reaches the new method.
- The probe cannot feed the term it avoids: [`cacheStore.Get`](https://github.com/gnolang/gno/blob/7834d5d7e/tm2/pkg/store/cache/store.go#L158) caches its result undirty and [`setCacheValue`](https://github.com/gnolang/gno/blob/7834d5d7e/tm2/pkg/store/cache/store.go#L462-L464) enters the unsorted set only for a dirty write. It costs 5.00 allocations and 208 bytes per balance, identical at 100,000 and at 1,000,000, which is the residual the description leaves unattributed at 14.8% and 21.3%.
</details>

## gno.land/pkg/gnoland/app.go:785 [gh](https://github.com/gnolang/gno/blob/7834d5d7e/gno.land/pkg/gnoland/app.go#L785) · [↗](../../../../../.worktrees/gno-review-6134/gno.land/pkg/gnoland/app.go#L785)
Missing test: substituting `firstSighting := true` here moves the committed genesis store hash while the whole `gno.land/pkg/gnoland` package stays green, because [`mockAuthKeeper.GetAccount`](https://github.com/gnolang/gno/blob/7834d5d7e/gno.land/pkg/gnoland/mock_test.go#L226) returns nil unconditionally and [`TestApplyBalanceWithARepeatedAddress`](https://github.com/gnolang/gno/blob/7834d5d7e/gno.land/pkg/gnoland/app_test.go#L3927) uses `ugnot`, the sole account-tier denom, so nothing in the package reaches the drain. The [mock-backed test](https://github.com/gnolang/gno/pull/6134#discussion_r3933470179) cannot be written as asked for the first reason; the one below asserts the effect instead, so `setCoinsCalls` keeps the meaning `setCoinsAtRecompute` depends on.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6134 -R gnolang/gno
cat > gno.land/pkg/gnoland/apphash_genesis_test.go <<'GO'
package gnoland

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gnolang/gno/gno.land/pkg/gnoland/ugnot"
	bft "github.com/gnolang/gno/tm2/pkg/bft/types"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/db/memdb"
	"github.com/gnolang/gno/tm2/pkg/log"
	"github.com/gnolang/gno/tm2/pkg/sdk"
	"github.com/gnolang/gno/tm2/pkg/sdk/auth"
	"github.com/gnolang/gno/tm2/pkg/sdk/bank"
	"github.com/gnolang/gno/tm2/pkg/sdk/params"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/gnolang/gno/tm2/pkg/store"
	storebptree "github.com/gnolang/gno/tm2/pkg/store/bptree"
	"github.com/gnolang/gno/tm2/pkg/store/dbadapter"
)

// recordingBankKeeper is a real BankKeeper whose InitCoins can be routed back to
// SetCoins, which is what applyBalance did before this change.
type recordingBankKeeper struct {
	bank.BankKeeper
	forceSetCoins bool
}

func (r *recordingBankKeeper) InitCoins(ctx sdk.Context, addr crypto.Address, amt std.Coins) error {
	if r.forceSetCoins {
		return r.BankKeeper.SetCoins(ctx, addr, amt)
	}
	return r.BankKeeper.InitCoins(ctx, addr, amt)
}

// For a genesis balance list carrying every shape a real one can, the committed
// store hash must equal the one unconditional SetCoins produces. A node operator
// has no other way to find out: a diverged hash surfaces as a consensus failure
// at the first block, after the genesis file has been distributed.
func TestGenesisBalanceLoadAppHashMatchesSetCoinsOnly(t *testing.T) {
	t.Parallel()

	a := crypto.AddressFromPreimage([]byte("a"))
	b := crypto.AddressFromPreimage([]byte("b"))
	c := crypto.AddressFromPreimage([]byte("c"))
	d := crypto.AddressFromPreimage([]byte("d"))
	realm := "/gno.land/r/x/tok:c"
	vest := std.Coins{{Denom: ugnot.Denom, Amount: 400}}
	sched := &std.VestingSchedule{OriginalVesting: vest, StartTime: 100, EndTime: 1_000_000}

	bals := []Balance{
		{Address: a, Amount: std.Coins{{Denom: ugnot.Denom, Amount: 100}}}, // account tier only
		{Address: b, Amount: std.Coins{{Denom: realm, Amount: 7}}},         // split tier only
		{Address: c, Amount: std.Coins{{Denom: realm, Amount: 7}, {Denom: ugnot.Denom, Amount: 100}}},
		{Address: d, Amount: vest, Vesting: sched},                                            // vesting
		{Address: a, Amount: std.Coins{{Denom: ugnot.Denom, Amount: 250}}},                    // repeat, same tier
		{Address: c, Amount: std.Coins{{Denom: ugnot.Denom, Amount: 250}}},                    // repeat dropping a split denom
		{Address: d, Amount: std.Coins{{Denom: ugnot.Denom, Amount: 500}}},                    // repeat clearing a schedule
		{Address: b, Amount: std.Coins{{Denom: realm, Amount: 7}, {Denom: "zzz", Amount: 1}}}, // repeat adding one
	}

	require.Equal(t,
		commitGenesisBalances(t, bals, true),  // every entry through SetCoins
		commitGenesisBalances(t, bals, false), // InitCoins on a first sighting
		"genesis store hash must not move")
}

func commitGenesisBalances(t *testing.T, bals []Balance, forceSetCoins bool) []byte {
	t.Helper()

	db := memdb.NewMemDB()
	baseKey := store.NewStoreKey("baseKey")
	mainKey := store.NewStoreKey("mainKey")
	cms := store.NewCommitMultiStore(db)
	cms.MountStoreWithDB(mainKey, storebptree.FastStoreConstructor, db)
	cms.MountStoreWithDB(baseKey, dbadapter.StoreConstructor, db)
	require.NoError(t, cms.LoadLatestVersion())

	// MultiCacheWrap mirrors BaseApp's deliverState, which is where InitChainer runs.
	ms := cms.MultiCacheWrap()
	ctx := sdk.NewContext(sdk.RunTxModeDeliver, ms, &bft.Header{ChainID: "test"}, log.NewNoopLogger())

	prmk := params.NewParamsKeeper(mainKey)
	acck := auth.NewAccountKeeper(mainKey, prmk.ForModule(auth.ModuleName), ProtoGnoAccount, std.ProtoBaseSessionAccount)
	bankk := bank.NewBankKeeper(acck, prmk.ForModule(bank.ModuleName), mainKey, []string{ugnot.Denom})
	prmk.Register(auth.ModuleName, acck)
	prmk.Register(bank.ModuleName, bankk)

	cfg := InitChainerConfig{acck: acck, bankk: &recordingBankKeeper{BankKeeper: bankk, forceSetCoins: forceSetCoins}}
	for _, bal := range bals {
		cfg.applyBalance(ctx, bal)
	}
	cfg.seedSupply(ctx)

	ms.MultiWrite()
	return cms.Commit().Hash
}
GO
go test -count=1 -run TestGenesisBalanceLoadAppHashMatchesSetCoinsOnly ./gno.land/pkg/gnoland/
sed -i 's|firstSighting := cfg.acck.GetAccount(ctx, bal.Address) == nil|firstSighting := true|' gno.land/pkg/gnoland/app.go
go test -count=1 ./gno.land/pkg/gnoland/
git checkout gno.land/pkg/gnoland/app.go
rm gno.land/pkg/gnoland/apphash_genesis_test.go
```

The second run is the finding: the branch is gone, the package is still green except for the added test, and the genesis a repeated address produces is a different chain.

```
ok  	github.com/gnolang/gno/gno.land/pkg/gnoland	0.080s

--- FAIL: TestGenesisBalanceLoadAppHashMatchesSetCoinsOnly (0.00s)
    	--- Expected
    	+++ Actual
    	 ([]uint8) (len=32) {
    	- 00000000  b1 43 60 84 fd 44 af ff  f7 5c bf ae 99 aa 73 55  |.C`..D...\....sU|
    	- 00000010  80 47 fe 0b 8c c5 90 27  e4 06 83 4b 25 01 df b4  |.G.....'...K%...|
    	+ 00000000  58 26 59 ff 0b c1 d5 12  47 55 ab 2e 4c 88 17 37  |X&Y.....GU..L..7|
    	+ 00000010  06 04 63 4c 40 f9 68 99  21 ae c1 f0 35 a5 b3 93  |..cL@.h.!...5...|
    	 }
    	Messages: genesis store hash must not move
FAIL	github.com/gnolang/gno/gno.land/pkg/gnoland	18.503s
```
</details>

## contribs/gnogenesis/internal/fork/test.go:228 [gh](https://github.com/gnolang/gno/blob/7834d5d7e/contribs/gnogenesis/internal/fork/test.go#L228) · [↗](../../../../../.worktrees/gno-review-6134/contribs/gnogenesis/internal/fork/test.go#L228)
The clock moved above `NewInMemoryNode` and three statements that belong with it stayed below: the [`--timeout`](https://github.com/gnolang/gno/blob/7834d5d7e/contribs/gnogenesis/internal/fork/test.go#L76) deadline at [`:254`](https://github.com/gnolang/gno/blob/7834d5d7e/contribs/gnogenesis/internal/fork/test.go#L254), the 30-second progress ticker at [`:251`](https://github.com/gnolang/gno/blob/7834d5d7e/contribs/gnogenesis/internal/fork/test.go#L251), and the [`Replaying %d txs (timeout: %s)`](https://github.com/gnolang/gno/blob/7834d5d7e/contribs/gnogenesis/internal/fork/test.go#L248) line, and the balance load this branch speeds up is the smaller half of what is inside that window: genesis transactions are delivered from `InitChainer` too, at [`app.go:615`](https://github.com/gnolang/gno/blob/7834d5d7e/gno.land/pkg/gnoland/app.go#L615) and [`:724`](https://github.com/gnolang/gno/blob/7834d5d7e/gno.land/pkg/gnoland/app.go#L724), which is why [`:288`](https://github.com/gnolang/gno/blob/7834d5d7e/contribs/gnogenesis/internal/fork/test.go#L288) can compare a final `txProcessed` the moment `n.Ready()` fires. Moving them up beside the clock spends the budget on the replay rather than restarting it afterwards, which is as far as it goes: `NewInMemoryNode` takes no context, so the replay itself still cannot be interrupted.

## tm2/pkg/sdk/bank/keeper.go:532-536 [gh](https://github.com/gnolang/gno/blob/7834d5d7e/tm2/pkg/sdk/bank/keeper.go#L532-L536) · [↗](../../../../../.worktrees/gno-review-6134/tm2/pkg/sdk/bank/keeper.go#L532-L536)
Missing test: the ordering this comment calls load-bearing is load-bearing and nothing holds it, since `ensureAccount` returns whatever account exists and [`BaseSessionAccount.SetCoins`](https://github.com/gnolang/gno/blob/7834d5d7e/tm2/pkg/std/account.go#L245-L250) rejects a non-zero amount, so on a session account the swapped order returns the error having already written the split key.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6134 -R gnolang/gno
cat > tm2/pkg/sdk/bank/initcoins_order_test.go <<'GO'
package bank

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gnolang/gno/tm2/pkg/std"
)

func TestInitCoinsWritesTheAccountTierFirst(t *testing.T) {
	t.Parallel()

	addr := addrN(4242)
	env := setupTestEnv()

	// A session account rejects any non-zero amount, which is the one reachable
	// way setAccountTierCoins fails.
	sess := &std.BaseSessionAccount{}
	require.NoError(t, sess.SetAddress(addr))
	env.acck.SetAccount(env.ctx, sess)

	amt := std.NewCoins(std.NewCoin(testAccountDenom, 5), std.NewCoin("zzzcoin", 9))
	require.Error(t, env.bankk.InitCoins(env.ctx, addr, amt))
	require.Equal(t, int64(0), env.bankk.GetCoin(env.ctx, addr, "zzzcoin"),
		"account tier first: the failed call must leave no split-tier key")
}
GO
go test -count=1 -run TestInitCoinsWritesTheAccountTierFirst ./tm2/pkg/sdk/bank/
python3 - <<'PY'
p="tm2/pkg/sdk/bank/keeper.go"; s=open(p).read()
old="""	// Account tier first, as in SetCoins: it is the only step that can fail.
	if err := bank.setAccountTierCoins(ctx, nil, addr, account); err != nil {
		return err
	}
	bank.writeSplitCoins(ctx, addr, split)
	return nil
}"""
new="""	bank.writeSplitCoins(ctx, addr, split)
	if err := bank.setAccountTierCoins(ctx, nil, addr, account); err != nil {
		return err
	}
	return nil
}"""
open(p,'w').write(s.replace(old,new,1))
PY
go test -count=1 -run TestInitCoinsWritesTheAccountTierFirst ./tm2/pkg/sdk/bank/
rm tm2/pkg/sdk/bank/initcoins_order_test.go
go test -count=1 ./tm2/pkg/sdk/bank/
git checkout tm2/pkg/sdk/bank/keeper.go
```

The last run is the finding: with the writes swapped and the new test removed, the shipped suite does not notice.

```
ok  	github.com/gnolang/gno/tm2/pkg/sdk/bank	0.018s

--- FAIL: TestInitCoinsWritesTheAccountTierFirst (0.00s)
        	Error:      	Not equal:
        	            	expected: 0
        	            	actual  : 9
        	Messages:   	account tier first: the failed call must leave no split-tier key

ok  	github.com/gnolang/gno/tm2/pkg/sdk/bank	0.702s
```
</details>

## gno.land/pkg/gnoland/app.go:773-776 [gh](https://github.com/gnolang/gno/blob/7834d5d7e/gno.land/pkg/gnoland/app.go#L773-L776) · [↗](../../../../../.worktrees/gno-review-6134/gno.land/pkg/gnoland/app.go#L773-L776)
Nit: these four lines repeat [`:787-790`](https://github.com/gnolang/gno/blob/7834d5d7e/gno.land/pkg/gnoland/app.go#L787-L790) word for word, so a paragraph about building the account also sits above the probe.

```suggestion
```

## tm2/pkg/sdk/bank/initcoins_bench_test.go:52 [gh](https://github.com/gnolang/gno/blob/7834d5d7e/tm2/pkg/sdk/bank/initcoins_bench_test.go#L52) · [↗](../../../../../.worktrees/gno-review-6134/tm2/pkg/sdk/bank/initcoins_bench_test.go#L52)
Suggestion: `-bench 'CoinsLoad'`, which [`:17`](https://github.com/gnolang/gno/blob/7834d5d7e/tm2/pkg/sdk/bank/initcoins_bench_test.go#L17) gives as the way to run this file, reaches these two after the four that finish in about a minute, and `-timeout` does not stop a benchmark in progress, so the run has no end and no signal.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno, in tm2/:
gh pr checkout 6134 -R gnolang/gno
go test ./pkg/sdk/bank/ -run XXX -bench '^BenchmarkCoinsLoadSetCoins20000$' -benchtime 1x -timeout 5s
```

A five-second timeout let a 37.7-second benchmark run to completion and report success, which is why the 100,000 pair cannot be bounded that way either.

```
BenchmarkCoinsLoadSetCoins20000-6   	       1	37720650596 ns/op
PASS
ok  	github.com/gnolang/gno/tm2/pkg/sdk/bank	37.749s
```
</details>

## tm2/pkg/sdk/bank/initcoins_test.go:61-65 [gh](https://github.com/gnolang/gno/blob/7834d5d7e/tm2/pkg/sdk/bank/initcoins_test.go#L61-L65) · [↗](../../../../../.worktrees/gno-review-6134/tm2/pkg/sdk/bank/initcoins_test.go#L61-L65)
Suggestion: the equivalence runs against the bare store `setupTestEnv` returns, and what the two runs compare is iteration order, which under a genesis load comes from the cache layer's `dirtyItems` rather than from the tree, the same reason [the benchmark in this branch](https://github.com/gnolang/gno/blob/7834d5d7e/tm2/pkg/sdk/bank/initcoins_bench_test.go#L28-L31) says measuring the bare store shows nothing.

```suggestion
			envSet := setupTestEnv()
			envSet.ctx = envSet.ctx.WithMultiStore(envSet.ctx.MultiStore().MultiCacheWrap())
			errSet := envSet.bankk.SetCoins(envSet.ctx, addr, tc.amt)

			envInit := setupTestEnv()
			envInit.ctx = envInit.ctx.WithMultiStore(envInit.ctx.MultiStore().MultiCacheWrap())
			errInit := envInit.bankk.InitCoins(envInit.ctx, addr, tc.amt)
```

`TestInitCoinsWholeLoadMatchesSetCoins` at [`:114-115`](https://github.com/gnolang/gno/blob/7834d5d7e/tm2/pkg/sdk/bank/initcoins_test.go#L114-L115) wants the same two lines.

## tm2/pkg/sdk/bank/initcoins_test.go:143-146 [gh](https://github.com/gnolang/gno/blob/7834d5d7e/tm2/pkg/sdk/bank/initcoins_test.go#L143-L146) · [↗](../../../../../.worktrees/gno-review-6134/tm2/pkg/sdk/bank/initcoins_test.go#L143-L146)
Suggestion: this is the only assertion in the tree that reddens when `InitCoins` starts draining, and `NotEqual` passes for a partial drain, a full wipe and a no-op alike, where the exact post-state is `1aaa,20bbb`.

```suggestion
	require.Equal(t, std.NewCoins(std.NewCoin("aaa", 1), std.NewCoin("bbb", 20)),
		envInit.bankk.GetCoins(envInit.ctx, addr),
		"InitCoins is documented not to drain; if bbb ever disappears here, the "+
			"precondition has been silently relaxed and the genesis fast path "+
			"needs rechecking")
```

## tm2/pkg/sdk/bank/keeper.go:533 [gh](https://github.com/gnolang/gno/blob/7834d5d7e/tm2/pkg/sdk/bank/keeper.go#L533) · [↗](../../../../../.worktrees/gno-review-6134/tm2/pkg/sdk/bank/keeper.go#L533)
Suggestion: `nil` sends `setAccountTierCoins` into [`ensureAccount`](https://github.com/gnolang/gno/blob/7834d5d7e/tm2/pkg/sdk/bank/keeper.go#L250-L257), which reads and amino-decodes the account [`applyBalance`](https://github.com/gnolang/gno/blob/7834d5d7e/gno.land/pkg/gnoland/app.go#L802) wrote three lines earlier and still holds, and the parameter exists for that case, per [`:259-260`](https://github.com/gnolang/gno/blob/7834d5d7e/tm2/pkg/sdk/bank/keeper.go#L259-L260). Giving `InitCoins` the account and passing `acc` saves 19 allocations and 744 bytes per balance, four times what the new probe costs, and it is free to do now while the method has one caller.

<details><summary>measurement</summary>

A harness replicating `applyBalance`'s loop at 100,000 balances against a cache-wrapped multistore, medians of three, allocation columns deterministic.

| Variant | Median | Allocations | Bytes |
| --- | ---: | ---: | ---: |
| As shipped | 3.32 s | 9,001,585 | 349,057,952 |
| Account handed to `setAccountTierCoins` | 1.99 s | 7,101,478 | 274,638,656 |

`SetCoins` passes `nil` from the same caller, so the same saving is available there; its own drain makes it the slow path either way.
</details>

## tm2/pkg/store/cache/bench_quadratic_test.go:43 [gh](https://github.com/gnolang/gno/blob/7834d5d7e/tm2/pkg/store/cache/bench_quadratic_test.go#L43) · [↗](../../../../../.worktrees/gno-review-6134/tm2/pkg/store/cache/bench_quadratic_test.go#L43)
Suggestion: one control follows this line rather than two, and `BenchmarkWritesOnly16000` is a single point, so it shows writes are fast rather than linear; the pair that would attribute the cost is the same iteration count over a clean cache against a dirty one, which separates "iteration is expensive" from "iteration over a dirty cache is expensive".

## SKIP gno.land/pkg/gnoland/mock_test.go:182 [gh](https://github.com/gnolang/gno/blob/7834d5d7e/gno.land/pkg/gnoland/mock_test.go#L182) · [↗](../../../../../.worktrees/gno-review-6134/gno.land/pkg/gnoland/mock_test.go#L182)
Already raised: https://github.com/gnolang/gno/pull/6134#discussion_r3933494727

Suggestion: the shared `setCoinsCalls` counter is what keeps `setCoinsAtRecompute` meaningful, and the test anchored at `app.go:785` asserts the effect rather than a call count, so no separate counter is needed for it.
