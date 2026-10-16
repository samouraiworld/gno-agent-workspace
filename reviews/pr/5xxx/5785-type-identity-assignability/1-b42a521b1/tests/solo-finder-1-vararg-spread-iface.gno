// Repro, gnolang/gno#5785 at b42a521b16655374e507bdb949ab5b8289bc8404:
//   git clone https://github.com/gnolang/gno && cd gno
//   git checkout b42a521b16655374e507bdb949ab5b8289bc8404
//   cp solo-finder-1-vararg-spread-iface.gno gnovm/tests/files/solo_finder_1_vararg_spread_iface.gno
//   go test ./gnovm/pkg/gnolang -run 'TestFiles/solo_finder_1_vararg_spread_iface' -count=1 -v
// PASS means checkAssignableTo's variadic-spread branch (type_check.go:584,
// `checkAssignableTo(n, st.Elt, cdt.Elt)`) accepts []int for ...any and []E
// for ...error, the element-assignability recursion this PR removed from the
// pointer case; the callee then writes a string into the caller's []int.
// Go rejects both calls at compile time.
package main

func f(xs ...any) { xs[0] = "hello" }

type E struct{}

func (E) Error() string { return "e" }

func g(xs ...error) int { return len(xs) }

func main() {
	s := []int{1, 2}
	f(s...)
	println(s[0])
	println(g([]E{{}, {}}...))
}

// Output:
// hello
// 2

// TypeCheckError:
// main/solo_finder_1_vararg_spread_iface.gno:23:4: cannot use s (variable of type []int) as []any value in argument to f; main/solo_finder_1_vararg_spread_iface.gno:25:12: cannot use []E{…} (value of type []E) as []error value in argument to g
