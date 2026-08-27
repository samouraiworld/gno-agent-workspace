// Asserts that assigning to a package-level func name is refused, by the
// preprocessor and by the go/types pass. No filetest covered this path at
// 687d9c51c; it passes there and on the merge base add66f24b.

/* Run: from a gno checkout:
gh pr checkout 6089 -R gnolang/gno && git checkout 687d9c51c
curl -fsSL -o gnovm/tests/files/assign_func_decl_err.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6089-derive-isassignable-from-namesources/1-687d9c51c/tests/assign_func_decl_err.gno
go test -v -run 'TestFiles/assign_func_decl_err.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/assign_func_decl_err.gno
*/

package main

func f() {}

func main() {
	f = nil
}

// Error:
// main/assign_func_decl_err.gno:18:2-9: not assignable

// TypeCheckError:
// main/assign_func_decl_err.gno:18:2: cannot assign to f (neither addressable nor a map index expression)
