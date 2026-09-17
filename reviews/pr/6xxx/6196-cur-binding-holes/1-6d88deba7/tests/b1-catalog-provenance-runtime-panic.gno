// Repro: the cur-call provenance check at preprocess.go:2155 uses the same
// declaring-node test, so a named result `cur realm` of a crossing function is
// accepted as a frame identity at preprocess and only rejected at runtime, by
// the stale-capture panic, whose text describes a different mistake.
//
// Head (6d88deba7) and base (b7845fb2d): runtime panic
//   "crossing function gno.land/r/demo/curzz2.Target was entered without
//    cross(), so it takes its caller's identity — but the value passed is not
//    the caller's own cur (a stale capture from another frame, ...)"
//
// From a plain clone:
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin pull/6196/head:pr6196 && git checkout 6d88deba7
//   cp <this file> gnovm/tests/files/zzb1provenance.gno
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zzb1provenance.gno$'

// PKGPATH: gno.land/r/demo/curzz2
package curzz2

func Target(cur realm) {
	println(cur.IsCurrent())
}

func F(_ realm) (cur realm) {
	Target(cur)
	return
}

func main(cur realm) {
	_ = F(cross(cur))
	println("ok")
}

// Output:
// ok
