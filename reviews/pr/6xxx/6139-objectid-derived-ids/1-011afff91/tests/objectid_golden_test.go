// Repro, from a plain clone of github.com/gnolang/gno:
//
//	cp objectid_golden_test.go gnovm/pkg/gnolang/
//	go test ./gnovm/pkg/gnolang/ -run TestDeriveObjectIDCryptoAddrGoldenVectors -count=1
//
// Asserts the object-address derivation itself, not that it agrees with
// itself: the preimage layout and the resulting bech32 addresses are written
// out as literals. It passes at the reviewed head and fails on any edit to the
// preimage prefix, the separator, the PkgID rendering, the PkgID flag nibble
// or the hash, all of which move addresses that already hold state.
package gnolang

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gnolang/gno/tm2/pkg/crypto"
)

// TestDeriveObjectIDCryptoAddrGoldenVectors pins the derivation end to end.
//
// Once an object's derived address is written into an event, a registry or a
// balance, changing any input to it is a state-breaking change: the same
// object answers with a different address, and whatever sat at the old one is
// unreachable. The literals below are that contract. Regenerating them to make
// this test pass is the thing the test exists to stop.
func TestDeriveObjectIDCryptoAddrGoldenVectors(t *testing.T) {
	t.Parallel()

	// PkgID is the first input, and it is not a plain hash: PkgIDFromPkgPath
	// clears the top nibble and writes IsStdlib/IsImmutable/IsInternal flag
	// bits into it, with bit 0x10 still reserved. Claiming that bit, or
	// reclassifying a path, moves every object address in that realm — so the
	// PkgID string is pinned here too, ahead of the address it feeds.
	pkgIDs := map[string]string{
		"gno.land/r/demo/foo20":        "RID05A95D3B90BADA501E6136CC5D7F4EF98DAB5B80",
		"gno.land/p/demo/tokens/grc20": "RID4FBD19DF645A50B847D72ED838E39F596646CAE1",
		"chain/runtime":                "RIDC09C8277A76BF0C457FDF56BD592EDCDCF839A50",
	}
	for path, want := range pkgIDs {
		require.Equal(t, want, PkgIDFromPkgPath(path).String(),
			"PkgID of %q moved; every object address in that realm moved with it", path)
	}

	vectors := []struct {
		pkgPath  string
		newTime  uint64
		preimage string
		addr     string
	}{
		{
			pkgPath:  "gno.land/r/demo/foo20",
			newTime:  1,
			preimage: "objectid:RID05A95D3B90BADA501E6136CC5D7F4EF98DAB5B80:1",
			addr:     "g1un39xxhnkdlj46586xvclnsk5xaw04jlrnyape",
		},
		{
			pkgPath:  "gno.land/r/demo/foo20",
			newTime:  2,
			preimage: "objectid:RID05A95D3B90BADA501E6136CC5D7F4EF98DAB5B80:2",
			addr:     "g1wr733ezlqykpj87fvl3p5n63u9nnq6vyuhju2a",
		},
		{
			pkgPath:  "gno.land/p/demo/tokens/grc20",
			newTime:  7,
			preimage: "objectid:RID4FBD19DF645A50B847D72ED838E39F596646CAE1:7",
			addr:     "g1dmfquaplgaf0wakjhx5ftlnkdntua67j6dtyyj",
		},
		{
			// The clock is a uint64 and the preimage renders it in decimal,
			// so the top of the range is part of the contract.
			pkgPath:  "chain/runtime",
			newTime:  1 << 32,
			preimage: "objectid:RIDC09C8277A76BF0C457FDF56BD592EDCDCF839A50:4294967296",
			addr:     "g1x53ple3fg0nghfnx0vu2xddnmd99h5mn5xv3nk",
		},
		{
			pkgPath:  "chain/runtime",
			newTime:  ^uint64(0),
			preimage: "objectid:RIDC09C8277A76BF0C457FDF56BD592EDCDCF839A50:18446744073709551615",
			addr:     "g18de5twlh8fpwgll5925uyf3faul4xwmn7uluhp",
		},
	}

	for _, v := range vectors {
		t.Run(v.pkgPath+":"+strconv.FormatUint(v.newTime, 10), func(t *testing.T) {
			t.Parallel()

			oid := ObjectID{PkgID: PkgIDFromPkgPath(v.pkgPath), NewTime: v.newTime}

			// The address the code produces.
			require.Equal(t, v.addr, DeriveObjectIDCryptoAddr(oid).String())
			require.Equal(t, v.addr, oid.DerivePath())

			// The preimage it produced it from, spelled out independently of
			// the function under test, so a changed prefix or separator is
			// reported as the layout change it is rather than as a hash miss.
			require.Equal(t, v.addr, crypto.AddressFromPreimage([]byte(v.preimage)).String(),
				"preimage layout changed: expected %q", v.preimage)
		})
	}
}

// TestDeriveObjectIDCryptoAddrIsSeparateFromPkgAddresses pins the domain
// separation. A realm address and an object address are drawn from one
// 20-byte space, and chain.PackageAddress lets a realm pick the pkgPath half
// of that space freely, so the two preimage prefixes are what stop a realm
// from naming an object address it does not own.
func TestDeriveObjectIDCryptoAddrIsSeparateFromPkgAddresses(t *testing.T) {
	t.Parallel()

	oid := ObjectID{PkgID: PkgIDFromPkgPath("gno.land/r/demo/foo20"), NewTime: 1}
	object := DeriveObjectIDCryptoAddr(oid).String()

	// The pkgPath preimage that would hit the same address if the "objectid:"
	// prefix were ever dropped or made a suffix.
	forged := "objectid:" + PkgIDFromPkgPath("gno.land/r/demo/foo20").String() + ":1"
	require.NotEqual(t, object, DerivePkgCryptoAddr(forged).String(),
		"a pkgPath-derived address reached an object address")
	require.NotEqual(t, object, DerivePkgCryptoAddr("gno.land/r/demo/foo20").String())
	require.NotEqual(t, object, DeriveStorageDepositCryptoAddr("gno.land/r/demo/foo20").String())
}
