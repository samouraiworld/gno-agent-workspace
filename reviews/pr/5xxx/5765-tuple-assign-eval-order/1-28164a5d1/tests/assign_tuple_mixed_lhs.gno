// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a local clone of gnolang/gno:
gh pr checkout 5765 -R gnolang/gno && git checkout 28164a5d1
curl -fsSL -o gnovm/tests/files/assign_tuple_mixed_lhs.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5765-tuple-assign-eval-order/1-28164a5d1/tests/assign_tuple_mixed_lhs.gno
go test -v -run 'TestFiles/assign_tuple_mixed_lhs.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/assign_tuple_mixed_lhs.gno
*/

// Stress doOpAssign's frame-buffer reorder with a 4-LHS tuple that mixes
// every operand-arity: IndexExpr (2 stack vals), SelectorExpr (1), StarExpr
// (1), NameExpr (0). PushForPointer stacks the operand frames in source
// order; doOpAssign pops them right-to-left into frames[] then re-pushes /
// resolves / assigns left-to-right. If the re-push corrupted a not-yet-
// consumed frame (a backing-array aliasing bug), the assigned values would
// scramble. Observed: a=10 s.f=20 *p=30 b=40 — exact, in order, no scramble.
// Flip the assertion by editing the stdout line if a regression appears.

package main

type S struct{ f int }

func main() {
	arr := []int{0}
	s := S{}
	x := 0
	p := &x
	b := 0

	arr[0], s.f, *p, b = 10, 20, 30, 40

	println(arr[0], s.f, x, b)
}

// Output:
// 10 20 30 40
