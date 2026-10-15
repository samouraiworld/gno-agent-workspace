// PR 6110 round 2, finder b2-lines: a map literal with an `any` key and a
// typed-nil slice/map/func constant key now fails at preprocess.
//
// preprocess.go:2510 runs ComputeMapKey on every ConstExpr key of a map
// literal; ComputeMapKey panics "hash of unhashable type" for a slice, map
// or func dynamic type. Go compiles this file and panics only if never()
// runs; the merge base's kset[cx.TypedValue] never panicked at preprocess.
//
// Repro from a plain clone:
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin pull/6110/head && git checkout c800a571ab179a07530dcf876142271afc64e40a
//   cp <this file> gnovm/tests/files/b2lines_maplit_a.gno
//   go test ./gnovm/pkg/gnolang -run 'TestFiles/b2lines_maplit_a' -count=1
// Head: FAIL, "unexpected panic: main/b2lines_maplit_a.gno:24:7-33:
//   &{("runtime error: hash of unhashable type []int" .uverse..runtimeError) ...}"
// Same file with gnovm/pkg/gnolang/preprocess.go restored from 0099b3138:
//   git show 0099b3138:gnovm/pkg/gnolang/preprocess.go > gnovm/pkg/gnolang/preprocess.go
//   go test ./gnovm/pkg/gnolang -run 'TestFiles/b2lines_maplit_a' -count=1   -> ok
// Same failure with map[int]int(nil) and (func())(nil) as the key.
// Go parity: `go run` of the same program prints "ok".
package main

func never() {
	m := map[any]int{[]int(nil): 1}
	_ = m
}

func main() {
	println("ok")
}

// Output:
// ok
