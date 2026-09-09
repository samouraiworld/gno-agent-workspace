<!-- NOT FOR POSTING: #6101 was closed unmerged on 2026-09-09. Kept as a record. -->

# Review: PR [#6101](https://github.com/gnolang/gno/pull/6101)
Event: REQUEST_CHANGES

## Body
This branch and [#6139](https://github.com/gnolang/gno/pull/6139) are two implementations of [issue 6026](https://github.com/gnolang/gno/issues/6026) that conflict over 19 files, all four stdlib files and the whole `grc20` tree among them, and I think #6139 is the one to keep, since deriving the identifier from `ObjectID` needs no second realm clock.

## examples/gno.land/p/demo/tokens/grc20/token.gno:46 [gh](https://github.com/gnolang/gno/blob/20d2a9f2e/examples/gno.land/p/demo/tokens/grc20/token.gno#L46) · [↗](../../../../../.worktrees/gno-review-6101/examples/gno.land/p/demo/tokens/grc20/token.gno#L46)
The prefix comes from [`m.Realm`](https://github.com/gnolang/gno/blob/20d2a9f2e/gnovm/stdlibs/chain/runtime/native.go#L59) rather than from [`origRealm`](https://github.com/gnolang/gno/blob/20d2a9f2e/examples/gno.land/p/demo/tokens/grc20/token.gno#L43), which carries the realm value [`IsCurrent`](https://github.com/gnolang/gno/blob/20d2a9f2e/examples/gno.land/p/demo/tokens/grc20/token.gno#L23-L25) authenticated, so a token built through [a plain function in another `/r/` package](https://github.com/gnolang/gno/blob/20d2a9f2e/gnovm/pkg/gnolang/machine.go#L2567-L2573) registers under one realm and names another in every event. Reject any identifier not starting with `origRealm` and a colon.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6101 -R gnolang/gno
cat > gno.land/pkg/integration/testdata/idprefix_probe.txtar <<'EOF'
loadpkg gno.land/r/demo/defi/grc20reg
loadpkg gno.land/r/demo/tests/bwrap $WORK/bwrap
loadpkg gno.land/r/demo/tests/amint $WORK/amint

gnoland start

gnokey maketx call -pkgpath gno.land/r/demo/tests/amint -func Mint -gas-fee 2000000ugnot -gas-wanted 20_000_000 -chainid=tendermint_test test1
stdout OK!
stdout 'origin=gno.land/r/demo/tests/amint id=gno.land/r/demo/tests/bwrap:[0-9]+ key=gno.land/r/demo/tests/amint\.AMT'

-- bwrap/gnomod.toml --
module = "gno.land/r/demo/tests/bwrap"
gno = "0.9"
-- bwrap/bwrap.gno --
package bwrap

import "gno.land/p/demo/tokens/grc20"

// Build forwards the caller's own realm to grc20.NewToken. Nothing here is
// privileged, and bwrap never sees amint's ledger.
func Build(name, symbol string, decimals int, rlm realm) (*grc20.Token, *grc20.PrivateLedger) {
	return grc20.NewToken(name, symbol, decimals, rlm)
}
-- amint/gnomod.toml --
module = "gno.land/r/demo/tests/amint"
gno = "0.9"
-- amint/amint.gno --
package amint

import (
	"gno.land/p/demo/tokens/grc20"
	"gno.land/r/demo/defi/grc20reg"
	"gno.land/r/demo/tests/bwrap"
)

var (
	Token  *grc20.Token
	ledger *grc20.PrivateLedger
)

func Mint(cur realm) string {
	Token, ledger = bwrap.Build("Amint", "AMT", 4, cur)
	key := grc20reg.Register(cross(cur), Token, "")
	ledger.Mint(cur.Address(), 1000)
	return "origin=" + Token.GetOriginRealm() + " id=" + Token.ID() + " key=" + key
}
EOF
go test ./gno.land/pkg/integration/ -run 'TestTestdata/idprefix_probe' -v -timeout 900s
rm gno.land/pkg/integration/testdata/idprefix_probe.txtar
```

[`Register`](https://github.com/gnolang/gno/blob/20d2a9f2e/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L40) accepts the token under `amint`, and both the `NewToken` and the `Transfer` event carry `bwrap:5`.

```
("origin=gno.land/r/demo/tests/amint id=gno.land/r/demo/tests/bwrap:5 key=gno.land/r/demo/tests/amint.AMT" string)
OK!
# … GAS WANTED, GAS USED and HEIGHT
STORAGE DELTA:  5267 bytes
# … STORAGE FEE and TOTAL TX COST
EVENTS:     [{"type":"NewToken","attrs":[{"key":"token","value":"gno.land/r/demo/tests/bwrap:5"},{"key":"name","value":"Amint"},{"key":"symbol","value":"AMT"},{"key":"decimals","value":"4"}],"pkg_path":"gno.land/p/demo/tokens/grc20"},{"type":"register","attrs":[{"key":"token_path","value":"gno.land/r/demo/tests/amint.AMT"},{"key":"pkgpath","value":"gno.land/r/demo/tests/amint"},{"key":"slug","value":""},{"key":"symbol","value":"AMT"}],"pkg_path":"gno.land/r/demo/defi/grc20reg"},{"type":"Transfer","attrs":[{"key":"token","value":"gno.land/r/demo/tests/bwrap:5"},{"key":"from","value":""},{"key":"to","value":"g1y2pdtnlqtgqxh5haky0rttetwnn7tud093gql9"},{"key":"value","value":"1000"}],"pkg_path":"gno.land/p/demo/tokens/grc20"},{"bytes_delta":1216,"fee_delta":{"denom":"ugnot","amount":121600},"pkg_path":"gno.land/r/demo/defi/grc20reg"},{"bytes_delta":273,"fee_delta":{"denom":"ugnot","amount":27300},"pkg_path":"gno.land/r/demo/tests/amint"},{"bytes_delta":3778,"fee_delta":{"denom":"ugnot","amount":377800},"pkg_path":"gno.land/r/demo/tests/bwrap"}]
# … INFO and TX HASH
--- PASS: TestTestdata/idprefix_probe (5.80s)
```

The same split is already in the shipped suite: [`grc20reg_test.gno`](https://github.com/gnolang/gno/blob/20d2a9f2e/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L14-L16) puts `testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/foo"))` in front of the constructor, and a probe in that shape answers `ID=gno.land/r/demo/defi/grc20reg:39 originRealm=gno.land/r/demo/probefoo`, because [`SetRealm`](https://github.com/gnolang/gno/blob/20d2a9f2e/gnovm/tests/stdlibs/testing/context_testing.gno#L96-L102) moves the threaded realm value and leaves `m.Realm` on `grc20reg`. Adding the prefix check leaves `p/demo/tokens/grc20` with its four filetests, `grc20factory` and `wugnot` green and reddens only those tests.
</details>

## gnovm/stdlibs/chain/runtime/native.go:55 [gh](https://github.com/gnolang/gno/blob/20d2a9f2e/gnovm/stdlibs/chain/runtime/native.go#L55) · [↗](../../../../../.worktrees/gno-review-6101/gnovm/stdlibs/chain/runtime/native.go#L55)
This increment and [the write-back under it](https://github.com/gnolang/gno/blob/20d2a9f2e/gnovm/stdlibs/chain/runtime/native.go#L58) follow `m.Realm`, so [a `/p/`-declared method running on another realm's stored object](https://github.com/gnolang/gno/blob/20d2a9f2e/gnovm/pkg/gnolang/machine.go#L2575-L2586) mints an identifier naming that realm and permanently advances the counter that [object finalisation](https://github.com/gnolang/gno/blob/20d2a9f2e/gnovm/pkg/gnolang/realm.go#L2033) draws identities from. Read the executing realm through [`execctx.CurrentRealm`](https://github.com/gnolang/gno/blob/20d2a9f2e/gnovm/stdlibs/internal/execctx/realm.go#L96-L98) instead, which is the realm [`chain/params`](https://github.com/gnolang/gno/blob/20d2a9f2e/gnovm/stdlibs/chain/params/params.go#L106) from the same frame already writes under.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6101 -R gnolang/gno
cat > gno.land/pkg/sdk/vm/foreign_probe_test.go <<'EOF'
package vm

import (
	"testing"

	"github.com/gnolang/gno/gnovm/pkg/gnolang"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/stretchr/testify/require"
)

func TestRealmIDSpendsForeignRealmCounter(t *testing.T) {
	env := setupTestEnv()
	addr := crypto.AddressFromPreimage([]byte("realm-id-foreign"))
	const (
		boxPath    = "gno.land/p/test/pbox"
		victimPath = "gno.land/r/test/victim"
		attackPath = "gno.land/r/test/attack"
	)

	ctx := env.vmk.MakeGnoTransactionStore(env.ctx)
	acc := env.acck.NewAccountWithAddress(ctx, addr)
	env.acck.SetAccount(ctx, acc)
	env.bankk.SetCoins(ctx, addr, initialBalance)

	require.NoError(t, env.vmk.AddPackage(ctx, NewMsgAddPackage(addr, boxPath, []*std.MemFile{
		{Name: "gnomod.toml", Body: gnolang.GenGnoModLatest(boxPath)},
		{Name: "pbox.gno", Body: `package pbox

import (
	"chain/params"
	"chain/runtime"
	"chain/runtime/unsafe"
)

type Box struct{ N int }

func (b *Box) Ping() string {
	params.SetString("probe", "written")
	return runtime.NewRealmID() + " identity=" + unsafe.CurrentRealm().PkgPath()
}`},
	})))
	require.NoError(t, env.vmk.AddPackage(ctx, NewMsgAddPackage(addr, victimPath, []*std.MemFile{
		{Name: "gnomod.toml", Body: gnolang.GenGnoModLatest(victimPath)},
		{Name: "victim.gno", Body: `package victim

import "gno.land/p/test/pbox"

var B = &pbox.Box{N: 1}`},
	})))
	require.NoError(t, env.vmk.AddPackage(ctx, NewMsgAddPackage(addr, attackPath, []*std.MemFile{
		{Name: "attack.gno", Body: `package attack

import (
	"chain/runtime"
	"gno.land/r/test/victim"
)

func Spend(cur realm) string {
	return "viaForeignReceiver=" + victim.B.Ping() + " own=" + runtime.NewRealmID()
}`},
		{Name: "gnomod.toml", Body: gnolang.GenGnoModLatest(attackPath)},
	})))
	env.vmk.CommitGnoTransactionStore(ctx)

	before := env.vmk.getGnoTransactionStore(env.vmk.MakeGnoTransactionStore(env.ctx)).GetPackageRealm(victimPath).Time

	callCtx := env.vmk.MakeGnoTransactionStore(env.ctx)
	res, err := env.vmk.Call(callCtx, NewMsgCall(addr, nil, attackPath, "Spend", nil))
	require.NoError(t, err)
	env.vmk.CommitGnoTransactionStore(callCtx)

	after := env.vmk.getGnoTransactionStore(env.vmk.MakeGnoTransactionStore(env.ctx)).GetPackageRealm(victimPath).Time
	t.Log(res)
	t.Logf("victimTimeBefore=%d victimTimeAfter=%d", before, after)

	var underVictim, underAttack string
	okVictim := env.prmk.GetString(env.ctx, "vm:"+victimPath+":probe", &underVictim)
	okAttack := env.prmk.GetString(env.ctx, "vm:"+attackPath+":probe", &underAttack)
	t.Logf("params under victim=%q(%v) under attack=%q(%v)", underVictim, okVictim, underAttack, okAttack)

	// IS:     the ID names victim, while the executing identity is attack.
	require.Contains(t, res, "viaForeignReceiver="+victimPath+":")
	require.Contains(t, res, "identity="+attackPath)
	require.Equal(t, before+1, after)
	require.False(t, okVictim)
	require.Equal(t, "written", underAttack)
	// SHOULD: the ID names the realm chain/params agrees is executing.
	// require.Contains(t, res, "viaForeignReceiver="+attackPath+":")
	// require.Equal(t, before, after)
}
EOF
go test -count=1 -v -run 'TestRealmIDSpendsForeignRealmCounter' ./gno.land/pkg/sdk/vm/
rm gno.land/pkg/sdk/vm/foreign_probe_test.go
```

`victim` holds a stored object, never calls the native, and pays for the call anyway: its saved counter moves while the param written from the same frame lands under `attack`.

```
("viaForeignReceiver=gno.land/r/test/victim:7 identity=gno.land/r/test/attack own=gno.land/r/test/attack:5" string)
victimTimeBefore=6 victimTimeAfter=7
params under victim=""(false) under attack="written"(true)
--- PASS: TestRealmIDSpendsForeignRealmCounter (3.64s)
```

The counter is read back through a fresh transaction store, so 7 is the persisted value rather than the one the call left in the cache. What `victim` loses is attribution plus a gap in its object-identity sequence: finalisation skips the consumed value rather than reusing it, so no object, coin or allowance moves.
</details>

## gnovm/stdlibs/native_gas.go:143 [gh](https://github.com/gnolang/gno/blob/20d2a9f2e/gnovm/stdlibs/native_gas.go#L143) · [↗](../../../../../.worktrees/gno-review-6101/gnovm/stdlibs/native_gas.go#L143)
Nothing under [`gnovm/cmd/calibrate/`](https://github.com/gnolang/gno/blob/20d2a9f2e/gnovm/cmd/calibrate/gen_native_table.py#L138-L151) benchmarks or fits `NewRealmID`, so regenerating the table [as this file's header describes](https://github.com/gnolang/gno/blob/20d2a9f2e/gnovm/stdlibs/native_gas.go#L72-L74) drops this row and makes [`chargeNativeGas`](https://github.com/gnolang/gno/blob/20d2a9f2e/gnovm/pkg/gnolang/native_gas.go#L131) panic on the first token creation under a real gas meter. Add the benchmark and the fitter spec, or record the row in the header beside [the six borrowed `chain/params` rows](https://github.com/gnolang/gno/blob/20d2a9f2e/gnovm/stdlibs/native_gas.go#L88-L102).

<details><summary>benchmark</summary>

The missing benchmark, paste-ready. It needs one line in `NATIVE_SPECS` after [`gen_native_table.py:151`](https://github.com/gnolang/gno/blob/20d2a9f2e/gnovm/cmd/calibrate/gen_native_table.py#L150-L151) to reach the fitter:

```python
    ("chain/runtime", "NewRealmID", None, "Flat",
     r"BenchmarkNative_Runtime_NewRealmID-\d+\s+\d+\s+([\d.]+)\s+ns/op"),
```

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6101 -R gnolang/gno
cat > gnovm/cmd/calibrate/newrealmid_bench_test.go <<'EOF'
package calibrate

import (
	"math"
	"testing"

	gno "github.com/gnolang/gno/gnovm/pkg/gnolang"
	"github.com/gnolang/gno/gnovm/stdlibs"
	"github.com/gnolang/gno/tm2/pkg/db/memdb"
	"github.com/gnolang/gno/tm2/pkg/store/dbadapter"
	storetypes "github.com/gnolang/gno/tm2/pkg/store/types"
)

// NewRealmID needs a persistent realm on the machine, a store to write the
// bumped counter back to, and RealmIDEnabled on the context.
func newRealmIDBench(b *testing.B) *dispatchHarness {
	b.Helper()
	const pkgPath = "gno.land/r/x"

	base := dbadapter.StoreConstructor(memdb.NewMemDB(), storetypes.StoreOptions{})
	iavl := dbadapter.StoreConstructor(memdb.NewMemDB(), storetypes.StoreOptions{})
	store := gno.NewStore(gno.NewAllocator(math.MaxInt64), base, iavl)
	rlm := gno.NewRealm(pkgPath)
	store.SetPackageRealm(rlm)
	tx := store.BeginTransaction(base.CacheWrap(), iavl.CacheWrap(), nil, nil)

	m := newDispatchMachine(0)
	addContextAndFrames(m, pkgPath)
	m.Store = tx
	m.Realm = tx.GetPackageRealm(pkgPath)
	ctx := m.Context.(stdlibs.ExecContext)
	ctx.RealmIDEnabled = true
	m.Context = ctx

	return &dispatchHarness{m: m, wrapper: resolveWrapper(b, "chain/runtime", "NewRealmID"), nReturns: 1}
}

func BenchmarkNative_Runtime_NewRealmID(b *testing.B) {
	h := newRealmIDBench(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.call()
	}
}
EOF
for i in $(seq 1 9); do
  for b in Params_SetString_1 Runtime_ChainID Runtime_NewRealmID; do
    go test -run XXX -bench "BenchmarkNative_${b}\$" -benchtime 400ms -count 1 ./gnovm/cmd/calibrate/
  done
done
rm gnovm/cmd/calibrate/newrealmid_bench_test.go
```

Medians over the nine interleaved runs that loop makes, on one AMD EPYC VM rather than the reference Xeon 8168 the header asks for. The table is derived from those runs, not printed by any of them:

| row | shipped `Base` | median ns/op |
| --- | --- | --- |
| [`chain/params.SetString`](https://github.com/gnolang/gno/blob/20d2a9f2e/gnovm/stdlibs/native_gas.go#L121) | 1772 | 3980 |
| [`chain/runtime.ChainID`](https://github.com/gnolang/gno/blob/20d2a9f2e/gnovm/stdlibs/native_gas.go#L140) | 45 | 137.7 |
| [`chain/runtime.NewRealmID`](https://github.com/gnolang/gno/blob/20d2a9f2e/gnovm/stdlibs/native_gas.go#L143) | 1772 | 4006 |

Scaling `NewRealmID` by each neighbour's own ratio of `Base` to ns lands it at 1309 on the `ChainID` anchor and 1784 on the `SetString` anchor, so the shipped 1772 sits inside that range and prices the native at parity with the `SetString` row it copies, on top of the 216 gas the 72-byte `gno.land/r/demo/defi/foo20` realm record's [amino encode](https://github.com/gnolang/gno/blob/20d2a9f2e/gnovm/pkg/gnolang/store.go#L496-L498) adds, 174 to 225 across the realms this branch touches. Those are one box's figures and its spread across the nine runs is wider than the 26 ns between the two 1772 rows, so what carries is each neighbour's own `Base`-to-ns ratio rather than the absolutes. The regenerated [`native_gas_table.go.txt`](https://github.com/gnolang/gno/blob/20d2a9f2e/gnovm/cmd/calibrate/native_gas_table.go.txt#L1) holds 46 rows against production's 73, 30 of those 31 missing rows are already missing on master, and the header's exception block covers the six `chain/params Get*` rows alone, so this row joins the three [`chain/runtime/unsafe` rows](https://github.com/gnolang/gno/blob/20d2a9f2e/gnovm/stdlibs/native_gas.go#L147-L149) that sit unfitted, unsampled and unnamed in the same position.
</details>

## gnovm/stdlibs/chain/runtime/native.go:59 [gh](https://github.com/gnolang/gno/blob/20d2a9f2e/gnovm/stdlibs/chain/runtime/native.go#L59) · [↗](../../../../../.worktrees/gno-review-6101/gnovm/stdlibs/chain/runtime/native.go#L59)
The number is the realm's object counter, so a realm redeployed after an unrelated edit mints a different identifier and nothing off chain can pin one. Say in the ADR's Consequences and in `NewToken`'s doc that an identifier belongs to one deployment.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6101 -R gnolang/gno
cat > gno.land/pkg/integration/testdata/idcount_probe.txtar <<'EOF'
loadpkg gno.land/r/demo/tests/idlean $WORK/idlean
loadpkg gno.land/r/demo/tests/idpad $WORK/idpad

gnoland start

gnokey query vm/qeval --data 'gno.land/r/demo/tests/idlean.First()'
stdout '"gno.land/r/demo/tests/idlean:7" string'

gnokey query vm/qeval --data 'gno.land/r/demo/tests/idpad.First()'
stdout '"gno.land/r/demo/tests/idpad:17" string'

-- idlean/gnomod.toml --
module = "gno.land/r/demo/tests/idlean"
gno = "0.9"
-- idlean/idlean.gno --
package idlean

import "chain/runtime"

var first string

func init(cur realm) {
	first = runtime.NewRealmID()
}

func First() string { return first }
-- idpad/gnomod.toml --
module = "gno.land/r/demo/tests/idpad"
gno = "0.9"
-- idpad/idpad.gno --
package idpad

import "chain/runtime"

type filler struct{ a, b, c string }

var (
	pad1 = &filler{"a", "b", "c"}
	pad2 = &filler{"d", "e", "f"}
	pad3 = []*filler{{"g", "h", "i"}}

	first string
)

func init(cur realm) {
	first = runtime.NewRealmID()
}

func First() string { return first }
EOF
go test ./gno.land/pkg/integration/ -run 'TestTestdata/idcount_probe' -v -timeout 900s
rm gno.land/pkg/integration/testdata/idcount_probe.txtar
```

The two realms' identifier code is byte-identical and their answers are ten apart.

```
data: ("gno.land/r/demo/tests/idlean:7" string)
data: ("gno.land/r/demo/tests/idpad:17" string)
--- PASS: TestTestdata/idcount_probe (4.47s)
```

The archives in this branch show the same spread on one realm: `foo20` is [`foo20:22`](https://github.com/gnolang/gno/blob/20d2a9f2e/gno.land/pkg/integration/testdata/grc20_registry_emit.txtar#L26) loaded at genesis and [`foo20:24`](https://github.com/gnolang/gno/blob/20d2a9f2e/gno.land/pkg/integration/testdata/grc20_id_persists_cross_realm.txtar#L17) deployed by transaction with one file added, and five assertions hard-code one of the two. Master's `gno.land/r/demo/defi/foo20.FOO.0000000` was derivable from the source alone.
</details>

## examples/gno.land/p/demo/tokens/grc20/token_test.gno:15-27 [gh](https://github.com/gnolang/gno/blob/20d2a9f2e/examples/gno.land/p/demo/tokens/grc20/token_test.gno#L15-L27) · [↗](../../../../../.worktrees/gno-review-6101/examples/gno.land/p/demo/tokens/grc20/token_test.gno#L15)
Missing test: this fixture builds the struct directly, so the package's own tests do not call [`grc20.NewToken`](https://github.com/gnolang/gno/blob/20d2a9f2e/examples/gno.land/p/demo/tokens/grc20/token.gno#L22) and its [`ErrInvalidDecimals`](https://github.com/gnolang/gno/blob/20d2a9f2e/examples/gno.land/p/demo/tokens/grc20/token.gno#L36-L38), [`ErrNotRealm`](https://github.com/gnolang/gno/blob/20d2a9f2e/examples/gno.land/p/demo/tokens/grc20/token.gno#L26-L28) and [`ErrSpoofedRealm`](https://github.com/gnolang/gno/blob/20d2a9f2e/examples/gno.land/p/demo/tokens/grc20/token.gno#L23-L25) paths have no assertion anywhere in `examples/`. The cases need to move to `filetests/`, since a `/p/` `PKGPATH` makes the constructor abort with [`realm ID issuance requires a persistent realm`](https://github.com/gnolang/gno/blob/20d2a9f2e/gnovm/stdlibs/chain/runtime/native.go#L49-L51) even after [`testing.SetRealm`](https://github.com/gnolang/gno/blob/20d2a9f2e/gnovm/tests/stdlibs/testing/context_testing.gno#L96-L102) runs.

<details><summary>test cases</summary>

Sibling [`grc721`](https://github.com/gnolang/gno/blob/20d2a9f2e/examples/gno.land/p/demo/tokens/grc721/token_test.gno#L106-L114) still covers the same set. Drop this in `examples/gno.land/p/demo/tokens/grc20/filetests/newtoken_rejects_filetest.gno` and run `gno test ./gno.land/p/demo/tokens/grc20` from `examples/`. It passes at this head.

```go
// PKGPATH: gno.land/r/demo/grc20reject
package grc20reject

import (
	"strings"

	"gno.land/p/demo/tokens/grc20"
)

// NewToken is a /p/ call from this realm's own frame, so its panic is not
// cross-realm and recover sees it.
func mustPanic(label string, want error, fn func()) {
	defer func() {
		r := recover()
		if r == nil {
			println(label, "no panic")
		} else if r != want {
			println(label, "wrong error")
		} else {
			println(label, "ok")
		}
	}()
	fn()
}

func main(cur realm) {
	mustPanic("empty name", grc20.ErrInvalidName, func() {
		grc20.NewToken("", "OK", 4, cur)
	})
	mustPanic("long name", grc20.ErrInvalidName, func() {
		grc20.NewToken(strings.Repeat("a", grc20.MaxNameLen+1), "OK", 4, cur)
	})
	mustPanic("control char in name", grc20.ErrInvalidName, func() {
		grc20.NewToken("bad\x01name", "OK", 4, cur)
	})
	mustPanic("empty symbol", grc20.ErrInvalidSymbol, func() {
		grc20.NewToken("Name", "", 4, cur)
	})
	mustPanic("dot in symbol", grc20.ErrInvalidSymbol, func() {
		grc20.NewToken("Name", "BA.D", 4, cur)
	})
	mustPanic("slash in symbol", grc20.ErrInvalidSymbol, func() {
		grc20.NewToken("Name", "BA/D", 4, cur)
	})
	mustPanic("negative decimals", grc20.ErrInvalidDecimals, func() {
		grc20.NewToken("Name", "OK", -1, cur)
	})
	mustPanic("decimals over cap", grc20.ErrInvalidDecimals, func() {
		grc20.NewToken("Name", "OK", grc20.MaxDecimals+1, cur)
	})

	tok, _ := grc20.NewToken(strings.Repeat("a", grc20.MaxNameLen), strings.Repeat("A", grc20.MaxSymbolLen), grc20.MaxDecimals, cur)
	println("boundary accepted", tok != nil)
	utf8Tok, _ := grc20.NewToken("Доллар", "RUB", 2, cur)
	println("utf8 name accepted", utf8Tok != nil)
}

// Output:
// empty name ok
// long name ok
// control char in name ok
// empty symbol ok
// dot in symbol ok
// slash in symbol ok
// negative decimals ok
// decimals over cap ok
// boundary accepted true
// utf8 name accepted true
```
</details>

## gno.land/pkg/sdk/vm/keeper_test.go:1151-1152 [gh](https://github.com/gnolang/gno/blob/20d2a9f2e/gno.land/pkg/sdk/vm/keeper_test.go#L1151-L1152) · [↗](../../../../../.worktrees/gno-review-6101/gno.land/pkg/sdk/vm/keeper_test.go#L1151)
Missing test: these rows leave out the `/r/` helper called plainly with a threaded realm, the one shape where the identifier and the realm [`NewToken`](https://github.com/gnolang/gno/blob/20d2a9f2e/examples/gno.land/p/demo/tokens/grc20/token.gno#L22) authenticated name different realms.

<details><summary>test cases</summary>

`NewToken` puts its `realm` parameter last, so it and any helper written the same way are non-crossing under [`FuncType.IsCrossing`](https://github.com/gnolang/gno/blob/20d2a9f2e/gnovm/pkg/gnolang/types.go#L1379-L1385), which is what puts this row inside borrow rule #1. Drop it in `gno.land/pkg/sdk/vm/` and run `go test -v -run TestVMKeeperNewRealmIDProvenanceThirdRow ./gno.land/pkg/sdk/vm/`; it passes at this head with the current expectation and reddens on the commented one.

```go
package vm

import (
	"testing"

	"github.com/gnolang/gno/gnovm/pkg/gnolang"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/stretchr/testify/require"
)

func TestVMKeeperNewRealmIDProvenanceThirdRow(t *testing.T) {
	env := setupTestEnv()
	addr := crypto.AddressFromPreimage([]byte("realm-id-third-row"))
	const (
		wrapperPath = "gno.land/r/test/cwrapper"
		callerPath  = "gno.land/r/test/dcaller"
	)

	ctx := env.vmk.MakeGnoTransactionStore(env.ctx)
	acc := env.acck.NewAccountWithAddress(ctx, addr)
	env.acck.SetAccount(ctx, acc)
	require.NoError(t, env.bankk.SetCoins(ctx, addr, initialBalance))

	// cwrapper.New is NOT a crossing function. It takes the realm value the
	// caller threads in, the same signature shape grc20.NewToken has.
	require.NoError(t, env.vmk.AddPackage(ctx, NewMsgAddPackage(addr, wrapperPath, []*std.MemFile{
		{Name: "cwrapper.gno", Body: `package cwrapper

import "chain/runtime"

func New(tag string, rlm realm) (string, string, bool) {
	return runtime.NewRealmID(), rlm.PkgPath(), rlm.IsCurrent()
}`},
		{Name: "gnomod.toml", Body: gnolang.GenGnoModLatest(wrapperPath)},
	})))
	require.NoError(t, env.vmk.AddPackage(ctx, NewMsgAddPackage(addr, callerPath, []*std.MemFile{
		{Name: "dcaller.gno", Body: `package dcaller

import "gno.land/r/test/cwrapper"

func NewFromWrapper(cur realm) string {
	id, verified, current := cwrapper.New("t", cur)
	if !current {
		return "id=" + id + " verified=" + verified + " current=false"
	}
	return "id=" + id + " verified=" + verified + " current=true"
}`},
		{Name: "gnomod.toml", Body: gnolang.GenGnoModLatest(callerPath)},
	})))
	env.vmk.CommitGnoTransactionStore(ctx)

	callCtx := env.vmk.MakeGnoTransactionStore(env.ctx)
	res, err := env.vmk.Call(callCtx, NewMsgCall(addr, nil, callerPath, "NewFromWrapper", nil))
	require.NoError(t, err)
	t.Log(res)

	// The realm value the callee authenticated is the caller's, and it is current.
	require.Contains(t, res, "verified="+callerPath)
	require.Contains(t, res, "current=true")

	// IS: the ID names the wrapper, not the realm whose cur passed IsCurrent.
	require.Contains(t, res, "id="+wrapperPath+":")
	// SHOULD: the ID names the realm the callee authenticated.
	// require.Contains(t, res, "id="+callerPath+":")
}
```

```
("id=gno.land/r/test/cwrapper:5 verified=gno.land/r/test/dcaller current=true" string)
--- PASS: TestVMKeeperNewRealmIDProvenanceThirdRow (0.16s)
```
</details>

## gnovm/stdlibs/chain/runtime/native.go:49-51 [gh](https://github.com/gnolang/gno/blob/20d2a9f2e/gnovm/stdlibs/chain/runtime/native.go#L49-L51) · [↗](../../../../../.worktrees/gno-review-6101/gnovm/stdlibs/chain/runtime/native.go#L49)
Nit: [`IsRealmPath`](https://github.com/gnolang/gno/blob/20d2a9f2e/gnovm/pkg/gnolang/mempackage.go#L81-L90) refuses the `/e/…/run` path of a `gnokey maketx run` script and the `<realm>_test` path of a realm's external test package, so both lose the constructor, and the ADR does not say so.

## gno.land/adr/prxxxx_grc20_realm_ids.md:1 [gh](https://github.com/gnolang/gno/blob/20d2a9f2e/gno.land/adr/prxxxx_grc20_realm_ids.md?plain=1#L1) · [↗](../../../../../.worktrees/gno-review-6101/gno.land/adr/prxxxx_grc20_realm_ids.md#L1)
Nit: this filename keeps the `prxxxx` placeholder [`AGENTS.md`](https://github.com/gnolang/gno/blob/20d2a9f2e/AGENTS.md?plain=1#L132) allows only while the number is unknown, unlike the twenty-nine other PR-scoped files in `gno.land/adr/`, [`pr6025_prod_only_typecheck_at_addpackage.md`](https://github.com/gnolang/gno/blob/20d2a9f2e/gno.land/adr/pr6025_prod_only_typecheck_at_addpackage.md?plain=1#L1) for one.

## gno.land/adr/prxxxx_grc20_realm_ids.md:22-24 [gh](https://github.com/gnolang/gno/blob/20d2a9f2e/gno.land/adr/prxxxx_grc20_realm_ids.md?plain=1#L22-L24) · [↗](../../../../../.worktrees/gno-review-6101/gno.land/adr/prxxxx_grc20_realm_ids.md#L22)
Nit: the code also enables issuance for [`Call`](https://github.com/gnolang/gno/blob/20d2a9f2e/gno.land/pkg/sdk/vm/keeper.go#L1186) and [`Run`](https://github.com/gnolang/gno/blob/20d2a9f2e/gno.land/pkg/sdk/vm/keeper.go#L1459), which this paragraph leaves out.

## SKIP examples/gno.land/p/demo/tokens/grc20/filetests/event_provenance_filetest.gno:5-16 [gh](https://github.com/gnolang/gno/blob/20d2a9f2e/examples/gno.land/p/demo/tokens/grc20/filetests/event_provenance_filetest.gno#L5-L16) · [↗](../../../../../.worktrees/gno-review-6101/examples/gno.land/p/demo/tokens/grc20/filetests/event_provenance_filetest.gno#L5)
Nit: this header opens "Three fields matter:" and lists two, and the second says the VM builds the identifier from the current realm path after `IsCurrent` verifies it, which the `token.gno:46` finding's runs contradict. Skipped: a finding about a code comment's own wording changes no behaviour, so the measurement stays in the review file.

## SKIP examples/gno.land/p/demo/tokens/grc20/token.gno:46 [gh](https://github.com/gnolang/gno/blob/20d2a9f2e/examples/gno.land/p/demo/tokens/grc20/token.gno#L46) · [↗](../../../../../.worktrees/gno-review-6101/examples/gno.land/p/demo/tokens/grc20/token.gno#L46)
Nit: this assignment changes the value inside the `token` attribute and [the emit under it](https://github.com/gnolang/gno/blob/20d2a9f2e/examples/gno.land/p/demo/tokens/grc20/token.gno#L55-L60) keeps the same four attributes it carried at the merge base, so the body's claim that the branch removes a redundant `realm` attribute from `NewToken` names a change that is not in the diff. Skipped: a mistake in the description stays in the review file.

## examples/gno.land/r/demo/tests/grc20xfer/grc20xfer.gno:16-20 [gh](https://github.com/gnolang/gno/blob/20d2a9f2e/examples/gno.land/r/demo/tests/grc20xfer/grc20xfer.gno#L16-L20) · [↗](../../../../../.worktrees/gno-review-6101/examples/gno.land/r/demo/tests/grc20xfer/grc20xfer.gno#L16)
Suggestion: this realm exists only to feed one archive, loads on every chain, and lets any caller spend the realm's own balance through [`RealmTeller`](https://github.com/gnolang/gno/blob/20d2a9f2e/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L59). Move both files into `grc20_id_persists_cross_realm.txtar` the way it already inlines [`idprobe.gno`](https://github.com/gnolang/gno/blob/20d2a9f2e/gno.land/pkg/integration/testdata/grc20_id_persists_cross_realm.txtar#L42-L49), and delete the realm.

## gnovm/pkg/gnolang/objectid_reservation_test.go:205-245 [gh](https://github.com/gnolang/gno/blob/20d2a9f2e/gnovm/pkg/gnolang/objectid_reservation_test.go#L205-L245) · [↗](../../../../../.worktrees/gno-review-6101/gnovm/pkg/gnolang/objectid_reservation_test.go#L205)
Refactor: this test tracks handed-out values in a map to assert none repeats, and strict monotonicity of the one counter gives the same property in 28 lines against 41, covering the closing check that a later object does not reuse a reserved value.

```suggestion
func TestObjectID_SharedRealmTimeModelNeverCollides(t *testing.T) {
	rlm := NewRealm("gno.land/r/demo/objid_reservation")
	alloc := NewAllocator(math.MaxInt64)
	alloc.currentRealmID = rlm.ID
	alloc.currentRealmPath = rlm.Path

	reserveID := func() uint64 {
		rlm.Time++
		return rlm.Time
	}

	finalizeOne := func() uint64 {
		sv := alloc.NewStruct(nil, nil)
		rlm.assignNewObjectID(nil, sv)
		return sv.GetObjectID().NewTime
	}

	var handed []uint64
	for _, next := range []func() uint64{finalizeOne, reserveID, reserveID, finalizeOne, reserveID, finalizeOne, finalizeOne} {
		v := next()
		if len(handed) > 0 {
			require.Greater(t, v, handed[len(handed)-1],
				"reservations and finalizations share one strictly increasing counter")
		}
		handed = append(handed, v)
	}
	require.Len(t, handed, 7)
}
```
