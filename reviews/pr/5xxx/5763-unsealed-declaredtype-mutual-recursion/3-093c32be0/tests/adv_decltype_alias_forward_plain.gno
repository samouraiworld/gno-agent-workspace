/* Run: from a gno checkout:
gh pr checkout 5763 -R gnolang/gno && git checkout 093c32be0
curl -fsSL -o gnovm/tests/files/adv_decltype_alias_forward_plain.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5763-unsealed-declaredtype-mutual-recursion/3-093c32be0/tests/adv_decltype_alias_forward_plain.gno
go test -v -run 'TestFiles/adv_decltype_alias_forward_plain.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/adv_decltype_alias_forward_plain.gno
*/
// Baseline for adv_decltype_alias_forward.gno with the mutual half removed.
// Valid Go, prints 4. Both 0397fc87f and 093c32be0 abort with
// "StaticBlock.Define2(TA) cannot change .V", so the forward alias is broken
// before this branch and independently of it.
package main

type TA = T1

type T1 struct {
	Val int
}

func main() {
	var a TA
	a.Val = 4
	println(a.Val)
}

// Output:
// 4
