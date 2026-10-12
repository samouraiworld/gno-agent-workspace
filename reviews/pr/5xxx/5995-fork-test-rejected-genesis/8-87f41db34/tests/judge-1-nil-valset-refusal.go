package node

// Asserts a genesis refused for having no validators can be corrected in place.
// Copy to tm2/pkg/bft/node/judge1_test.go, then:
//   go test ./tm2/pkg/bft/node -run TestJudge1NilValsetRefusalCorrected -v
// Fails at 87f41db34: the retry gets "validator set is nil in genesis and still empty after InitChain".

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	abci "github.com/gnolang/gno/tm2/pkg/bft/abci/types"
	cfg "github.com/gnolang/gno/tm2/pkg/bft/config"
	"github.com/gnolang/gno/tm2/pkg/bft/types"
	dbm "github.com/gnolang/gno/tm2/pkg/db"
	"github.com/gnolang/gno/tm2/pkg/db/memdb"
	"github.com/gnolang/gno/tm2/pkg/events"
	"github.com/gnolang/gno/tm2/pkg/log"
)

// Returns the request's validators, as gno.land's InitChainer does.
type echoValsApp struct {
	abci.BaseApplication
	seen []int
}

func (a *echoValsApp) InitChain(req abci.RequestInitChain) abci.ResponseInitChain {
	a.seen = append(a.seen, len(req.Validators))
	return abci.ResponseInitChain{Validators: req.Validators}
}

func TestJudge1NilValsetRefusalCorrected(t *testing.T) {
	config, genesisFile := cfg.ResetTestRoot("judge1_nil_valset")
	defer os.RemoveAll(config.RootDir)

	app := &echoValsApp{}
	config.LocalApp = app

	opened := map[string]dbm.DB{}
	dbs := func(ctx *DBContext) (dbm.DB, error) {
		if db, ok := opened[ctx.ID]; ok {
			return db, nil
		}
		db := memdb.NewMemDB()
		opened[ctx.ID] = db
		return db, nil
	}

	build := func(withVals bool) (*Node, error) {
		provider := func() (*types.GenesisDoc, error) {
			doc, derr := types.GenesisDocFromFile(genesisFile)
			if derr != nil {
				return nil, derr
			}
			if !withVals {
				doc.Validators = nil
			}
			doc.AppState = "app-state"
			return doc, nil
		}
		return newNodeWithDBProvider(
			config, provider, dbs, events.NewEventSwitch(), log.NewNoopLogger())
	}

	_, err := build(false)
	require.Error(t, err, "a genesis with no validators must not boot")
	t.Logf("first boot: %v", err)

	n, err := build(true)
	t.Logf("validators seen by InitChain per boot: %v", app.seen)
	require.NoError(t, err, "the corrected genesis must boot")
	require.NotNil(t, n)
}
