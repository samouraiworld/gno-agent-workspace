// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a gno checkout:
gh pr checkout 5808 -R gnolang/gno && git checkout 17b76f841
curl -fsSL -o gnovm/tests/files/opus2_slicekey_recover_probe.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5808-pin-nil-map-delete/1-17b76f841/tests/opus2_slicekey_recover_probe.gno
go test -v -run 'TestFiles/opus2_slicekey_recover_probe.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/opus2_slicekey_recover_probe.gno
*/
// Probes whether gno's "slice type cannot be used as map key" panic (raised by
// TypedValue.ComputeMapKey via `panic(&Exception{...})`) is recoverable by a gno
// defer/recover. ADR pr5808_delete_nil_map.md consequence 2 asserts this panic
// "does not go through the recoverable exception path", justifying the nil-map
// unhashable-key no-op over gc parity. If recover() catches it below, that
// premise is false: hashing-before-nil-return would be a *recoverable* panic.
package main

func main() {
	defer func() {
		if r := recover(); r != nil {
			println("recovered:", r)
		} else {
			println("no panic")
		}
	}()
	m := map[interface{}]int{1: 1} // non-nil map
	delete(m, []int{1})            // unhashable slice key on a NON-nil map
	println("delete returned normally")
}

// Output:
// recovered: runtime error: slice type cannot be used as map key
