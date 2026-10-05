// Multibase claims of gnolang/gno PR 6271 at 2a84c1dcd: gas figures, caps
// checked before decoding, and one accepted string per CID per multibase.
//
// Repro, from a plain clone of gnolang/gno:
//   git fetch origin pull/6271/head && git checkout --detach 2a84c1dcdbd91a15a18923f56180b6367de97e21
//   go install ./gnovm/cmd/gno
//   cp <this file> examples/gno.land/p/omarsy/cid/v0/zz_claims_test.gno
//   cd examples/gno.land/p/omarsy/cid/v0 && gno test -v -run 'TestGas|TestCanonicalRoundTrip' .
//
// Result at 2a84c1dcd (GAS minus TestGasEmpty 592478; worst cases minus
// TestGasRepeatOnly 887643, which builds the input):
//   Parse base32 CIDv1 0.74M, base58btc CIDv1 1.26M, CIDv0 1.17M
//   Parse "z"+204: 8.65M; "k"+231: 7.77M; "b"+239: 2.78M; "f"+298: 3.11M
//   Parse "z"+205: about 2K over building the input, so rejected before decoding
//   TestCanonicalRoundTrip: z 615, k 696, b 286, B 283, f 450, F 450 accepted, 0 mismatches
//
// Mutations of multibase.gno, each run with
//   gno test -run 'TestMultibase|TestDigits|TestMaxCID|TestParse|TestCID|TestDecode' .
// all killed: drop `bits >= 5`, drop `acc != 0`, drop the hex parity check,
// drop the length cap, each cap raised by one (z, k, b) or two (f), each cap
// lowered (z 202, k 229, b 238: TestMaxCID), base32 and hex accepting the
// other case, base36 accepting upper case, base58 accepting 'l'.
package cid

import (
	"strings"
	"testing"
)

func TestGasEmpty(t *testing.T) {}

func TestGasParseB32(t *testing.T) {
	if _, err := Parse("bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi"); err != nil {
		t.Fatal(err)
	}
}

func TestGasParseB58v1(t *testing.T) {
	if _, err := Parse("zdj7WWeQ43G6JJvLWQWZpyHuAMq6uYWRjkBXFad11vE2LHhQ7"); err != nil {
		t.Fatal(err)
	}
}

func TestGasParseV0(t *testing.T) {
	if _, err := Parse("QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG"); err != nil {
		t.Fatal(err)
	}
}

func TestGasRepeatOnly(t *testing.T) { _ = "k" + strings.Repeat("z", 231) }

func TestGasWorstZ204(t *testing.T) {
	if _, err := Parse("z" + strings.Repeat("z", 204)); err == nil {
		t.Fatal("accepted")
	}
}

func TestGasWorstK231(t *testing.T) {
	if _, err := Parse("k" + strings.Repeat("z", 231)); err == nil {
		t.Fatal("accepted")
	}
}

func TestGasWorstB239(t *testing.T) {
	if _, err := Parse("b" + strings.Repeat("7", 239)); err == nil {
		t.Fatal("accepted")
	}
}

func TestGasWorstF298(t *testing.T) {
	if _, err := Parse("f" + strings.Repeat("f", 298)); err == nil {
		t.Fatal("accepted")
	}
}

func TestGasCapZ205(t *testing.T) {
	if _, err := Parse("z" + strings.Repeat("z", 205)); err == nil {
		t.Fatal("accepted")
	}
}

// Every string decodeMultibase accepts re-encodes to itself, so no two
// strings in one base decode to the same bytes.
func TestCanonicalRoundTrip(t *testing.T) {
	seed := uint32(7)
	next := func() uint32 {
		seed = seed*1664525 + 1013904223
		return seed >> 8
	}
	type tc struct {
		prefix   byte
		alphabet string
		max      int
	}
	for _, c := range []tc{
		{'z', base58Alphabet, 204}, {'k', base36Alphabet, 231},
		{'b', base32Lower, 239}, {'B', base32Upper, 239},
		{'f', base16Lower, 298}, {'F', base16Upper, 298},
	} {
		accepted, bad := 0, 0
		for n := 0; n <= c.max; n++ {
			for rep := 0; rep < 3; rep++ {
				buf := make([]byte, n+1)
				buf[0] = c.prefix
				for i := 1; i <= n; i++ {
					r := next()
					if r%4 == 0 {
						buf[i] = c.alphabet[0]
					} else {
						buf[i] = c.alphabet[int(r)%len(c.alphabet)]
					}
				}
				if rep == 2 && n > 0 {
					buf[n] = c.alphabet[0]
				}
				s := string(buf)
				b, err := decodeMultibase(s)
				if err != nil {
					continue
				}
				accepted++
				back, err := encodeMultibase(c.prefix, b)
				if err != nil || back != s {
					bad++
					t.Errorf("%s re-encodes as %s", s, back)
				}
			}
		}
		t.Logf("%s accepted=%d mismatches=%d", string(c.prefix), accepted, bad)
	}
}
