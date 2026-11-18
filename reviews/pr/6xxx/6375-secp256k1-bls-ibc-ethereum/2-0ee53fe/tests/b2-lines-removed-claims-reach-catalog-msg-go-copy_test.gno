// b2-lines-removed-claims-reach-catalog-msg-go-copy: the generated dispatcher
// copies msg into a fresh Go []byte (generated.go:1426-1428 for
// fastAggregateVerify, :1474-1476 for the Excluding twin; gonative.go
// Gno2GoValue: data := make([]byte, svl); copy(...)), outside the VM
// allocator, and the bls12381 rows (native_gas.go:196, :201) charge nothing for
// len(msg). One call therefore buys a Go allocation and memcpy of len(msg)
// bytes, the shape adr/acd01fa29_gno2go_slice_gas.md closed for capacity,
// on top of the HashToG2 time round 1 measured.
//
// Repro, from a plain clone of gnolang/gno at 0ee53fefc:
//   (cd gnovm && go build -o /tmp/gno ./cmd/gno)
//   mkdir -p /tmp/blsmem && cp <this file> /tmp/blsmem/mem_test.gno
//   printf 'module = "gno.land/p/demo/blsmem"\ngno = "0.9"\n' > /tmp/blsmem/gnomod.toml
//   R=$PWD; for t in TestAllocOnly TestAllocVerify TestSmallVerify; do
//     (cd /tmp/blsmem && GNOROOT=$R python3 -I -c 'import resource,subprocess,sys; subprocess.run(sys.argv[1:]); print("maxrss_kb", resource.getrusage(resource.RUSAGE_CHILDREN).ru_maxrss)' \
//       /tmp/gno test -v -run "^$t\$" .)
//   done
// Each test runs in its own process so ru_maxrss is that test's peak.
// TestAllocOnly holds the same 256 MiB msg live without the call: the control.
// Measured at fd7e75152 (0ee53fefc slice), two runs each:
//   TestAllocOnly    maxrss_kb 233056 / 239280   GAS 392899
//   TestSmallVerify  maxrss_kb 234128            GAS 5119566 (32-byte msg)
//   TestAllocVerify  maxrss_kb 495268 / 493160   GAS 5120542 (256 MiB msg)
// The call adds ~256 MiB of resident Go heap (the copy) for 976 more gas than
// the 32-byte call; the untouched Gno buffer itself is not resident.
package blsmem

import (
	"crypto/bls12381"
	"encoding/hex"
	"testing"
)

const (
	pkHex  = "820ad0f24a42c82129fef2a137f7b7c230c2aaffb78ffd82f6cbdcd2bfbf3560435a35c62d3ff66ad696b78f8c6c6c68"
	sigHex = "857cbc9c0976c88648089b0598cdc7b8c75a96f8e4f06ea0f0bbd0e2d5ac74d9dae7d4c479a7b573f0d7f7e3b875a6b9" +
		"039083a95ea6f3af6d432056337bc115ae50023fa289c320322941a3b07db7a710b438f643432a1a56f64b121931357e"
	big = 256 << 20
)

func TestAllocOnly(t *testing.T) {
	msg := make([]byte, big)
	msg[big-1] = 1
	if msg[big-1] != 1 {
		t.Fatal("unreachable")
	}
}

func TestAllocVerify(t *testing.T) {
	pk, _ := hex.DecodeString(pkHex)
	sig, _ := hex.DecodeString(sigHex)
	msg := make([]byte, big)
	msg[big-1] = 1
	println("ok", bls12381.FastAggregateVerify(pk, msg, sig))
}

func TestSmallVerify(t *testing.T) {
	pk, _ := hex.DecodeString(pkHex)
	sig, _ := hex.DecodeString(sigHex)
	msg := make([]byte, 32)
	println("ok", bls12381.FastAggregateVerify(pk, msg, sig))
}
