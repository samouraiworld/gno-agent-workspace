// topic5-edge-semantics: the crypto/bls12381 natives of gnolang/gno#6375 at the
// edges of the IETF draft and Ethereum's spec: duplicate keys in pubkeys and
// in excluded, excluded holding the aggregate, the documented
// FastAggregateVerifyExcluding forgery, the identity signature under a
// non-identity key, stray-bit and non-canonical encodings, and Ethereum
// consensus-spec BLS vectors (keys, sign, fast_aggregate_verify).
//
// Every case logs its result and asserts what the IETF draft / Ethereum spec
// expects, so a FAIL line names a divergence. At 0ee53fefc every test passes
// except TestTopic5IdentitySumCommittee, which fails by design: a committee
// summing to the identity is refused by bls12381.go:86 although its signers'
// FastAggregateVerify passes.
//
// Run from a plain clone at 0ee53fefc (or the synthetic slice fd7e75152):
//
//	cp topic5-edge-semantics_test.go gnovm/stdlibs/crypto/bls12381/
//	go test -run 'TestTopic5' -v -timeout 2m ./gnovm/stdlibs/crypto/bls12381/
package bls12381

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"testing"

	bls "github.com/consensys/gnark-crypto/ecc/bls12-381"
	"github.com/consensys/gnark-crypto/ecc/bls12-381/fp"
)

func t5sk(hexs string) *big.Int {
	v, _ := new(big.Int).SetString(hexs, 16)
	return v
}

func t5pk(sk *big.Int) bls.G1Affine {
	g1 := g1Generator()
	var p bls.G1Affine
	p.ScalarMultiplication(&g1, sk)
	return p
}

func t5sig(sk *big.Int, msg []byte) bls.G2Affine {
	h, _ := bls.HashToG2(msg, ethDST)
	var s bls.G2Affine
	s.ScalarMultiplication(&h, sk)
	return s
}

func t5g1b(p bls.G1Affine) []byte { b := p.Bytes(); return b[:] }
func t5g2b(p bls.G2Affine) []byte { b := p.Bytes(); return b[:] }

func t5addG2(ps ...bls.G2Affine) bls.G2Affine {
	var j bls.G2Jac
	for i := range ps {
		j.AddMixed(&ps[i])
	}
	var a bls.G2Affine
	a.FromJacobian(&j)
	return a
}

func t5addG1(ps ...bls.G1Affine) bls.G1Affine {
	var j bls.G1Jac
	for i := range ps {
		j.AddMixed(&ps[i])
	}
	var a bls.G1Affine
	a.FromJacobian(&j)
	return a
}

func t5cat(bs ...[]byte) []byte {
	var out []byte
	for _, b := range bs {
		out = append(out, b...)
	}
	return out
}

func t5expect(t *testing.T, name string, got, want bool) {
	t.Helper()
	t.Logf("%-58s got=%v want=%v", name, got, want)
	if got != want {
		t.Errorf("DIVERGENCE %s: got %v want %v", name, got, want)
	}
}

