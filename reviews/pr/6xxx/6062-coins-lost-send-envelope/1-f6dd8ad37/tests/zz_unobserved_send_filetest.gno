// PKGPATH: gno.land/r/test/zzunobserved
// SEND: 400ugnot

// A SEND: envelope no code reads passes under `gno test`, while MsgCall
// refuses the same call with "never read the send-envelope". The 400ugnot
// strands in the realm's address and the harness reports green.
//
// Run: from a gno checkout:
// gh pr checkout 6062 -R gnolang/gno && git checkout f6dd8ad37
// cp zz_unobserved_send_filetest.gno gnovm/tests/files/
// go test -v -run 'TestFiles/zz_unobserved_send_filetest.gno$' ./gnovm/pkg/gnolang/
// rm gnovm/tests/files/zz_unobserved_send_filetest.gno

package zzunobserved

import (
	"chain/banker"
	"chain/runtime/unsafe"
)

// ignores has no notion of being paid.
func ignores(cur realm) {}

func main(cur realm) {
	ignores(cur)
	me := unsafe.CurrentRealm().Address()
	println("stranded=", banker.NewReadonlyBanker().GetCoins(me).AmountOf("ugnot"))
}

// Output:
// stranded= 400
