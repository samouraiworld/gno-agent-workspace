// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.

/* Run: from a local clone of gnolang/gno:
gh pr checkout 5710 -R gnolang/gno && git checkout 55d874f6
curl -fsSL -o gnovm/pkg/gnolang/transactionstore_meter_sync_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5710-computemapkey-via-store-meter/1-55d874f6/tests/transactionstore_meter_sync_test.go
go test -v -run TestNewMachine_TransactionStoreMeterSync ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/transactionstore_meter_sync_test.go
*/

package gnolang

import (
	"testing"

	storetypes "github.com/gnolang/gno/tm2/pkg/store/types"
	"github.com/stretchr/testify/require"
)

// TestNewMachine_TransactionStoreMeterSync exercises the defensive sync in
// NewMachineWithOptions that promises: "if the Machine has a meter, the store
// must too." (machine.go:199-205). The sync uses store.(*defaultStore), which
// silently fails on the production wrapper transactionStore{*defaultStore} —
// the concrete type returned by BeginTransaction.
//
// IS:     Machine has GasMeter, transactionStore has nil meter — sync no-ops.
//         ComputeMapKey on this Machine charges zero (silent).
// SHOULD: After NewMachineWithOptions, store.GetMeter() == the passed meter,
//         and a probe ComputeMapKey call charges OpCPUComputeMapKey.
//
// Flipping the assertion: when the sync is extended to handle transactionStore
// (e.g. via SetMeter on the Store interface or unwrapping in the type assert),
// `wantSynced` flips to true and both subtests pass.
func TestNewMachine_TransactionStoreMeterSync(t *testing.T) {
	// Build a transactionStore via the production path: BeginTransaction with
	// nil meter, then hand the result to NewMachineWithOptions with a non-nil
	// meter — mirrors gnovm/pkg/test/test.go:335+513.
	alloc := NewAllocator(1 << 30)
	parent := NewStore(alloc, nil, nil)
	txs := parent.BeginTransaction(nil, nil, nil, nil) // nil meter
	gm := storetypes.NewGasMeter(1 << 30)

	m := NewMachineWithOptions(MachineOptions{
		PkgPath:     "",
		Output:      nil,
		Store:       txs,
		Alloc:       alloc,
		GasMeter:    gm,
		SkipPackage: true,
	})
	defer m.Release()

	t.Run("store_meter_after_sync", func(t *testing.T) {
		got := m.Store.GetMeter()
		// IS:     got == nil (defensive sync missed transactionStore)
		// SHOULD: got == gm (sync extended to handle the wrapper)
		require.Nil(t, got,
			"IS: defensive sync only handles *defaultStore, transactionStore silently misses")
		// require.Same(t, gm, got,
		// 	"SHOULD: sync covers transactionStore too — meters are aligned")
	})

	t.Run("computemapkey_charges_via_store", func(t *testing.T) {
		tv := typedInt(42)
		before := gm.GasConsumed()
		_, isNaN := tv.ComputeMapKey(m.Store, false)
		require.False(t, isNaN)
		delta := gm.GasConsumed() - before
		// IS:     delta == 0 — store has no meter, ComputeMapKey skips charge
		// SHOULD: delta >= OpCPUComputeMapKey — store-meter populated, charge fires
		require.Equal(t, int64(0), delta,
			"IS: ComputeMapKey via transactionStore charges zero because store.GetMeter() is nil")
		// require.GreaterOrEqual(t, delta, int64(OpCPUComputeMapKey),
		// 	"SHOULD: ComputeMapKey via transactionStore charges per-call constant")
	})
}
