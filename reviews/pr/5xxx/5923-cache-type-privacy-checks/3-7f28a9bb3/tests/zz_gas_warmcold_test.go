// Does type-privacy cache warmth change billed gas? The ADR for PR 5923
// claims "No gas or consensus impact: this whole path is unmetered before
// and after". assertTypeIsPublic reaches isPkgPrivateFromPkgPath ->
// Store.GetPackage -> GetObjectSafe -> loadObjectSafe, which calls
// baseStore.Get(ds.gctx, ...) and ds.consumeGas(GasAminoDecode...). On a
// cold cache the walk runs and pays those reads; on a warm cache
// typeHasPrivateDep short-circuits and pays nothing.
//
// Run: from a local clone of gnolang/gno at the PR head, drop this file
// into gnovm/pkg/gnolang/ and run:
//
//	go test ./gnovm/pkg/gnolang/ -run TestGasWarmCold -v
//
// Head-only: typeHasPrivateDep does not exist at the merge base. The
// merge-base number is the "cold" column, which is what master always
// pays.
package gnolang

import (
	"io"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/db/memdb"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/gnolang/gno/tm2/pkg/store/dbadapter"
	storetypes "github.com/gnolang/gno/tm2/pkg/store/types"
)

func TestGasWarmCold(t *testing.T) {
	db := memdb.NewMemDB()
	tm2Store := dbadapter.StoreConstructor(db, storetypes.StoreOptions{})
	st := NewStore(nil, tm2Store, tm2Store)

	// Deploy a package holding a type, and commit it to the backend.
	w1 := tm2Store.CacheWrap()
	tx1 := st.BeginTransaction(w1, w1, nil, nil)
	m := NewMachineWithOptions(MachineOptions{PkgPath: "gno.vm/t/thing", Store: tx1, Output: io.Discard})
	m.RunMemPackage(&std.MemPackage{
		Type: MPUserProd, Name: "thing", Path: "gno.vm/t/thing",
		Files: []*std.MemFile{{Name: "thing.gno", Body: "package thing\n\ntype Thing struct{ A int }\n"}},
	}, true)
	tx1.Write()
	w1.Write()

	// The realm doing the saving is NOT the package declaring the type,
	// so the merge base also reaches isPkgPrivateFromPkgPath here.
	rlm := NewRealm("gno.vm/t/holder")

	measure := func(w storetypes.Store) (gas int64, tx TransactionStore) {
		meter := storetypes.NewInfiniteGasMeter()
		gctx := &storetypes.GasContext{Meter: meter, Config: storetypes.DefaultGasConfig()}
		tx = st.BeginTransaction(w, w, gctx, meter)
		typ := tx.GetType("gno.vm/t/thing.Thing")
		before := meter.GasConsumed() // exclude GetType's own I/O
		rlm.assertTypeIsPublic(tx, typ, map[TypeID]struct{}{})
		return meter.GasConsumed() - before, tx
	}

	w2 := tm2Store.CacheWrap()
	cold, tx2 := measure(w2)
	tx2.Write() // commit: promotes the verdict to the shared cache

	w3 := tm2Store.CacheWrap()
	warm, _ := measure(w3)

	t.Logf("gas consumed by assertTypeIsPublic: cold=%d warm=%d", cold, warm)
	if cold != warm {
		t.Errorf("cache warmth is visible in billed gas: cold=%d warm=%d", cold, warm)
	}
}
