// PR 6206 narrows the address check in gno.land/pkg/gnoweb/handler_http.go from
// tm2/pkg/bech32.Decode to tm2/pkg/crypto.AddressFromBech32. No test the PR adds
// feeds CreateUsernameFromBech32 a string the two disagree on, so reverting the
// body leaves every test in gno.land/pkg/gnoweb green. The three cases below are
// the disagreement: they pass on head 876762bd and fail on the reverted body.
//
// Repro from a plain clone (go1.25.9, the version go.mod pins):
//
//	git clone https://github.com/gnolang/gno.git && cd gno
//	git fetch origin pull/6206/head && git checkout 876762bdf2ea6635b27e2b0a42f26f9fdc54af24
//	cp <this file> gno.land/pkg/gnoweb/b1_bech32_username_test.go
//	go test ./gno.land/pkg/gnoweb/ -run TestB1CreateUsernameFromBech32 -count=1
//	# ok  github.com/gnolang/gno/gno.land/pkg/gnoweb  0.029s
//
// Now revert the fix and run the whole package. Only this file fails; every test
// the PR adds stays green, which is what makes the change unpinned:
//
//	python3 - gno.land/pkg/gnoweb/handler_http.go <<'PY'
//	import sys
//	p = sys.argv[1]; s = open(p).read()
//	new = '''func CreateUsernameFromBech32(username string) string {
//		if _, err := crypto.AddressFromBech32(username); err != nil {
//			return username
//		}
//
//		return username[:4] + "..." + username[len(username)-4:]
//	}'''
//	old = '''func CreateUsernameFromBech32(username string) string {
//		_, _, err := bech32.Decode(username)
//		if err == nil {
//			username = username[:4] + "..." + username[len(username)-4:]
//		}
//
//		return username
//	}'''
//	s = s.replace(new, old).replace('"github.com/gnolang/gno/tm2/pkg/crypto"',
//		'"github.com/gnolang/gno/tm2/pkg/bech32"\n\t"github.com/gnolang/gno/tm2/pkg/crypto"', 1)
//	open(p, "w").write(s)
//	PY
//	go test ./gno.land/pkg/gnoweb/ -count=1
//	# --- FAIL: TestB1CreateUsernameFromBech32_OnlyGnoAddressesAreTruncated (0.00s)
//	# FAIL  github.com/gnolang/gno/gno.land/pkg/gnoweb  7.926s
package gnoweb_test

import (
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb"
	"github.com/gnolang/gno/tm2/pkg/bech32"
	"github.com/stretchr/testify/require"
)

func TestB1CreateUsernameFromBech32_OnlyGnoAddressesAreTruncated(t *testing.T) {
	t.Parallel()

	gnoAddr, err := bech32.Encode("g", make([]byte, 20))
	require.NoError(t, err)
	foreignHRP, err := bech32.Encode("cosmos", make([]byte, 20))
	require.NoError(t, err)
	wrongLen, err := bech32.Encode("g", make([]byte, 32))
	require.NoError(t, err)

	require.Equal(t, gnoAddr[:4]+"..."+gnoAddr[len(gnoAddr)-4:],
		gnoweb.CreateUsernameFromBech32(gnoAddr), "a real gno address still truncates")
	require.Equal(t, foreignHRP, gnoweb.CreateUsernameFromBech32(foreignHRP),
		"a foreign-HRP bech32 string is a name, not an address")
	require.Equal(t, wrongLen, gnoweb.CreateUsernameFromBech32(wrongLen),
		"a g-prefixed bech32 string of the wrong length is not an address")
}
