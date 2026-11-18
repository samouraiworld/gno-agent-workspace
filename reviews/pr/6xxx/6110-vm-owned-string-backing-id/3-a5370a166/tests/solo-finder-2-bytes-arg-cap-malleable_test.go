// Repro (from a plain gno clone; run at a5370a166 and at merge base 6c49f5420):
//   cp solo-finder-2-bytes-arg-cap-malleable_test.go gno.land/pkg/sdk/vm/zz_sf2cap_test.go
//   go test ./gno.land/pkg/sdk/vm -run TestSF2BytesArgCapMalleable -count=1 -v -timeout 2m
// Claim: a []byte MsgCall argument now gets cap = base64.DecodedLen(len(arg)),
// not len(decoded), so one byte string arrives with a cap the encoding's
// ignored \r\n padding chooses, visible to the realm and persisted whole.
package vm

import (
	"testing"

	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/stretchr/testify/require"
)

func TestSF2BytesArgCapMalleable(t *testing.T) {
	env := setupTestEnv()
	addr := crypto.AddressFromPreimage([]byte("addr1"))
	ctx := env.vmk.MakeGnoTransactionStore(env.ctx)
	acc := env.acck.NewAccountWithAddress(ctx, addr)
	env.acck.SetAccount(ctx, acc)
	env.bankk.SetCoins(ctx, addr, initialBalance)
	deployJSONTestPkg(t, env, env.vmk.MakeGnoTransactionStore(env.ctx), addr, "gno.land/r/cc",
		"package cc\n\nvar Stored []byte\n\nfunc Probe(cur realm, b []byte) (int, int) { Stored = b; return len(b), cap(b) }\n")
	for _, arg := range []string{"AAAA", "AAAA\n\n\n\n", "AAAA" + string(make([]byte, 0)) + "\r\n\r\n\r\n\r\n\r\n\r\n\r\n\r\n"} {
		tx := env.vmk.MakeGnoTransactionStore(env.ctx)
		res, err := env.vmk.Call(tx, NewMsgCall(addr, nil, "gno.land/r/cc", "Probe", []string{arg}))
		require.NoError(t, err)
		t.Logf("arg=%q -> %q", arg, res)
	}
}
