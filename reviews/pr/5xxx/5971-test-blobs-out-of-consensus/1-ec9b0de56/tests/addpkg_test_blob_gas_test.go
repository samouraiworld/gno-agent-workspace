/* Run: from a gno checkout:
gh pr checkout 5971 -R gnolang/gno && git checkout ec9b0de56
curl -fsSL -o gno.land/pkg/sdk/vm/addpkg_test_blob_gas_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5971-test-blobs-out-of-consensus/1-ec9b0de56/tests/addpkg_test_blob_gas_test.go
go test -v -run 'TestAddPkgWithTestBlobGas' ./gno.land/pkg/sdk/vm/
rm gno.land/pkg/sdk/vm/addpkg_test_blob_gas_test.go
*/

// The #allbutprod blob is the only part of a package whose destination store
// AddMemPackage changes, so an addpkg that ships a _test.gno file is the only
// shape whose charged gas moves. At ec9b0de56 the deploy costs 21680911;
// routing the blob back to iavlStore costs 21904511.
package vm

import (
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoland/ugnot"
	"github.com/gnolang/gno/gnovm/pkg/gnolang"
	bft "github.com/gnolang/gno/tm2/pkg/bft/types"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/sdk"
	"github.com/gnolang/gno/tm2/pkg/sdk/auth"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/stretchr/testify/require"
)

// expectedAddPkgWithTestBlobGas pins DeliverTx gas for deploying a package that
// ships a test file. Deploying the same package without the test file writes no
// #allbutprod blob and is unaffected by the store move.
const expectedAddPkgWithTestBlobGas = int64(21680911)

func TestAddPkgWithTestBlobGas(t *testing.T) {
	env := setupTestEnv()
	// A non-genesis block: genesis uses an infinite gas meter.
	ctx := env.ctx.WithBlockHeader(&bft.Header{Height: int64(1)}).WithMode(sdk.RunTxModeDeliver)

	addr := crypto.AddressFromPreimage([]byte("test1"))
	acc := env.acck.NewAccountWithAddress(ctx, addr)
	env.acck.SetAccount(ctx, acc)
	env.bankk.SetCoins(ctx, addr, std.MustParseCoins(ugnot.ValueString(100000000)))

	const pkgPath = "gno.land/r/hellotest"
	files := []*std.MemFile{
		{Name: "gnomod.toml", Body: gnolang.GenGnoModLatest(pkgPath)},
		{Name: "hello.gno", Body: "package hellotest\n\nfunc Echo() string { return \"hello world\" }\n"},
		{Name: "hello_test.gno", Body: "package hellotest\n\nimport \"testing\"\n\nfunc TestEcho(t *testing.T) {\n\tif Echo() != \"hello world\" {\n\t\tt.Fatal(\"bad\")\n\t}\n}\n"},
	}
	msg := NewMsgAddPackage(addr, pkgPath, files)
	fee := std.NewFee(50000000, std.MustParseCoin(ugnot.ValueString(1)))
	tx := std.NewTx([]std.Msg{msg}, fee, []std.Signature{}, "")

	gctx := auth.SetGasMeter(ctx, tx.Fee.GasWanted)
	gctx, _ = gctx.CacheContext()
	gctx = env.vmh.vm.MakeGnoTransactionStore(gctx)

	res := env.vmh.Process(gctx, tx.GetMsgs()[0])
	require.True(t, res.IsOK(), "addpkg must succeed: %v", res)

	require.Equal(t, expectedAddPkgWithTestBlobGas, gctx.GasMeter().GasConsumed(),
		"addpkg gas for a package shipping a test file moved; the #allbutprod blob's "+
			"store write is metered by its destination store, so this is consensus-affecting")
}
