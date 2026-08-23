/* Run: from a gno checkout:
gh pr checkout 5763 -R gnolang/gno && git checkout 093c32be0
curl -fsSL -o gnovm/tests/files/adv_decltype_chain3_source_first.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5763-unsealed-declaredtype-mutual-recursion/3-093c32be0/tests/adv_decltype_chain3_source_first.gno
go test -v -run 'TestFiles/adv_decltype_chain3_source_first.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/adv_decltype_chain3_source_first.gno
*/
// Control for adv_decltype_chain3_dependent_first.gno: the same three types with
// the source declared first. 0397fc87f rejects it with "should not happen";
// 093c32be0 prints 7, 8, 9. This is the order the fix reaches.
package main

type T1 struct {
	Next *T3
	Val  int
}

type T2 T1

type T3 T2

func main() {
	var a T1
	a.Next = &T3{Val: 7}
	println(a.Next.Val)
	var b T2
	b.Val = 8
	println(b.Val)
	var c T3
	c.Val = 9
	println(c.Val)
}

// Output:
// 7
// 8
// 9
