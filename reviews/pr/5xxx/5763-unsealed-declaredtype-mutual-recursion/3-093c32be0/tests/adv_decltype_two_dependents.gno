/* Run: from a gno checkout:
gh pr checkout 5763 -R gnolang/gno && git checkout 093c32be0
curl -fsSL -o gnovm/tests/files/adv_decltype_two_dependents.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5763-unsealed-declaredtype-mutual-recursion/3-093c32be0/tests/adv_decltype_two_dependents.gno
go test -v -run 'TestFiles/adv_decltype_two_dependents.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/adv_decltype_two_dependents.gno
*/
// Two defined types over one source. Both captured the same provisional base
// pointer at predefine, so one in-place fill has to serve both, and the two
// named types have to stay distinct.
package main

type T1 struct {
	Next *T2
	Val  int
}

type T2 T1

type T3 T1

func main() {
	var a T1
	a.Next = &T2{Val: 1}
	println(a.Next.Val)
	var c T3
	c.Val = 2
	c.Next = &T2{Val: 3}
	println(c.Val, c.Next.Val)
	var d T1 = T1(c)
	println(d.Val)
}

// Output:
// 1
// 2 3
// 2
