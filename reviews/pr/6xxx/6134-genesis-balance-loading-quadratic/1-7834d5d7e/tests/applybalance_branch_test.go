package gnoland

// Which keeper method each genesis balance entry takes, and what a repeated
// address ends up holding.
//
// PR 6134 gives applyBalance two paths -- InitCoins on a first sighting,
// SetCoins on a repeat -- and nothing in gno.land/pkg/gnoland distinguishes
// them: mockBankKeeper.InitCoins increments the same setCoinsCalls counter as
// SetCoins, and mockAuthKeeper.GetAccount returns nil unconditionally, so every
// mock-backed test takes the InitCoins path and no test notices. The second test
// here is the one with teeth: it fails at head only if the repeat detection is
// wrong, and the third pins the committed store hash against the one master's
// unconditional SetCoins produces.
//
// All three pass at 7834d5d7e. With `firstSighting := true` substituted at
// gno.land/pkg/gnoland/app.go:785 the whole gno.land/pkg/gnoland package still
// passes and only these three fail.
//
/* Run: from a gno checkout:
gh pr checkout 6134 -R gnolang/gno && git checkout 7834d5d7e
curl -fsSL -o gno.land/pkg/gnoland/applybalance_branch_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6134-genesis-balance-loading-quadratic/1-7834d5d7e/tests/applybalance_branch_test.go
go test -race -count=1 -run 'TestApplyBalanceTakesTheRightKeeperPath|TestApplyBalanceRepeatDrainsAStaleSplitDenom|TestGenesisBalanceLoadAppHashMatchesSetCoinsOnly' ./gno.land/pkg/gnoland/
rm gno.land/pkg/gnoland/applybalance_branch_test.go
*/

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

// recordingBankKeeper is a real BankKeeper that also records which of the two
// write methods each call site chose. Embedding the real keeper rather than
// stubbing it keeps the store effects real, so the same fixture can assert both
// the path taken and the balance it left behind.
type recordingBankKeeper struct {
	bank.BankKeeper
	calls []string // "Set:<addr>" / "Init:<addr>", in order
	// forceSetCoins reproduces master, where applyBalance called SetCoins for
	// every entry.
	forceSetCoins bool
}

func (r *recordingBankKeeper) SetCoins(ctx sdk.Context, addr crypto.Address, amt std.Coins) error {
	r.calls = append(r.calls, "Set:"+addr.String())
	return r.BankKeeper.SetCoins(ctx, addr, amt)
}

func (r *recordingBankKeeper) InitCoins(ctx sdk.Context, addr crypto.Address, amt std.Coins) error {
	r.calls = append(r.calls, "Init:"+addr.String())
	if r.forceSetCoins {
		return r.BankKeeper.SetCoins(ctx, addr, amt)
	}
	return r.BankKeeper.InitCoins(ctx, addr, amt)
}

// realGenesisEnv wires the real auth and bank keepers, since the branch under
// test reads account existence and mockAuthKeeper never has any.
func realGenesisEnv(t *testing.T) (sdk.Context, InitChainerConfig, auth.AccountKeeper, *recordingBankKeeper) {
	t.Helper()

	db := memdb.NewMemDB()
	baseKey := store.NewStoreKey("baseKey")
	mainKey := store.NewStoreKey("mainKey")
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(mainKey, storebptree.FastStoreConstructor, db)
	ms.MountStoreWithDB(baseKey, dbadapter.StoreConstructor, db)
	require.NoError(t, ms.LoadLatestVersion())
	// MultiCacheWrap mirrors BaseApp's deliverState, which is where InitChainer runs.
	ctx := sdk.NewContext(sdk.RunTxModeDeliver, ms.MultiCacheWrap(), &bft.Header{ChainID: "test"}, log.NewNoopLogger())

	prmk := params.NewParamsKeeper(mainKey)
	acck := auth.NewAccountKeeper(mainKey, prmk.ForModule(auth.ModuleName), ProtoGnoAccount, std.ProtoBaseSessionAccount)
	bankk := bank.NewBankKeeper(acck, prmk.ForModule(bank.ModuleName), mainKey, []string{ugnot.Denom})
	prmk.Register(auth.ModuleName, acck)
	prmk.Register(bank.ModuleName, bankk)

	rec := &recordingBankKeeper{BankKeeper: bankk}
	return ctx, InitChainerConfig{acck: acck, bankk: rec}, acck, rec
}

