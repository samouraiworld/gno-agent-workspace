// Repro, from a plain clone of gnolang/gno:
//   git fetch origin pull/5995/head && git checkout 1a1a313105e35c8d31fabac32413d0558a9ad476
//   cp solo-finder-gnoland-retry_test.go gno.land/pkg/gnoland/zz_solo_retry_test.go
//   go test ./gno.land/pkg/gnoland -run TestSoloGnolandRejectThenCorrected -count=1 -v
// Checks the ADR consequence "An operator can correct genesis.json and restart
// on the same data dir" against gno.land's real InitChainer, not a mock app:
// the same DB is handed to a refused boot and then to a corrected one.
package gnoland

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/gnolang/gno/gnovm/pkg/gnoenv"
	abci "github.com/gnolang/gno/tm2/pkg/bft/abci/types"
	bft "github.com/gnolang/gno/tm2/pkg/bft/types"
	"github.com/gnolang/gno/tm2/pkg/db/memdb"
	"github.com/gnolang/gno/tm2/pkg/log"
)

func TestSoloGnolandRejectThenCorrected(t *testing.T) {
	for _, tc := range []struct {
		name                   string
		badDoc, badApp, goodDoc, goodApp int64
	}{
		{"app side corrected, doc stays 100", 100, 999, 100, 100},
		{"doc side corrected, 100 to 1", 100, 1, 1, 1},
		{"doc side corrected, 1 to 100", 1, 100, 100, 100},
	} {
		t.Run(tc.name, func(t *testing.T) {
			td := t.TempDir()
			tmcfg := NewDefaultTMConfig(td)
			pv := bft.NewMockPV()
			pk := pv.PubKey()
			db := memdb.NewMemDB()
			gtime := time.Now()

			mk := func(docH, appH int64) *InMemoryNodeConfig {
				st := DefaultGenState()
				st.InitialHeight = appH
				return &InMemoryNodeConfig{
					PrivValidator: pv,
					Genesis: &bft.GenesisDoc{
						GenesisTime: gtime, ChainID: tmcfg.ChainID(), InitialHeight: docH,
						ConsensusParams: abci.ConsensusParams{Block: defaultBlockParams()},
						Validators:      []bft.GenesisValidator{{Address: pk.Address(), PubKey: pk, Power: 10, Name: "self"}},
						AppState:        st,
					},
					TMConfig: tmcfg,
					DB:       db,
					InitChainerConfig: InitChainerConfig{
						GenesisTxResultHandler: PanicOnFailingTxResultHandler,
						StdlibDir:              filepath.Join(gnoenv.RootDir(), "gnovm", "stdlibs"),
						CacheStdlibLoad:        true,
					},
				}
			}

			_, err := NewInMemoryNode(log.NewNoopLogger(), mk(tc.badDoc, tc.badApp))
			require.Error(t, err)
			t.Logf("refused boot: %v", err)

			n, err := NewInMemoryNode(log.NewNoopLogger(), mk(tc.goodDoc, tc.goodApp))
			require.NoError(t, err, "corrected genesis on the same DB must boot")
			require.NoError(t, n.Start())
			t.Cleanup(func() { _ = n.Stop() })
			select {
			case <-n.Ready():
			case <-time.After(60 * time.Second):
				t.Fatal("timeout waiting for first block")
			}
			want := tc.goodDoc
			if want == 0 {
				want = 1
			}
			t.Logf("first block height=%d, genesis doc InitialHeight=%d", n.BlockStore().Height(), n.GenesisDoc().InitialHeight)
			require.Equal(t, want, n.BlockStore().Height())
		})
	}
}
