/* Run: from a gno checkout:
gh pr checkout 6037 -R gnolang/gno && git checkout 390bffe90
curl -fsSL -o gnovm/tests/files/loopvar_composite_key_nested.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6037-map-composite-key-in-loop/1-390bffe90/tests/loopvar_composite_key_nested.gno
go test -v -run 'TestFiles/loopvar_composite_key_nested.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/loopvar_composite_key_nested.gno
*/

// One literal holds both halves of the loop-variable rename: the outer key is a
// map key that must keep its ".loopvar" suffix, the elided inner key is a struct
// field name that must lose it.
// At merge base fb02547fd this fails with "name i not declared"; with the
// exclusion dropped but no suffix trim it fails with "has no field i.loopvar".
// It passes only at 390bffe90.

package main

type T struct{ i int }

type M map[int]T

func main() {
	for i := 0; i < 2; i++ {
		m := M{i: {i: i + 4}}
		println(m[i].i)
	}
}

// Output:
// 4
// 5
