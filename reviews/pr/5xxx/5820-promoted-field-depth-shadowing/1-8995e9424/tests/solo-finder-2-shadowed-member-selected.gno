// Repro, from a plain clone of gnolang/gno:
//   git fetch origin pull/5820/head && git checkout 8995e9424b363e8fc70c1c285087300f9627f68a
//   cp solo-finder-2-shadowed-member-selected.gno gnovm/tests/files/solo-finder-2-shadowed-member-selected.gno
//   (cd gnovm/pkg/gnolang && go test -run 'TestFiles/solo-finder-2-shadowed-member-selected.gno' -count=1 -v .)
// Merge base: git show c775354dda49fb0c8c2411a26e8f6bd52213c834:gnovm/pkg/gnolang/types.go > gnovm/pkg/gnolang/types.go, re-run, then git checkout -- gnovm/pkg/gnolang/types.go
// The Output block below is what Go prints (go/types accepts or rejects as stated).
// Head 8995e94: FAIL, prints 'iface: A / method: A / field: a' (A.M and A.V, two levels deep, win over C.M and C.V one level deep).
// Base c775354: FAIL, preprocess panic 'main.T does not implement main.I (missing method M)'.
// go/types accepts the program (no TypeCheckError).

package main

type I interface{ M() string }

type A struct{ V string }

func (A) M() string { return "A" }

type C struct{ V string }

func (C) M() string { return "C" }

type W struct{ C }

// Inside Sub, A.M (depth 1) shadows W.C.M (depth 2).
type Sub struct {
	A
	W
}

// From T: C.M at depth 1, Sub.A.M at depth 2, Sub.W.C.M at depth 3.
// Go resolves T.M to C.M and T.V to C.V.
type T struct {
	Sub
	C
}

func main() {
	t := T{Sub{A{"a"}, W{C{"wc"}}}, C{"c"}}
	var i I = t
	println("iface:", i.M())
	println("method:", t.M())
	println("field:", t.V)
}

// Output:
// iface: C
// method: C
// field: c
