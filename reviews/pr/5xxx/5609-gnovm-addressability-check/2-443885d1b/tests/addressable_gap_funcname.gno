// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a local clone of gnolang/gno:
gh pr checkout 5609 -R gnolang/gno && git checkout 443885d1b
curl -fsSL -o gnovm/tests/files/addressable_gap_funcname.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5609-gnovm-addressability-check/2-443885d1b/tests/addressable_gap_funcname.gno
go test -v -run 'TestFiles/addressable_gap_funcname.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/addressable_gap_funcname.gno
*/

// Demonstrates that PR #5609's `isAddressable` accepts `&funcName` for a
// user-defined package-level function. `*NameExpr` returns true
// unconditionally; the Kind() of the name is never inspected, so a function
// value reaches doOpRef and a real *func() is materialized. Go rejects this
// at compile time.
//
// Observed: `// Output: ok` passes — `&bar` produces a usable pointer and the
// runtime invokes the function through it. Flip the assertion to // Error:
// once a FuncKind exclusion is added to *NameExpr (or NameExpr lookup goes
// through evalStaticTypeOf and returns false for FuncKind).

package main

func bar() { println("ok") }

func main() {
	p := &bar
	(*p)()
}

// Output:
// ok

// TypeCheckError:
// main/addressable_gap_funcname.gno:26:8: invalid operation: cannot take address of bar (value of type func())
