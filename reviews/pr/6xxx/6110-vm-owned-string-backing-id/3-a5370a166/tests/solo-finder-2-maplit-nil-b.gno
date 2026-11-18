// Repro (from a plain gno clone; at a5370a166 and at merge base 6c49f5420):
//   cp solo-finder-2-maplit-nil-b.gno gnovm/tests/files/sf2_maplit_nil_b.gno
//   go test ./gnovm/pkg/gnolang -run TestFiles/sf2_maplit_nil_b -count=1 -timeout 2m
// Go (go run) prints the Output below; gno at head panics "duplicate key (nil *int) in map literal".
// Head: b and f fail. Base: b fails, f passes (f is new on this branch).
package main

func main() {
	m := map[*int]int{nil: 1, nil: 2}; println(len(m), m[nil])
}

// Output:
// 1 2
