package gnolang

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/db/memdb"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/gnolang/gno/tm2/pkg/store/dbadapter"
	"github.com/gnolang/gno/tm2/pkg/store/iavl"
	stypes "github.com/gnolang/gno/tm2/pkg/store/types"
)

// zzChain builds a switch of n clauses, each declaring k locals of the same
// count so the merge base does not hit its block-shrinkage panic, all chained
// by fallthrough from clause 0.
func zzChain(n, k int) string {
	var b strings.Builder
	b.WriteString("package main\n\nfunc main() {\n\tswitch 1 {\n")
	for c := range n {
		if c == 0 {
			b.WriteString("\tcase 1:\n")
		} else {
			fmt.Fprintf(&b, "\tcase %d:\n", c+1)
		}
		for v := range k {
			fmt.Fprintf(&b, "\t\tv%d_%d := %d\n\t\t_ = v%d_%d\n", c, v, c*10+v, c, v)
		}
		if c < n-1 {
			b.WriteString("\t\tfallthrough\n")
		}
	}
	b.WriteString("\t}\n\tprintln(\"done\")\n}\n")
	return b.String()
}

func zzMeasure(t *testing.T, label, body string) {
	db := memdb.NewMemDB()
	store := NewStore(nil,
		dbadapter.StoreConstructor(db, stypes.StoreOptions{}),
		iavl.StoreConstructor(db, stypes.StoreOptions{}))
	gm := stypes.NewInfiniteGasMeter()
	alloc := NewAllocator(200 * 1024 * 1024)
	alloc.SetGasMeter(gm)
	m := NewMachineWithOptions(MachineOptions{
		PkgPath:  "gno.land/r/zz/main",
		Output:   io.Discard,
		Store:    store,
		Alloc:    alloc,
		GasMeter: gm,
	})
	defer m.Release()
	m.RunMemPackage(&std.MemPackage{
		Type:  MPUserProd,
		Name:  "main",
		Path:  "gno.land/r/zz/main",
		Files: []*std.MemFile{{Name: "m.gno", Body: body}},
	}, true)
	_, pre := m.Alloc.Status()
	preGas := gm.GasConsumed()
	preCycles := m.Cycles
	m.RunMain()
	_, post := m.Alloc.Status()
	fmt.Printf("ZZ %-18s allocDelta=%-7d gasDelta=%-9d cycleDelta=%d\n",
		label, post-pre, gm.GasConsumed()-preGas, m.Cycles-preCycles)
}

func TestZZFallthroughAlloc(t *testing.T) {
	zzMeasure(t, "chain-1x4", zzChain(1, 4))
	zzMeasure(t, "chain-2x4", zzChain(2, 4))
	zzMeasure(t, "chain-10x4", zzChain(10, 4))
	zzMeasure(t, "chain-50x4", zzChain(50, 4))
}
