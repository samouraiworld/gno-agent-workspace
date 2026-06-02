// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a local clone of gnolang/gno:
gh pr checkout 5763 -R gnolang/gno && git checkout 61fc396e4
curl -fsSL -o gnovm/tests/files/adv_decltype_3way.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5763-unsealed-declaredtype-mutual-recursion/1-61fc396e4/tests/adv_decltype_3way.gno
go test -v -run 'TestFiles/adv_decltype_3way.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/adv_decltype_3way.gno
*/

// Control: 3-way `type A B; type B C; type C struct{*A}` chain. Each RHS
// resolves to a still-unsealed *DeclaredType during predefine. PR #5763
// must accept this (master accepts it too — the removed guard never fired
// here because no base collapses to struct{}). Passes on 61fc396e4 with
// "ok"; flip nothing — this is the baseline that must keep working.

package main

// 3-way mutual recursion: A -> B -> C -> back to A via pointer field.
// During predefine, B's RHS resolves to unsealed A; C's struct field
// resolves to unsealed A.

type A B
type B C
type C struct {
	Self *A
}

func main() {
	var c C
	var a A = A(c)
	a.Self = &A{}
	println("ok")
}

// Output:
// ok
