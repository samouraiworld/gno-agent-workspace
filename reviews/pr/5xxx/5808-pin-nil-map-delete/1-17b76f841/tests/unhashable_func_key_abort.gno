// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a gno checkout:
gh pr checkout 5808 -R gnolang/gno && git checkout 17b76f841
curl -fsSL -o gnovm/tests/files/unhashable_func_key_abort.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5808-pin-nil-map-delete/1-17b76f841/tests/unhashable_func_key_abort.gno
go test -v -run 'TestFiles/unhashable_func_key_abort.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/unhashable_func_key_abort.gno
*/
// Mechanism: delete on a NON-nil map[interface{}]int with a func key reaches
// ComputeMapKey's default case, a plain Go panic "unexpected map key type
// func()" (gnovm/pkg/gnolang/values.go:1683-1686). Unlike the *SliceType
// case (&Exception, recoverable), plain panics re-raise through
// Machine.runOnce (machine.go:1661) and abort the VM: recover() does NOT
// catch them. This test FAILS BY DESIGN, demonstrating the abort.
// Observed at 17b76f841: harness error "unexpected panic: unexpected map
// key type func()" with a Go stacktrace; no "recovered:" line is printed.
// Flip: once func-key hashing panics recoverably, set the Output block's
// first line to the recovered message and the test passes.
package main

func main() {
	mi := map[interface{}]int{"a": 1}
	func() {
		defer func() {
			println("recovered:", recover())
		}()
		delete(mi, func() {})
	}()
	println("done")
}

// Output:
// recovered: unexpected map key type func()
// done
