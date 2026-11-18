// solo-finder-1: a map literal Go accepts is rejected at preprocess.
//
// Repro from a plain clone of gnolang/gno:
//   git fetch origin pull/6110/head && git checkout --detach a5370a166b459689a3febf222f3a5edd8739058e
//   cp <this file> gnovm/tests/files/zz_finder1_nk1.gno
//   go test ./gnovm/pkg/gnolang -run 'TestFiles/zz_finder1_nk1' -count=1 -timeout 2m
// Head a5370a166: FAIL unexpected panic: duplicate key (nil *int) in map literal
// Merge base 6c49f5420: ok.  Go 1.25 (go run): prints 1.
// Sibling map[*int]int{nil: 1, nil: 2} fails at both head and base (pre-existing);
// map[any]int{nil: 1, nil: 2} and {any(nil): 1, any(nil): 2} pass at both.
package main

func main() {
	a := map[any]int{(*int)(nil): 1, (*int)(nil): 2}
	println(len(a))
}

// Output:
// 1
