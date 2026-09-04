package main

// Asserts the premise broadcastWasFree rests on for the arm it keeps counted:
// a MsgEnablePackage that clears CheckTx and fails inside a block is charged
// the flat gas fee, so the debit taken before the broadcast stays.
// Passes at b51a78a8a; no test in the pull request reaches this arm.

/* Run: from a gno checkout:
gh pr checkout 6111 -R gnolang/gno && git checkout b51a78a8a
curl -fsSL -o contribs/gpao/zz_delivertxcharge_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6111-debit-max-spend-at-broadcast/1-b51a78a8a/tests/zz_delivertxcharge_test.go
cd contribs/gpao && go test -v -run 'TestDeliverTxFailureIsCharged' .
rm zz_delivertxcharge_test.go
*/

import (
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoclient"
	"github.com/gnolang/gno/gno.land/pkg/gnoland"
	"github.com/gnolang/gno/gno.land/pkg/integration"
	vm "github.com/gnolang/gno/gno.land/pkg/sdk/vm"
	"github.com/gnolang/gno/gnovm/pkg/gnoenv"
	gno "github.com/gnolang/gno/gnovm/pkg/gnolang"
	rpcclient "github.com/gnolang/gno/tm2/pkg/bft/rpc/client"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/log"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeliverTxFailureIsCharged pins the other half of the debit rule to the
// money.
//
// broadcastWasFree keeps the debit for a DeliverTx failure because "a DeliverTx
// failure ran in a block and was charged". TestBroadcastWasFree asserts the
// function returns false for that shape; nothing asserts a chain charges for
// it. The two node tests cover a return before the broadcast and a CheckTx
// rejection, so this arm is the one the rule turns on and the one no test
// reaches.
//
// The enable is sent with a GasWanted that covers the ante and not the
// package's own execution: CheckTx runs the ante alone and passes, DeliverTx
// runs the enable and runs out of gas inside a block.
func TestDeliverTxFailureIsCharged(t *testing.T) {
	const pkgPath = "gno.land/r/test/deliverfail"
	const gasFee = 1_000_000

	gnoroot := gnoenv.RootDir()
	cfg := integration.TestingMinimalNodeConfig(gnoroot)
	cfg.SkipGenesisSigVerification = true

	signer, err := gnoclient.SignerFromBip39(
		integration.DefaultAccount_Seed, cfg.Genesis.ChainID, "", 0, 0)
	require.NoError(t, err)
	info, err := signer.Info()
	require.NoError(t, err)
	who := info.GetAddress()

	ggs := cfg.Genesis.AppState.(gnoland.GnoGenesisState)
	ggs.Balances = []gnoland.Balance{{
		Address: who,
		Amount:  std.NewCoins(std.NewCoin("ugnot", 100_000_000_000)),
	}}
	ggs.VM.Params.CodeSubmissionPolicy = "inert"
	ggs.VM.Params.PkgApprovers = []crypto.Address{who}
	cfg.Genesis.AppState = ggs

	node, remote := integration.TestingInMemoryNode(t, log.NewNoopLogger(), cfg)
	defer node.Stop()

	rpc, err := rpcclient.NewHTTPClient(remote)
	require.NoError(t, err)
	client := gnoclient.Client{Signer: signer, RPCClient: rpc}

	mpkg := &std.MemPackage{
		Name: "deliverfail",
		Path: pkgPath,
		// Sorted by name, which the keeper requires.
		Files: []*std.MemFile{
			{Name: "deliverfail.gno", Body: "package deliverfail\n\nfunc F(cur realm) string { return \"ok\" }\n"},
			{Name: "gnomod.toml", Body: gno.GenGnoModLatest(pkgPath)},
		},
	}
	addTx := std.Tx{
		Msgs: []std.Msg{vm.MsgAddPackage{Creator: who, Package: mpkg}},
		Fee:  std.NewFee(20_000_000, std.MustParseCoin("1000000ugnot")),
	}
	signedAdd, err := client.SignTx(addTx, 0, 0)
	require.NoError(t, err)
	addRes, err := client.BroadcastTxCommit(signedAdd)
	require.NoError(t, err)
	require.True(t, addRes.CheckTx.IsOK(), "park checkTx: %v", addRes.CheckTx.Error)
	require.True(t, addRes.DeliverTx.IsOK(), "park deliverTx: %v", addRes.DeliverTx.Error)

	before, _, err := client.QueryBalance(who)
	require.NoError(t, err)

	enableTx := std.Tx{
		Msgs: []std.Msg{vm.MsgEnablePackage{
			Approver: who,
			PkgPath:  pkgPath,
			PkgHash:  vm.PackageContentHash(mpkg),
		}},
		// Enough for the ante, short of running the enable. 100_000 runs the
		// ante out of gas in CheckTx, which is the refunded arm; 5_000_000
		// lets the enable succeed.
		Fee: std.NewFee(1_000_000, std.MustParseCoin("1000000ugnot")),
	}
	signedEnable, err := client.SignTx(enableTx, 0, 0)
	require.NoError(t, err)
	res, err := client.BroadcastTxCommit(signedEnable)

	require.Error(t, err, "the enable has to fail, or the arm under test was never reached")
	require.NotNil(t, res)
	require.True(t, res.CheckTx.IsOK(),
		"the failure has to be DeliverTx's: a CheckTx rejection is the refunded arm, checkTx: %v", res.CheckTx.Error)
	require.True(t, res.DeliverTx.IsErr(), "deliverTx: %v", res.DeliverTx.Error)

	assert.False(t, broadcastWasFree(res),
		"a DeliverTx failure keeps the debit")

	after, _, err := client.QueryBalance(who)
	require.NoError(t, err)
	assert.Equal(t, before.AmountOf("ugnot")-gasFee, after.AmountOf("ugnot"),
		"the ante charged the flat gas fee for a transaction that ran in a block and failed")
}
