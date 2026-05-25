// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.

/* Run: from a local clone of gnolang/gno:
gh pr checkout 5155 -R gnolang/gno && git checkout 9ae784f3
curl -fsSL -o gnovm/tests/files/print_nested_blowup.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5155-print-truncation/1-9ae784f3/tests/print_nested_blowup.gno
go test -v -run 'TestFiles/print_nested_blowup.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/print_nested_blowup.gno
*/

// Goal: probe whether the printOutputLimit=64_000 safety net actually fires
// on the print() path. The per-slice cap (printLimit=256) renders one
// "slice[...(256 elements)]" string per slice, so a top-level slice of N
// already-truncated child slices produces O(N * 30) bytes — well under 64KB
// at typical N. Combinatorial blowup needs nested arrays/structs that the
// per-level cap doesn't truncate (structs have no field cap), but the
// per-level slice/map cap of 256 makes the realistic blowup small for slice
// print paths. This filetest just exercises the print() call path with a
// modestly large nested shape and asserts the per-level truncation strings
// appear; it does NOT prove the top-level safety net is reachable from print().

package main

func main() {
	// 5 outer * 5 inner * 300 leaf — outer/inner under the 256 cap so they
	// expand and each leaf hits the 256 cap.
	outer := make([][][]int, 5)
	for i := 0; i < 5; i++ {
		outer[i] = make([][]int, 5)
		for j := 0; j < 5; j++ {
			outer[i][j] = make([]int, 300)
		}
	}
	println(outer)
}

// Output:
// slice[(slice[(slice[...(300 elements)] []int),(slice[...(300 elements)] []int),(slice[...(300 elements)] []int),(slice[...(300 elements)] []int),(slice[...(300 elements)] []int)] [][]int),(slice[(slice[...(300 elements)] []int),(slice[...(300 elements)] []int),(slice[...(300 elements)] []int),(slice[...(300 elements)] []int),(slice[...(300 elements)] []int)] [][]int),(slice[(slice[...(300 elements)] []int),(slice[...(300 elements)] []int),(slice[...(300 elements)] []int),(slice[...(300 elements)] []int),(slice[...(300 elements)] []int)] [][]int),(slice[(slice[...(300 elements)] []int),(slice[...(300 elements)] []int),(slice[...(300 elements)] []int),(slice[...(300 elements)] []int),(slice[...(300 elements)] []int)] [][]int),(slice[(slice[...(300 elements)] []int),(slice[...(300 elements)] []int),(slice[...(300 elements)] []int),(slice[...(300 elements)] []int),(slice[...(300 elements)] []int)] [][]int)]
