// PKGPATH: gno.land/r/demo/curnofwd
package curnofwd

// PR 6196, bundle 2 (claims angle), finder b2-claims-r2.
//
// CLAIM UNDER TEST (PR 6196 body, "Compatibility"):
//   "No `.gno` source that compiled and ran before is rejected now: the write
//    rules only got narrower through the declaration test, and the `DEFINE`
//    rebind they still reject was already fail-closed at call time."
//
// The `DEFINE` rebind is fail-closed at call time ONLY when the rebound `cur`
// is later forwarded to a crossing function: installInheritedCur's identity
// check runs on a crossing entry and nowhere else. A rebind that never
// forwards reaches no crossing entry, so on the merge base it compiled AND
// ran. At head the moved write rule (preprocess.go, the `for _, lh := range
// n.Lhs` loop after the DEFINE/ASSIGN branch) rejects it at preprocess.
//
// REPRO FROM A PLAIN CLONE:
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin 6d88deba71165b165641b65541259d00aba20377
//   # head (PR 6196):
//   git checkout 6d88deba71165b165641b65541259d00aba20377
//   cp <this file> gnovm/tests/files/zrealm_cur_nofwd.gno
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zrealm_cur_nofwd.gno$'
//   #   FAIL: unexpected panic: .../zrealm_cur_nofwd.gno:9:2-29: cannot
//   #   reassign the crossing `cur` parameter: ...
//   # merge base (b7845fb2da9a21bfb895e1c008f016776f8342d0, already carries #6193):
//   git checkout b7845fb2da9a21bfb895e1c008f016776f8342d0
//   cp <this file> gnovm/tests/files/zrealm_cur_nofwd.gno
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zrealm_cur_nofwd.gno$'
//   #   ok  github.com/gnolang/gno/gnovm/pkg/gnolang
//
// The file is written to PASS on the merge base. At head it fails at
// preprocess, which is the delta: source that compiled and ran is now
// rejected, against the Compatibility claim.

func main(cur realm) {
	cur, x := cur.Previous(), 1
	_ = x
	_ = cur
	println("ok")
}

// Output:
// ok
