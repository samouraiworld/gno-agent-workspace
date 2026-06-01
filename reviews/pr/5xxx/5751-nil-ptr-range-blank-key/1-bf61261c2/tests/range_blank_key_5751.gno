// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a local clone of gnolang/gno:
gh pr checkout 5751 -R gnolang/gno && git checkout bf61261c2
cat > gnovm/tests/files/range_blank_key_5751.gno <<'EOF'
<this file, from the package line down>
EOF
go test -v -run 'TestFiles/range_blank_key_5751.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/range_blank_key_5751.gno
*/

// PR #5751 fixes `for _ = range slice` (blank key, value absent) by widening the
// early-return in (*RangeStmt).AssertCompatible. But the sibling form
// `for _, v = range slice` (blank key, value PRESENT) takes a different path:
// it skips the early return, evalStaticTypeOf(`_`) yields nil, and the SliceType
// branch calls assertIndexTypeIsInt(nil) at type_check.go:853 -> 823, which
// dereferences nil. Same crash as issue #5664, different range form.
//
// Observed (PR head bf61261c2): uncaught nil pointer dereference, stack ends in
//   gnolang.assertIndexTypeIsInt({0x0, 0x0})  type_check.go:823
//   gnolang.(*RangeStmt).AssertCompatible(...)  type_check.go:853
//
// Flip the assertion: once the slice/array branch guards a nil key type
// (as the StringKind branch already does with `if kt != nil`), this prints `ok`
// and the `// Output: ok` below holds.

package main

func main() {
	a := []int{1, 2, 3}
	var v int
	for _, v = range a {
	}
	_ = v
	println("ok")
}

// Output:
// ok