func TestTopic5Duplicates(t *testing.T) {
	msg := []byte("signing root")
	sk1, sk2, sk3 := t5sk("1234567"), t5sk("89abcdef"), t5sk("7654321")
	pk1, pk2, pk3 := t5pk(sk1), t5pk(sk2), t5pk(sk3)
	s1, s2, s3 := t5sig(sk1, msg), t5sig(sk2, msg), t5sig(sk3, msg)

	// IETF FastAggregateVerify sums duplicates: [pk1, pk1] verifies 2*s1.
	t5expect(t, "FAV [pk1,pk1] sig 2*s1", X_fastAggregateVerify(t5cat(t5g1b(pk1), t5g1b(pk1)), msg, t5g2b(t5addG2(s1, s1))), true)
	t5expect(t, "FAV [pk1,pk1] sig s1", X_fastAggregateVerify(t5cat(t5g1b(pk1), t5g1b(pk1)), msg, t5g2b(s1)), false)
	t5expect(t, "FAV [pk1,pk2,pk1] sig 2*s1+s2", X_fastAggregateVerify(t5cat(t5g1b(pk1), t5g1b(pk2), t5g1b(pk1)), msg, t5g2b(t5addG2(s1, s2, s1))), true)

	// Committee [pk1, pk2, pk2, pk3] (pk2 sampled twice, as Ethereum's
	// sync committee allows), pk2 absent at both positions.
	agg := t5addG1(pk1, pk2, pk2, pk3)
	sig13 := t5g2b(t5addG2(s1, s3))
	t5expect(t, "Excl agg(1,2,2,3) excl [pk2,pk2] sig s1+s3", X_fastAggregateVerifyExcluding(t5g1b(agg), t5cat(t5g1b(pk2), t5g1b(pk2)), msg, sig13), true)
	t5expect(t, "Excl agg(1,2,2,3) excl [pk2] (deduped) sig s1+s3", X_fastAggregateVerifyExcluding(t5g1b(agg), t5g1b(pk2), msg, sig13), false)
	// pk2 absent at one position only: excluded [pk2], signers 1,2,3.
	t5expect(t, "Excl agg(1,2,2,3) excl [pk2] sig s1+s2+s3", X_fastAggregateVerifyExcluding(t5g1b(agg), t5g1b(pk2), msg, t5g2b(t5addG2(s1, s2, s3))), true)
	// Same member excluded twice in a committee holding it once: signers'
	// key is pk1+pk3-pk2; only a holder of sk2 can sign for it.
	agg123 := t5addG1(pk1, pk2, pk3)
	var negS2 bls.G2Affine
	negS2.Neg(&s2)
	t5expect(t, "Excl agg(1,2,3) excl [pk2,pk2] sig s1+s3-s2", X_fastAggregateVerifyExcluding(t5g1b(agg123), t5cat(t5g1b(pk2), t5g1b(pk2)), msg, t5g2b(t5addG2(s1, s3, negS2))), true)
	t5expect(t, "Excl agg(1,2,3) excl [pk2,pk2] sig s1+s3", X_fastAggregateVerifyExcluding(t5g1b(agg123), t5cat(t5g1b(pk2), t5g1b(pk2)), msg, t5g2b(t5addG2(s1, s3))), false)
	// The aggregate itself in excluded: identity, refused.
	t5expect(t, "Excl excl [agg] any sig", X_fastAggregateVerifyExcluding(t5g1b(agg123), t5g1b(agg123), msg, t5g2b(bls.G2Affine{})), false)
	// Excluding equals FAV over the signers.
	t5expect(t, "Excl agg(1,2,3) excl [] == FAV [1,2,3]", X_fastAggregateVerifyExcluding(t5g1b(agg123), nil, msg, t5g2b(t5addG2(s1, s2, s3))), X_fastAggregateVerify(t5cat(t5g1b(pk1), t5g1b(pk2), t5g1b(pk3)), msg, t5g2b(t5addG2(s1, s2, s3))))
}

// bls12381.go:67-69 says Excluding equals FastAggregateVerify over the
// signers' keys when aggregate is the committee's sum. A committee {pk, -pk}
// (one operator holding sk and -sk, each key with a valid PoP) sums to the
// identity: line 86 refuses the aggregate before excluded is applied.
func TestTopic5IdentitySumCommittee(t *testing.T) {
	msg := []byte("signing root")
	sk := t5sk("1234567")
	pk := t5pk(sk)
	var npk bls.G1Affine
	npk.Neg(&pk)
	agg := t5addG1(pk, npk)
	t.Logf("committee {pk,-pk} aggregate is identity: %v", agg.IsInfinity())
	s := t5g2b(t5sig(sk, msg))
	fav := X_fastAggregateVerify(t5g1b(pk), msg, s)
	excl := X_fastAggregateVerifyExcluding(t5g1b(agg), t5g1b(npk), msg, s)
	t.Logf("FAV over signers [pk]=%v, Excluding(agg={pk,-pk}, excl [-pk])=%v", fav, excl)
	t5expect(t, "Excluding equals FAV over signers (identity-sum committee)", excl, fav)
}

