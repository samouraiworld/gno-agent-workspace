// b1-claims-xeon-scaling: the ADR (gnovm/adr/pr6375_bls12381_native.md:51-56)
// and native_gas.go:193-195 say the bls12381 rows were measured on an Apple M4
// Pro and "scaled by 2.3x toward the Xeon Silver 4114 the rows around it use".
// If that scaling is right, a 1-key FastAggregateVerify and a 2-pair
// bn254.pairingCheck (a neighbour row fitted on that Xeon) cost, on any x86
// host, roughly the ratio their two rows charge. This prints both ratios; only
// ratios are reported, never a gas rate derived from a timing.
//
// Rows, Base + Slope*bytes/1024, every reference a Xeon Silver 4114 draft row:
//   bls12381.fastAggregateVerify, 1 key (48 B):      2024000 + 1962000*48/1024    = 2115968
//   bls12381.fastAggregateVerify, 512 keys (24576 B): 2024000 + 1962000*24576/1024 = 49112000
//   bn254.pairingCheck, 2 pairs (384 B):              457574 + 1786890*384/1024    = 1127657
//   bn254.g1Mul (flat, 96 B):                         44465
//   keccak256.sum256, 16384 B:                        4323 + 23654*16384/1024      = 382787
// priced/measured below 1 means the bls row charges less per unit of work
// than the neighbour row it was scaled toward.
//
// Repro, from a plain clone of gnolang/gno at 0ee53fefc:
//   cp <this file> gnovm/stdlibs/crypto/bls12381/b1_claims_xeon_scaling_test.go
//   go test -run 'TestB1ClaimsXeonScaling$' -v -timeout 2m ./gnovm/stdlibs/crypto/bls12381/
package bls12381

import (
	"testing"

	bn "github.com/consensys/gnark-crypto/ecc/bn254"
	libbn254 "github.com/gnolang/gno/gnovm/stdlibs/crypto/bn254"
	libkeccak "github.com/gnolang/gno/gnovm/stdlibs/crypto/keccak256"
)

func keccak(b []byte) { _ = libkeccak.X_sum256(b) }

func bnPairInput(t *testing.T) []byte {
	t.Helper()
	_, _, g1, g2 := bn.Generators()
	var ng1 bn.G1Affine
	ng1.Neg(&g1)
	var out []byte
	for _, p := range []bn.G1Affine{g1, ng1} {
		x, y := p.X.Bytes(), p.Y.Bytes()
		out = append(out, x[:]...)
		out = append(out, y[:]...)
		// EIP-197 G2 order: x.imag, x.real, y.imag, y.real.
		xa1, xa0, ya1, ya0 := g2.X.A1.Bytes(), g2.X.A0.Bytes(), g2.Y.A1.Bytes(), g2.Y.A0.Bytes()
		out = append(out, xa1[:]...)
		out = append(out, xa0[:]...)
		out = append(out, ya1[:]...)
		out = append(out, ya0[:]...)
	}
	return out
}

func TestB1ClaimsXeonScaling(t *testing.T) {
	msg := make([]byte, 32)
	pubkeys, sig := signers(t, 1, msg)
	if !X_fastAggregateVerify(pubkeys, msg, sig) {
		t.Fatal("bls 1-key vector does not verify")
	}
	bnIn := bnPairInput(t)
	if got := libbn254.X_pairingCheck(bnIn); len(got) != 32 || got[31] != 1 {
		t.Fatalf("bn254 2-pair input does not pass: %x", got)
	}

	pk512, sig512 := signers(t, 512, msg)
	if !X_fastAggregateVerify(pk512, msg, sig512) {
		t.Fatal("bls 512-key vector does not verify")
	}
	mulIn := append(append([]byte{}, bnIn[:64]...), pubkeys[:32]...)
	if got := libbn254.X_g1Mul(mulIn); len(got) != 64 {
		t.Fatalf("bn254 g1Mul input rejected: %x", got)
	}
	kecIn := make([]byte, 16384)

	bench := func(f func()) float64 {
		return float64(testing.Benchmark(func(b *testing.B) {
			for range b.N {
				f()
			}
		}).NsPerOp())
	}
	type side struct {
		name string
		row  float64
		run  func()
	}
	blsSides := []side{
		{"bls 1 key", 2115968, func() { X_fastAggregateVerify(pubkeys, msg, sig) }},
		{"bls 512 keys", 49112000, func() { X_fastAggregateVerify(pk512, msg, sig512) }},
	}
	refs := []side{
		{"bn254.pairingCheck 2 pairs", 1127657, func() { libbn254.X_pairingCheck(bnIn) }},
		{"bn254.g1Mul", 44465, func() { libbn254.X_g1Mul(mulIn) }},
		{"keccak256.sum256 16 KiB", 382787, func() { keccak(kecIn) }},
	}
	for round := 0; round < 2; round++ {
		tb := make([]float64, len(blsSides))
		for i, s := range blsSides {
			tb[i] = bench(s.run)
		}
		for _, r := range refs {
			tr := bench(r.run)
			for i, s := range blsSides {
				measured, priced := tb[i]/tr, s.row/r.row
				t.Logf("round %d: %s / %s: work %.2fx, rows charge %.2fx, priced/measured %.2f",
					round, s.name, r.name, measured, priced, priced/measured)
			}
		}
	}
}
