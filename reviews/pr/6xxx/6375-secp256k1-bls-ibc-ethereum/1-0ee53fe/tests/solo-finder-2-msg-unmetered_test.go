// solo-finder-2-msg-unmetered (native timing half): growth of one call's
// native work across msg sizes, one key, valid encodings.
//
// Repro, from a plain clone of gnolang/gno at 0ee53fefc:
//   cp <this file> gnovm/stdlibs/crypto/bls12381/msg_growth_test.go
//   cd gnovm && go test -run '^$' -bench BenchmarkMsgGrowth -benchtime 5x ./stdlibs/crypto/bls12381/
// Measured growth ratio of ns/op against msg=32 (gas ratio 1.00 at every size):
//   fastAggregateVerify           1MiB 1.6x  16MiB 7.8x  64MiB 26.6x
//   fastAggregateVerifyExcluding  1MiB 1.4x  16MiB 6.6x  64MiB 22.4x
package bls12381

import (
	"encoding/hex"
	"fmt"
	"testing"
)

// The native's gas row slopes on pubkeys alone; this times one call with one
// key across msg sizes, so the growth ratio can be read against flat gas.
func BenchmarkMsgGrowth(b *testing.B) {
	pk, _ := hex.DecodeString("820ad0f24a42c82129fef2a137f7b7c230c2aaffb78ffd82f6cbdcd2bfbf3560435a35c62d3ff66ad696b78f8c6c6c68")
	sig, _ := hex.DecodeString("857cbc9c0976c88648089b0598cdc7b8c75a96f8e4f06ea0f0bbd0e2d5ac74d9dae7d4c479a7b573f0d7f7e3b875a6b9" +
		"039083a95ea6f3af6d432056337bc115ae50023fa289c320322941a3b07db7a710b438f643432a1a56f64b121931357e")
	for _, n := range []int{32, 1 << 20, 16 << 20, 64 << 20} {
		msg := make([]byte, n)
		b.Run(fmt.Sprintf("msg=%d", n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				X_fastAggregateVerify(pk, msg, sig)
			}
		})
		b.Run(fmt.Sprintf("excl/msg=%d", n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				X_fastAggregateVerifyExcluding(pk, nil, msg, sig)
			}
		})
	}
}
