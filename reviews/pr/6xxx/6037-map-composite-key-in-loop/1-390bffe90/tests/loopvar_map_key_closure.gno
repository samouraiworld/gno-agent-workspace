/* Run: from a gno checkout:
gh pr checkout 6037 -R gnolang/gno && git checkout 390bffe90
curl -fsSL -o gnovm/tests/files/loopvar_map_key_closure.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6037-map-composite-key-in-loop/1-390bffe90/tests/loopvar_map_key_closure.gno
go test -v -run 'TestFiles/loopvar_map_key_closure.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/loopvar_map_key_closure.gno
*/

// A map key reading a loop variable that the closure also captures, so the key
// must resolve to the heap-promoted per-iteration binding rather than the last one.
// At merge base fb02547fd this fails with "name i not declared"; it passes at
// 390bffe90 and prints what Go prints.

package main

func main() {
	var fns []func()
	for i := 0; i < 3; i++ {
		fns = append(fns, func() {
			println(map[int]int{i: i * 2}[i])
		})
	}
	for _, fn := range fns {
		fn()
	}
}

// Output:
// 0
// 2
// 4
