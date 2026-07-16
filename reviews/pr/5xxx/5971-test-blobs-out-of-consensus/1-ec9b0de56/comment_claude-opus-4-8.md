# Review: PR [#5971](https://github.com/gnolang/gno/pull/5971)
Event: REQUEST_CHANGES

## Body
Verified on ec9b0de56: appending a test function to `gnovm/stdlibs/chain/address_test.gno` leaves `TestAppHashCrossrealm38` at its pinned value. Changing the line that writes the blob to `ds.baseStore` back to `ds.iavlStore` restores master's own `058910b2…`, and that same test edit then moves the hash. So the new pin is the blob move and nothing else.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5971-test-blobs-out-of-consensus/1-ec9b0de56/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/pkg/gnolang/store.go:1026 [↗](../../../../../.worktrees/gno-review-5971/gnovm/pkg/gnolang/store.go#L1026)
Sending the blob to `baseStore` also drops a flat 223,600 gas off deploying any package that ships a test file, so the consensus impact is not app-hash-only. The write is metered by its destination: [`cacheStore.Set`](https://github.com/gnolang/gno/blob/ec9b0de56/tm2/pkg/store/cache/store.go#L186-L196) charges depth-scaled gas behind [`bptree`](https://github.com/gnolang/gno/blob/ec9b0de56/tm2/pkg/store/bptree/store.go#L46), which implements [`DepthEstimator`](https://github.com/gnolang/gno/blob/ec9b0de56/tm2/pkg/store/cache/store.go#L79-L80), and a flat cost behind `dbadapter`, which does not. Under [the default depth params](https://github.com/gnolang/gno/blob/ec9b0de56/gno.land/pkg/sdk/vm/params.go#L40-L42) that is `2.0×ReadCostFlat + 5.4×WriteCostFlat` against a bare `WriteCostFlat`, so the per-byte term cancels and the delta is a constant no matter how large the test files are.

<details><summary>repro</summary>

The amino-encode gas inside [`setMemPackageBlob`](https://github.com/gnolang/gno/blob/ec9b0de56/gnovm/pkg/gnolang/store.go#L1049-L1056) is length-driven and unchanged, but it is not the only charge on the path. Closed form, with [`ReadCostFlat` 59,000 and `WriteCostFlat` 24,000](https://github.com/gnolang/gno/blob/ec9b0de56/tm2/pkg/store/types/gas.go#L404-L407):

```
depth (bptree)    = 2.0×59,000 + 5.4×24,000 + 14×len = 247,600 + 14×len
flat  (dbadapter) =              1.0×24,000 + 14×len =  24,000 + 14×len
delta                                                = 223,600, for any len
```

Under `-tags gastrace` the sole `Set` on `pkg:gno.land/r/hellotest#allbutprod` charges 26,926 (`info=depth=false`) at head and 250,526 (`info=depth=true`) when routed back. The tx-level measurement:

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5971 -R gnolang/gno

cat > gno.land/pkg/sdk/vm/zz_blobgas_test.go <<'EOF'
package vm

import (
	"fmt"
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoland/ugnot"
	"github.com/gnolang/gno/gnovm/pkg/gnolang"
	bft "github.com/gnolang/gno/tm2/pkg/bft/types"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/sdk"
	"github.com/gnolang/gno/tm2/pkg/sdk/auth"
	"github.com/gnolang/gno/tm2/pkg/std"
)

func TestZZBlobGas(t *testing.T) {
	for _, withTest := range []bool{true, false} {
		env := setupTestEnv()
		ctx := env.ctx.WithBlockHeader(&bft.Header{Height: int64(1)}).WithMode(sdk.RunTxModeDeliver)
		addr := crypto.AddressFromPreimage([]byte("test1"))
		env.acck.SetAccount(ctx, env.acck.NewAccountWithAddress(ctx, addr))
		env.bankk.SetCoins(ctx, addr, std.MustParseCoins(ugnot.ValueString(100000000)))

		const pkgPath = "gno.land/r/hellotest"
		files := []*std.MemFile{
			{Name: "gnomod.toml", Body: gnolang.GenGnoModLatest(pkgPath)},
			{Name: "hello.gno", Body: "package hellotest\n\nfunc Echo() string { return \"hello world\" }\n"},
		}
		if withTest {
			files = append(files, &std.MemFile{Name: "hello_test.gno", Body: "package hellotest\n\nimport \"testing\"\n\nfunc TestEcho(t *testing.T) {\n\tif Echo() != \"hello world\" {\n\t\tt.Fatal(\"bad\")\n\t}\n}\n"})
		}
		fee := std.NewFee(50000000, std.MustParseCoin(ugnot.ValueString(1)))
		tx := std.NewTx([]std.Msg{NewMsgAddPackage(addr, pkgPath, files)}, fee, []std.Signature{}, "")

		gctx := auth.SetGasMeter(ctx, tx.Fee.GasWanted)
		gctx, _ = gctx.CacheContext()
		gctx = env.vmh.vm.MakeGnoTransactionStore(gctx)
		if res := env.vmh.Process(gctx, tx.GetMsgs()[0]); !res.IsOK() {
			t.Fatalf("addpkg failed: %v", res)
		}
		fmt.Printf("GAS with_test_file=%v -> %d\n", withTest, gctx.GasMeter().GasConsumed())
	}
}
EOF

echo "--- as merged:"
go test -run TestZZBlobGas -v ./gno.land/pkg/sdk/vm/ 2>&1 | grep '^GAS'

echo "--- blob routed back to iavlStore:"
sed -i 's|setMemPackageBlob(ds.baseStore, \[\]byte(backendPackageAllButProdKey|setMemPackageBlob(ds.iavlStore, []byte(backendPackageAllButProdKey|' gnovm/pkg/gnolang/store.go
go test -run TestZZBlobGas -v ./gno.land/pkg/sdk/vm/ 2>&1 | grep '^GAS'

rm gno.land/pkg/sdk/vm/zz_blobgas_test.go
git checkout HEAD -- gnovm/pkg/gnolang/store.go
```

```
--- as merged:
GAS with_test_file=true -> 21680911
GAS with_test_file=false -> 3771428
--- blob routed back to iavlStore:
GAS with_test_file=true -> 21904511
GAS with_test_file=false -> 3771428
```
</details>

## gno.land/pkg/sdk/vm/gas_test.go:333 [↗](../../../../../.worktrees/gno-review-5971/gno.land/pkg/sdk/vm/gas_test.go#L333)
Missing test: no golden deploys a package that ships a test file, the one shape whose gas this changes. [`setupAddPkg`](https://github.com/gnolang/gno/blob/ec9b0de56/gno.land/pkg/sdk/vm/gas_test.go#L333-L374) builds `gnomod.toml` plus one `.gno` file, so no `#allbutprod` blob is written, and [`addpkg_import_testdep_gas.txtar`](https://github.com/gnolang/gno/blob/ec9b0de56/gno.land/pkg/integration/testdata/addpkg_import_testdep_gas.txtar) carries a `_test.gno` but pins equal gas across two importers, not the deploy's own cost. Nothing holds the 223,600-gas delta.

<details><summary>test cases</summary>

```go
// gno.land/pkg/sdk/vm/addpkg_test_blob_gas_test.go
package vm

import (
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoland/ugnot"
	"github.com/gnolang/gno/gnovm/pkg/gnolang"
	bft "github.com/gnolang/gno/tm2/pkg/bft/types"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/sdk"
	"github.com/gnolang/gno/tm2/pkg/sdk/auth"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/stretchr/testify/require"
)

// expectedAddPkgWithTestBlobGas pins DeliverTx gas for deploying a package that
// ships a test file. Deploying the same package without the test file writes no
// #allbutprod blob and is unaffected by the store move.
const expectedAddPkgWithTestBlobGas = int64(21680911)

func TestAddPkgWithTestBlobGas(t *testing.T) {
	env := setupTestEnv()
	// A non-genesis block: genesis uses an infinite gas meter.
	ctx := env.ctx.WithBlockHeader(&bft.Header{Height: int64(1)}).WithMode(sdk.RunTxModeDeliver)

	addr := crypto.AddressFromPreimage([]byte("test1"))
	acc := env.acck.NewAccountWithAddress(ctx, addr)
	env.acck.SetAccount(ctx, acc)
	env.bankk.SetCoins(ctx, addr, std.MustParseCoins(ugnot.ValueString(100000000)))

	const pkgPath = "gno.land/r/hellotest"
	files := []*std.MemFile{
		{Name: "gnomod.toml", Body: gnolang.GenGnoModLatest(pkgPath)},
		{Name: "hello.gno", Body: "package hellotest\n\nfunc Echo() string { return \"hello world\" }\n"},
		{Name: "hello_test.gno", Body: "package hellotest\n\nimport \"testing\"\n\nfunc TestEcho(t *testing.T) {\n\tif Echo() != \"hello world\" {\n\t\tt.Fatal(\"bad\")\n\t}\n}\n"},
	}
	msg := NewMsgAddPackage(addr, pkgPath, files)
	fee := std.NewFee(50000000, std.MustParseCoin(ugnot.ValueString(1)))
	tx := std.NewTx([]std.Msg{msg}, fee, []std.Signature{}, "")

	gctx := auth.SetGasMeter(ctx, tx.Fee.GasWanted)
	gctx, _ = gctx.CacheContext()
	gctx = env.vmh.vm.MakeGnoTransactionStore(gctx)

	res := env.vmh.Process(gctx, tx.GetMsgs()[0])
	require.True(t, res.IsOK(), "addpkg must succeed: %v", res)

	require.Equal(t, expectedAddPkgWithTestBlobGas, gctx.GasMeter().GasConsumed(),
		"addpkg gas for a package shipping a test file moved; the #allbutprod blob's "+
			"store write is metered by its destination store, so this is consensus-affecting")
}
```
</details>

## gnovm/pkg/gnolang/store.go:1271 [↗](../../../../../.worktrees/gno-review-5971/gnovm/pkg/gnolang/store.go#L1271)
Missing test: nothing lists paths on the same-store shape this branch exists for. [`TestTransactionStore`](https://github.com/gnolang/gno/blob/ec9b0de56/gnovm/pkg/gnolang/store_test.go#L16-L22) builds that shape but never calls `FindPathsByPrefix`, the three call sites that do ([244](https://github.com/gnolang/gno/blob/ec9b0de56/gnovm/pkg/gnolang/store_test.go#L244), [273](https://github.com/gnolang/gno/blob/ec9b0de56/gnovm/pkg/gnolang/store_test.go#L273), [288](https://github.com/gnolang/gno/blob/ec9b0de56/gnovm/pkg/gnolang/store_test.go#L288)) use two distinct backends, and the only production caller passes distinct stores too. So the lockstep argument in [the comment above](https://github.com/gnolang/gno/blob/ec9b0de56/gnovm/pkg/gnolang/store.go#L1217-L1223) is unchecked.

<details><summary>test cases</summary>

```go
// gnovm/pkg/gnolang/store_test.go
func TestFindByPrefixSameStoreBackend(t *testing.T) {
	db := memdb.NewMemDB()
	one := dbadapter.StoreConstructor(db, storetypes.StoreOptions{})
	store := NewStore(nil, one, one)

	add := func(name string, files ...*std.MemFile) {
		store.AddMemPackage(&std.MemPackage{
			Type:  MPUserAll,
			Name:  name,
			Path:  "gno.land/r/demo/" + name,
			Files: files,
		}, MPUserAll)
	}
	// Production file plus test file: a prod blob and an #allbutprod blob.
	add("alpha", &std.MemFile{Name: "alpha.gno", Body: "package alpha\n"},
		&std.MemFile{Name: "alpha_test.gno", Body: "package alpha\n"})
	// Test file only: an #allbutprod blob and no prod blob.
	add("beta", &std.MemFile{Name: "beta_test.gno", Body: "package beta\n"})
	// Production file only: a prod blob and no #allbutprod blob.
	add("gamma", &std.MemFile{Name: "gamma.gno", Body: "package gamma\n"})

	var got []string
	store.FindPathsByPrefix("gno.land")(func(p string) bool {
		got = append(got, p)
		return true
	})
	require.Equal(t, []string{
		"gno.land/r/demo/alpha",
		"gno.land/r/demo/beta",
		"gno.land/r/demo/gamma",
	}, got, "each package must be listed exactly once")
}
```
</details>

## config/addrbook.json:1 [↗](../../../../../.worktrees/gno-review-5971/config/addrbook.json#L1)
Nothing ignores `config/addrbook.json`, so `gnoland start` from the repo root re-dirties the tree for everyone. It is a node runtime artifact and the only tracked file under `config/`, committed in db302a1e0 alongside the store change.
