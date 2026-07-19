/* Run: from a gno checkout:
gh pr checkout 5923 -R gnolang/gno && git checkout dcd6db417
curl -fsSL -o gnovm/pkg/gnolang/privdep_reach_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5923-cache-type-privacy-checks/1-dcd6db417/tests/privdep_reach_test.go
go test -v -run 'TestPrivateDepCache' ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/privdep_reach_test.go
*/

// The privateDep memo lives on the Type object itself, so it only helps
// when the same object is handed to assertTypeIsPublic twice. Both tests
// fail at dcd6db417: a type reloaded in a later transaction is a fresh
// object, and a declared type carrying any method is walked as a cycle
// and never committed to the memo.
package gnolang

import (
	"io"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/db/memdb"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/gnolang/gno/tm2/pkg/store/dbadapter"
	storetypes "github.com/gnolang/gno/tm2/pkg/store/types"
)

func TestPrivateDepCacheSurvivesTransaction(t *testing.T) {
	db := memdb.NewMemDB()
	tm2Store := dbadapter.StoreConstructor(db, storetypes.StoreOptions{})
	st := NewStore(nil, tm2Store, tm2Store)

	w1 := tm2Store.CacheWrap()
	tx1 := st.BeginTransaction(w1, w1, nil, nil)
	m := NewMachineWithOptions(MachineOptions{PkgPath: "gno.vm/t/hello", Store: tx1, Output: io.Discard})
	m.RunMemPackage(&std.MemPackage{
		Type: MPUserProd, Name: "hello", Path: "gno.vm/t/hello",
		Files: []*std.MemFile{{Name: "hello.gno", Body: "package hello\n\ntype Coin struct{ Denom string; Amount int }\n"}},
	}, true)
	tx1.Write()
	w1.Write()

	first := st.GetType("gno.vm/t/hello.Coin").(*DeclaredType)
	typeHasPrivateDep(st, first)
	if first.privateDep == 0 {
		t.Fatalf("privateDep = 0 after the first walk, want it memoized")
	}

	// A second transaction reloads the type through GetType.
	w2 := tm2Store.CacheWrap()
	tx2 := st.BeginTransaction(w2, w2, nil, nil)
	second := tx2.GetType("gno.vm/t/hello.Coin").(*DeclaredType)

	t.Logf("tx1 %p privateDep=%d / tx2 %p privateDep=%d", first, first.privateDep, second, second.privateDep)
	if second.privateDep == 0 {
		t.Fatalf("privateDep = 0 in the next transaction: the memo does not survive a transaction boundary, so the realm save path re-walks every type on every commit")
	}
}

func TestPrivateDepCacheWithMethods(t *testing.T) {
	// Get's signature never names Coin, but the method value carries the
	// receiver, so walking Coin reaches Coin again.
	for _, tc := range []struct{ name, body string }{
		{"no method", "package hello\n\ntype Coin struct{ Amount int }\n"},
		{"one method", "package hello\n\ntype Coin struct{ Amount int }\n\nfunc (c Coin) Get() int { return c.Amount }\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := memdb.NewMemDB()
			tm2Store := dbadapter.StoreConstructor(db, storetypes.StoreOptions{})
			st := NewStore(nil, tm2Store, tm2Store)
			w := tm2Store.CacheWrap()
			tx := st.BeginTransaction(w, w, nil, nil)
			m := NewMachineWithOptions(MachineOptions{PkgPath: "gno.vm/t/hello", Store: tx, Output: io.Discard})
			m.RunMemPackage(&std.MemPackage{
				Type: MPUserProd, Name: "hello", Path: "gno.vm/t/hello",
				Files: []*std.MemFile{{Name: "hello.gno", Body: tc.body}},
			}, true)

			dt := tx.GetType("gno.vm/t/hello.Coin").(*DeclaredType)
			typeHasPrivateDep(tx, dt)
			t.Logf("%s: methods=%d privateDep=%d", tc.name, len(dt.Methods), dt.privateDep)
			if dt.privateDep == 0 {
				t.Fatalf("privateDep = 0: a type with %d method(s) is never memoized", len(dt.Methods))
			}
		})
	}
}
