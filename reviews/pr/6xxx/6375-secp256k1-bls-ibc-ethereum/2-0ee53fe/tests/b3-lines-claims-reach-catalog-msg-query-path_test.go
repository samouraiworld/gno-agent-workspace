// b3-lines-claims-reach-catalog-msg-query-path: extends round 1's msg Warning
// (native_gas.go:196) to its extremes. bls12381.go:123 hashes the whole msg
// with bls.HashToG2, and nothing bounds msg but the allocator: a transaction
// admits a msg up to maxAllocTx (500e6 bytes), qeval up to maxAllocQuery
// (1.5e9 bytes), and qeval charges nobody: its only bound is maxGasQuery
// (3e9), on the read-only query connection, which admits a bounded number of
// concurrent callers (tm2/pkg/bft/appconn/multi_app_conn.go:84-86).
//
// Measured at fd7e75152 (the 0ee53fefc BLS slice):
//
//	tx    msg=32B    per-call gas 2130918
//	tx    msg=256MiB per-call gas 2130918 (ratio 1.0000), per-call wall time 130x
//	tx    msg=1GiB   allocation limit exceeded
//	qeval msg=1GiB   2 calls succeed, wall time 563x the same query on 32 B
//
// Repro, from a plain clone of gnolang/gno at 0ee53fefc:
//
//	cp <this file> gno.land/pkg/sdk/vm/blsmsg_query_test.go
//	cd gno.land && go test -count=1 -timeout 2m -v -run 'TestBLSMsg' ./pkg/sdk/vm/
//
// Each test prints gas (tx path) and the wall-time growth ratio against the
// same call count on a 32-byte msg.
package vm

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/gnolang/gno/gno.land/pkg/gnoland/ugnot"
	"github.com/gnolang/gno/gnovm/pkg/gnolang"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/sdk"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/gnolang/gno/tm2/pkg/store"
)

const blsBurnPath = "gno.land/r/test/blsburn"

// One valid compressed G1 key and G2 signature (bls12381_test.gno's first
// vector key and aggregate): the call decodes both, hashes msg, pairs.
const blsBurnSrc = `package blsburn

import (
	"crypto/bls12381"
	"encoding/hex"
)

const (
	pkHex  = "820ad0f24a42c82129fef2a137f7b7c230c2aaffb78ffd82f6cbdcd2bfbf3560435a35c62d3ff66ad696b78f8c6c6c68"
	sigHex = "857cbc9c0976c88648089b0598cdc7b8c75a96f8e4f06ea0f0bbd0e2d5ac74d9dae7d4c479a7b573f0d7f7e3b875a6b9" +
		"039083a95ea6f3af6d432056337bc115ae50023fa289c320322941a3b07db7a710b438f643432a1a56f64b121931357e"
)

// Burn makes one msg of n bytes and verifies one signature over it calls times.
func Burn(n, calls int) int {
	pk, _ := hex.DecodeString(pkHex)
	sig, _ := hex.DecodeString(sigHex)
	msg := make([]byte, n)
	done := 0
	for i := 0; i < calls; i++ {
		bls12381.FastAggregateVerify(pk, msg, sig)
		done++
	}
	return done
}

func BurnTx(cur realm, n, calls int) int { return Burn(n, calls) }
`

func deployBLSBurn(t *testing.T) (testEnv, crypto.Address) {
	t.Helper()
	env := setupTestEnv()
	ctx := env.vmk.MakeGnoTransactionStore(env.ctx)
	addr := crypto.AddressFromPreimage([]byte("addr1"))
	acc := env.acck.NewAccountWithAddress(ctx, addr)
	env.acck.SetAccount(ctx, acc)
	env.bankk.SetCoins(ctx, addr, std.MustParseCoins(ugnot.ValueString(9000000)))
	files := []*std.MemFile{
		{Name: "blsburn.gno", Body: blsBurnSrc},
		{Name: "gnomod.toml", Body: gnolang.GenGnoModLatest(blsBurnPath)},
	}
	require.NoError(t, env.vmk.AddPackage(ctx, NewMsgAddPackage(addr, blsBurnPath, files)))
	env.vmk.CommitGnoTransactionStore(ctx)
	return env, addr
}

func burnTx(t *testing.T, env testEnv, addr crypto.Address, n, calls int) (gas int64, d time.Duration, err error) {
	t.Helper()
	ctx := env.vmk.MakeGnoTransactionStore(env.ctx)
	ctx = ctx.WithGasMeter(store.NewGasMeter(3_000_000_000))
	msg := NewMsgCall(addr, nil, blsBurnPath, "BurnTx", []string{fmt.Sprint(n), fmt.Sprint(calls)})
	start := time.Now()
	_, err = env.vmk.Call(ctx, msg)
	return ctx.GasMeter().GasConsumed(), time.Since(start), err
}

func burnQuery(t *testing.T, env testEnv, n, calls int) (string, time.Duration, error) {
	t.Helper()
	start := time.Now()
	res, err := env.vmk.QueryEval(sdk.Context(env.ctx), blsBurnPath, fmt.Sprintf("Burn(%d, %d)", n, calls))
	return res, time.Since(start), err
}

// Transaction path: same gas at 32 B and at 256 MiB, the most a transaction's
// allocator admits rounded down to a power of two.
func TestBLSMsgTxPath(t *testing.T) {
	env, addr := deployBLSBurn(t)
	// Per-call figures are (x8 - x1) / 7, so the msg allocation, charged
	// once per transaction, drops out.
	var perGas [2]int64
	var perDur [2]time.Duration
	for i, n := range []int{32, 256 << 20} {
		g1, d1, err := burnTx(t, env, addr, n, 1)
		require.NoError(t, err)
		g8, d8, err := burnTx(t, env, addr, n, 8)
		require.NoError(t, err)
		perGas[i], perDur[i] = (g8-g1)/7, (d8-d1)/7
		t.Logf("tx  msg=%d x1 gas=%d x8 gas=%d per-call gas=%d", n, g1, g8, perGas[i])
	}
	t.Logf("tx  per-call gas ratio 256MiB/32B %.4f, per-call wall-time growth %.0fx",
		float64(perGas[1])/float64(perGas[0]), float64(perDur[1])/float64(perDur[0]))
	// A msg past maxAllocTx is refused in a transaction.
	_, _, err := burnTx(t, env, addr, 1<<30, 1)
	t.Logf("tx  msg=1GiB err=%v", err)
	require.Error(t, err)
}

// Query path: qeval admits 1 GiB, which the transaction path refuses, and
// charges the caller nothing for it.
func TestBLSMsgQueryPath(t *testing.T) {
	env, _ := deployBLSBurn(t)
	const calls = 2
	r0, d0, err := burnQuery(t, env, 32, calls)
	require.NoError(t, err)
	r1, d1, err := burnQuery(t, env, 1<<30, calls)
	require.NoError(t, err)
	t.Logf("qeval msg=32B  calls=%d res=%s", calls, r0)
	t.Logf("qeval msg=1GiB calls=%d res=%s", calls, r1)
	t.Logf("qeval wall-time growth %.0fx at the same call count", float64(d1)/float64(d0))
}