func TestTopic5DocumentedForgery(t *testing.T) {
	// bls12381.gno:54-58: excluded = e1 || (aggregate - pk_a - e1) makes the
	// signers' key exactly pk_a, so pk_a's own signature verifies.
	sk1, sk2, sk3, ska, ske := t5sk("1234567"), t5sk("89abcdef"), t5sk("7654321"), t5sk("abcdef01"), t5sk("1111")
	agg := t5addG1(t5pk(sk1), t5pk(sk2), t5pk(sk3))
	pka, e1 := t5pk(ska), t5pk(ske)
	var na, ne bls.G1Affine
	na.Neg(&pka)
	ne.Neg(&e1)
	e2 := t5addG1(agg, na, ne)
	m := []byte("a message the committee never signed")
	t5expect(t, "documented forgery holds as written", X_fastAggregateVerifyExcluding(t5g1b(agg), t5cat(t5g1b(e1), t5g1b(e2)), m, t5g2b(t5sig(ska, m))), true)
	// One excluded key suffices: aggregate - pk_a.
	e := t5addG1(agg, na)
	t5expect(t, "single-key variant of the forgery", X_fastAggregateVerifyExcluding(t5g1b(agg), t5g1b(e), m, t5g2b(t5sig(ska, m))), true)
}

func TestTopic5IdentitySignature(t *testing.T) {
	msg := []byte("signing root")
	pk := t5pk(t5sk("1234567"))
	var inf bls.G2Affine
	t5expect(t, "FAV [pk] identity sig", X_fastAggregateVerify(t5g1b(pk), msg, t5g2b(inf)), false)
	t5expect(t, "Excl agg=pk excl [] identity sig", X_fastAggregateVerifyExcluding(t5g1b(pk), nil, msg, t5g2b(inf)), false)
	t5expect(t, "FAV [pk] identity sig empty msg", X_fastAggregateVerify(t5g1b(pk), nil, t5g2b(inf)), false)
}

func TestTopic5Encodings(t *testing.T) {
	msg := []byte("signing root")
	sk := t5sk("1234567")
	pk := t5g1b(t5pk(sk))
	sig := t5g2b(t5sig(sk, msg))
	t5expect(t, "baseline valid", X_fastAggregateVerify(pk, msg, sig), true)

	setFlags := func(b []byte, flags byte) []byte {
		c := append([]byte{}, b...)
		c[0] = (c[0] & 0x1f) | flags
		return c
	}
	for _, f := range []byte{0x00, 0x20, 0x40, 0x60, 0xc0, 0xe0} {
		t5expect(t, "pubkey X with flags "+hex.EncodeToString([]byte{f}), X_fastAggregateVerify(setFlags(pk, f), msg, sig), false)
		t5expect(t, "sig X with flags "+hex.EncodeToString([]byte{f}), X_fastAggregateVerify(pk, msg, setFlags(sig, f)), false)
		t5expect(t, "Excl aggregate with flags "+hex.EncodeToString([]byte{f}), X_fastAggregateVerifyExcluding(setFlags(pk, f), nil, msg, sig), false)
	}
	// Infinity with stray bits, in every slot.
	inf48 := make([]byte, 48)
	inf48[0] = 0xc0
	inf96 := make([]byte, 96)
	inf96[0] = 0xc0
	for _, pos := range []int{0, 1, 47} {
		k := append([]byte{}, inf48...)
		if pos == 0 {
			k[0] |= 0x01
		} else {
			k[pos] = 0x01
		}
		t5expect(t, "pubkey infinity stray bit at byte "+string(rune('0'+pos%10)), X_fastAggregateVerify(t5cat(pk, k), msg, sig), false)
		t5expect(t, "excluded infinity stray bit", X_fastAggregateVerifyExcluding(pk, k, msg, sig), false)
	}
	for _, pos := range []int{0, 1, 48, 95} {
		s := append([]byte{}, inf96...)
		if pos == 0 {
			s[0] |= 0x01
		} else {
			s[pos] = 0x01
		}
		t5expect(t, "sig infinity stray bit", X_fastAggregateVerify(pk, msg, s), false)
	}
	// 0xe0 || 0: infinity + sort flag.
	e0 := append([]byte{}, inf96...)
	e0[0] = 0xe0
	t5expect(t, "sig 0xe0 zeros", X_fastAggregateVerify(pk, msg, e0), false)
	// Non-canonical X: X + p fits in 381 bits when X < 2^381 - p.
	var x big.Int
	xb := append([]byte{}, pk...)
	flags := xb[0] & 0xe0
	xb[0] &= 0x1f
	x.SetBytes(xb)
	x.Add(&x, fp.Modulus())
	if x.BitLen() <= 381 {
		nc := make([]byte, 48)
		x.FillBytes(nc)
		nc[0] |= flags
		t5expect(t, "pubkey non-canonical X+p", X_fastAggregateVerify(nc, msg, sig), false)
	} else {
		t.Logf("pubkey X+p exceeds 381 bits, case not constructible for this key")
	}
	// Sort flag flipped: decodes to -pk, a valid key, must not verify sig.
	flip := append([]byte{}, pk...)
	flip[0] ^= 0x20
	t5expect(t, "pubkey sort flag flipped (-pk)", X_fastAggregateVerify(flip, msg, sig), false)
}

