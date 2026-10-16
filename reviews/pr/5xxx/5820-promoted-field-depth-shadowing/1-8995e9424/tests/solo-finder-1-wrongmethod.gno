// Repro, from a plain clone of github.com/gnolang/gno:
//   git checkout 8995e9424b363e8fc70c1c285087300f9627f68a
//   cp <this file> gnovm/tests/files/solo1_wrongmethod.gno
//   cd gnovm/pkg/gnolang && go test -run 'TestFiles/solo1_wrongmethod' -v .
// Base: git checkout c775354dda49fb0c8c2411a26e8f6bd52213c834 -- gnovm/pkg/gnolang/types.go, rerun.
// Go twin: same program with println -> fmt.Println, go run main.go.
// Head 8995e9424: prints C.M / C.M; go/types accepts (no TypeCheckError)
// Base c775354dd: unexpected panic: missing field M in main.T
// Go 1.27:        prints R.M outer / R.M outer
package main

// Go resolves T.M to R.M (depth 1). The S subtree reaches R at depth 3
// first, marking R visited, so the direct R embed is pruned.

type I interface{ M() string }

type C struct{}

func (C) M() string { return "C.M" }

type R struct{ V string }

func (r R) M() string { return "R.M " + r.V }

type X struct{ R }

type S struct {
	C
	X
}

type T struct {
	S
	R
}

func main() {
	t := T{S{C{}, X{R{"inner"}}}, R{"outer"}}
	println(t.M())
	var i I = t
	println(i.M())
}

// Output:
// R.M outer
// R.M outer
