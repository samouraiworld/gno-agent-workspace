/* Run: from a gno checkout:
gh pr checkout 6058 -R gnolang/gno && git checkout 391840aa7
curl -fsSL -o gnovm/tests/files/switch58.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6058-switch-case-shadowing/1-391840aa7/tests/switch58.gno
go test -v -run 'TestFiles/switch58.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/switch58.gno
*/

package main

// A `const` shadow in a clause body must leave the outer name alone at a point
// textually before it. `// Output:` is what `go run` prints; 391840aa7 prints 0
// on the first line and fails this file.
func main() {
	switch v := 1; v {
	case 1:
		println(v)
		const v = "c"
		println(v)
	}
}

// Output:
// 1
// c
