// Repro (from a plain clone of gnolang/gno):
//   git fetch origin pull/6110/head:pr6110 && git checkout c800a571ab179a07530dcf876142271afc64e40a
//   cp <this file> gno.land/pkg/sdk/vm/zz_b2catalog_test.go
//   cd gno.land/pkg/sdk/vm && go test -count=1 -run TestB2CatalogConstBackingDependsOnRestart -v .
// Observed at c800a571a:
//   long-running node: qq.B and rr.C share a backing = false, recount = 2096
//   restarted node:    qq.B and rr.C share a backing = true, recount = 1096
//   --- FAIL (expected: 2096, actual: 1096)
// Two realms each declaring `const X = pp.A` hold the const in a ConstExpr whose
// stringBacking comes from the pp PackageValue load that preprocessed them: one
// load per addpkg tx on the node that ran them, one shared load in
// PreprocessAllFilesAndSaveBlockNodes after a restart. The GC dedups by that
// pointer, so the same live values recount 1000 bytes apart across nodes.

package vm

import (
	"strings"
	"testing"

	"github.com/gnolang/gno/gnovm/pkg/gnolang"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/log"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/stretchr/testify/require"
)

// Whether two packages' ConstExpr copies of one imported string const share a
// backing depends on the node's cache history, not on VM execution alone.
func TestB2CatalogConstBackingDependsOnRestart(t *testing.T) {
	env := setupTestEnv()
	addr := crypto.AddressFromPreimage([]byte("addr1"))
	ctx0 := env.vmk.MakeGnoTransactionStore(env.ctx)
	acc := env.acck.NewAccountWithAddress(ctx0, addr)
	env.acck.SetAccount(ctx0, acc)
	env.bankk.SetCoins(ctx0, addr, initialBalance)

	add := func(path, body string) {
		ctx := env.vmk.MakeGnoTransactionStore(env.ctx) // one tx store per addpkg
		files := []*std.MemFile{
			{Name: "a.gno", Body: body},
			{Name: "gnomod.toml", Body: gnolang.GenGnoModLatest(path)},
		}
		require.NoError(t, env.vmk.AddPackage(ctx, NewMsgAddPackage(addr, path, files)))
		env.vmk.CommitGnoTransactionStore(ctx)
	}
	add("gno.land/r/pp", "package pp\n\nconst A = \""+strings.Repeat("a", 1000)+"\"\n")
	add("gno.land/r/qq", "package qq\n\nimport \"gno.land/r/pp\"\n\nconst B = pp.A\n")
	add("gno.land/r/rr", "package rr\n\nimport \"gno.land/r/pp\"\n\nconst C = pp.A\n")

	value := func(path string, name gnolang.Name) gnolang.StringValue {
		pn := env.vmk.gnoStore.GetBlockNode(gnolang.PackageNodeLocation(path)).(*gnolang.PackageNode)
		idx, ok := pn.GetLocalIndex(name)
		require.True(t, ok)
		return pn.StaticBlock.Block.Values[idx].V.(gnolang.StringValue)
	}
	backing := func(path string, name gnolang.Name) any { return value(path, name).B }
	// GC recount of the two live const values, as one GC run sees them.
	recount := func() int64 {
		alloc := gnolang.NewAllocator(1 << 30)
		var n int64
		vis := gnolang.GCVisitorFn(1, alloc, &n)
		vis(value("gno.land/r/qq", "B"))
		vis(value("gno.land/r/rr", "C"))
		_, b := alloc.Status()
		return b
	}
	before := backing("gno.land/r/qq", "B") == backing("gno.land/r/rr", "C")
	rcBefore := recount()
	t.Logf("long-running node: qq.B and rr.C share a backing = %v, recount = %d", before, rcBefore)

	env.vmk.gnoStore = nil // restart, as TestVMKeeperReinitialize does
	mcw := env.ctx.MultiStore().MultiCacheWrap()
	env.vmk.Initialize(log.NewNoopLogger(), mcw)
	mcw.MultiWrite()

	after := backing("gno.land/r/qq", "B") == backing("gno.land/r/rr", "C")
	rcAfter := recount()
	t.Logf("restarted node:    qq.B and rr.C share a backing = %v, recount = %d", after, rcAfter)
	require.Equal(t, rcBefore, rcAfter, "GC recount differs between a long-running and a restarted node")
	require.Equal(t, before, after, "GC dedup partition differs between a long-running and a restarted node")
}
