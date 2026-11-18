// b2-lines-removed-claims-reach-catalog-key-growth: does the native time of
// fastAggregateVerify grow with the key count the way the draft row
// (native_gas.go:196, Base 2024000, Slope 1962000 per 1024 bytes of pubkeys)
// charges it? The row's gas ratio of 512 keys over 1 key is
// (2024000 + 1962000*512*48/1024) / (2024000 + 1962000*48/1024) = 23.2.
// This prints the measured time ratio over the same key counts; growth
// ratios only, no ns-to-gas rate.
//
// Repro, from a plain clone of gnolang/gno at 0ee53fefc:
//   cp <this file> gnovm/stdlibs/crypto/bls12381/key_growth_test.go
//   cd gnovm && go test -run '^$' -bench BenchmarkKeyGrowth -benchtime 20x -timeout 2m ./stdlibs/crypto/bls12381/
// Measured at fd7e75152 (0ee53fefc slice), AMD Ryzen 7 7840HS: time ratio
// 512 keys over 1 key is 24.9x for fastAggregateVerify and 24.6x for the
// Excluding twin, against the row's 23.2x: the per-key slope tracks the base.
// The 1-key call runs 1.2x and the 512-key call 1.28x the M4 figures the row
// scales (0.88ms + 40us/key), on the host class (Ryzen 7 7840U) the 2.3x
// factor of ibc_native_bench_test.go:76-85 was derived on.
package bls12381

import (
	"fmt"
	"math/big"
	"testing"

	bls "github.com/consensys/gnark-crypto/ecc/bls12-381"
)

func keyGrowthKeys(n int) []byte {
	_, _, g1, _ := bls.Generators()
	out := make([]byte, 0, n*pubKeySize)
	for i := 0; i < n; i++ {
		var p bls.G1Affine
		p.ScalarMultiplication(&g1, big.NewInt(int64(1000003*i+7)))
		b := p.Bytes()
		out = append(out, b[:]...)
	}
	return out
}

func BenchmarkKeyGrowth(b *testing.B) {
	_, _, g1, g2 := bls.Generators()
	aggb := g1.Bytes()
	sigb := g2.Bytes()
	sig := sigb[:]
	msg := make([]byte, 32)
	for _, n := range []int{1, 8, 64, 512} {
		keys := keyGrowthKeys(n)
		if !(len(keys) == n*pubKeySize) {
			b.Fatal("bad keys")
		}
		b.Run(fmt.Sprintf("fav/keys=%d", n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				X_fastAggregateVerify(keys, msg, sig)
			}
		})
		agg := aggb[:] // not among keys, so aggregate minus excluded is never the identity
		b.Run(fmt.Sprintf("excl/excluded=%d", n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				X_fastAggregateVerifyExcluding(agg, keys, msg, sig)
			}
		})
	}
}
