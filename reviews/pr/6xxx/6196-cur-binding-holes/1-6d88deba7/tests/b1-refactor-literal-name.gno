// PKGPATH: gno.land/r/demo/curlitname
package curlitname

// Repro from a plain clone:
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin pull/6196/head
//   git checkout 6d88deba71165b165641b65541259d00aba20377
//   cp <this file> gnovm/tests/files/zrealm_cur_literal_name.gno
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zrealm_cur_literal_name.gno$'
//
// It PASSES at the reviewed sha: the branch it covers works. The point is that
// nothing in the tree covered it — `grep -rn "func literal at" gnovm/tests/` is
// empty there, and TestFuncDisplayName (op_call_test.go) asserts only the
// Name != "" and the Source == nil arms, not the source-location arm that PR
// 6196 adds for exactly this shape: a crossing function LITERAL entered
// without cross() with a cur captured in another frame.
//
// The pinned line also records the rendering, "<pkgpath> func literal at
// <basename>:<line>", which is not the "<pkgpath>/<file>:<span>" that
// Location.String() gives the stacktrace printed right under it.

func exec(cur realm, cb func()) {
	cb()
}

func main(cur realm) {
	f := func(cur realm) { println("f:", cur.PkgPath()) }
	cb := func() { f(cur) } // captures main's cur, calls the crossing literal
	exec(cross(cur), cb)    // ...but runs inside exec's frame
}

// Error:
// crossing function gno.land/r/demo/curlitname func literal at zrealm_cur_literal_name.gno:27 was entered without cross(), so it takes its caller's identity — but the value passed is not the caller's own cur (a stale capture from another frame, a sibling frame, or a rebound cur). Pass the caller's own `cur` unchanged, or use cross(cur) to enter as this realm
