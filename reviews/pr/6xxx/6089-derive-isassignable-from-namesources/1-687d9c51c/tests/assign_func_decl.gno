// Asserts the two names the func-decl refusal must not swallow: a package-level
// var holding a func value, and a local that shadows a package-level func.
// Both stay assignable. Passes at 687d9c51c and on the merge base add66f24b.

/* Run: from a gno checkout:
gh pr checkout 6089 -R gnolang/gno && git checkout 687d9c51c
curl -fsSL -o gnovm/tests/files/assign_func_decl.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6089-derive-isassignable-from-namesources/1-687d9c51c/tests/assign_func_decl.gno
go test -v -run 'TestFiles/assign_func_decl.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/assign_func_decl.gno
*/

package main

func f() { println("f") }

var g = func() { println("g") }

func main() {
	g = f
	g()
	f := 1
	f = 2
	println(f)
}

// Output:
// f
// 2
