// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a gno checkout:
gh pr checkout 5808 -R gnolang/gno && git checkout 17b76f841
curl -fsSL -o gnovm/tests/files/opus2_funckey_recover_probe.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5808-pin-nil-map-delete/1-17b76f841/tests/opus2_funckey_recover_probe.gno
go test -v -run 'TestFiles/opus2_funckey_recover_probe.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/opus2_funckey_recover_probe.gno
*/
// Splits "unhashable map key" into its two gno panic regimes, the distinction
// ADR pr5808_delete_nil_map.md consequence 2 collapses. TypedValue.ComputeMapKey
// (gnovm/pkg/gnolang/values.go) raises a *recoverable* panic(&Exception{}) for a
// slice key (values.go:1662, "slice type cannot be used as map key"), but for a
// func key falls to the default arm (values.go:1683-1686) and raises a *raw Go*
// panic("unexpected map key type func()") — an unrecoverable VM abort.
//
// Observed: this filetest does NOT print "recovered:" or "no panic". The runner
// reports `unexpected panic: unexpected map key type func()` with a Go stack
// through Machine.RunMain — the host aborts; the gno defer/recover never runs.
// Contrast opus2_slicekey_recover_probe.gno, where recover() catches the slice
// key. So the ADR's "does not go through the recoverable exception path" holds
// for func keys but is false for the slice key delete1.gno actually pins.
//
// To flip the assertion: if the default arm ever wraps its panic in &Exception
// (recoverable), this filetest would instead print "recovered: ..." and the
// `// Error:` directive below would need to become an `// Output:` directive.
package main

func main() {
	defer func() {
		if r := recover(); r != nil {
			println("recovered:", r)
		} else {
			println("no panic")
		}
	}()
	m := map[interface{}]int{1: 1} // non-nil
	delete(m, func() {})           // func key: unhashable, raw-Go-panic arm
	println("delete returned normally")
}

// Error:
// unexpected map key type func()
