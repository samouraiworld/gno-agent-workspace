// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a local clone of gnolang/gno:
gh pr checkout 5765 -R gnolang/gno && git checkout 28164a5d1
curl -fsSL -o gnovm/tests/files/assign_tuple_alias_stress.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5765-tuple-assign-eval-order/1-28164a5d1/tests/assign_tuple_alias_stress.gno
go test -v -run 'TestFiles/assign_tuple_alias_stress.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/assign_tuple_alias_stress.gno
*/

// Worst case for the frames[] backing-array reuse: 8 IndexExpr LHS targets,
// each contributing 2 stack values (X + Index). doOpAssign pops all 16 raw
// values into frames[] (which slice into m.Values) then re-pushes 2-at-a-time
// per target and resolves. If any re-push clobbered a not-yet-consumed frame,
// some arr[i] would land the wrong value. Distinct indices into one slice plus
// distinct RHS make every scramble observable. Observed: 0 1 2 3 4 5 6 7.
// Equivalent Go prints the same; flip the stdout line to assert a regression.

package main

func main() {
	a := []int{-1, -1, -1, -1, -1, -1, -1, -1}
	a[0], a[1], a[2], a[3], a[4], a[5], a[6], a[7] = 0, 1, 2, 3, 4, 5, 6, 7
	println(a[0], a[1], a[2], a[3], a[4], a[5], a[6], a[7])
}

// Output:
// 0 1 2 3 4 5 6 7
