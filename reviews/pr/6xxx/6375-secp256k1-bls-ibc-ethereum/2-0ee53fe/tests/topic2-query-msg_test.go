// topic2-query-msg: the unsloped msg of crypto/bls12381 reached through a free
// vm/qeval query, which runs under maxGasQuery (3_000_000_000) and
// maxAllocQuery (1_500_000_000) instead of a fee-paying transaction's limits.
//
// Repro, from a plain clone of gnolang/gno at 0ee53fefc:
//   cp <this file> gno.land/pkg/sdk/vm/topic2_query_msg_test.go
//   cd gno.land && go test -run 'TestTopic2QueryMsg$' -v -timeout 2m ./pkg/sdk/vm/
//   cd gno.land && go test -run 'TestTopic2QueryPastTxAlloc$' -v -timeout 2m ./pkg/sdk/vm/
//   cd gno.land && go test -run 'TestTopic2TxBlockGas$' -v -timeout 2m ./pkg/sdk/vm/
//   cd gno.land && go test -run 'TestTopic2TxNearAllocCap$' -v -timeout 2m ./pkg/sdk/vm/
//
// TestTopic2QueryMsg: the same query, Burn(n, k), with k calls on one key, at
// msg 32 B and 64 MiB. The out-of-gas threshold on k is the same at both sizes
// (gas flat in msg); the wall-time ratio of the k=1300 query is the native work
// a free query buys per unit of gas, against the 32 B case.
// TestTopic2QueryPastTxAlloc: one call with a 768 MiB msg, past maxAllocTx
// (500_000_000), completes inside a query.
//
// Measured at fd7e75152 (the BLS slice of 0ee53fefc), growth ratios only:
//   QueryMsg:        k=1300 ok and k=1500 "out of gas in location: CPUCycles" at both 32 B and 64 MiB;
//                    wall-time ratio of the k=1300 query, 64MiB over 32B: 35.0x
//   TxBlockGas:      gas 2772488179 (32 B) vs 2782277240 (64 MiB), ratio 1.0035; wall-time ratio 34.9x
//   TxNearAllocCap:  k=4, 448MiB over 64MiB: gas ratio 3.7623 (the msg allocation), wall-time ratio 5.8x
//   QueryPastTxAlloc: msg=768MiB k=1 -> (1 int), PASS
package vm

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/gnolang/gno/gno.land/pkg/gnoland/ugnot"
	"github.com/gnolang/gno/gnovm/pkg/gnolang"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/gnolang/gno/tm2/pkg/store"
)

const topic2Body = `package blsburn

import (
	"crypto/bls12381"
	"encoding/hex"
)

// Burn calls FastAggregateVerify k times with one valid key, a valid G2
// signature encoding and an n-byte msg, so each call decodes, hashes msg to G2
// and runs the pairing; the result is false and the work is full.
func Burn(n, k int) int {
	pk, _ := hex.DecodeString("820ad0f24a42c82129fef2a137f7b7c230c2aaffb78ffd82f6cbdcd2bfbf3560435a35c62d3ff66ad696b78f8c6c6c68")
	sig, _ := hex.DecodeString("857cbc9c0976c88648089b0598cdc7b8c75a96f8e4f06ea0f0bbd0e2d5ac74d9dae7d4c479a7b573f0d7f7e3b875a6b9" +
		"039083a95ea6f3af6d432056337bc115ae50023fa289c320322941a3b07db7a710b438f643432a1a56f64b121931357e")
	msg := make([]byte, n)
	calls := 0
	for i := 0; i < k; i++ {
		bls12381.FastAggregateVerify(pk, msg, sig)
		calls++
	}
	return calls
}

// BurnTx is Burn behind a MsgCall.
func BurnTx(cur realm, n, k int) int { return Burn(n, k) }
`

func topic2Env(t *testing.T) (testEnv, string) {
	t.Helper()
	env := setupTestEnv()
	ctx := env.vmk.MakeGnoTransactionStore(env.ctx)
	addr := crypto.AddressFromPreimage([]byte("addr1"))
	acc := env.acck.NewAccountWithAddress(ctx, addr)
	env.acck.SetAccount(ctx, acc)
	env.bankk.SetCoins(ctx, addr, std.MustParseCoins(ugnot.ValueString(20000000)))
	pkgPath := "gno.land/r/test/blsburn"
	files := []*std.MemFile{
		{Name: "a.gno", Body: topic2Body},
		{Name: "gnomod.toml", Body: gnolang.GenGnoModLatest(pkgPath)},
	}
	require.NoError(t, env.vmk.AddPackage(ctx, NewMsgAddPackage(addr, pkgPath, files)))
	env.vmk.CommitGnoTransactionStore(ctx)
	return env, pkgPath
}

