// gnogenesis verify reports success for a genesis tx with a zero fee, which a
// node on the default (closed) credit window rejects at InitChain.
//
// Repro from a plain clone of gnolang/gno:
//
//	git checkout 1f043ddb131e54e1410aebabd41d04ed49e37e1e
//	cp b1-claims-zero-fee-genesis-verify_test.go contribs/gnogenesis/internal/verify/b1_claims_zero_fee_genesis_test.go
//	(cd contribs/gnogenesis && go test -count=1 -run TestB1ClaimsVerifyAcceptsZeroFeeGenesisTx ./internal/verify/)
//	go test -count=1 -run TestAnteHandlerRejectsZeroFeeWhenCreditWindowDisabled ./tm2/pkg/sdk/auth/
//
// Head: the first test PASSES (verify accepts the zero-fee tx) and the second
// PASSES (the ante rejects a zero-fee tx in DeliverTx with the window closed;
// gno.land's default GenesisTxResultHandler, PanicOnFailingTxResultHandler,
// then panics InitChain). At the merge base 3cc494ec4 the first test FAILS:
// Tx.ValidateBasic rejected the empty fee, so verify refused the genesis.
package verify

import (
	"context"
	"testing"
	"time"

	"github.com/gnolang/gno/gno.land/pkg/gnoland"
	"github.com/gnolang/gno/tm2/pkg/bft/types"
	"github.com/gnolang/gno/tm2/pkg/commands"
	"github.com/gnolang/gno/tm2/pkg/crypto/ed25519"
	"github.com/gnolang/gno/tm2/pkg/crypto/mock"
	"github.com/gnolang/gno/tm2/pkg/sdk/bank"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/gnolang/gno/tm2/pkg/testutils"
	"github.com/stretchr/testify/require"
)

func TestB1ClaimsVerifyAcceptsZeroFeeGenesisTx(t *testing.T) {
	key := mock.GenPrivKey().PubKey()
	g := &types.GenesisDoc{
		GenesisTime:     time.Now(),
		ChainID:         "valid-chain-id",
		ConsensusParams: types.DefaultConsensusParams(),
		Validators: []types.GenesisValidator{
			{Address: key.Address(), PubKey: key, Power: 1, Name: "valid validator"},
		},
		AppState: gnoland.DefaultGenState(),
	}
	// The shipped default: credit window closed.
	require.Zero(t, g.ConsensusParams.Block.MaxGasCreditPerTx)

	signer := ed25519.GenPrivKey()
	tx := std.Tx{
		Msgs: []std.Msg{bank.MsgSend{
			FromAddress: signer.PubKey().Address(),
			ToAddress:   signer.PubKey().Address(),
			Amount:      std.NewCoins(std.NewCoin("ugnot", 10)),
		}},
		// A zero fee as it decodes off the wire or from JSON: the empty Coin.
		Fee:        std.Fee{GasWanted: 1000000, GasFee: std.Coin{}},
		Signatures: []std.Signature{{}},
	}
	t.Logf("tx.ValidateBasic() = %v", tx.ValidateBasic())

	appState := g.AppState.(gnoland.GnoGenesisState)
	appState.Txs = []gnoland.TxWithMetadata{{Tx: tx}}
	g.AppState = appState

	tempFile, cleanup := testutils.NewTestFile(t)
	t.Cleanup(cleanup)
	require.NoError(t, g.SaveAs(tempFile.Name()))

	cmd := NewVerifyCmd(commands.NewTestIO())
	err := cmd.ParseAndRun(context.Background(), []string{
		"--genesis-path", tempFile.Name(), "--skip-signature-check",
	})
	t.Logf("gnogenesis verify error = %v", err)
	require.NoError(t, err, "verify accepts a zero-fee genesis tx")
}
