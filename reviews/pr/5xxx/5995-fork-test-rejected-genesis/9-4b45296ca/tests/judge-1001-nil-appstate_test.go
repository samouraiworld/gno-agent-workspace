package gnoland

// Judge probe for gnolang/gno#5995 round 9, candidate 1001: boot with a nil AppState.

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/gnolang/gno/gnovm/pkg/gnoenv"
	abci "github.com/gnolang/gno/tm2/pkg/bft/abci/types"
	bft "github.com/gnolang/gno/tm2/pkg/bft/types"
	"github.com/gnolang/gno/tm2/pkg/db/memdb"
	"github.com/gnolang/gno/tm2/pkg/log"
)

func TestJudge1001NilAppState(t *testing.T) {
	tmcfg := NewDefaultTMConfig(t.TempDir())
	pv := bft.NewMockPV()
	pk := pv.PubKey()
	genesis := &bft.GenesisDoc{
		GenesisTime: time.Now(), ChainID: tmcfg.ChainID(),
		ConsensusParams: abci.ConsensusParams{Block: defaultBlockParams()},
		Validators:      []bft.GenesisValidator{{Address: pk.Address(), PubKey: pk, Power: 10, Name: "self"}},
	}
	n, err := NewInMemoryNode(log.NewNoopLogger(), &InMemoryNodeConfig{
		PrivValidator: pv, Genesis: genesis, TMConfig: tmcfg, DB: memdb.NewMemDB(),
		InitChainerConfig: InitChainerConfig{
			GenesisTxResultHandler: PanicOnFailingTxResultHandler,
			StdlibDir:              filepath.Join(gnoenv.RootDir(), "gnovm", "stdlibs"),
			CacheStdlibLoad:        true,
		},
	})
	t.Logf("JUDGE1001 err=%v node_built=%v", err, n != nil)
}
