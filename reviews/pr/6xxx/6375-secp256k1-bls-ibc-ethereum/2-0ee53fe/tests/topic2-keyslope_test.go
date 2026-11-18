// topic2-keyslope: native work of both bls12381 natives against their gas
// rows along the priced dimension (keys), at 1, 512 and 4096 keys, plus the
// early-failing call (first key undecodable) that pays the full charge.
//
// Repro, from a plain clone of gnolang/gno at 0ee53fefc:
//   cp <this file> gnovm/stdlibs/crypto/bls12381/topic2_keyslope_test.go
//   cd gnovm && go test -run '^$' -bench 'BenchmarkTopic2' -benchtime 20x -timeout 2m ./stdlibs/crypto/bls12381/
//
// Gas per call from the rows at native_gas.go:196 and :201 (Base + Slope*bytes/1024):
//   fastAggregateVerify          1 key 2115968, 512 keys 49112000 (23.2x), 4096 keys 378728000 (179.0x)
//   fastAggregateVerifyExcluding 0 excl 2116000, 512 excl 49204000 (23.3x), 4096 excl 378820000 (179.0x)
// Compare the ns/op growth ratio over the 1-key (0-excluded) case with those.
//
// Measured at fd7e75152, ns/op growth over the smallest case:
//   fastAggregateVerify          512 keys 24.0x, 4096 keys 181.4x; 512 keys first-invalid ~1/740000 of the full call
//   fastAggregateVerifyExcluding 512 excl 25.4x, 4096 excl 198.3x; per excluded key within 2% of a FAV key
package bls12381

import (
	"fmt"
	"testing"

	bls "github.com/consensys/gnark-crypto/ecc/bls12-381"
)

// keysOf returns n distinct valid compressed G1 keys: k*G for k = 1..n.
func keysOf(n int) []byte {
	g := g1Generator()
	var acc bls.G1Jac
	var out []byte
	for range n {
		acc.AddMixed(&g)
		var a bls.G1Affine
		a.FromJacobian(&acc)
		b := a.Bytes()
		out = append(out, b[:]...)
	}
	return out
}

func BenchmarkTopic2FAV(b *testing.B) {
	msg := make([]byte, 32)
	_, sig := signers(b, 1, msg) // a valid G2 point; the pairing result does not change the work
	for _, n := range []int{1, 512, 4096} {
		pks := keysOf(n)
		b.Run(fmt.Sprintf("keys=%d", n), func(b *testing.B) {
			for range b.N {
				X_fastAggregateVerify(pks, msg, sig)
			}
		})
	}
	bad := keysOf(512)
	for i := range 48 {
		bad[i] = 0 // uncompressed flag on a 48-byte key: SetBytes errors at once
	}
	b.Run("keys=512/first-invalid", func(b *testing.B) {
		for range b.N {
			if X_fastAggregateVerify(bad, msg, sig) {
				b.Fatal("accepted")
			}
		}
	})
}

func BenchmarkTopic2Excluding(b *testing.B) {
	msg := make([]byte, 32)
	_, sig := signers(b, 1, msg)
	agg := keysOf(1)
	for _, n := range []int{0, 512, 4096} {
		var excl []byte
		if n > 0 {
			excl = keysOf(n + 1)[48:] // 2G..(n+1)G, never summing to the aggregate G
		}
		b.Run(fmt.Sprintf("excluded=%d", n), func(b *testing.B) {
			for range b.N {
				X_fastAggregateVerifyExcluding(agg, excl, msg, sig)
			}
		})
	}
}
