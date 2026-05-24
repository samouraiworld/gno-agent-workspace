# PR #5710: fix(gnovm): route ComputeMapKey gas via store.GetMeter()

URL: https://github.com/gnolang/gno/pull/5710
Author: ltzmaxwell | Base: master | Files: 10 | +179 -53
Reviewed by: davd-gzl | Model: claude-opus-4-7

**Verdict: REQUEST CHANGES** — design is correct and closes the write/restore gas asymmetry the description claims, but the defensive `*defaultStore` type-assert in `NewMachineWithOptions` silently no-ops on the `transactionStore` wrapper, leaving `gno test` and any sub-tx Machine with `m.GasMeter` set but `m.Store.GetMeter() == nil` — `ComputeMapKey` then charges zero.

## Summary

Follow-up to #5127. #5127 added gas charging for `ComputeMapKey` but threaded the meter through `*Machine`, so realm restore (`fillTypesOfValue` rebuilding `vmap` on cache miss) — which has a `Store` in scope but no `*Machine` — passed `nil` and charged zero. Write path: charged. Restore path: free. Asymmetric, attacker-controllable via cache pressure.

This PR moves the meter onto the `Store` interface (`GetMeter() store.GasMeter`), `ComputeMapKey` consults it directly, and the `*Machine` parameter is dropped from `ComputeMapKey` / `GetPointerForKey` / `GetValueForKey` / `DeleteForKey` / `GetPointerAtIndexInt`. The tx meter is already installed on the store by `BeginTransaction` (it was used for amino encode/decode), so both paths now consult one source of truth.

```
before #5710:                        after #5710:
  write path:                          write/restore both:
    m -> m.GasMeter -> ConsumeGas        store -> store.GetMeter() -> ConsumeGas
  restore path (fillTypesOfValue):
    nil -> (no charge)                  ^ symmetric
```

## Glossary

- `ComputeMapKey` — encodes a `TypedValue` to a `MapKey` string used for `MapValue.vmap` lookup; recursive on arrays/structs.
- `fillTypesOfValue` — `realm.go` deserialization path; rebuilds `MapValue.vmap` after loading from disk by re-running `ComputeMapKey` on every entry.
- `BeginTransaction` — forks a `defaultStore` into a `transactionStore` with per-tx caches; takes `gasMeter` arg and writes it onto the sub-store.
- `transactionStore` — wrapper struct `{*defaultStore}` returned by `BeginTransaction`; methods dispatch to embedded `*defaultStore`.

## Fix

