// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a gno checkout:
gh pr checkout 5808 -R gnolang/gno && git checkout 17b76f841
curl -fsSL -o gnovm/tests/files/nilmap_read_unhashable.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5808-pin-nil-map-delete/1-17b76f841/tests/nilmap_read_unhashable.gno
go test -v -run 'TestFiles/nilmap_read_unhashable.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/nilmap_read_unhashable.gno
*/
// Mechanism: reading a nil map with an unhashable (slice) key. delete1.gno's
// comment justifies the pinned delete no-op by "gno's own nil-map read
// behavior", but no filetest pins that read behavior. gc panics
// "hash of unhashable type: []int" here; gno returns zero values for both
// 1-value and comma-ok reads without hashing the key.
// Observed at 17b76f841: PASS — prints "read: 0 0 false".
// Flip: assert gc behavior instead (panic) — fails, gno returns zero values.
package main

func main() {
	var mi map[interface{}]int
	v := mi[[]int{1}]
	v2, ok := mi[[]int{2}]
	println("read:", v, v2, ok)
}

// Output:
// read: 0 0 false
