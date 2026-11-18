// b2-lines-removed-claims-reach-catalog-query-msg: the unpaid query path of the
// bls12381 msg hole. vm/qeval runs under maxGasQuery (3e9) and maxAllocQuery
// (1.5e9) with no fee (keeper.go:51-52, withQueryEvalMachine). The bls12381
// rows (native_gas.go:196, :201) charge nothing for len(msg), and the generated
// dispatcher (generated.go:1426-1428) copies msg into a fresh Go []byte that the
// VM allocator never sees (gonative.go Gno2GoValue: make([]byte, svl)). So one
// free query holding a 1 GiB msg can call the native ~1400 times, each call a
// 1 GiB Go allocation plus HashToG2 over 1 GiB, past the allocation budget.
//
// Repro, from a plain clone of gnolang/gno at 0ee53fefc:
//   cp <this file> gno.land/pkg/sdk/vm/zz_bls_query_test.go
//   go test -run 'TestBLSQueryMsg' -v -timeout 2m ./gno.land/pkg/sdk/vm/
// Reads: whether each query succeeds, the Go heap it allocated (TotalAlloc
// delta) against maxAllocQuery, and the wall-time growth from a 32-byte msg.
// Measured at fd7e75152 (0ee53fefc slice):
//   msg=32 calls=1300 err=<nil>
//   msg=32 calls=1500 err=out of gas in location: CPUCycles
//   msg=1GiB calls=3 err=<nil> goHeapAlloc=4295503920 bytes (maxAllocQuery=1500000000)
//   wall-time growth 1GiB/32B at 3 calls: 506x
package vm

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/gnolang/gno/gno.land/pkg/gnoland/ugnot"
	"github.com/gnolang/gno/gnovm/pkg/gnolang"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/stretchr/testify/require"
)

const blsQueryRealm = `package blsq

import (
	"crypto/bls12381"
	"encoding/hex"
)

const (
	pkHex  = "820ad0f24a42c82129fef2a137f7b7c230c2aaffb78ffd82f6cbdcd2bfbf3560435a35c62d3ff66ad696b78f8c6c6c68"
	sigHex = "857cbc9c0976c88648089b0598cdc7b8c75a96f8e4f06ea0f0bbd0e2d5ac74d9dae7d4c479a7b573f0d7f7e3b875a6b9" +
		"039083a95ea6f3af6d432056337bc115ae50023fa289c320322941a3b07db7a710b438f643432a1a56f64b121931357e"
)

// Spin holds one msg of n bytes and calls the native k times on it.
func Spin(n int, k int) string {
	pk, _ := hex.DecodeString(pkHex)
	sig, _ := hex.DecodeString(sigHex)
	msg := make([]byte, n)
	c := 0
	for i := 0; i < k; i++ {
		if !bls12381.FastAggregateVerify(pk, msg, sig) {
			c++
		}
	}
	if c != k {
		return "unexpected"
	}
	return "done"
}
`

func TestBLSQueryMsg(t *testing.T) {
	env := setupTestEnv()
	ctx := env.vmk.MakeGnoTransactionStore(env.ctx)
	addr := crypto.AddressFromPreimage([]byte("addr1"))
	acc := env.acck.NewAccountWithAddress(ctx, addr)
	env.acck.SetAccount(ctx, acc)
	env.bankk.SetCoins(ctx, addr, std.MustParseCoins(ugnot.ValueString(20000000)))
	pkgPath := "gno.land/r/test/blsq"
	files := []*std.MemFile{
		{Name: "gnomod.toml", Body: gnolang.GenGnoModLatest(pkgPath)},
		{Name: "x.gno", Body: blsQueryRealm},
	}
	require.NoError(t, env.vmk.AddPackage(ctx, NewMsgAddPackage(addr, pkgPath, files)))
	env.vmk.CommitGnoTransactionStore(ctx)

	run := func(n, k int) (time.Duration, uint64, error) {
		runtime.GC()
		var a, b runtime.MemStats
		runtime.ReadMemStats(&a)
		start := time.Now()
		_, err := env.vmk.QueryEval(env.ctx, pkgPath, fmt.Sprintf("Spin(%d, %d)", n, k))
		d := time.Since(start)
		runtime.ReadMemStats(&b)
		return d, b.TotalAlloc - a.TotalAlloc, err
	}
	// How many calls one free query buys at a 32-byte msg.
	for _, k := range []int{1300, 1500} {
		_, _, err := run(32, k)
		t.Logf("msg=32 calls=%d err=%v", k, err)
	}
	// Same call count, 32 B against 1 GiB.
	small, smallAlloc, err := run(32, 3)
	require.NoError(t, err)
	big, bigAlloc, err := run(1<<30, 3)
	t.Logf("msg=1GiB calls=3 err=%v goHeapAlloc=%d bytes (maxAllocQuery=%d) small goHeapAlloc=%d", err, bigAlloc, int64(maxAllocQuery), smallAlloc)
	t.Logf("wall-time growth 1GiB/32B at 3 calls: %.0fx", float64(big)/float64(small))
}
