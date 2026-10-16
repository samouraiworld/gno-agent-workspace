// solo-finder-2: a []T passed as ...I is accepted (Go rejects), and the callee
// writes a non-T value into the caller's []T.
//
// Repro from a plain clone of gnolang/gno:
//   git checkout b42a521b16655374e507bdb949ab5b8289bc8404
//   cd gnovm && make install   # or use a prebuilt gno at that sha
//   mkdir /tmp/vararg && cp <this file> /tmp/vararg/main.gno
//   gno run /tmp/vararg/main.gno
// Observed at head b42a521b1:
//   hello      <- ts[0].x, an int field, reads U's string
//   5          <- ts[0].M() runs U.M
// Go (go vet / go build): cannot use ts (variable of type []T) as []I value in argument to f
// Cause: type_check.go:585, `if cdt.Vrd { return checkAssignableTo(n, st.Elt, cdt.Elt) }`,
// the same recursion-into-assignability shape the diff removed from the *PointerType case.
// Same line at merge base 4c9de5225 (type_check.go:577-578): pre-existing.
package main

type I interface{ M() int }

type T struct{ x int }

func (t T) M() int { return t.x }

type U struct{ s string }

func (u U) M() int { return len(u.s) }

func f(xs ...I) {
	xs[0] = U{"hello"}
}

func main() {
	ts := []T{{1}}
	f(ts...)
	println(ts[0].x)
	println(ts[0].M())
}
