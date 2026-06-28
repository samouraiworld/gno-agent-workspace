/* Run: from a gno checkout:
gh pr checkout 5763 -R gnolang/gno && git checkout 093c32be0
curl -fsSL -o gnovm/tests/files/decltype_derived_empty_base.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5763-unsealed-declaredtype-mutual-recursion/2-093c32be0/tests/decltype_derived_empty_base.gno
go test -v -run 'TestFiles/decltype_derived_empty_base.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/decltype_derived_empty_base.gno
*/
// Round-1 Critical regression: `type T2 T1` where T1 is declared first and
// carries data fields. On 61fc396e4 T2.Base collapsed to struct{} and reading
// a field through T2 panicked "struct type struct{} has no field Val".
// fillTypeInPlace now fills T1's original base pointer, so T2 observes the
// completed struct and prints 9.
package main

type T1 struct {
	Next *T2
	Val  int
}

type T2 T1

func main() {
	var a T1
	a.Next = &T2{Val: 9}
	println(a.Next.Val)
	var b T2
	b.Val = 7
	println(b.Val)
	println("ok")
}

// Output:
// 9
// 7
// ok
