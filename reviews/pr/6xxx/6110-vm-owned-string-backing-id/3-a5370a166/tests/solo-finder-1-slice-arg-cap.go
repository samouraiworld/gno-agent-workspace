// solo-finder-1: a []byte MsgCall argument now arrives with cap > len.
//
// Repro from a plain clone of gnolang/gno:
//   git fetch origin pull/6110/head && git checkout --detach a5370a166b459689a3febf222f3a5edd8739058e
//   cp <this file> gno.land/pkg/sdk/vm/zz_finder1_capcall_test.go
//   go test ./gno.land/pkg/sdk/vm -run TestFinder1SliceArgCapCall -v -count=1 -timeout 2m
// Head a5370a166:  arg "AQ==" -> cap(b) = (3 int); "AQ==" + 400 x "\n" -> cap(b) = (303 int)
// Merge base 6c49f5420 (same file, same command): (1 int) for both.
// Cause: convert.go:216 alloc.NewSliceFromData(bz) sets Maxcap = cap(bz), and
// base64.DecodeString returns a buffer of DecodedLen(len(arg)), not len(bz).
package vm

import (
	"strings"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/stretchr/testify/require"
)

func TestFinder1SliceArgCapCall(t *testing.T) {
	env := setupTestEnv()
	ctx := env.vmk.MakeGnoTransactionStore(env.ctx)
	addr := crypto.AddressFromPreimage([]byte("addr1"))
	acc := env.acck.NewAccountWithAddress(ctx, addr)
	env.acck.SetAccount(ctx, acc)
	env.bankk.SetCoins(ctx, addr, initialBalance)
	deployJSONTestPkg(t, env, ctx, addr, "gno.land/r/capr",
		"package capr\n\nfunc Cap(cur realm, b []byte) int { return cap(b) }\n")
	for _, arg := range []string{"AQ==", "AQ==" + strings.Repeat("\n", 400)} {
		ctx := env.vmk.MakeGnoTransactionStore(env.ctx)
		res, err := env.vmk.Call(ctx, NewMsgCall(addr, nil, "gno.land/r/capr", "Cap", []string{arg}))
		require.NoError(t, err)
		t.Logf("arg len %d -> cap(b) = %s", len(arg), res)
	}
}
