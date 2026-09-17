// PKGPATH: gno.land/r/demo/curlitname
package curlitname

// Repro, from a plain clone at 6d88deba71165b165641b65541259d00aba20377:
//
//   git clone https://github.com/gnolang/gno && cd gno
//   git checkout 6d88deba71165b165641b65541259d00aba20377
//   cp <this file> gnovm/tests/files/zrealm_cur_litname.gno
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zrealm_cur_litname.gno$'   # PASS at head
//
// Mutation that must turn it red (nothing in the tree at this sha does):
//
//   sed -i 's|"%s func literal at %s:%d"|"MUTATED-%s-%s-%d"|' gnovm/pkg/gnolang/op_call.go
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zrealm_cur_litname.gno$'   # FAIL with the mutation
//   go test ./gnovm/pkg/gnolang/ -run 'TestFuncDisplayName'                 # ok WITH the mutation
//
// The callee of the identity panic is a crossing function LITERAL, which is
// the shape funcDisplayName's Source fallback was written for. The named-func
// branch is pinned by zrealm_cur_backstop.gno; this pins the literal branch.

func exec(cur realm, cb func()) {
	cb()
}

func main(cur realm) {
	tgt := func(cur realm) { println("tgt:", cur.PkgPath()) }
	cb := func() { tgt(cur) } // captures main's cur
	exec(cross(cur), cb)      // ...but runs inside exec's frame
}

// Error:
// crossing function gno.land/r/demo/curlitname func literal at zrealm_cur_litname.gno:26 was entered without cross(), so it takes its caller's identity — but the value passed is not the caller's own cur (a stale capture from another frame, a sibling frame, or a rebound cur). Pass the caller's own `cur` unchanged, or use cross(cur) to enter as this realm
