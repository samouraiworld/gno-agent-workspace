// PKGPATH: gno.land/r/demo/curlitname
package curlitname

// PR 6196, bundle 2 (claims angle), finder b2-claims-r2.
//
// CLAIM UNDER TEST (ADR "Diagnostics name anonymous callees"; PR body fix 4):
//   "`funcDisplayName` falls back to the source location for a function
//    literal and to `<func literal>` when there is no source, so the panic
//    never renders as a bare trailing dot."
//
// The claim HOLDS at head: this file renders
//   crossing function gno.land/r/demo/curlitname func literal at
//   zrealm_cur_litname.gno:7 was entered without cross(), ...
//
// What nothing pins is the branch that produces it. op_call_test.go's
// TestFuncDisplayName exercises only Name != "" and Source == nil; a real
// function-literal FuncValue always has Source != nil, so the ONE branch that
// fires in production is untested, and no fixture anywhere in the tree
// contains the string "func literal at".
//
// MUTATION THAT MUST TURN A TEST RED AND DOES NOT (run at head):
//   delete the `if fv.Source != nil { ... }` block from funcDisplayName
//   (gnovm/pkg/gnolang/op_call.go:571-574), then:
//     go test ./gnovm/pkg/gnolang/ -run 'TestFuncDisplayName'   # ok
//     go test ./gnovm/pkg/gnolang/ -run 'TestHarnessSeedsCur'   # ok
//     go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zrealm_cur'  # ok
//   The message silently regresses to the bare ".<func literal>" form that
//   fix 4 exists to replace.
//
// REPRO FROM A PLAIN CLONE:
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin 6d88deba71165b165641b65541259d00aba20377
//   git checkout 6d88deba71165b165641b65541259d00aba20377
//   cp <this file> gnovm/tests/files/zrealm_cur_litname.gno
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zrealm_cur_litname.gno$'
//   # passes as written; delete the Source branch above and it fails,
//   # which is the coverage this fixture adds.

func exec(cur realm, cb func()) { cb() }

func main(cur realm) {
	f := func(cur realm) { println("f:", cur.PkgPath()) }
	cb := func() { f(cur) }
	exec(cross(cur), cb)
}

// Error:
// crossing function gno.land/r/demo/curlitname func literal at zrealm_cur_litname.gno:7 was entered without cross(), so it takes its caller's identity — but the value passed is not the caller's own cur (a stale capture from another frame, a sibling frame, or a rebound cur). Pass the caller's own `cur` unchanged, or use cross(cur) to enter as this realm
