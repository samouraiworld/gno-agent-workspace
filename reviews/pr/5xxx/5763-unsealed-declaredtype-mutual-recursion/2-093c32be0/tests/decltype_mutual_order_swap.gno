/* Run: from a gno checkout:
gh pr checkout 5763 -R gnolang/gno && git checkout 093c32be0
curl -fsSL -o gnovm/tests/files/decltype_mutual_order_swap.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5763-unsealed-declaredtype-mutual-recursion/2-093c32be0/tests/decltype_mutual_order_swap.gno
go test -v -run 'TestFiles/decltype_mutual_order_swap.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/decltype_mutual_order_swap.gno
*/
// Declaration order swapped: T2 declared before T1. The PR's own filetest only
// covers the T1-first order; this pins that both orders resolve to T1's struct.
package main

type T2 T1
type T1 struct {
	Next *T2
	Val  int
}

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