`Store` interface gains `GetMeter() store.GasMeter` ([`store.go:73-77`](../../../../../.worktrees/gno-review-5710/gnovm/pkg/gnolang/store.go#L73-L77)), implemented as a one-line accessor on `*defaultStore` ([`store.go:325-327`](../../../../../.worktrees/gno-review-5710/gnovm/pkg/gnolang/store.go#L325-L327)). `ComputeMapKey` drops its `*Machine` parameter and reads the meter through `store.GetMeter()` instead ([`values.go:1597-1604`](../../../../../.worktrees/gno-review-5710/gnovm/pkg/gnolang/values.go#L1597-L1604)). `fillTypesOfValue` — which previously passed `nil` machine with an apology comment — now just passes the store ([`realm.go:1879`](../../../../../.worktrees/gno-review-5710/gnovm/pkg/gnolang/realm.go#L1879)). A defensive sync in `NewMachineWithOptions` writes the Machine's meter onto the store if the store doesn't already have one ([`machine.go:199-205`](../../../../../.worktrees/gno-review-5710/gnovm/pkg/gnolang/machine.go#L199-L205)). The load-bearing invariant: the tx-scoped meter is on the store from `BeginTransaction` onward, so any reader (write or restore) finds it via `Store`.

## Benchmarks / Numbers

Restore-path txtar (new `compute_map_key_restore_gas.txtar`):

| Tx | Path | GAS USED |
|---|---|---:|
| Tx1 | `Insert` (write path, charges via Machine→Store→Meter) | 2,595,329 |
| Tx2 | `Lookup` (restore path, fresh tx, forces `fillTypesOfValue`) | 1,861,974 |

Pre-#5710, Tx2's `ComputeMapKey` calls would have charged zero — the restore-path delta is the load-bearing signal.

## Critical (must fix)

- **[defensive sync skips transactionStore]** [`machine.go:201-205`](../../../../../.worktrees/gno-review-5710/gnovm/pkg/gnolang/machine.go#L201-L205) — `store.(*defaultStore)` type-asserts the concrete type; the production wrapper is `transactionStore{*defaultStore}`, a different concrete type, so the assert fails and the sync silently no-ops on every wrapped store.
  <details><summary>details</summary>

  **Shape:** `store.(*defaultStore)` matches the raw struct only. `transactionStore` embeds `*defaultStore` for method promotion but the concrete type returned by `BeginTransaction` is `transactionStore`, not `*defaultStore`. The assert returns `ok=false`.

  **Mechanism:** Real callsite at [`test.go:513`](../../../../../.worktrees/gno-review-5710/gnovm/pkg/test/test.go#L513): `m = Machine(tgs, ..., store.NewInfiniteGasMeter())` where `tgs` came from [`test.go:335`](../../../../../.worktrees/gno-review-5710/gnovm/pkg/test/test.go#L335) — `opts.TestStore.BeginTransaction(tcw, tcw, nil, nil)` (nil meter at tx creation). Reaching `NewMachineWithOptions`: `vmGasMeter != nil`, `store.GetMeter() == nil` (sub-store has no meter), `store.(*defaultStore)` fails → defensive code skipped. Net result: `m.GasMeter` is set but `m.Store.GetMeter()` returns nil; `ComputeMapKey` charges zero.

  **Result:** Every `gno test` run with map operations undercharges `ComputeMapKey` relative to production. Same shape at [`filetest.go:454-455`](../../../../../.worktrees/gno-review-5710/gnovm/pkg/test/filetest.go#L454-L455) for the realm filetest path. Production keeper is safe (`BeginTransaction` always receives `ctx.GasMeter()`), but the test-path divergence means CI gas pinning under `gno test` would silently drift from on-chain gas.

  **Fix:** Either unwrap inside the assert (`if ts, ok := store.(transactionStore); ok { ts.gasMeter = vmGasMeter }` covers the embedded field via promotion since `transactionStore.defaultStore.gasMeter` is the same field), or — preferred — add `SetMeter(store.GasMeter)` to the `Store` interface and call it unconditionally. The latter mirrors the existing `SetAllocator` shape and avoids the assert dance entirely. Add a unit test that constructs `transactionStore{...}` with a nil meter, runs the sync, and asserts `store.GetMeter() != nil` afterward.

  **Repro:** (paste into a local clone of `gnolang/gno`)
  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5710 -R gnolang/gno
  cat > gnovm/pkg/gnolang/tx_meter_sync_test.go <<'EOF'
  package gnolang

  import (
  	"testing"

  	storetypes "github.com/gnolang/gno/tm2/pkg/store/types"
  	"github.com/stretchr/testify/require"
  )

  func TestTxStoreMeterSync(t *testing.T) {
  	alloc := NewAllocator(1 << 30)
  	parent := NewStore(alloc, nil, nil)
  	txs := parent.BeginTransaction(nil, nil, nil, nil) // nil meter
  	gm := storetypes.NewGasMeter(1 << 30)

  	m := NewMachineWithOptions(MachineOptions{
  		Store: txs, Alloc: alloc, GasMeter: gm, SkipPackage: true,
  	})
  	defer m.Release()

  	// IS: store-meter is nil because the defensive sync's type assert
  	// matches *defaultStore but the store is transactionStore.
  	require.Nil(t, m.Store.GetMeter())

  	// IS: ComputeMapKey via this store charges zero, even though m.GasMeter is set.
  	before := gm.GasConsumed()
  	tv := typedInt(42)
  	_, _ = tv.ComputeMapKey(m.Store, false)
  	require.Equal(t, int64(0), gm.GasConsumed()-before)
  }
  EOF
  go test -v -run TestTxStoreMeterSync ./gnovm/pkg/gnolang/
  rm gnovm/pkg/gnolang/tx_meter_sync_test.go
  ```
  Passing both assertions confirms the silent gas drop. Flip to `require.Same(t, gm, m.Store.GetMeter())` and `require.GreaterOrEqual(t, ..., int64(OpCPUComputeMapKey))` after a fix — both should then pass.
  </details>

## Warnings (should fix)

- **[behavior change for half-wired Machines]** [`values.go:1601`](../../../../../.worktrees/gno-review-5710/gnovm/pkg/gnolang/values.go#L1601) — pre-PR, a Machine with `m.GasMeter == nil` but a store-meter set wouldn't charge `ComputeMapKey`; post-PR, it does. Mirror-image: a Machine with `m.GasMeter != nil` and store-meter nil charged pre-PR, doesn't post-PR (see Critical).
  <details><summary>details</summary>

  The two failure modes are now coupled through the store alone. This is a contract tightening (good in production: write and restore are now symmetric) but a silent behavior change for any test/tool that relied on the old `m.GasMeter`-only route. The PR description should call this out as a breaking change for direct `ComputeMapKey` callers in `gnovm/pkg/gnolang` consumers. Worth at least one inline doc-comment on `Store.GetMeter()` stating "this is the authoritative meter for VM-CPU work that doesn't have a `*Machine` in scope".
  </details>

- **[no SetMeter symmetry]** [`store.go:73-77`](../../../../../.worktrees/gno-review-5710/gnovm/pkg/gnolang/store.go#L73-L77) — `GetMeter` is added without a `SetMeter` counterpart, unlike `GetAllocator`/`SetAllocator`. The defensive sync in `NewMachineWithOptions` reaches in via direct field access (`ds.gasMeter = vmGasMeter`), which only works from within `package gnolang` — fragile.
  <details><summary>details</summary>

  Adding `SetMeter(store.GasMeter)` to the interface costs one method on `*defaultStore` and lets the defensive sync be a single unconditional call instead of a type assert. This also resolves the Critical above (the type assert is what's broken). Same naming question: `GetGasMeter` / `SetGasMeter` would be more consistent with `BeginTransaction`'s `gasMeter` parameter name and the underlying field name. Minor — `GetMeter` is fine if applied consistently.
  </details>

- **[gctx not synced alongside meter]** [`machine.go:199-205`](../../../../../.worktrees/gno-review-5710/gnovm/pkg/gnolang/machine.go#L199-L205) — the defensive sync copies `vmGasMeter` onto the store but not `gctx` (the storage-I/O gas context). For half-wired test paths this means VM-CPU charges via `ComputeMapKey` would land on the meter, but amino encode/decode storage charges would still no-op. Not a correctness bug today (test paths use infinite meters), but the asymmetry is the kind of trap that bites later. At least a comment naming it as deliberate would help.

## Nits

- [`values.go:1597`](../../../../../.worktrees/gno-review-5710/gnovm/pkg/gnolang/values.go#L1597) — `ComputeMapKey` now silently panics if `store == nil` (pre-PR was `m != nil && m.GasMeter != nil` guarded). All current callers pass a non-nil store, but a one-line `if store == nil { ... }` guard or a doc-comment "store must not be nil" is cheap insurance.
- [`store.go:73-77`](../../../../../.worktrees/gno-review-5710/gnovm/pkg/gnolang/store.go#L73-L77) — the comment says "Passive accessor — Store does not charge gas itself" but `defaultStore.consumeGas` does charge gas (amino encode/decode). The comment means "this accessor doesn't charge" — phrasing could be tighter.
- [`values_test.go:404-409`](../../../../../.worktrees/gno-review-5710/gnovm/pkg/gnolang/values_test.go#L404-L409) — `TestComputeMapKey_GasViaStoreMeter` exercises a bare `*defaultStore` with manually set `ds.gasMeter`; doesn't exercise the production shape (`transactionStore` from `BeginTransaction`). Add a sibling that runs `ComputeMapKey(ds.BeginTransaction(nil, nil, nil, gm), ...)` to catch the Critical above directly at unit-test speed.

## Missing Tests

- **[transactionStore defensive sync]** [`machine.go:201-205`](../../../../../.worktrees/gno-review-5710/gnovm/pkg/gnolang/machine.go#L201-L205) — no test asserts that `NewMachineWithOptions(MachineOptions{Store: someTransactionStore, GasMeter: gm})` results in `m.Store.GetMeter() == gm` when the tx had no meter. Direct unit test would fail today, which is the point.
  <details><summary>details</summary>

  Suggested test shape: create a `defaultStore`, call `BeginTransaction(nil, nil, nil, nil)`, hand the result and a fresh meter to `NewMachineWithOptions`, then assert `m.Store.GetMeter() != nil`. This is the canary for the Critical above.
  </details>

- **[restore-path nested key]** new txtar [`compute_map_key_restore_gas.txtar`](../../../../../.worktrees/gno-review-5710/gno.land/pkg/integration/testdata/compute_map_key_restore_gas.txtar) — exercises a single-field-dominant struct key (`bigKey{A: ...}`). The recursive-charging change in #5127's tail (`165f98496` "charge `len(bz)` at end") was specifically about depth-N nested keys re-copying child bytes. A nested key (e.g. `map[[2]bigKey]string`) would catch any regression where the recursive call's `mk` re-append no longer charges. Optional but cheap.

## Suggestions

- [`values.go:1601-1623`](../../../../../.worktrees/gno-review-5710/gnovm/pkg/gnolang/values.go#L1601-L1623) — `gm := store.GetMeter()` is called once at the top of every recursive call, then again on every subcall. Each call into a 1000-element array pays N store-method dispatches just to look up the meter. Caching the meter on the `Machine` and threading it through the recursion would be one solution, but the cleaner alternative — and what makes this PR's design clean overall — is that the per-call dispatch cost is dwarfed by the per-call gas charge it produces. Leaving as-is; flagging only because a future profiler may notice.

## Questions for Author

- The defensive sync in `NewMachineWithOptions` is described in the comment as "If the Machine has a meter, the store must too — otherwise store-routed charges silently drop." Was the `transactionStore` case considered? If yes, what's the rationale for not handling it (the type assert specifically excludes the wrapper)?
- Why no `SetMeter` on the `Store` interface? `SetAllocator` exists; the asymmetry feels accidental.
- Tx1 vs Tx2 gas pinning in the new txtar (2,595,329 vs 1,861,974): is the intent that any future recalibration of `OpCPUComputeMapKey` / `OpCPUSlopeComputeMapKeyByte` will update both proportionally? The txtar's regression doc says "Tx2 drops noticeably below pinned value" — what's "noticeably" in practice (50%? 90%?), and would a one-step-recalibration ever trip it as a false positive?
