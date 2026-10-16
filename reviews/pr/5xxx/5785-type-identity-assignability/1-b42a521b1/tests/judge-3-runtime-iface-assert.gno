/* Run: from a gno checkout:
gh pr checkout 5785 -R gnolang/gno
cp judge-3-runtime-iface-assert.gno gnovm/tests/files/judge_3_runtime_iface_assert.gno
go test -v -run 'TestFiles/judge_3_runtime_iface_assert.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/judge_3_runtime_iface_assert.gno
*/
// Judge, candidates 3 and 4: runtime x.(I), comma-ok and type switch against an
// interface whose method differs from the concrete one only in variadicity or
// in a struct tag. Go answers false/default/false; the merge base answers
// true/case I/true, the reviewed head false/default/false.
package main

type T struct{}

func (T) M(xs []int) {}

type S struct{}

func (S) N(s struct{ A int "a" }) {}

func main() {
	var x any = T{}
	_, ok := x.(interface{ M(...int) })
	println(ok)
	switch x.(type) {
	case interface{ M(...int) }:
		println("case I")
	default:
		println("default")
	}
	var y any = S{}
	_, ok2 := y.(interface{ N(struct{ A int "b" }) })
	println(ok2)
}

// Output:
// false
// default
// false
