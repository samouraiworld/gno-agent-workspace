# Review: PR [#5924](https://github.com/gnolang/gno/pull/5924)
Event: APPROVE

## Body
Looks good. Both fixes make a genesis replay that aborts before loading state fail loudly instead of passing silently.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5924-surface-genesis-replay-failures/1-f79f1ae62/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## contribs/gnogenesis/internal/fork/test.go:283 [↗](../../../../../.worktrees/gno-review-5924/contribs/gnogenesis/internal/fork/test.go#L283)
Missing test: no test drives `execTest` to the new incomplete-replay failure. Only the `countDeliverableTxs` helper is covered, so a regression that dropped the guard keeps every test green. A genesis whose `GnoGenesisState.InitialHeight` mismatches the `GenesisDoc` aborts [`loadAppState`](https://github.com/gnolang/gno/blob/f79f1ae62/gno.land/pkg/gnoland/app.go#L494-L499) · [↗](../../../../../.worktrees/gno-review-5924/gno.land/pkg/gnoland/app.go#L494) before the tx loop while the node still boots to Ready.

<details><summary>test cases</summary>

Green at f79f1ae62 with the guard present, red before the guard commit. Paste into the `fork` package:

```go
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
		Balances:      []gnoland.Balance{},
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
```
</details>

## contribs/gnogenesis/internal/fork/test.go:262 [↗](../../../../../.worktrees/gno-review-5924/contribs/gnogenesis/internal/fork/test.go#L262)
Suggestion: the "Txs processed" line divides by `len(appState.Txs)`, which counts the `Metadata.Failed` entries the replay [intentionally skips](https://github.com/gnolang/gno/blob/f79f1ae62/gno.land/pkg/gnoland/app.go#L792-L800) · [↗](../../../../../.worktrees/gno-review-5924/gno.land/pkg/gnoland/app.go#L792). A healthy genesis carrying source-failed txs prints "Txs processed: 3 / 5" right before "PASS", the same shape as a partial failure. Divide by `countDeliverableTxs` so the denominator matches what the guard measures.
