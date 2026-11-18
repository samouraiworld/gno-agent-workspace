// b4-lines-claims-tests-reach-catalog-vector-provenance: recomputes the
// hard-coded vectors of gnovm/stdlibs/crypto/bls12381/bls12381_test.gno
// (PR 6375 at 0ee53fefc) from the secret keys its comments name, to check
// the provenance those comments state:
//
//	bls12381_test.gno:9-10   three signers 0x1234567, 0x89abcdef, 0x7654321;
//	                         vecSig is their aggregate over vecMsg
//	bls12381_test.gno:74-75  committee of four (adds 0x5555); second and
//	                         fourth did not sign
//
// Repro, from a plain clone of gnolang/gno at 0ee53fefc:
//
//	cp <this file> gnovm/stdlibs/crypto/bls12381/b4_provenance_test.go
//	go test -count=1 -timeout 2m -run 'TestB4VectorProvenance' -v ./gnovm/stdlibs/crypto/bls12381/
//	go test -count=1 -race -timeout 2m -run 'TestFastAggregateVerify' ./gnovm/stdlibs/crypto/bls12381/
//
// The second line is the catalog's concurrency check: the package's own
// tests run t.Parallel subtests that share the package vars ethDST and
// negG1Gen.
package bls12381

import (
	"encoding/hex"
	"math/big"
	"testing"

	bls "github.com/consensys/gnark-crypto/ecc/bls12-381"
)

// Copied byte for byte from bls12381_test.gno at 0ee53fefc.
const (
	b4VecPubKeys = "820ad0f24a42c82129fef2a137f7b7c230c2aaffb78ffd82f6cbdcd2bfbf3560435a35c62d3ff66ad696b78f8c6c6c68" +
		"b6a005f54655f0a670599f6b9de9edffc39c99799a482f62fa5e42bfe986a93ae2c52360a96139ed82c4c4d6672695e5" +
		"ac7563bb556a1687d1dfc2b5fa2d5891c0756ba960f3cfc62de348a5549ddf65121efdbc06050988704674d7820f4a3a"
	b4VecSig = "857cbc9c0976c88648089b0598cdc7b8c75a96f8e4f06ea0f0bbd0e2d5ac74d9dae7d4c479a7b573f0d7f7e3b875a6b9" +
		"039083a95ea6f3af6d432056337bc115ae50023fa289c320322941a3b07db7a710b438f643432a1a56f64b121931357e"
	b4VecMsg       = "gno.land bls12381 test vector"
	b4VecAggregate = "8b38c72176718cbaa7ba3fb02c062ce2a116cbd72791c1af8d4fc9ff84f0ba265dfac905e4cbc8120b5cd01ac95c3aaf"
	b4VecExcluded  = "b6a005f54655f0a670599f6b9de9edffc39c99799a482f62fa5e42bfe986a93ae2c52360a96139ed82c4c4d6672695e5" +
		"b900fd910bb898cd4fb2a689bd59751e450105efc3b2be03c6a9545b7221fa2f5320d284cc731dc49c93262459138da0"
	b4VecPartialSig = "951e05eff3af6074e89a657c9d36f9d01c0b463e33ea5cfb64c19d822b52055c816414e78bd833da3ad50364bb1e358b" +
		"0e322f576108346548a273637c42b7056e6d0777458215800ed23ff4d320ea3f5350df732af8d2b9d26eb6397292403f"
)

func TestB4VectorProvenance(t *testing.T) {
	sks := []int64{0x1234567, 0x89abcdef, 0x7654321, 0x5555}
	g1 := g1Generator()
	pks := make([]bls.G1Affine, len(sks))
	for i, sk := range sks {
		pks[i].ScalarMultiplication(&g1, big.NewInt(sk))
	}
	enc1 := func(p bls.G1Affine) string { b := p.Bytes(); return hex.EncodeToString(b[:]) }

	var aggJ bls.G1Jac
	for i := range pks {
		aggJ.AddMixed(&pks[i])
	}
	var agg bls.G1Affine
	agg.FromJacobian(&aggJ)

	h, err := bls.HashToG2([]byte(b4VecMsg), ethDST)
	if err != nil {
		t.Fatal(err)
	}
	sigOf := func(sum int64) string {
		var s bls.G2Affine
		s.ScalarMultiplication(&h, big.NewInt(sum))
		b := s.Bytes()
		return hex.EncodeToString(b[:])
	}

	check := func(name, got, want string) {
		if got != want {
			t.Errorf("%s: MISMATCH\n got  %s\n want %s", name, got, want)
		} else {
			t.Logf("%s: matches", name)
		}
	}
	check("vecPubKeys = pk(0x1234567)||pk(0x89abcdef)||pk(0x7654321)", enc1(pks[0])+enc1(pks[1])+enc1(pks[2]), b4VecPubKeys)
	check("vecSig = (sk1+sk2+sk3)*H(vecMsg)", sigOf(sks[0]+sks[1]+sks[2]), b4VecSig)
	check("vecAggregate = sum of the four keys", enc1(agg), b4VecAggregate)
	check("vecExcluded = pk(0x89abcdef)||pk(0x5555)", enc1(pks[1])+enc1(pks[3]), b4VecExcluded)
	check("vecPartialSig = (sk1+sk3)*H(vecMsg)", sigOf(sks[0]+sks[2]), b4VecPartialSig)
}
