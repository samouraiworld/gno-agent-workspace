// Repro, gnolang/gno#5785 at b42a521b16655374e507bdb949ab5b8289bc8404:
//   git clone https://github.com/gnolang/gno && cd gno
//   git checkout b42a521b16655374e507bdb949ab5b8289bc8404
//   cp solo-finder-1-copy-elem-mismatch.gno gnovm/tests/files/solo_finder_1_copy_elem_mismatch.gno
//   go test ./gnovm/pkg/gnolang -run 'TestFiles/solo_finder_1_copy_elem_mismatch' -count=1 -v
// PASS means the GnoVM preprocesses and runs copy() between slices whose
// element types differ (Go rejects at compile time); dst[0], static type int,
// then holds the string "x". Only the go/types TypeCheckError rejects it.
// copy_struct_tags.gno, listed in the PR body as a static rejection, is the
// struct-tag instance of the same shape and also has no // Error: line.
package main

func main() {
	dst := make([]int, 1)
	src := []string{"x"}
	n := copy(dst, src)
	println(n, dst[0])
}

// Output:
// 1 x

// TypeCheckError:
// main/solo_finder_1_copy_elem_mismatch.gno:16:12: invalid copy: arguments dst (variable of type []int) and src (variable of type []string) have different element types int and string
