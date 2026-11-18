// b4-lines-claims-tests-reach-catalog-slice-shapes: the sibling shapes of
// the []byte arguments crypto/bls12381's Gno tests pass (PR 6375 at
// 0ee53fefc). Every Gno test that slices an input at an offset expects
// false, so none shows the native reading a sub-slice at its offset rather
// than at its backing array's start. Each case here builds a valid input in
// a different shape and expects true.
//
// Repro, from a plain clone of gnolang/gno at 0ee53fefc:
//
//	mkdir -p /tmp/b4shapes && cp <this file> /tmp/b4shapes/shapes_test.gno
//	printf 'module = "gno.land/p/demo/b4shapes"\ngno = "0.9"\n' > /tmp/b4shapes/gnomod.toml
//	(cd gnovm && go build -o /tmp/gno ./cmd/gno) && GNOROOT=$PWD /tmp/gno test -v /tmp/b4shapes
package b4shapes

import (
	"crypto/bls12381"
	"encoding/hex"
	"testing"
)

// The three-signer vector and the four-member committee vector of
// bls12381_test.gno at 0ee53fefc, byte for byte.
const (
	vecPubKeys = "820ad0f24a42c82129fef2a137f7b7c230c2aaffb78ffd82f6cbdcd2bfbf3560435a35c62d3ff66ad696b78f8c6c6c68" +
		"b6a005f54655f0a670599f6b9de9edffc39c99799a482f62fa5e42bfe986a93ae2c52360a96139ed82c4c4d6672695e5" +
		"ac7563bb556a1687d1dfc2b5fa2d5891c0756ba960f3cfc62de348a5549ddf65121efdbc06050988704674d7820f4a3a"
	vecSig = "857cbc9c0976c88648089b0598cdc7b8c75a96f8e4f06ea0f0bbd0e2d5ac74d9dae7d4c479a7b573f0d7f7e3b875a6b9" +
		"039083a95ea6f3af6d432056337bc115ae50023fa289c320322941a3b07db7a710b438f643432a1a56f64b121931357e"
	vecMsg       = "gno.land bls12381 test vector"
	vecAggregate = "8b38c72176718cbaa7ba3fb02c062ce2a116cbd72791c1af8d4fc9ff84f0ba265dfac905e4cbc8120b5cd01ac95c3aaf"
	vecExcluded  = "b6a005f54655f0a670599f6b9de9edffc39c99799a482f62fa5e42bfe986a93ae2c52360a96139ed82c4c4d6672695e5" +
		"b900fd910bb898cd4fb2a689bd59751e450105efc3b2be03c6a9545b7221fa2f5320d284cc731dc49c93262459138da0"
	vecPartialSig = "951e05eff3af6074e89a657c9d36f9d01c0b463e33ea5cfb64c19d822b52055c816414e78bd833da3ad50364bb1e358b" +
		"0e322f576108346548a273637c42b7056e6d0777458215800ed23ff4d320ea3f5350df732af8d2b9d26eb6397292403f"
)

func dec(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

// window returns b placed at offset 7 of a larger buffer, with spare
// capacity after it, sliced back to len(b).
func window(b []byte) []byte {
	buf := make([]byte, 7+len(b)+13)
	for i := range buf {
		buf[i] = 0xa5
	}
	copy(buf[7:], b)
	return buf[7 : 7+len(b)]
}

func TestShapesFastAggregateVerify(t *testing.T) {
	pks, sig, msg := dec(vecPubKeys), dec(vecSig), []byte(vecMsg)

	if !bls12381.FastAggregateVerify(pks, msg, sig) {
		t.Fatal("control: fresh slices rejected")
	}
	if !bls12381.FastAggregateVerify(window(pks), window(msg), window(sig)) {
		t.Error("offset sub-slices rejected")
	}
	var pkArr [3 * bls12381.PubKeySize]byte
	var sigArr [bls12381.SignatureSize]byte
	copy(pkArr[:], pks)
	copy(sigArr[:], sig)
	if !bls12381.FastAggregateVerify(pkArr[:], msg, sigArr[:]) {
		t.Error("slices of fixed arrays rejected")
	}
	// One backing array carrying key, then signature, then message.
	all := append(append(append([]byte{0xff}, pks...), sig...), msg...)
	p0 := 1
	s0 := p0 + len(pks)
	m0 := s0 + len(sig)
	if !bls12381.FastAggregateVerify(all[p0:s0], all[m0:], all[s0:m0]) {
		t.Error("three windows of one backing array rejected")
	}
}

func TestShapesFastAggregateVerifyExcluding(t *testing.T) {
	agg, ex, sig, msg := dec(vecAggregate), dec(vecExcluded), dec(vecPartialSig), []byte(vecMsg)

	if !bls12381.FastAggregateVerifyExcluding(agg, ex, msg, sig) {
		t.Fatal("control: fresh slices rejected")
	}
	if !bls12381.FastAggregateVerifyExcluding(window(agg), window(ex), window(msg), window(sig)) {
		t.Error("offset sub-slices rejected")
	}
	var aggArr [bls12381.PubKeySize]byte
	copy(aggArr[:], agg)
	if !bls12381.FastAggregateVerifyExcluding(aggArr[:], ex, msg, sig) {
		t.Error("slice of fixed array rejected")
	}
}
