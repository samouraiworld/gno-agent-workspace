// Repro, from a plain clone of gnolang/gno:
//   git fetch origin pull/5820/head && git checkout 8995e9424b363e8fc70c1c285087300f9627f68a
//   cp solo-finder-2-valid-program-rejected.gno gnovm/tests/files/solo-finder-2-valid-program-rejected.gno
//   (cd gnovm/pkg/gnolang && go test -run 'TestFiles/solo-finder-2-valid-program-rejected.gno' -count=1 -v .)
// Merge base: git show c775354dda49fb0c8c2411a26e8f6bd52213c834:gnovm/pkg/gnolang/types.go > gnovm/pkg/gnolang/types.go, re-run, then git checkout -- gnovm/pkg/gnolang/types.go
// The Output block below is what Go prints (go/types accepts or rejects as stated).
// Head 8995e94: FAIL, preprocess panic 'missing field M in main.T'.
// Base c775354: PASS, prints C.
// go/types accepts the program (no TypeCheckError).

package main

type A struct{}

func (A) M() string { return "A" }

type B struct{}

func (B) M() string { return "B" }

type C struct{}

func (C) M() string { return "C" }

type W struct{ C }

// Inside Amb, A.M and B.M are ambiguous at depth 1.
type Amb struct {
	A
	B
	W
}

// From T: C.M at depth 1 is the only member at the shallowest depth.
// Go resolves T.M to C.M.
type T struct {
	Amb
	C
}

func main() {
	println(T{}.M())
}

// Output:
// C
