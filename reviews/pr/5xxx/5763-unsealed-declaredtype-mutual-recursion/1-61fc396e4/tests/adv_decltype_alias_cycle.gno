// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a local clone of gnolang/gno:
gh pr checkout 5763 -R gnolang/gno && git checkout 61fc396e4
curl -fsSL -o gnovm/tests/files/adv_decltype_alias_cycle.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5763-unsealed-declaredtype-mutual-recursion/1-61fc396e4/tests/adv_decltype_alias_cycle.gno
go test -v -run 'TestFiles/adv_decltype_alias_cycle.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/adv_decltype_alias_cycle.gno
*/

// Control: alias `type A2 = A1` into a symmetric struct cycle (A1<->B1).
// Alias path through an unsealed named type must resolve. Passes on
// 61fc396e4 ("ok"); baseline that must keep working.

package main

// Alias pointing into a mutual struct cycle. Alias A2 = A1 must resolve
// to the same underlying named type while A1 is still being defined.

type A1 struct {
	Peer *B1
}

type B1 struct {
	Peer *A1
}

type A2 = A1

func main() {
	var a A2
	a.Peer = &B1{}
	println("ok")
}

// Output:
// ok
