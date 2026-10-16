// Repro, from a plain clone of gnolang/gno:
//   git fetch origin pull/5820/head && git checkout 8995e9424b363e8fc70c1c285087300f9627f68a
//   cp solo-finder-2-diamond-ambiguity-accepted.gno gnovm/tests/files/solo-finder-2-diamond-ambiguity-accepted.gno
//   (cd gnovm/pkg/gnolang && go test -run 'TestFiles/solo-finder-2-diamond-ambiguity-accepted.gno' -count=1 -v .)
// Merge base: git show c775354dda49fb0c8c2411a26e8f6bd52213c834:gnovm/pkg/gnolang/types.go > gnovm/pkg/gnolang/types.go, re-run, then git checkout -- gnovm/pkg/gnolang/types.go
// The Output block below is what Go prints (go/types accepts or rejects as stated).
// Head 8995e94: runs and prints 'base'; the harness then reports TypeCheckError 'ambiguous selector T{}.M' from go/types.
// Base c775354: preprocess panic 'missing field M in main.T', matching Go's rejection.
// Go rejects the program; the Output block records what head prints.

package main

type Base struct{}

func (Base) M() string { return "base" }

type A struct{ Base }

type B struct{ Base }

type E struct{}

func (E) M() string { return "E" }

type D struct{ E }

type C struct{ D }

// From T: Base.M twice at depth 2 (via A and via B), E.M at depth 3.
// Go rejects T.M as an ambiguous selector.
type T struct {
	A
	B
	C
}

func main() {
	println(T{}.M())
}

// Output:
// base
