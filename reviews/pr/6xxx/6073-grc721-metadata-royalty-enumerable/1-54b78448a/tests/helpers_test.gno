// Shared helper for the probe tests beside this file. NewToken requires a
// current realm, so the token is built through a same-realm cross.
package zprobe

import (
	"gno.land/p/demo/tokens/grc721"
	"gno.land/p/nt/seqid/v0"
)

func newCore(name, symbol string, id seqid.ID, rlm realm) (tok *grc721.Token, led *grc721.PrivateLedger) {
	func(cur realm) {
		tok, led = grc721.NewToken(name, symbol, id, cur)
	}(cross(rlm))

	return
}
