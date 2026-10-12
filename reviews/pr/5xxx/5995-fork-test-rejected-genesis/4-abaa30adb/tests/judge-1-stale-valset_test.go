package node

// Run from a gno checkout at abaa30adb, file placed in tm2/pkg/bft/node/:
//   go test -v -run 'TestJudgeStaleValsetAfterReset$' ./tm2/pkg/bft/node/

import (
	"testing"

	"github.com/stretchr/testify/require"

	sm "github.com/gnolang/gno/tm2/pkg/bft/state"
	"github.com/gnolang/gno/tm2/pkg/crypto/ed25519"
	dbm "github.com/gnolang/gno/tm2/pkg/db"
	"github.com/gnolang/gno/tm2/pkg/db/memdb"
)

// A first boot at InitialHeight 1 that InitChain rejects (ReplayBlocks writes
// nothing at InitialHeight 1 before InitChain), then a corrected restart at 100.
func TestJudgeStaleValsetAfterReset(t *testing.T) {
	pk := ed25519.GenPrivKey().PubKey()

	reset := dbm.DB(memdb.NewMemDB())
	_, _, err := LoadStateFromDBOrGenesisDocProvider(reset, probeGenesisDocProvider(pk, 1))
	require.NoError(t, err)
	st, doc, err := LoadStateFromDBOrGenesisDocProvider(reset, probeGenesisDocProvider(pk, 100))
	require.NoError(t, err)
	t.Logf("reset node: doc.InitialHeight=%d state.LastBlockHeight=%d", doc.InitialHeight, st.LastBlockHeight)

	fresh := dbm.DB(memdb.NewMemDB())
	_, _, err = LoadStateFromDBOrGenesisDocProvider(fresh, probeGenesisDocProvider(pk, 100))
	require.NoError(t, err)

	for _, h := range []int64{1, 2, 100, 101} {
		rv, rerr := sm.LoadValidators(reset, h)
		_, ferr := sm.LoadValidators(fresh, h)
		_, rcerr := sm.LoadConsensusParams(reset, h)
		_, fcerr := sm.LoadConsensusParams(fresh, h)
		n := 0
		if rv != nil {
			n = rv.Size()
		}
		t.Logf("h=%d reset: vals=%d err=%v params_err=%v | fresh: vals_err=%v params_err=%v", h, n, rerr, rcerr, ferr, fcerr)
	}
}
