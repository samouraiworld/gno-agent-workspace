package gnolang_test

// Judge probe for PR 6272 candidates 1 and 3: which package-block shapes
// GetNameSourceForPath resolves without panicking, and whether the PR's
// test needs its store. Fails at head on every shape that panics.

import (
	"fmt"
	"testing"

	gno "github.com/gnolang/gno/gnovm/pkg/gnolang"
)

func zzProbe(t *testing.T, label string, pn *gno.PackageNode, store gno.Store, path gno.ValuePath) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("%s: PANIC %v\n", label, r)
			t.Errorf("%s panicked", label)
		}
	}()
	_, fn, nsrc := pn.GetNameSourceForPath(store, path)
	fmt.Printf("%s: ok fnNil=%v type=%v\n", label, fn == nil, nsrc.Type)
}

func zzPath(pn *gno.PackageNode, n gno.Name) gno.ValuePath {
	idx, ok := pn.GetLocalIndex(n)
	if !ok {
		panic("no index for " + string(n))
	}
	p := gno.ValuePath{Type: gno.VPBlock, Name: n}
	p.SetDepth(1)
	p.Index = idx
	return p
}

func TestZZJudgeNameSourceShapes(t *testing.T) {
	// PR's own shape, nil store: empty FileSet, Reserve with a NameExpr.
	pn := gno.NewPackageNode("a", "gno.land/r/a", &gno.FileSet{})
	pn.Reserve(false, &gno.NameExpr{Name: "outVar"}, &gno.ValueDecl{}, gno.NSValueDecl, 0)
	zzProbe(t, "emptyFileSet+Reserve, nil store", pn, nil, zzPath(pn, "outVar"))

	// Empty FileSet, Define (zero NameSource).
	pn2 := gno.NewPackageNode("b", "gno.land/r/b", &gno.FileSet{})
	pn2.Define("pkg", gno.TypedValue{T: &gno.PackageType{}})
	zzProbe(t, "emptyFileSet+Define", pn2, nil, zzPath(pn2, "pkg"))

	// nil FileSet, Reserve with a NameExpr.
	pn3 := gno.NewPackageNode("c", "gno.land/r/c", nil)
	pn3.Reserve(false, &gno.NameExpr{Name: "w"}, &gno.ValueDecl{}, gno.NSValueDecl, 0)
	zzProbe(t, "nilFileSet+Reserve", pn3, nil, zzPath(pn3, "w"))

	// Uverse path for `true`.
	un := gno.UverseNode()
	idx, ok := un.GetLocalIndex("true")
	if !ok {
		t.Fatal("no true in uverse")
	}
	zzProbe(t, "uverse true", un, nil, gno.ValuePath{Type: gno.VPUverse, Name: "true", Index: idx})
}
