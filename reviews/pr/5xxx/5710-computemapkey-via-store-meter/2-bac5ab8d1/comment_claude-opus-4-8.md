# Review: PR #5710
Event: COMMENT

## Body
Reviewed independently; the implementation is correct and closes the write/restore asymmetry. Verified on bac5ab8d1: disabling the `ComputeMapKey` charge drops the read-only `Lookup` tx by 5516 gas (1861974 → 1856458), billing the restore path that was previously free. The write-path `Insert` tx loses the identical 5516 over the same cold-loaded keys, so write and rehydrate charge the same. The charge math is byte-identical to master; only the meter source moved.

One call for the maintainers, no file anchor: charging gas on the realm-restore path is consensus-affecting (it raises cost for every realm that persists a map), worth settling explicitly before merge.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5710-computemapkey-via-store-meter/2-bac5ab8d1/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/pkg/gnolang/machine.go:201-205 [↗](../../../../../.worktrees/gno-review-5710/gnovm/pkg/gnolang/machine.go#L201)
This defensive sync type-asserts `store.(*defaultStore)`, which never matches the production wrapper `transactionStore{*defaultStore}`, so it no-ops on every wrapped store and the comment's promise that store-routed charges otherwise "silently drop" does not hold. No consensus impact, since production and the gas-pinning filetests install the meter at [`BeginTransaction`](https://github.com/gnolang/gno/blob/bac5ab8d1/gno.land/pkg/sdk/vm/keeper.go#L395-L397) · [↗](../../../../../.worktrees/gno-review-5710/gno.land/pkg/sdk/vm/keeper.go#L395-L397), but the assert *would* match the shared base `*defaultStore`, so a future caller pairing it with a meter leaks a tx meter into shared state. Add `SetMeter` to the `Store` interface and call it unconditionally, or drop the block and rely on `BeginTransaction`.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5710 -R gnolang/gno
cat > gnovm/pkg/gnolang/zz_meter_sync_test.go <<'EOF'
package gnolang

import (
	"testing"

	storetypes "github.com/gnolang/gno/tm2/pkg/store/types"
	"github.com/stretchr/testify/require"
)

func TestMeterSyncGap(t *testing.T) {
	alloc := NewAllocator(1 << 30)
	parent := NewStore(alloc, nil, nil)
	txs := parent.BeginTransaction(nil, nil, nil, nil) // transactionStore, nil meter
	gm := storetypes.NewGasMeter(1 << 30)
	m := NewMachineWithOptions(MachineOptions{Store: txs, Alloc: alloc, GasMeter: gm, SkipPackage: true})
	defer m.Release()
	require.Nil(t, m.Store.GetMeter()) // defensive sync missed the wrapper
	before := gm.GasConsumed()
	_, _ = typedInt(42).ComputeMapKey(m.Store, false)
	require.Equal(t, int64(0), gm.GasConsumed()-before) // charges zero
}
EOF
go test -run TestMeterSyncGap ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/zz_meter_sync_test.go
```

```
ok  	github.com/gnolang/gno/gnovm/pkg/gnolang
```
</details>

## gno.land/pkg/integration/testdata/compute_map_key_restore_gas.txtar:35-47 [↗](../../../../../.worktrees/gno-review-5710/gno.land/pkg/integration/testdata/compute_map_key_restore_gas.txtar#L35)
These `GAS USED` pins are full-tx totals, so unrelated gas-model drift on master reds this test even when the restore fix is intact, and they prove the totals rather than the asymmetry (the [`compute_map_key_big_bytes`](https://github.com/gnolang/gno/blob/bac5ab8d1/gnovm/tests/files/gas/compute_map_key_big_bytes.gno) · [↗](../../../../../.worktrees/gno-review-5710/gnovm/tests/files/gas/compute_map_key_big_bytes.gno) / `compute_map_key_small_bytes` filetests are already off by +113 on master). A differential assertion — two txs differing by one cold-loaded entry, asserting the per-entry delta — would check the property directly. Separately, the header's "Tx2 by ~4728" undercounts the measured ~5516, since the 3-field Insert key costs more than a single-field one.

## gnovm/pkg/gnolang/values_test.go:409-435 [↗](../../../../../.worktrees/gno-review-5710/gnovm/pkg/gnolang/values_test.go#L409)
`TestComputeMapKey_GasViaStoreMeter` only lower-bounds the charge for one scalar `int` key; it never charges the same key on a write-shaped and a restore-shaped call and asserts equality, which is the symmetry this PR claims. Add a composite-key case (struct or array) asserting equal write and restore deltas, so a future path-specific branch can't silently break symmetry.

## gnovm/pkg/gnolang/store.go:73-77 [↗](../../../../../.worktrees/gno-review-5710/gnovm/pkg/gnolang/store.go#L73)
The comment says "Store does not charge gas itself," but [`consumeGas`](https://github.com/gnolang/gno/blob/bac5ab8d1/gnovm/pkg/gnolang/store.go#L1104-L1108) · [↗](../../../../../.worktrees/gno-review-5710/gnovm/pkg/gnolang/store.go#L1104) charges amino gas through this same store. The accessor is passive; reword to "this accessor does not itself charge."
