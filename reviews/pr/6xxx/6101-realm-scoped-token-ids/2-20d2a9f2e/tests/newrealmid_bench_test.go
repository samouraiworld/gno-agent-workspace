// The calibration benchmark chain/runtime.NewRealmID ships without, which is
// why re-running gen_native_table.py drops its row. Add the matching one-line
// NATIVE_SPECS entry after gen_native_table.py:151:
//
//	("chain/runtime", "NewRealmID", None, "Flat",
//	 r"BenchmarkNative_Runtime_NewRealmID-\d+\s+\d+\s+([\d.]+)\s+ns/op"),
//
// Medians of 9 interleaved 400ms runs on one AMD EPYC VM at 20d2a9f2e, beside
// the two rows the shipped Base is argued from. The NewRealmID figure carries
// the SetPackageRealm amino encode and KVStore write, which production meters
// separately through GasAminoEncode.
//
//	row                       shipped Base   measured here
//	chain/runtime.ChainID               45       137.7 ns
//	chain/params.SetString            1772      3980.0 ns
//	chain/runtime.NewRealmID          1772      4006.0 ns
//
/* Run: from a gno checkout:
gh pr checkout 6101 -R gnolang/gno && git checkout 20d2a9f2e
curl -fsSL -o gnovm/cmd/calibrate/newrealmid_bench_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6101-realm-scoped-token-ids/2-20d2a9f2e/tests/newrealmid_bench_test.go
for i in $(seq 1 9); do
  for b in Params_SetString_1 Runtime_ChainID Runtime_NewRealmID; do
    go test -run XXX -bench "BenchmarkNative_${b}\$" -benchtime 400ms -count 1 ./gnovm/cmd/calibrate/
  done
done
rm gnovm/cmd/calibrate/newrealmid_bench_test.go
*/
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
