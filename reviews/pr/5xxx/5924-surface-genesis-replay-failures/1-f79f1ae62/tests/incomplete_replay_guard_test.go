/* Run: from a gno checkout:
gh pr checkout 5924 -R gnolang/gno && git checkout f79f1ae62
curl -fsSL -o contribs/gnogenesis/internal/fork/incomplete_replay_guard_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5924-surface-genesis-replay-failures/1-f79f1ae62/tests/incomplete_replay_guard_test.go
cd contribs/gnogenesis && go test -v -run 'TestExecTest_IncompleteReplayFails' ./internal/fork/
rm internal/fork/incomplete_replay_guard_test.go
*/

// A genesis whose GnoGenesisState.InitialHeight mismatches the GenesisDoc
// makes loadAppState return before the tx loop, so no deliverable tx reaches
// the result handler. The node still boots to Ready. execTest's guard
// (processed < countDeliverableTxs) must fail loudly. Green at f79f1ae62;
// red before the guard commit (execTest returned nil / printed PASS).

package fork

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gnolang/gno/gno.land/pkg/gnoland"
	vmm "github.com/gnolang/gno/gno.land/pkg/sdk/vm"
	"github.com/gnolang/gno/tm2/pkg/amino"
	abci "github.com/gnolang/gno/tm2/pkg/bft/abci/types"
	bft "github.com/gnolang/gno/tm2/pkg/bft/types"
	"github.com/gnolang/gno/tm2/pkg/commands"
	"github.com/gnolang/gno/tm2/pkg/sdk/auth"
	"github.com/gnolang/gno/tm2/pkg/sdk/bank"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/stretchr/testify/require"
)

func TestExecTest_IncompleteReplayFails(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode — requires loading stdlibs")
	}

	appState := gnoland.GnoGenesisState{
		Balances: []gnoland.Balance{},
		// One deliverable tx: countDeliverableTxs == 1, but loadAppState
		// aborts before delivering it, so processed stays 0.
		Txs:           []gnoland.TxWithMetadata{{Tx: std.Tx{}, Metadata: nil}},
		Auth:          auth.DefaultGenesisState(),
		Bank:          bank.DefaultGenesisState(),
		VM:            vmm.DefaultGenesisState(),
		InitialHeight: 50, // != GenesisDoc.InitialHeight below -> loadAppState errors early
	}

	pv := bft.NewMockPV()
	pk := pv.PubKey()
	genDoc := bft.GenesisDoc{
		GenesisTime:   time.Now(),
		ChainID:       "test-hardfork-1",
		InitialHeight: 100,
		ConsensusParams: abci.ConsensusParams{
			Block: &abci.BlockParams{
				MaxTxBytes:   1_000_000,
				MaxDataBytes: 2_000_000,
				MaxGas:       3_000_000_000,
				TimeIotaMS:   100,
			},
		},
		Validators: []bft.GenesisValidator{
			{Address: pk.Address(), PubKey: pk, Power: 10, Name: "test-validator"},
		},
		AppState: appState,
	}

	data, err := amino.MarshalJSONIndent(genDoc, "", "  ")
	require.NoError(t, err)

	dir := t.TempDir()
	path := filepath.Join(dir, "genesis.json")
	require.NoError(t, os.WriteFile(path, data, 0o644))

	io := commands.NewTestIO()
	cfg := &testCfg{genesis: path, timeout: 3 * time.Minute}

	err = execTest(context.Background(), cfg, io)
	require.Error(t, err, "incomplete replay must fail the test, not print PASS")
	require.ErrorContains(t, err, "genesis replay delivered 0 of 1 expected txs")
}