// A first sighting takes InitCoins and a repeat takes SetCoins, vesting or not.
func TestApplyBalanceTakesTheRightKeeperPath(t *testing.T) {
	t.Parallel()

	plain := crypto.AddressFromPreimage([]byte("plain"))
	vester := crypto.AddressFromPreimage([]byte("vester"))
	amt := std.Coins{{Denom: ugnot.Denom, Amount: 1000}}
	sched := &std.VestingSchedule{OriginalVesting: amt, StartTime: 100, EndTime: 1_000_000}

	cases := []struct {
		name string
		bals []Balance
		want []string
	}{
		{
			"fresh address",
			[]Balance{{Address: plain, Amount: amt}},
			[]string{"Init:" + plain.String()},
		},
		{
			"fresh vesting address",
			[]Balance{{Address: vester, Amount: amt, Vesting: sched}},
			[]string{"Init:" + vester.String()},
		},
		{
			"repeated address",
			[]Balance{{Address: plain, Amount: amt}, {Address: plain, Amount: amt}},
			[]string{"Init:" + plain.String(), "Set:" + plain.String()},
		},
		{
			"repeated vesting address",
			[]Balance{
				{Address: vester, Amount: amt, Vesting: sched},
				{Address: vester, Amount: amt, Vesting: sched},
			},
			[]string{"Init:" + vester.String(), "Set:" + vester.String()},
		},
		{
			"plain entry after a vesting one",
			[]Balance{{Address: vester, Amount: amt, Vesting: sched}, {Address: vester, Amount: amt}},
			[]string{"Init:" + vester.String(), "Set:" + vester.String()},
		},
		{
			"two distinct addresses",
			[]Balance{{Address: plain, Amount: amt}, {Address: vester, Amount: amt}},
			[]string{"Init:" + plain.String(), "Init:" + vester.String()},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx, cfg, _, rec := realGenesisEnv(t)
			for _, bal := range tc.bals {
				cfg.applyBalance(ctx, bal)
			}
			require.Equal(t, tc.want, rec.calls)
		})
	}
}

// The repeat path must still drain a split-tier denom the later entry drops.
// InitCoins skips that drain, so taking it on a repeat leaves the address
// holding a coin the genesis file does not grant it.
func TestApplyBalanceRepeatDrainsAStaleSplitDenom(t *testing.T) {
	t.Parallel()

	ctx, cfg, acck, rec := realGenesisEnv(t)
	addr := crypto.AddressFromPreimage([]byte("dup-split"))
	realm := "/gno.land/r/x/tok:c" // not in the account-tier allowlist: its own store key

	// First entry grants both tiers; the second grants ugnot only.
	cfg.applyBalance(ctx, Balance{Address: addr, Amount: std.Coins{
		{Denom: realm, Amount: 7}, {Denom: ugnot.Denom, Amount: 100},
	}})
	cfg.applyBalance(ctx, Balance{Address: addr, Amount: std.Coins{
		{Denom: ugnot.Denom, Amount: 250},
	}})

	// Balance first: this is what an operator loses if the repeat detection breaks.
	require.Equal(t, int64(0), rec.GetCoin(ctx, addr, realm),
		"the dropped split-tier denom must be drained by the repeat")
	require.Equal(t, std.Coins{{Denom: ugnot.Denom, Amount: 250}}, rec.GetCoins(ctx, addr),
		"the last entry's amount wins, in full")
	require.Equal(t, []string{"Init:" + addr.String(), "Set:" + addr.String()}, rec.calls,
		"a repeat must not take the InitCoins fast path")
	require.NotNil(t, acck.GetAccount(ctx, addr), "one account must survive")

	// Supply is a sweep over what is held, so it agrees only if the drain happened.
	rec.RecomputeSupply(ctx)
	require.Equal(t, int64(250), rec.TotalSupply(ctx, ugnot.Denom))
	require.Equal(t, int64(0), rec.TotalSupply(ctx, realm))
	msg, broken := bank.AllInvariants(rec.ViewKeeper)(ctx)
	require.False(t, broken, "invariants must be clean after a repeated entry:\n%s", msg)
}

// The determinism guard the change needs: for a genesis balance list carrying
// every shape a real one can, the committed store hash must equal the one
// master's unconditional SetCoins produces. A node operator has no other way to
// find out -- a diverged app hash surfaces as a consensus failure at the first
// block, long after the genesis file has been distributed.
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
		commitGenesisBalances(t, bals, true),  // master: SetCoins for every entry
		commitGenesisBalances(t, bals, false), // head: InitCoins on first sighting
		"genesis store hash must not move")
}

// commitGenesisBalances loads bals through applyBalance and returns the
// committed store hash. forceSetCoins routes InitCoins to SetCoins, which is
// exactly what applyBalance did before this change.
func commitGenesisBalances(t *testing.T, bals []Balance, forceSetCoins bool) []byte {
	t.Helper()

	db := memdb.NewMemDB()
	baseKey := store.NewStoreKey("baseKey")
	mainKey := store.NewStoreKey("mainKey")
	cms := store.NewCommitMultiStore(db)
	cms.MountStoreWithDB(mainKey, storebptree.FastStoreConstructor, db)
	cms.MountStoreWithDB(baseKey, dbadapter.StoreConstructor, db)
	require.NoError(t, cms.LoadLatestVersion())

	ms := cms.MultiCacheWrap()
	ctx := sdk.NewContext(sdk.RunTxModeDeliver, ms, &bft.Header{ChainID: "test"}, log.NewNoopLogger())

	prmk := params.NewParamsKeeper(mainKey)
	acck := auth.NewAccountKeeper(mainKey, prmk.ForModule(auth.ModuleName), ProtoGnoAccount, std.ProtoBaseSessionAccount)
	bankk := bank.NewBankKeeper(acck, prmk.ForModule(bank.ModuleName), mainKey, []string{ugnot.Denom})
	prmk.Register(auth.ModuleName, acck)
	prmk.Register(bank.ModuleName, bankk)

	rec := &recordingBankKeeper{BankKeeper: bankk, forceSetCoins: forceSetCoins}
	cfg := InitChainerConfig{acck: acck, bankk: rec}
	for _, bal := range bals {
		cfg.applyBalance(ctx, bal)
	}
	cfg.seedSupply(ctx)

	ms.MultiWrite()
	return cms.Commit().Hash
}
