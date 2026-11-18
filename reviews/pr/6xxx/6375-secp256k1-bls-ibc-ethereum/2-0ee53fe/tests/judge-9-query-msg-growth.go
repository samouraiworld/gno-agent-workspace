package vm

// judge-9: crypto/bls12381.FastAggregateVerify reached through a free vm/qeval
// query. Asserts the per-call gas is flat in len(msg) (same out-of-gas call
// count at 32 B and 4 MiB) while wall time and the Go heap the dispatcher
// allocates outside the VM allocator grow with len(msg). Fails at 0ee53fefc.
//
// Repro, from a plain clone of gnolang/gno:
//   git fetch origin pull/6375/head:pr6375 && git checkout 0ee53fefc
//   cp judge-9-query-msg-growth.go gno.land/pkg/sdk/vm/judge9_query_msg_test.go
//   go test ./gno.land/pkg/sdk/vm -run 'TestJudge9QueryMsgGrowth' -count=1 -v -timeout 2m
//   rm gno.land/pkg/sdk/vm/judge9_query_msg_test.go

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

const judge9Realm = `package blsj

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
func Spin(n int, k int) int {
	pk, _ := hex.DecodeString(pkHex)
	sig, _ := hex.DecodeString(sigHex)
	msg := make([]byte, n)
	c := 0
	for i := 0; i < k; i++ {
		if !bls12381.FastAggregateVerify(pk, msg, sig) {
			c++
		}
	}
	return c
}

// Hold allocates the same msg and makes no native call.
func Hold(n int) int {
	pk, _ := hex.DecodeString(pkHex)
	sig, _ := hex.DecodeString(sigHex)
	msg := make([]byte, n)
	return len(pk) + len(sig) + len(msg)
}
`

func TestJudge9QueryMsgGrowth(t *testing.T) {
	env := setupTestEnv()
	ctx := env.vmk.MakeGnoTransactionStore(env.ctx)
	addr := crypto.AddressFromPreimage([]byte("addr1"))
	acc := env.acck.NewAccountWithAddress(ctx, addr)
	env.acck.SetAccount(ctx, acc)
	env.bankk.SetCoins(ctx, addr, std.MustParseCoins(ugnot.ValueString(20000000)))
	pkgPath := "gno.land/r/test/blsj"
	files := []*std.MemFile{
		{Name: "gnomod.toml", Body: gnolang.GenGnoModLatest(pkgPath)},
		{Name: "x.gno", Body: judge9Realm},
	}
	require.NoError(t, env.vmk.AddPackage(ctx, NewMsgAddPackage(addr, pkgPath, files)))
	env.vmk.CommitGnoTransactionStore(ctx)

	q := func(expr string) (time.Duration, uint64, error) {
		runtime.GC()
		var a, b runtime.MemStats
		runtime.ReadMemStats(&a)
		start := time.Now()
		_, err := env.vmk.QueryEval(env.ctx, pkgPath, expr)
		d := time.Since(start)
		runtime.ReadMemStats(&b)
		return d, b.TotalAlloc - a.TotalAlloc, err
	}

	// Gas: the out-of-gas call count of one free query, at two msg sizes.
	oog := map[int]string{}
	for _, n := range []int{32, 4 << 20} {
		for _, k := range []int{1300, 1500} {
			_, _, err := q(fmt.Sprintf("Spin(%d, %d)", n, k))
			oog[n] += fmt.Sprintf("k=%d err=%v; ", k, err)
		}
		t.Logf("msg=%d: %s", n, oog[n])
	}

	// Time and Go heap: 3 calls per query, against the same msg held with no call.
	const k = 3
	var base time.Duration
	for _, n := range []int{32, 1 << 20, 16 << 20, 64 << 20} {
		d, spinAlloc, err := q(fmt.Sprintf("Spin(%d, %d)", n, k))
		require.NoError(t, err)
		_, holdAlloc, err := q(fmt.Sprintf("Hold(%d)", n))
		require.NoError(t, err)
		if n == 32 {
			base = d
		}
		perCall := (float64(spinAlloc) - float64(holdAlloc)) / k
		t.Logf("msg=%9d B: wall-time ratio to 32 B %6.1fx; Go heap per call beyond Hold %.0f B (%.2f x len(msg))",
			n, float64(d)/float64(base), perCall, perCall/float64(n))
	}
	// IS:     the out-of-gas threshold is the same at 32 B and 4 MiB.
	// SHOULD: a 4 MiB msg runs out of gas at fewer calls than a 32 B one.
	if oog[32] == oog[4<<20] {
		t.Errorf("per-call gas is flat in len(msg) on the free query path")
	}
}
