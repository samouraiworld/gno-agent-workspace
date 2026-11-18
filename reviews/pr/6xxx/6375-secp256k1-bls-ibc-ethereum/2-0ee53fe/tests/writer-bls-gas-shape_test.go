package stdlibs

import "testing"

// Both BLS natives do one key decoding and subgroup check per 48-byte key, so
// each must meter its key list per byte: pubkeys (param 0) for
// fastAggregateVerify, excluded (param 1) for fastAggregateVerifyExcluding.
func TestBLS12381GasMetersKeys(t *testing.T) {
	t.Parallel()
	want := map[string]int8{"fastAggregateVerify": 0, "fastAggregateVerifyExcluding": 1}
	for i := range calibratedNativeGas {
		e := &calibratedNativeGas[i]
		idx, ok := want[e.Fn]
		if e.Pkg != "crypto/bls12381" || !ok {
			continue
		}
		delete(want, e.Fn)
		if e.SlopeKind != SizeLenBytes || e.SlopeIdx != idx || e.Slope == 0 {
			t.Errorf("%s must meter param %d per byte; got Slope=%d SlopeIdx=%d SlopeKind=%d",
				e.Fn, idx, e.Slope, e.SlopeIdx, e.SlopeKind)
		}
	}
	if len(want) != 0 {
		t.Fatalf("crypto/bls12381 rows missing from calibratedNativeGas: %v", want)
	}
}