func topic2Query(t *testing.T, env testEnv, pkgPath string, n, k int) (string, error, time.Duration) {
	t.Helper()
	start := time.Now()
	res, err := env.vmk.QueryEval(env.ctx, pkgPath, fmt.Sprintf("Burn(%d, %d)", n, k))
	return res, err, time.Since(start)
}

func TestTopic2QueryMsg(t *testing.T) {
	env, pkgPath := topic2Env(t)

	res, err, small := topic2Query(t, env, pkgPath, 32, 1300)
	require.NoError(t, err)
	t.Logf("msg=32B k=1300: %s", res)
	_, err, _ = topic2Query(t, env, pkgPath, 32, 1500)
	require.ErrorContains(t, err, "out of gas")
	t.Logf("msg=32B k=1500: %v", err)

	res, err, big := topic2Query(t, env, pkgPath, 64<<20, 1300)
	require.NoError(t, err)
	t.Logf("msg=64MiB k=1300: %s", res)
	_, err, _ = topic2Query(t, env, pkgPath, 64<<20, 1500)
	require.ErrorContains(t, err, "out of gas")
	t.Logf("msg=64MiB k=1500: %v", err)

	t.Logf("wall-time ratio of the k=1300 query, 64MiB over 32B: %.1fx", float64(big)/float64(small))
}

// TestTopic2TxBlockGas: the same loop as one MsgCall whose gas meter is
// MaxBlockMaxGas (3_000_000_000), the most one transaction can buy in a block.
// Gas consumed is read off the meter at both msg sizes.
func TestTopic2TxBlockGas(t *testing.T) {
	env, pkgPath := topic2Env(t)
	addr := crypto.AddressFromPreimage([]byte("addr1"))
	run := func(n int) (int64, time.Duration) {
		ctx := env.vmk.MakeGnoTransactionStore(env.ctx.WithGasMeter(store.NewGasMeter(3_000_000_000)))
		start := time.Now()
		res, err := env.vmk.Call(ctx, NewMsgCall(addr, nil, pkgPath, "BurnTx", []string{fmt.Sprint(n), "1300"}))
		elapsed := time.Since(start)
		require.NoError(t, err)
		t.Logf("tx msg=%d k=1300: %q gas=%d", n, res, ctx.GasMeter().GasConsumed())
		return ctx.GasMeter().GasConsumed(), elapsed
	}
	gSmall, tSmall := run(32)
	gBig, tBig := run(64 << 20)
	t.Logf("gas ratio 64MiB/32B: %.4f, wall-time ratio: %.1fx", float64(gBig)/float64(gSmall), float64(tBig)/float64(tSmall))
}

// TestTopic2TxNearAllocCap: per-call growth from 64 MiB to 448 MiB, the
// largest power-of-two-ish msg under maxAllocTx, as a MsgCall of 4 calls each.
func TestTopic2TxNearAllocCap(t *testing.T) {
	env, pkgPath := topic2Env(t)
	addr := crypto.AddressFromPreimage([]byte("addr1"))
	run := func(n int) (int64, time.Duration) {
		ctx := env.vmk.MakeGnoTransactionStore(env.ctx.WithGasMeter(store.NewGasMeter(3_000_000_000)))
		start := time.Now()
		_, err := env.vmk.Call(ctx, NewMsgCall(addr, nil, pkgPath, "BurnTx", []string{fmt.Sprint(n), "4"}))
		elapsed := time.Since(start)
		require.NoError(t, err)
		return ctx.GasMeter().GasConsumed(), elapsed
	}
	g64, t64 := run(64 << 20)
	g448, t448 := run(448 << 20)
	t.Logf("k=4, 448MiB over 64MiB: gas ratio %.4f, wall-time ratio %.1fx", float64(g448)/float64(g64), float64(t448)/float64(t64))
}

func TestTopic2QueryPastTxAlloc(t *testing.T) {
	env, pkgPath := topic2Env(t)
	res, err, _ := topic2Query(t, env, pkgPath, 768<<20, 1)
	require.NoError(t, err)
	t.Logf("msg=768MiB k=1 (maxAllocTx is 500_000_000): %s", res)
}
