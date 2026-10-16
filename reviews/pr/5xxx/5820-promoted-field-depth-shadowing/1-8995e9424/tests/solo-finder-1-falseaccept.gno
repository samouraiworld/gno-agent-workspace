// Repro, from a plain clone of github.com/gnolang/gno:
//   git checkout 8995e9424b363e8fc70c1c285087300f9627f68a
//   cp <this file> gnovm/tests/files/solo1_falseaccept.gno
//   cd gnovm/pkg/gnolang && go test -run 'TestFiles/solo1_falseaccept' -v .
// Base: git checkout c775354dda49fb0c8c2411a26e8f6bd52213c834 -- gnovm/pkg/gnolang/types.go, rerun.
// Go twin: same program with println -> fmt.Println, go run main.go.
// Head 8995e9424: prints C.M; go/types rejects: ambiguous selector t.M
// Base c775354dd: unexpected panic: missing field M in main.T
// Go 1.27:        compile error: ambiguous selector t.M
package main

// Go: T.A.C.M and T.B.D.M are both at depth 2, so T.M is ambiguous.
// The A subtree reaches D at depth 3 first, so T.B.D is pruned.

type C struct{}

func (C) M() string { return "C.M" }

type D struct{}

func (D) M() string { return "D.M" }

type X struct{ D }

type A struct {
	C
	X
}

type B struct{ D }

type T struct {
	A
	B
}

func main() {
	t := T{}
	println(t.M())
}

// Output:
// ambiguous
