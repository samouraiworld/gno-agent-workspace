package gnoland

// Repro: does each TestSponsorshipUseCaseFitsWindow case, admitted by CheckTx at
// its pinned window, also succeed at DeliverTx with the same gas, and does the
// realm actually pay at delivery?
//
// From a plain clone of gnolang/gno at ba63646a512bdea71d0b1603caa64aec1cbe331b:
//
//	cp b2-lines-removed-reach-catalog-usecase-deliver_test.go gno.land/pkg/gnoland/b2_usecase_deliver_test.go
//	cd gno.land && go test -count=1 -run TestB2UseCaseDeliverMatchesCheck -v ./pkg/gnoland/

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/gnolang/gno/gno.land/pkg/sdk/vm"
	"github.com/gnolang/gno/gnovm/pkg/gnoenv"
	"github.com/gnolang/gno/gnovm/pkg/gnolang"
	"github.com/gnolang/gno/tm2/pkg/amino"
	abci "github.com/gnolang/gno/tm2/pkg/bft/abci/types"
	bft "github.com/gnolang/gno/tm2/pkg/bft/types"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/crypto/ed25519"
	"github.com/gnolang/gno/tm2/pkg/db/memdb"
	"github.com/gnolang/gno/tm2/pkg/sdk"
	"github.com/gnolang/gno/tm2/pkg/std"
)

func TestB2UseCaseDeliverMatchesCheck(t *testing.T) {
	cases := []struct {
		name    string
		src     string
		withAVL bool
		withGRC bool
		window  int64
	}{
		{"paygas-only", paygasOnlyRealm, false, false, 1_000_000},
		{"map", mapLedgerRealm, false, false, 2_000_000},
		{"avl", paymasterRealm, true, false, 7_000_000},
		{"grc20", grc20Realm, true, true, 6_000_000},
		{"grc20-approve", grc20ApproveInTxRealm, true, true, 6_000_000},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts := TestAppOptions(memdb.NewMemDB())
			opts.AllowZeroFeeTxs = true
			app, err := NewAppWithOptions(opts)
			require.NoError(t, err)
			bapp := app.(*sdk.BaseApp)

			deployer := crypto.AddressFromPreimage([]byte("uc_deployer"))
			priv := ed25519.GenPrivKey()
			addr := priv.PubKey().Address()
			realmAddr := gnolang.DerivePkgCryptoAddr("gno.land/r/uc/paymaster")

			st := DefaultGenState()
			st.Balances = []Balance{
				{Address: deployer, Amount: []std.Coin{{Amount: 1e15, Denom: "ugnot"}}},
				{Address: addr, Amount: []std.Coin{{Amount: 1e12, Denom: "ugnot"}}},
				{Address: realmAddr, Amount: []std.Coin{{Amount: 1e14, Denom: "ugnot"}}},
			}
			ex := filepath.Join(gnoenv.RootDir(), "examples", "gno.land", "p", "nt")
			if tc.withAVL {
				st.Txs = append(st.Txs,
					TxWithMetadata{Tx: deployExampleTx(t, deployer, filepath.Join(ex, "ufmt", "v0"), "gno.land/p/nt/ufmt/v0")},
					TxWithMetadata{Tx: deployExampleTx(t, deployer, filepath.Join(ex, "avl", "v0"), "gno.land/p/nt/avl/v0")},
				)
			}
			if tc.withGRC {
				st.Txs = append(st.Txs,
					TxWithMetadata{Tx: deployExampleTx(t, deployer, filepath.Join(ex, "cford32", "v0"), "gno.land/p/nt/cford32/v0")},
					TxWithMetadata{Tx: deployExampleTx(t, deployer, filepath.Join(ex, "seqid", "v0"), "gno.land/p/nt/seqid/v0")},
					TxWithMetadata{Tx: deployExampleTx(t, deployer, filepath.Join(ex, "grc20", "v0"), "gno.land/p/nt/grc20/v0")},
				)
			}
			st.Txs = append(st.Txs, TxWithMetadata{Tx: mustDeployTx(deployer, "gno.land/r/uc/paymaster", "paymaster.gno", tc.src)})
			if tc.withGRC {
				st.Txs = append(st.Txs, TxWithMetadata{Tx: std.Tx{
					Msgs:       []std.Msg{vm.NewMsgCall(deployer, nil, "gno.land/r/uc/paymaster", "Setup", []string{addr.String()})},
					Fee:        std.Fee{GasWanted: 5e8, GasFee: std.Coin{Denom: "ugnot", Amount: 1e6}},
					Signatures: []std.Signature{{}},
				}})
			}
			bp := defaultBlockParams()
			bp.MaxGasCreditPerTx = tc.window
			r := bapp.InitChain(abci.RequestInitChain{Time: time.Now(), ChainID: "uc", ConsensusParams: &abci.ConsensusParams{Block: bp}, AppState: st})
			require.True(t, r.IsOK(), "InitChain: %v", r)
			bapp.BeginBlock(abci.RequestBeginBlock{Header: &bft.Header{ChainID: "uc", Height: 1}})
			bapp.EndBlock(abci.RequestEndBlock{})
			bapp.Commit()

			q := bapp.Query(abci.RequestQuery{Path: "auth/accounts/" + addr.String()})
			require.True(t, q.IsOK())
			var acc GnoAccount
			require.NoError(t, amino.UnmarshalJSON(q.Data, &acc))

			tx := std.Tx{
				Msgs: []std.Msg{vm.NewMsgCall(addr, nil, "gno.land/r/uc/paymaster", "Sponsor", []string{"ab"})},
				Fee:  std.Fee{GasWanted: tc.window, GasFee: std.Coin{Denom: "ugnot", Amount: 0}},
			}
			sb, err := tx.GetSignBytes("uc", acc.GetAccountNumber(), 0)
			require.NoError(t, err)
			sig, err := priv.Sign(sb)
			require.NoError(t, err)
			tx.Signatures = []std.Signature{{PubKey: priv.PubKey(), Signature: sig}}
			bz := amino.MustMarshal(tx)

			chk := bapp.CheckTx(abci.RequestCheckTx{Tx: bz})
			require.True(t, chk.IsOK(), "check: %v | %.200s", chk.Error, chk.Log)

			balQ := func() string {
				q := bapp.Query(abci.RequestQuery{Path: "bank/balances/" + realmAddr.String()})
				return string(q.Data)
			}
			before := balQ()
			bapp.BeginBlock(abci.RequestBeginBlock{Header: &bft.Header{ChainID: "uc", Height: 2, Time: time.Now()}})
			del := bapp.DeliverTx(abci.RequestDeliverTx{Tx: bz})
			bapp.EndBlock(abci.RequestEndBlock{})
			bapp.Commit()
			after := balQ()
			t.Logf("%-14s check=%d deliver=%d ok=%v realm %s -> %s err=%v %.200s", tc.name, chk.GasUsed, del.GasUsed, del.IsOK(), before, after, del.Error, del.Log)
			require.True(t, del.IsOK(), "deliver failed at a window CheckTx admitted")
			require.Equal(t, chk.GasUsed, del.GasUsed, "deliver gas differs from check gas")
		})
	}
}
