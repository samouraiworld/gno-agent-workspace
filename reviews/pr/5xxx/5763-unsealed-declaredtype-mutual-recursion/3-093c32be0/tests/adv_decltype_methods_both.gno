/* Run: from a gno checkout:
gh pr checkout 5763 -R gnolang/gno && git checkout 093c32be0
curl -fsSL -o gnovm/tests/files/adv_decltype_methods_both.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5763-unsealed-declaredtype-mutual-recursion/3-093c32be0/tests/adv_decltype_methods_both.gno
go test -v -run 'TestFiles/adv_decltype_methods_both.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/adv_decltype_methods_both.gno
*/
// A defined type does not inherit its source type's methods, and each half of a
// mutual pair carries its own. Nothing in the PR asserts that the in-place base
// fill leaves the two method sets separate.
package main

type T1 struct {
	Next *T2
	Val  int
}

func (t T1) Who() string { return "T1" }

type T2 T1

func (t T2) Who() string { return "T2" }

func main() {
	a := T1{Val: 1}
	a.Next = &T2{Val: 2}
	println(a.Who(), a.Val)
	println(a.Next.Who(), a.Next.Val)
}

// Output:
// T1 1
// T2 2
