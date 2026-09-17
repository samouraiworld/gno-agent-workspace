// Repro: a named result `cur realm` in a crossing function is treated as the
// crossing parameter, so writing it is refused. `isCrossingCurParam` resolves
// the name to its declaring node and asks only `IsCrossing()`, never whether
// the first parameter is actually named `cur`.
//
// Head (6d88deba7) and base (b7845fb2d): preprocess panic
//   "cannot reassign the crossing `cur` parameter: ..."
// Pre-existing, but the class this PR's fix 2 sweeps ("the write rules refused
// more than the parameter"): `func F(x int) (cur realm)` was fixed,
// `func F(_ realm) (cur realm)` was not.
//
// From a plain clone:
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin pull/6196/head:pr6196 && git checkout 6d88deba7
//   cp <this file> gnovm/tests/files/zzb1namedres.gno
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zzb1namedres.gno$'

// PKGPATH: gno.land/r/demo/curzz1
package curzz1

func F(_ realm) (cur realm) {
	cur = nil
	return cur
}

func main(cur realm) {
	_ = F(cross(cur))
	println("ok")
}

// Output:
// ok
