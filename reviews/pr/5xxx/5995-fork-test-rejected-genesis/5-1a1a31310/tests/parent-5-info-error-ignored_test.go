package consensus

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	abci "github.com/gnolang/gno/tm2/pkg/bft/abci/types"
	"github.com/gnolang/gno/tm2/pkg/bft/appconn"
	"github.com/gnolang/gno/tm2/pkg/bft/proxy"
	sm "github.com/gnolang/gno/tm2/pkg/bft/state"
)

type infoErrApp struct{ abci.BaseApplication }

func (infoErrApp) Info(abci.RequestInfo) abci.ResponseInfo {
	return abci.ResponseInfo{ResponseBase: abci.ResponseBase{Error: abci.StringError("info refused")}}
}

func TestZZInfoErrorIgnored(t *testing.T) {
	config, genesisFile := ResetConfig("zz_info_")
	defer os.RemoveAll(config.RootDir)
	stateDB, state, store := makeStateAndStore(config, genesisFile, "v0.0.0-test")
	genDoc, _ := sm.MakeGenesisDocFromFile(genesisFile)
	proxyApp := appconn.NewAppConns(proxy.NewLocalClientCreator(infoErrApp{}))
	require.NoError(t, proxyApp.Start())
	defer proxyApp.Stop()
	err := NewHandshaker(stateDB, state, store, genDoc).Handshake(proxyApp)
	t.Logf("handshake err=%v", err)
}
