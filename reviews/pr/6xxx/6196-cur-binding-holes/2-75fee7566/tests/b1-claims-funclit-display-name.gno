// PKGPATH: gno.land/r/demo/b1lit
package b1lit

// Drives funcDisplayName's source-location branch, the one every real function
// literal takes (fv.Source is set for a literal built by doOpFuncLit). No test
// in the tree covers it: `func literal at` matches only op_call.go, and
// TestFuncDisplayName pins the fv.Source == nil fallback instead.
//
// Repro from a plain clone:
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin 6d88deba71165b165641b65541259d00aba20377
//   git checkout 6d88deba71165b165641b65541259d00aba20377
//   cp <this file> gnovm/tests/files/zz_b1_litname.gno
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zz_b1_litname.gno$'
//   grep -rn 'func literal at' gnovm/          # only op_call.go:573
//
// head 6d88deba7: FAIL, with the branch's output:
//   unexpected panic: crossing function gno.land/r/demo/b1lit func literal at
//   zz_b1_litname.gno:10 was entered without cross(), ...

func B(cur realm, cb func()) { cb() }

func main(cur realm) {
	f := func(cur realm) { println(cur.IsCurrent()) }
	cb := func() { f(cur) }
	B(cross(cur), cb)
}

// Output:
// ok
