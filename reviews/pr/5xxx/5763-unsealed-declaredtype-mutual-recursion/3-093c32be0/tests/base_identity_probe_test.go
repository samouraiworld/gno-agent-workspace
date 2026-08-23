/* Run: from a gno checkout:
gh pr checkout 5763 -R gnolang/gno && git checkout 093c32be0
curl -fsSL -o gnovm/pkg/gnolang/zz_base_identity_probe_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5763-unsealed-declaredtype-mutual-recursion/3-093c32be0/tests/base_identity_probe_test.go
go test -v -run 'TestRev5763' ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/zz_base_identity_probe_test.go
*/

// Asserts the three type-identity facts the PR rests on: T1.Base and T2.Base
// are the same pointer after `type T2 T1`, the two named types keep distinct
// TypeIDs, and no package-level *DeclaredType survives preprocessing unsealed.
// Passes at 093c32be0.

package gnolang

import (
	"testing"

	"github.com/gnolang/gno/tm2/pkg/db/memdb"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/gnolang/gno/tm2/pkg/store/dbadapter"
	"github.com/gnolang/gno/tm2/pkg/store/iavl"
	stypes "github.com/gnolang/gno/tm2/pkg/store/types"
)

func rev5763Run(t *testing.T, name, body string) (*Machine, *PackageNode) {
	t.Helper()
	db := memdb.NewMemDB()
	store := NewStore(nil,
		dbadapter.StoreConstructor(db, stypes.StoreOptions{}),
		iavl.StoreConstructor(db, stypes.StoreOptions{}))
	m := NewMachine(name, store)
	m.RunMemPackage(&std.MemPackage{
		Type:  MPStdlibProd,
		Name:  name,
		Path:  name,
		Files: []*std.MemFile{{Name: "a.gno", Body: body}},
	}, true)
	return m, m.Package.GetPackageNode(m.Store)
}

func rev5763Named(t *testing.T, pn *PackageNode, store Store, n Name) *DeclaredType {
	t.Helper()
	tv := pn.GetSlot(store, n, true)
	if tv == nil {
		t.Fatalf("no slot for %s", n)
	}
	dt, ok := tv.GetType().(*DeclaredType)
	if !ok {
		t.Fatalf("%s is %T, not *DeclaredType", n, tv.GetType())
	}
	return dt
}

// Round-2 claim: T1.Base == T2.Base, and the named types stay distinct.
func TestRev5763_BaseIdentity(t *testing.T) {
	m, pn := rev5763Run(t, "rev", `package rev
type T1 struct {
	Next *T2
	Val  int
}
type T2 T1
`)
	t1 := rev5763Named(t, pn, m.Store, "T1")
	t2 := rev5763Named(t, pn, m.Store, "T2")
	if t1.Base != t2.Base {
		t.Fatalf("bases differ: %p vs %p", t1.Base, t2.Base)
	}
	if got := len(t1.Base.(*StructType).Fields); got != 2 {
		t.Fatalf("shared base has %d fields, want 2", got)
	}
	if t1.TypeID() == t2.TypeID() {
		t.Fatalf("named types collapsed to one TypeID: %s", t1.TypeID())
	}
	if t1.Base.TypeID() != t2.Base.TypeID() {
		t.Fatalf("underlying TypeIDs differ: %s vs %s", t1.Base.TypeID(), t2.Base.TypeID())
	}
	t.Logf("shared base %p (%s); named ids %s / %s",
		t1.Base, t1.Base.TypeID(), t1.TypeID(), t2.TypeID())
}

// Round-2 claim: a PrimitiveType base is left alone by fillTypeInPlace.
func TestRev5763_PrimitiveBase(t *testing.T) {
	m, pn := rev5763Run(t, "revp", `package revp
type P1 int
type P2 P1
`)
	p1 := rev5763Named(t, pn, m.Store, "P1")
	p2 := rev5763Named(t, pn, m.Store, "P2")
	if _, ok := p1.Base.(PrimitiveType); !ok {
		t.Fatalf("P1.Base is %T, want PrimitiveType", p1.Base)
	}
	if p1.Base != p2.Base {
		t.Fatalf("primitive bases differ: %v vs %v", p1.Base, p2.Base)
	}
	t.Logf("P1.Base = P2.Base = %v (%s)", p1.Base, p1.Base.TypeID())
}

// The tryPredefine comment claims TRANS_LEAVE seals the type later. Assert no
// package-level *DeclaredType is left unsealed once preprocessing returns.
func TestRev5763_NothingLeftUnsealed(t *testing.T) {
	programs := map[string]string{
		"structptr": "type T1 struct{ Next *T2; Val int }\ntype T2 T1\ntype T3 T2\n",
		"slice":     "type T1 []T2\ntype T2 T1\n",
		"map":       "type T1 map[string]T2\ntype T2 T1\n",
		"funct":     "type T1 func(int) T2\ntype T2 T1\n",
		"iface":     "type T1 interface{ M() *T2 }\ntype T2 T1\n",
		"ptr":       "type T1 *T2\ntype T2 T1\n",
		"arrayptr":  "type T1 [2]*T2\ntype T2 T1\n",
		"prim":      "type T1 int\ntype T2 T1\n",
		"alias":     "type T1 struct{ Next *T2; Val int }\ntype T2 = T1\n",
	}
	for name, decls := range programs {
		t.Run(name, func(t *testing.T) {
			m, pn := rev5763Run(t, "p"+name, "package p"+name+"\n"+decls)
			seen := 0
			for _, n := range pn.GetBlockNames() {
				tv := pn.GetSlot(m.Store, n, true)
				if tv == nil {
					continue
				}
				dt, ok := tv.GetType().(*DeclaredType)
				if !ok {
					continue
				}
				seen++
				if !dt.sealed {
					t.Fatalf("%s left unsealed after preprocessing", n)
				}
			}
			// Guard against a vacuous pass: the walk must reach the
			// declared types the program defines.
			if seen < 2 {
				t.Fatalf("walk reached %d *DeclaredType, want >= 2", seen)
			}
			t.Logf("%d sealed *DeclaredType", seen)
		})
	}
}
