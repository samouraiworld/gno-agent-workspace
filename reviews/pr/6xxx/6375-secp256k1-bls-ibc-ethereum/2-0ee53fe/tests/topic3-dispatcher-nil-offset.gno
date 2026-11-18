// Round 2, topic 3: the generated dispatcher with nil, empty, offset and
// array-backed []byte arguments, through the VM (Gno2GoValue) at 0ee53fefc.
// Repro from a plain clone of gnolang/gno at 0ee53fefc:
//   go install ./gnovm/cmd/gno && GNOROOT=$PWD gno run topic3-dispatcher-nil-offset.gno
// Measured: base true, offset53 true, cap-two-keys false, array true,
// sig-offset true, nilmsg true, emptymsg true, emptymsg-wrongsig false,
// excl-nil-vs-empty true.
package main

import (
	"crypto/bls12381"
	"encoding/hex"
)

const (
	vecPubKeys = "820ad0f24a42c82129fef2a137f7b7c230c2aaffb78ffd82f6cbdcd2bfbf3560435a35c62d3ff66ad696b78f8c6c6c68" +
		"b6a005f54655f0a670599f6b9de9edffc39c99799a482f62fa5e42bfe986a93ae2c52360a96139ed82c4c4d6672695e5" +
		"ac7563bb556a1687d1dfc2b5fa2d5891c0756ba960f3cfc62de348a5549ddf65121efdbc06050988704674d7820f4a3a"
	vecSig = "857cbc9c0976c88648089b0598cdc7b8c75a96f8e4f06ea0f0bbd0e2d5ac74d9dae7d4c479a7b573f0d7f7e3b875a6b9" +
		"039083a95ea6f3af6d432056337bc115ae50023fa289c320322941a3b07db7a710b438f643432a1a56f64b121931357e"
	emptySig = "85f10c089a4b2702c8217da6b23b74b34b4381180d54195c72466be065d78d536c2b715d625206267f4a0f8e2db359b9014ad8f2bf61f72c911dc6ef8e72e8f27c1cbc8c8f52aa1b0af6c1f33b53def397da200555b184e840cb3dd5de77b0a4"
	vecMsg = "gno.land bls12381 test vector"
)

func d(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

func main() {
	pks, sig := d(vecPubKeys), d(vecSig)
	println("base", bls12381.FastAggregateVerify(pks, []byte(vecMsg), sig))
	buf := append(make([]byte, 53), pks...)
	println("offset53", bls12381.FastAggregateVerify(buf[53:], []byte(vecMsg), sig))
	println("cap-two-keys", bls12381.FastAggregateVerify(pks[:96], []byte(vecMsg), sig))
	var arr [144]byte
	copy(arr[:], pks)
	println("array", bls12381.FastAggregateVerify(arr[:], []byte(vecMsg), sig))
	sbuf := append(append(make([]byte, 7), sig...), 1, 2, 3)
	println("sig-offset", bls12381.FastAggregateVerify(pks, []byte(vecMsg), sbuf[7:103]))
	es := d(emptySig)
	println("nilmsg", bls12381.FastAggregateVerify(pks, nil, es))
	println("emptymsg", bls12381.FastAggregateVerify(pks, []byte{}, es))
	println("emptymsg-wrongsig", bls12381.FastAggregateVerify(pks, nil, sig))
	println("excl-nil-vs-empty", bls12381.FastAggregateVerifyExcluding(pks[:48], nil, nil, es) == bls12381.FastAggregateVerifyExcluding(pks[:48], []byte{}, nil, es))
}