// Ethereum consensus-spec BLS vectors (ethereum/bls12-381-tests): the three
// standard private keys, sign_case on 0x56*32, and fast_aggregate_verify_valid
// on 0xab*32.
func TestTopic5EthereumVectors(t *testing.T) {
	sks := []string{
		"263dbd792f5b1be47ed85f8938c0f29586af0d3ac7b977f21c278fe1462040e3",
		"47b8192d77bf871b62e87859d653922725724a5c031afeabc60bcef5ff665138",
		"328388aff0d4a5b7dc9205abd374e7e98f3cd9f3418edb4eafda5fb16473d216",
	}
	wantPks := []string{
		"a491d1b0ecd9bb917989f0e74f0dea0422eac4a873e5e2644f368dffb9a6e20fd6e10c1b77654d067c0618f6e5a7f79a",
		"b301803f8b5ac4a1133581fc676dfedc60d891dd5fa99028805e5ea5b08d3491af75d0707adab3b70c6a6a580217bf81",
		"b53d21a4cfd562c469cc81514d4ce5a6b577d8403d32a394dc265dd190b47fa9f829fdd7963afdf972e5e77854051f6f",
	}
	for i := range sks {
		got := hex.EncodeToString(t5g1b(t5pk(t5sk(sks[i]))))
		t5expect(t, "eth pubkey "+wantPks[i][:8], got == wantPks[i], true)
	}
	m56 := bytes.Repeat([]byte{0x56}, 32)
	wantSign := "882730e5d03f6b42c3abc26d3372625034e1d871b65a8a6b900a56dae22da98abbe1b68f85e49fe7652a55ec3d0591c20767677e33e5cbb1207315c41a9ac03be39c2e7668edc043d6cb1d9fd93033caa8a1c5b0e84bedaeb6c64972503a43eb"
	gotSign := hex.EncodeToString(t5g2b(t5sig(t5sk(sks[0]), m56)))
	t.Logf("gnark sign(sk0, 0x56*32) = %s", gotSign)
	t5expect(t, "eth sign_case sk0 0x56*32 matches", gotSign == wantSign, true)

	mab := bytes.Repeat([]byte{0xab}, 32)
	var pks []byte
	for _, p := range wantPks {
		b, _ := hex.DecodeString(p)
		pks = append(pks, b...)
	}
	favSig, _ := hex.DecodeString("9712c3edd73a209c742b8250759db12549b3eaf43b5ca61376d9f30e2747dbcf842d8b2ac0901d2a093713e20284a7670fcf6954e9ab93de991bb9b313e664785a075fc285806fa5224c82bde146561b446ccfc706a64b8579513cfc4ff1d930")
	var gotAgg []bls.G2Affine
	for _, s := range sks {
		gotAgg = append(gotAgg, t5sig(t5sk(s), mab))
	}
	t.Logf("gnark aggregate sign(sk0..2, 0xab*32) = %s", hex.EncodeToString(t5g2b(t5addG2(gotAgg...))))
	t5expect(t, "eth fast_aggregate_verify_valid (0xab*32)", X_fastAggregateVerify(pks, mab, favSig), true)
}
