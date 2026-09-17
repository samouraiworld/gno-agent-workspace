// The read-back zrealm_cur_shadow.gno stops short of, at
// 6d88deba71165b165641b65541259d00aba20377.
//
// This PR drops the `evalStaticTypeOf(store, last, ne) == gRealmType` test from
// the write rule (gnovm/pkg/gnolang/preprocess.go:3011) and resolves the name to
// its declaring block instead, which legalises rebinding a realm-typed shadow of
// `cur` for the first time: at the merge base `cur := cur; cur = nil` inside a
// crossing function was refused, and zrealm_cur_shadow.gno flipped from an
// expected rejection to a positive control in the same commit. That control ends
// on println("ok") and never touches the parameter after the block, so it cannot
// tell a separate slot from one aliasing the frame's identity. This file adds the
// missing half: after the shadow has been nil'ed, cur-call a crossing helper with
// the real parameter.
//
// Repro from a plain clone:
//
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin pull/6196/head:pr6196 && git checkout 6d88deba71165b165641b65541259d00aba20377
//   cp <this file> gnovm/tests/files/zz_cur_shadow_readback.gno
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zz_cur_shadow_readback.gno$' -v
//
// Observed at head:
//   --- PASS: TestFiles/zz_cur_shadow_readback.gno (0.00s)
//
// So the shadow write does not reach the parameter slot and the behaviour is
// right; what is missing is the assertion, which costs the two lines below.

// PKGPATH: gno.land/r/demo/curshadowreadback
package curshadowreadback

func probe(cur realm) {
	println(cur.IsCurrent())
}

func main(cur realm) {
	{
		cur := cur
		cur = nil
		p := &cur
		*p = nil
		_ = cur
	}
	probe(cur)
	println("ok")
}

// Output:
// true
// ok
