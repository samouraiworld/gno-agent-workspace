// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a local clone of gnolang/gno:
gh pr checkout 5765 -R gnolang/gno && git checkout 28164a5d1
curl -fsSL -o gnovm/tests/files/assign_tuple_operand_sideeffects.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5765-tuple-assign-eval-order/1-28164a5d1/tests/assign_tuple_operand_sideeffects.gno
go test -v -run 'TestFiles/assign_tuple_operand_sideeffects.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/assign_tuple_operand_sideeffects.gno
*/

// Go spec: the operands of LHS index/pointer expressions are evaluated
// (left-to-right) BEFORE any assignment. doOpAssign reorders pointer
// RESOLUTION/assignment, not operand evaluation — PushForPointer already ran
// the index/X expressions during op_exec. This checks the observable operand
// order is unchanged: idx() is called for arr[idx()] and brr[idx()] before
// the writes, in source order. Equivalent Go prints "idx0 idx1 done 100 200".
// Observed under PR 5765: idx0 idx1 done 100 200 — operand order preserved.

package main

var calls = 0

func idx() int {
	println("idx", calls)
	calls++
	return 0
}

func main() {
	arr := []int{-1}
	brr := []int{-1}
	arr[idx()], brr[idx()] = 100, 200
	println("done", arr[0], brr[0])
}

// Output:
// idx 0
// idx 1
// done 100 200
