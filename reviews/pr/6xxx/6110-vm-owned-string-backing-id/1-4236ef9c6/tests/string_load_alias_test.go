/* Run: from a gno checkout:
gh pr checkout 6110 -R gnolang/gno && git checkout 4236ef9c6
curl -fsSL -o gnovm/pkg/gnolang/string_load_alias_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6110-vm-owned-string-backing-id/1-4236ef9c6/tests/string_load_alias_test.go
go test -v -run TestZZAliasSurvivesLoad ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/string_load_alias_test.go
*/

// Two package-level pointers to one persisted object. After a cold load under a
// tight allocator cap, a write through A must be visible through B. It is not:
// the mint added to the load path can trigger a GC that evicts the object from
// the store cache between the insert and the caller wiring it in, so the second
// reference loads an independent copy. The transaction still succeeds.
// Fails at 4236ef9c6 for 56 caps; passes at 2ed70a202 over the same sweep.

package gnolang

import (
	"fmt"
	"io"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/db/memdb"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/gnolang/gno/tm2/pkg/store/dbadapter"
	storetypes "github.com/gnolang/gno/tm2/pkg/store/types"
)

var zzUsed int64

var zzBroken []int64

const zzPkg = "gno.land/r/zzalias"

const zzSrc = `package zzalias

type Node struct{ Name string }

var A *Node
var B *Node

func init() {
	n := &Node{Name: "original"}
	A = n
	B = n
}

func Probe() string {
	A.Name = "MUTATED"
	return B.Name
}
`

// zzProbe persists the realm once, then reloads it in a fresh machine whose
// allocator cap is maxAlloc, mutates through A and reads through B. A and B
// are one object, so the read must observe the write.
func zzProbe(t *testing.T, maxAlloc int64) (got string, err any) {
	t.Helper()
	db := memdb.NewMemDB()
	base := dbadapter.StoreConstructor(db, storetypes.StoreOptions{})
	iavl := dbadapter.StoreConstructor(memdb.NewMemDB(), storetypes.StoreOptions{})
	st := NewStore(nil, base, iavl)

	// Phase 1: deploy and persist, with room to spare.
	{
		w1, w2 := base.CacheWrap(), iavl.CacheWrap()
		txSt := st.BeginTransaction(w1, w2, nil, nil)
		m := NewMachineWithOptions(MachineOptions{
			PkgPath: "", Output: io.Discard, Store: txSt,
			Alloc: NewAllocator(64 * 1024 * 1024),
		})
		m.RunMemPackage(&std.MemPackage{
			Type: MPUserProd, Name: "zzalias", Path: zzPkg,
			Files: []*std.MemFile{{Name: "a.gno", Body: zzSrc}},
		}, true)
		m.Release()
		txSt.Write()
		w1.Write()
		w2.Write()
	}

	// Phase 2: cold load under a tight cap.
	w1, w2 := base.CacheWrap(), iavl.CacheWrap()
	txSt := st.BeginTransaction(w1, w2, nil, nil)
	defer func() { err = recover() }()
	m := NewMachineWithOptions(MachineOptions{
		PkgPath: "", Output: io.Discard, Store: txSt,
		Alloc: NewAllocator(maxAlloc),
	})
	defer m.Release()
	pv := txSt.GetPackage(zzPkg, false)
	m.SetActivePackage(pv)
	res := m.Eval(Call(Nx("Probe")))
	_, used := txSt.GetAllocator().Status()
	zzUsed = used
	return res[0].GetString(), nil
}

func TestZZAliasSurvivesLoad(t *testing.T) {
	if _, perr := zzProbe(t, 1<<30); perr != nil {
		t.Fatalf("uncapped probe panicked: %v", perr)
	}
	base := zzUsed
	fmt.Printf("ZZ cold-load charge = %d bytes\n", base)
	ok, broken, aborted := 0, 0, 0
	for cap := base - 3000; cap <= base+3000; cap++ {
		got, perr := zzProbe(t, cap)
		switch {
		case perr != nil:
			aborted++
		case got == "MUTATED":
			ok++
		default:
			broken++
			zzBroken = append(zzBroken, cap)
		}
	}
	fmt.Printf("ZZ ok=%d broken=%d aborted=%d\n", ok, broken, aborted)
	if len(zzBroken) > 0 {
		fmt.Printf("ZZ broken caps: %d..%d\n", zzBroken[0], zzBroken[len(zzBroken)-1])
		t.Errorf("aliasing broken at %d allocator caps, first %d", len(zzBroken), zzBroken[0])
	}
}
