// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a local clone of gnolang/gno:
gh pr checkout 5763 -R gnolang/gno && git checkout 61fc396e4
curl -fsSL -o gnovm/tests/files/adv_decltype_invalid_cycle.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5763-unsealed-declaredtype-mutual-recursion/1-61fc396e4/tests/adv_decltype_invalid_cycle.gno
go test -v -run 'TestFiles/adv_decltype_invalid_cycle.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/adv_decltype_invalid_cycle.gno
*/

// Control: a genuinely illegal direct value cycle `type X Y; type Y X`
// (no pointer indirection) must still error cleanly after the guard is
// removed, not crash. The go/types checker catches it (TypeCheckError);
// the VM cycle detector is the fallback (Error). Confirms PR #5763 does
// not open a path that swallows invalid cycles. Passes on 61fc396e4.

package main

type X Y
type Y X

func main() {
	var x X
	_ = x
	println("unreachable")
}

// Error:
// main/adv_decltype_invalid_cycle.gno:18:6-9: invalid recursive type: X -> Y -> X

// TypeCheckError:
// main/adv_decltype_invalid_cycle.gno:18:6: invalid recursive type X; main/adv_decltype_invalid_cycle.gno:18:6: 	X refers to Y; main/adv_decltype_invalid_cycle.gno:19:6: 	Y refers to X
