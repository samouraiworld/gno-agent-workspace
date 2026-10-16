// Repro (from a plain clone of gnolang/gno):
//   git fetch origin pull/6272/head && git checkout --detach 03363f296312b625bf4620b8a5fba0927983a1ed
//   cp <this file> gnovm/pkg/repl/zz_namesource_test.go
//   go test ./gnovm/pkg/repl -run TestZZNameSourceShapes -count=1 -v
// Observed at 03363f296:
//   nilFileSet+Define pkg PANIC: runtime error: invalid memory address or nil pointer dereference
//   nilFileSet+Reserve w PANIC: runtime error: invalid memory address or nil pointer dereference
//   emptyFileSet+Define pkg PANIC: runtime error: invalid memory address or nil pointer dereference
//   uverse println PANIC: runtime error: invalid memory address or nil pointer dereference
//   name=strings/f/v/x/T/c/a/b (every name the REPL defines): ok fnNil=true, no panic
// Reading: the REPL shape the PR targets is fixed; a nil *FileSet package and a
// name Define'd with a zero NameSource still panic on nodes.go:2085.
package repl

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	gno "github.com/gnolang/gno/gnovm/pkg/gnolang"
)

// Probe: for every package-block name the REPL defines, call
// GetNameSourceForPath and report the NameSource and any panic.
func TestZZNameSourceShapes(t *testing.T) {
	var out, errb bytes.Buffer
	r := NewRepl(WithIO(os.Stdin, &out, &errb))
	inputs := []string{
		`import "strings"`,
		`func f() int { return 1 }`,
		`var v = 1`,
		`x := 2`,
		`type T int`,
		`const c = 3`,
		`var a, b string = "a", "b"`,
	}
	for _, in := range inputs {
		r.RunStatements(in)
		if errb.Len() > 0 {
			t.Logf("input %q errput: %s", in, errb.String())
			errb.Reset()
		}
	}
	probe := func(label string, pn *gno.PackageNode, path gno.ValuePath) {
		defer func() {
			if rec := recover(); rec != nil {
				fmt.Printf("%s PANIC: %v\n", label, rec)
			}
		}()
		_, fn, nsrc := pn.GetNameSourceForPath(r.store, path)
		fmt.Printf("%s ok fnNil=%v type=%v\n", label, fn == nil, nsrc.Type)
	}
	// nil-FileSet synthetic package, name Define'd (MsgCall's main shape).
	mpn := gno.NewPackageNode("main", "", nil)
	mpn.Define("pkg", gno.TypedValue{T: &gno.PackageType{}, V: r.pv})
	pidx, _ := mpn.GetLocalIndex("pkg")
	pp := gno.ValuePath{Type: gno.VPBlock, Name: "pkg"}
	pp.SetDepth(1)
	pp.Index = pidx
	probe("nilFileSet+Define pkg", mpn, pp)
	// nil-FileSet synthetic package, name Reserve'd with a NameSource.
	mpn2 := gno.NewPackageNode("main", "", nil)
	mpn2.Reserve(false, &gno.NameExpr{Name: "w"}, &gno.ValueDecl{}, gno.NSValueDecl, 0)
	widx, _ := mpn2.GetLocalIndex("w")
	wp := gno.ValuePath{Type: gno.VPBlock, Name: "w"}
	wp.SetDepth(1)
	wp.Index = widx
	probe("nilFileSet+Reserve w", mpn2, wp)
	// empty-FileSet package, name Define'd with zero NameSource.
	mpn3 := gno.NewPackageNode("main", "", &gno.FileSet{})
	mpn3.Define("pkg", gno.TypedValue{T: &gno.PackageType{}, V: r.pv})
	p3 := gno.ValuePath{Type: gno.VPBlock, Name: "pkg"}
	p3.SetDepth(1)
	p3.Index, _ = mpn3.GetLocalIndex("pkg")
	probe("emptyFileSet+Define pkg", mpn3, p3)
	// uverse path.
	uidx, _ := gno.UverseNode().GetLocalIndex("println")
	probe("uverse println", r.pn, gno.NewValuePathUverse(uidx, "println"))
	for _, n := range r.pn.GetBlockNames() {
		idx, ok := r.pn.GetLocalIndex(n)
		if !ok {
			continue
		}
		path := gno.ValuePath{Type: gno.VPBlock, Name: n}
		path.SetDepth(1)
		path.Index = idx
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					fmt.Printf("name=%-8s PANIC: %v\n", n, rec)
				}
			}()
			ns := r.pn.GetNameSources()[idx]
			fmt.Printf("name=%-8s nsrc.Type=%v nameExprNil=%v ", n, ns.Type, ns.NameExpr == nil)
			_, fn, nsrc := r.pn.GetNameSourceForPath(r.store, path)
			fmt.Printf("ok fnNil=%v type=%v\n", fn == nil, nsrc.Type)
		}()
	}
}
