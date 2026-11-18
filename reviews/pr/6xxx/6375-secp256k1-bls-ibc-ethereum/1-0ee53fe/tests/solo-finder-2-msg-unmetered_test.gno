// solo-finder-2-msg-unmetered: the msg parameter of crypto/bls12381's two
// natives is neither bounded in the .gno wrapper nor sloped in native_gas.go,
// so gas per call is flat in len(msg) while HashToG2 SHA-256s all of it.
//
// Repro, from a plain clone of gnolang/gno at 0ee53fefc:
//   mkdir -p /tmp/blsmsg && cp <this file> /tmp/blsmsg/msg_test.gno
//   printf 'module = "gno.land/p/demo/blsmsg"\ngno = "0.9"\n' > /tmp/blsmsg/gnomod.toml
//   (cd gnovm && go build -o /tmp/gno ./cmd/gno) && GNOROOT=$PWD /tmp/gno test -v /tmp/blsmsg
// Per-call gas is (xN5 - xN1)/4. Measured at fd7e75152 (0ee53fefc slice):
//   32B   x1 5043024  x5 13566180  -> 2130789 per call
//   1MiB  x1 4712724  x5 13235880  (earlier run, same per-call delta)
//   16MiB x1 4712748  x5 13235904
//   64MiB x1 5038025  x5 13561659  -> 2130908 per call; wall 0.06s / 0.25s
// The merge base c32ca7b60 has no crypto/bls12381: the diff adds the gap.
package blsmsg

import (
	"crypto/bls12381"
	"encoding/hex"
	"testing"
)

// One valid compressed G1 key and one valid compressed G2 signature: the
// call decodes both, then hashes msg to G2 and runs the pairing whatever msg is.
const (
	pkHex  = "820ad0f24a42c82129fef2a137f7b7c230c2aaffb78ffd82f6cbdcd2bfbf3560435a35c62d3ff66ad696b78f8c6c6c68"
	sigHex = "857cbc9c0976c88648089b0598cdc7b8c75a96f8e4f06ea0f0bbd0e2d5ac74d9dae7d4c479a7b573f0d7f7e3b875a6b9" +
		"039083a95ea6f3af6d432056337bc115ae50023fa289c320322941a3b07db7a710b438f643432a1a56f64b121931357e"
)

func run(n, calls int) {
	pk, _ := hex.DecodeString(pkHex)
	sig, _ := hex.DecodeString(sigHex)
	msg := make([]byte, n)
	for i := 0; i < calls; i++ {
		bls12381.FastAggregateVerify(pk, msg, sig)
	}
}

func TestMsg32x1(t *testing.T)   { run(32, 1) }
func TestMsg32x5(t *testing.T)   { run(32, 5) }
func TestMsg1Mx1(t *testing.T)   { run(1<<20, 1) }
func TestMsg1Mx5(t *testing.T)   { run(1<<20, 5) }
func TestMsg16Mx1(t *testing.T)  { run(16<<20, 1) }
func TestMsg16Mx5(t *testing.T)  { run(16<<20, 5) }
func TestMsg64Mx1(t *testing.T)  { run(64<<20, 1) }
func TestMsg64Mx5(t *testing.T)  { run(64<<20, 5) }
