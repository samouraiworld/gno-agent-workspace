// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a gno checkout:
gh pr checkout 5808 -R gnolang/gno && git checkout 17b76f841
curl -fsSL -o gnovm/tests/files/unhashable_delete_recoverable.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5808-pin-nil-map-delete/1-17b76f841/tests/unhashable_delete_recoverable.gno
go test -v -run 'TestFiles/unhashable_delete_recoverable.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/unhashable_delete_recoverable.gno
*/
// Mechanism: delete on a NON-nil map[interface{}]int with a slice key reaches
// TypedValue.ComputeMapKey, whose *SliceType case panics with
// &Exception{"runtime error: slice type cannot be used as map key"}
// (gnovm/pkg/gnolang/values.go:1662). Machine.runOnce converts Go-level
// *Exception panics into recoverable gno panics (machine.go:1655-1663),
// so recover() catches it. At this PR's pre-merge base (59df7d868) the same
// site was a plain Go panic (values.go:1647 there) = unrecoverable VM abort;
// #5501 (326832e56, on this head via merge bf1467158) made it recoverable.
// Observed at 17b76f841: PASS — recover() returns the runtime error and the
// map is untouched (len 1). This is the premise check for the divergence
// pinned by delete1.gno:24 (delete on a NIL map with the same key no-ops).
// Flip: change the Output block's "recovered:" line to assert a VM abort —
// the harness then fails with the recovered runtime error instead.
package main

func main() {
	mi := map[interface{}]int{"a": 1}
	func() {
		defer func() {
			println("recovered:", recover())
		}()
		delete(mi, []int{1})
	}()
	println("after, len:", len(mi))
}

// Output:
// recovered: runtime error: slice type cannot be used as map key
// after, len: 1
