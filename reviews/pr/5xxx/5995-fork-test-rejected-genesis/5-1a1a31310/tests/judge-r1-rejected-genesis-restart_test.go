package gnoland

// Asserts: a gno.land genesis InitChainer refuses (InitialHeight mismatch) fails
// boot, and a restart on the same DB after correcting either side boots at the
// corrected doc's InitialHeight. Copy into gno.land/pkg/gnoland and run
// go test -run TestJudgeRejectedGenesisRestart -v ./gno.land/pkg/gnoland

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/gnolang/gno/gnovm/pkg/gnoenv"
	abci "github.com/gnolang/gno/tm2/pkg/bft/abci/types"
	bft "github.com/gnolang/gno/tm2/pkg/bft/types"
	"github.com/gnolang/gno/tm2/pkg/db"
	"github.com/gnolang/gno/tm2/pkg/db/memdb"
	"github.com/gnolang/gno/tm2/pkg/log"
)

func TestJudgeRejectedGenesisRestart(t *testing.T) {
	cases := []struct {
		name                 string
		fixDoc, fixApp, want int64
	}{
		{"correct the app state", 100, 100, 100},
		{"correct the genesis doc", 999, 999, 999},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			td := t.TempDir()
			pv := bft.NewMockPV()
			pk := pv.PubKey()
			shared := memdb.NewMemDB()
			gen := time.Now()
			boot := func(docH, appH int64) (bool, int64, int64) {
				tmcfg := NewDefaultTMConfig(td)
				app := DefaultGenState()
				app.InitialHeight = appH
				cfg := &InMemoryNodeConfig{
					PrivValidator: pv,
					Genesis: &bft.GenesisDoc{
						GenesisTime: gen, ChainID: tmcfg.ChainID(), InitialHeight: docH,
						ConsensusParams: abci.ConsensusParams{Block: defaultBlockParams()},
						Validators:      []bft.GenesisValidator{{Address: pk.Address(), PubKey: pk, Power: 10, Name: "self"}},
						AppState:        app,
					},
					TMConfig: tmcfg,
					DB:       db.DB(shared),
					InitChainerConfig: InitChainerConfig{
						GenesisTxResultHandler: PanicOnFailingTxResultHandler,
						StdlibDir:              filepath.Join(gnoenv.RootDir(), "gnovm", "stdlibs"),
						CacheStdlibLoad:        true,
					},
				}
				n, err := NewInMemoryNode(log.NewNoopLogger(), cfg)
				if err != nil {
					t.Logf("boot doc=%d app=%d: err=%v", docH, appH, err)
					return false, 0, 0
				}
				require.NoError(t, n.Start())
				defer n.Stop()
				select {
				case <-n.Ready():
				case <-time.After(60 * time.Second):
					t.Fatal("no first block")
				}
				h := n.BlockStore().Height()
				t.Logf("boot doc=%d app=%d: booted, docInitialHeight=%d firstBlock=%d", docH, appH, n.GenesisDoc().InitialHeight, h)
				return true, n.GenesisDoc().InitialHeight, h
			}
			ok1, _, _ := boot(100, 999)
			require.False(t, ok1, "a refused genesis must not boot")
			ok2, docH, h := boot(tc.fixDoc, tc.fixApp)
			require.True(t, ok2, "the corrected genesis must boot")
			require.Equal(t, tc.want, docH)
			require.Equal(t, tc.want, h)
		})
	}
}
