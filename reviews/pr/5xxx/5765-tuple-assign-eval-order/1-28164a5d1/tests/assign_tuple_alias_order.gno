// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a local clone of gnolang/gno:
gh pr checkout 5765 -R gnolang/gno && git checkout 28164a5d1
curl -fsSL -o gnovm/tests/files/assign_tuple_alias_order.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5765-tuple-assign-eval-order/1-28164a5d1/tests/assign_tuple_alias_order.gno
go test -v -run 'TestFiles/assign_tuple_alias_order.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/assign_tuple_alias_order.gno
*/

// Go spec: assignments proceed left-to-right, so when LHS targets alias the
// same memory the LAST write wins. doOpAssign must resolve+assign in forward
// LHS order. Three aliasing shapes in one statement: same map key, same slice
// index, same deref. Equivalent Go (go run) prints 3 30 300. Observed under
// PR 5765: 3 30 300 — last-writer-wins parity. Under old master (reverse
// assign) this prints 1 10 100 (first-writer-wins); flip the comment to
// assert the regression.

package main

func main() {
	m := map[string]int{}
	arr := []int{0}
	x := 0
	p := &x

	m["k"], m["k"], m["k"] = 1, 2, 3
	arr[0], arr[0], arr[0] = 10, 20, 30
	*p, *p, *p = 100, 200, 300

	println(m["k"], arr[0], x)
}

// Output:
// 3 30 300
