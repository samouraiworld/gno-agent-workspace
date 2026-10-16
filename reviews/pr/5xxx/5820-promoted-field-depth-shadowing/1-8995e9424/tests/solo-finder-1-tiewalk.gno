// Repro, from a plain clone of github.com/gnolang/gno:
//   git checkout 8995e9424b363e8fc70c1c285087300f9627f68a
//   cp <this file> gnovm/tests/files/solo1_tiewalk.gno
//   cd gnovm/pkg/gnolang && go test -run 'TestFiles/solo1_tiewalk' -v .
// Base: git checkout c775354dda49fb0c8c2411a26e8f6bd52213c834 -- gnovm/pkg/gnolang/types.go, rerun.
// Go twin: same program with println -> fmt.Println, go run main.go.
// Head 8995e9424: FAIL, unexpected panic: main.T does not implement main.I (missing method M)
// Base c775354dd: PASS, prints R.M
// Go 1.27:        prints R.M; go/types accepts (no TypeCheckError)
package main

// Go resolves T.M to R.M (depth 1); S.A.M and S.B.M at depth 2 are
// shadowed. Inside S, the tie between A and B no longer returns early,
// so the walk continues into Z and marks R visited before T reaches R.

type I interface{ M() string }

type A struct{}

func (A) M() string { return "A.M" }

type B struct{}

func (B) M() string { return "B.M" }

type R struct{}

func (R) M() string { return "R.M" }

type Z struct{ R }

type S struct {
	A
	B
	Z
}

type T struct {
	S
	R
}

func main() {
	var i I = T{}
	println(i.M())
}

// Output:
// R.M
