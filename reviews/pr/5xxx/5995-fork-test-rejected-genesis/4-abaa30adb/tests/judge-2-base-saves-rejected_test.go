package consensus

// Run from a gno checkout, file placed in tm2/pkg/bft/consensus/, at the merge
// base cfcbaf484 and at the head abaa30adb:
//   go test -v -run 'TestJudgeRejectedInitChainMarker$' ./tm2/pkg/bft/consensus/

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	abci "github.com/gnolang/gno/tm2/pkg/bft/abci/types"
	"github.com/gnolang/gno/tm2/pkg/bft/appconn"
	"github.com/gnolang/gno/tm2/pkg/bft/proxy"
	sm "github.com/gnolang/gno/tm2/pkg/bft/state"
)

func TestJudgeRejectedInitChainMarker(t *testing.T) {
	app := initChainApp{initChain: func(req abci.RequestInitChain) abci.ResponseInitChain {
		return abci.ResponseInitChain{ResponseBase: abci.ResponseBase{Error: abci.StringError("InitialHeight mismatch")}}
	}}
	config, genesisFile := ResetConfig("judge_marker_")
	defer os.RemoveAll(config.RootDir)
	stateDB, state, store := makeStateAndStore(config, genesisFile, "v0.0.0-test")
	genDoc, _ := sm.MakeGenesisDocFromFile(genesisFile)
	proxyApp := appconn.NewAppConns(proxy.NewLocalClientCreator(app))
	require.NoError(t, proxyApp.Start())
	defer proxyApp.Stop()
	err := NewHandshaker(stateDB, state, store, genDoc).Handshake(proxyApp)
	t.Logf("handshake err=%v", err)
	has, herr := stateDB.Has(sm.CalcABCIResponsesKey(0))
	t.Logf("height-0 marker present=%v err=%v", has, herr)
}
