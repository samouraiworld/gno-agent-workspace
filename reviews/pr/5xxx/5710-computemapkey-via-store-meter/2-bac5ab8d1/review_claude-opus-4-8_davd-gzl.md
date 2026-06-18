# PR #5710: fix(gnovm): route ComputeMapKey gas via store.GetMeter()

URL: https://github.com/gnolang/gno/pull/5710
Author: ltzmaxwell | Base: master | Files: 10 | +193 -53
Reviewed by: davd-gzl | Model: claude-opus-4-8 (deep) | Commit: `bac5ab8d1` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5710 bac5ab8d1`

> Round 2 (head advanced `55d874f6` → `bac5ab8d1`, +15 commits). The GnoVM code (`values.go`, `machine.go`, `store.go`, `realm.go`, `values_test.go`) is byte-identical to round 1; the only PR-content change since is the `compute_map_key_restore_gas.txtar` comment rewrite (commit `ee230482`), which quantified the regression signal (~788 gas/entry). **Round 1's headline Critical (the `transactionStore` meter-sync gap) is re-assessed down to a Warning here:** verified that no gas-enforcing or consensus path is affected (production and the gas-pinning filetests both install the meter at `BeginTransaction`); the gap's only effect is a cosmetic `gno test` gas undercount. New since round 1: two unresolved maintainer comments by @thehowl on the current head, one of which questions whether the restore path should charge at all. Verdict shifts REQUEST CHANGES → NEEDS DISCUSSION accordingly.

**TL;DR:** Map lookups encode the key into a string (`ComputeMapKey`); #5127 made that charge gas, but only when a `*Machine` was in scope, so reloading a map from disk (which has no `*Machine`) was free while writing it was not. This PR removes the `*Machine` plumbing and reads the gas meter off the `Store` instead, so both paths charge the same.

**Verdict: NEEDS DISCUSSION** — implementation is correct, deterministic, and genuinely closes the asymmetry, but two design questions are open and unanswered on the current head: whether the restore path *should* charge gas at all ([@thehowl](https://github.com/gnolang/gno/pull/5710#discussion_r3341682416)), and whether the meter should come from the Store or an explicit `GasMeter` parameter ([@thehowl](https://github.com/gnolang/gno/pull/5710#discussion_r3341663206)); separately, the defensive meter-sync in `NewMachineWithOptions` does not work for the wrapped store type and should be fixed or removed.

## Summary

Follow-up to #5127. #5127 added a per-call CPU charge plus a per-byte slope to `ComputeMapKey`, but threaded the gas meter through `*Machine`. The realm-restore path (`fillTypesOfValue` rebuilding a `MapValue.vmap` after a cold load from disk) has a `Store` but no `*Machine`, so it passed `nil` and charged zero, while the write path charged. This PR drops the `*Machine` parameter from `ComputeMapKey` / `GetPointerForKey` / `GetValueForKey` / `DeleteForKey` / `GetPointerAtIndexInt` and routes the charge through a new `Store.GetMeter()` accessor. The tx-scoped meter is already on the store from `BeginTransaction` (it backs amino encode/decode gas), so both paths now read one source of truth. Gas constants and charge math are unchanged from master; what changes is that cold map loads now bill the user.

```
master:                                  this PR:
  write path:   m -> m.GasMeter            write + restore:
  restore path: nil -> (free)                store -> store.GetMeter()  [symmetric]
```

## Glossary

- ComputeMapKey — encodes a `TypedValue` to the `MapKey` string for `vmap` lookup; recursive on arrays/structs; charges CPU gas via the meter read from the Store.
- fillTypesOfValue — `realm.go` deserialization step that rebuilds a loaded map's `vmap` by re-running `ComputeMapKey` per entry (the restore path).
- transactionStore — per-tx Store wrapper `{*defaultStore}` from `BeginTransaction`; promotes methods to the embedded pointer, so `store.(*defaultStore)` does NOT match it.

## Fix

`Store` gains `GetMeter() store.GasMeter` ([`store.go:73-77`](https://github.com/gnolang/gno/blob/bac5ab8d1/gnovm/pkg/gnolang/store.go#L73-L77) · [↗](../../../../../.worktrees/gno-review-5710/gnovm/pkg/gnolang/store.go#L73-L77)), a one-line accessor on `*defaultStore` ([`store.go:325-327`](https://github.com/gnolang/gno/blob/bac5ab8d1/gnovm/pkg/gnolang/store.go#L325-L327) · [↗](../../../../../.worktrees/gno-review-5710/gnovm/pkg/gnolang/store.go#L325-L327)). `ComputeMapKey` drops `*Machine` and reads the meter via `store.GetMeter()` ([`values.go:1597-1604`](https://github.com/gnolang/gno/blob/bac5ab8d1/gnovm/pkg/gnolang/values.go#L1597-L1604) · [↗](../../../../../.worktrees/gno-review-5710/gnovm/pkg/gnolang/values.go#L1597-L1604)); `fillTypesOfValue` drops its nil-machine call ([`realm.go:1879`](https://github.com/gnolang/gno/blob/bac5ab8d1/gnovm/pkg/gnolang/realm.go#L1879) · [↗](../../../../../.worktrees/gno-review-5710/gnovm/pkg/gnolang/realm.go#L1879)). A defensive block in `NewMachineWithOptions` tries to copy the Machine's meter onto a meter-less store ([`machine.go:199-205`](https://github.com/gnolang/gno/blob/bac5ab8d1/gnovm/pkg/gnolang/machine.go#L199-L205) · [↗](../../../../../.worktrees/gno-review-5710/gnovm/pkg/gnolang/machine.go#L199-L205)). The load-bearing invariant in production: `BeginTransaction` installs the tx meter on the store ([`keeper.go:395-397`](https://github.com/gnolang/gno/blob/bac5ab8d1/gno.land/pkg/sdk/vm/keeper.go#L395-L397) · [↗](../../../../../.worktrees/gno-review-5710/gno.land/pkg/sdk/vm/keeper.go#L395-L397)) and every Machine is built with the same `ctx.GasMeter()`, so both meters are the same object.

## Benchmarks / Numbers

Total `ComputeMapKey` gas per tx, measured by disabling both charge sites in `ComputeMapKey` and re-running the txtar (worktree, charge on vs off):

| Tx | Path | GAS USED (PR) | charge off | ComputeMapKey gas |
|---|---|---:|---:|---:|
| Tx1 `Insert` | restore 5 init keys + write 1 | 2,595,329 | 2,589,813 | 5,516 |
| Tx2 `Lookup` | restore 6 keys, no write | 1,861,974 | 1,856,458 | 5,516 |

Tx2 does no writes, so its 5,516 is purely restore-path gas that was **zero** before this PR — direct confirmation the asymmetry is closed. Both txs lose the identical 5,516 over the same five cold-loaded init keys, so writing the Insert key (Tx1) and rehydrating it (Tx2) cost the same: the paths charge symmetrically. Decomposes as ~3,940 for the five single-field init keys (788 each, per the txtar header) plus ~1,576 for the 3-field Insert key, which costs more than a single-field key — so the header's "Tx2 by ~4728" (6 × 788) undercounts the real ~5,516 (see Suggestions).

## Critical (must fix)

None.

## Warnings (should fix)

- **[defensive meter-sync can't see the store type it targets]** [`machine.go:201-205`](https://github.com/gnolang/gno/blob/bac5ab8d1/gnovm/pkg/gnolang/machine.go#L201-L205) · [↗](../../../../../.worktrees/gno-review-5710/gnovm/pkg/gnolang/machine.go#L201-L205) — `store.(*defaultStore)` never matches the production wrapper `transactionStore{*defaultStore}`, so the block silently no-ops on every wrapped store; its comment ("the store must too — otherwise store-routed charges silently drop") promises a guarantee it does not provide. Fix: add `SetMeter` to the `Store` interface and call it unconditionally, or remove the block and rely on `BeginTransaction`.
  <details><summary>details</summary>

  **Shape:** `transactionStore` embeds `*defaultStore` for method promotion but is a distinct concrete type; the assert returns `ok=false`. A Machine built with `m.GasMeter` set but a `transactionStore` whose meter is nil ends up with `m.Store.GetMeter() == nil`, and `ComputeMapKey` charges zero.

  **Impact (corrected from round 1):** no gas-enforcing or consensus path is affected. Production installs the meter at `BeginTransaction` ([`keeper.go:395-397`](https://github.com/gnolang/gno/blob/bac5ab8d1/gno.land/pkg/sdk/vm/keeper.go#L395-L397) · [↗](../../../../../.worktrees/gno-review-5710/gno.land/pkg/sdk/vm/keeper.go#L395-L397)), and the gas-pinning filetests under `gnovm/tests/files/gas/` run via `runFiletest`, which also passes the meter at [`filetest.go:83`](https://github.com/gnolang/gno/blob/bac5ab8d1/gnovm/pkg/test/filetest.go#L83) · [↗](../../../../../.worktrees/gno-review-5710/gnovm/pkg/test/filetest.go#L83), so `store.GetMeter()` is non-nil and they charge correctly (those tests are `package main` and skip the realm re-wrap). The only foothold is the `gno test` unit path ([`test.go:335`](https://github.com/gnolang/gno/blob/bac5ab8d1/gnovm/pkg/test/test.go#L335) · [↗](../../../../../.worktrees/gno-review-5710/gnovm/pkg/test/test.go#L335) creates the tx store with a nil meter, [`test.go:513`](https://github.com/gnolang/gno/blob/bac5ab8d1/gnovm/pkg/test/test.go#L513) · [↗](../../../../../.worktrees/gno-review-5710/gnovm/pkg/test/test.go#L513) gives the Machine an `InfiniteGasMeter`) and the realm-filetest restore phase ([`filetest.go:454`](https://github.com/gnolang/gno/blob/bac5ab8d1/gnovm/pkg/test/filetest.go#L454) · [↗](../../../../../.worktrees/gno-review-5710/gnovm/pkg/test/filetest.go#L454) re-wraps with a nil meter): both undercharge `ComputeMapKey`, but the meter is infinite and the `--- GAS:` line ([`test.go:613`](https://github.com/gnolang/gno/blob/bac5ab8d1/gnovm/pkg/test/test.go#L613) · [↗](../../../../../.worktrees/gno-review-5710/gnovm/pkg/test/test.go#L613)) is `Fprintf`-only, never asserted. So `gno test` gas estimates for map-heavy realms read low vs on-chain, which is the should-fix.

  **Latent footgun:** the assert *does* match a bare `*defaultStore`. The shared base `vm.gnoStore` is a bare `*defaultStore` ([`keeper.go:158-164`](https://github.com/gnolang/gno/blob/bac5ab8d1/gno.land/pkg/sdk/vm/keeper.go#L158-L164) · [↗](../../../../../.worktrees/gno-review-5710/gno.land/pkg/sdk/vm/keeper.go#L158-L164)); it is never paired with a meter today, but if a future caller ever builds a metered Machine on it, this block would persist a tx-scoped meter into the long-lived shared store.

  Fix: `SetMeter(store.GasMeter)` on the interface (mirrors `SetAllocator`) called unconditionally, which also lets `gno test` match on-chain gas; or drop the block and document `gno test` gas as informational. Verified the block is dead today: removing it leaves all `gnovm/pkg/gnolang` tests green and `benchMachineWithGas` still charging (it sets `ds.gasMeter` itself). Repro of the gap is the round-1 adversarial test (`../1-55d874f6/tests/transactionstore_meter_sync_test.go`), which passes the IS-state assertions on this head.
  </details>

- **[restore-path gas charge is an unresolved design decision]** [`realm.go:1879`](https://github.com/gnolang/gno/blob/bac5ab8d1/gnovm/pkg/gnolang/realm.go#L1879) · [↗](../../../../../.worktrees/gno-review-5710/gnovm/pkg/gnolang/realm.go#L1879) — [@thehowl](https://github.com/gnolang/gno/pull/5710#discussion_r3341682416) left an open, unanswered comment on this exact line reading that the restore path is "more correct ... to use a `nil` gasMeter" — i.e. it should not charge — directly opposing the PR's premise. This is a consensus-affecting gas increase (any realm persisting a non-trivial map pays more after this lands), so the charge-vs-don't decision should be settled in-thread before merge.
  <details><summary>details</summary>

  The factual half of the concern is confirmed: across the three states (pre-#5127: neither charges; master: write charges, restore zero; this PR: both charge), this PR strictly increases gas for cold map loads. The change is not height- or version-gated (the constants are unconditional), which matches gno's coordinated-upgrade model but means historical replay cost changes. The comment is ambiguous (it could be read as merely flagging the bump), but the stronger reading attaches "more correct" to "nil gasMeter," so I'm treating the direction as genuinely contested. Not a code defect; a maintainer decision.
  </details>

- **[meter-source API shape contested]** [`values.go:1601`](https://github.com/gnolang/gno/blob/bac5ab8d1/gnovm/pkg/gnolang/values.go#L1601) · [↗](../../../../../.worktrees/gno-review-5710/gnovm/pkg/gnolang/values.go#L1601) — [@thehowl](https://github.com/gnolang/gno/pull/5710#discussion_r3341663206) prefers `ComputeMapKey` take a `GasMeter` parameter directly rather than widening `Store`, so the signature communicates both permissions (Store for data, GasMeter to count). Open and unanswered. Resolving toward the explicit parameter would also delete the `GetMeter()` accessor and the defensive sync above, collapsing three findings at once; resolving toward the Store accessor is defensible but should be a deliberate, documented choice given the maintainer's stated preference.

## Nits

- [`store.go:73-77`](https://github.com/gnolang/gno/blob/bac5ab8d1/gnovm/pkg/gnolang/store.go#L73-L77) · [↗](../../../../../.worktrees/gno-review-5710/gnovm/pkg/gnolang/store.go#L73-L77) — the `GetMeter` doc says "Passive accessor — Store does not charge gas itself," but `defaultStore.consumeGas` ([`store.go:1104-1108`](https://github.com/gnolang/gno/blob/bac5ab8d1/gnovm/pkg/gnolang/store.go#L1104-L1108) · [↗](../../../../../.worktrees/gno-review-5710/gnovm/pkg/gnolang/store.go#L1104-L1108)) charges amino encode/decode gas through `ds.gasMeter`. The accessor is passive; the sentence over-claims about the Store. Tighten to "this accessor does not itself charge."
- [`values.go:1597`](https://github.com/gnolang/gno/blob/bac5ab8d1/gnovm/pkg/gnolang/values.go#L1597) · [↗](../../../../../.worktrees/gno-review-5710/gnovm/pkg/gnolang/values.go#L1597) — `ComputeMapKey` now dereferences `store.GetMeter()` unconditionally, so a `nil` store panics for every key shape (master tolerated a nil store for the `tv.T == nil` and primitive cases). No caller passes nil today (verified by grep), so this is a doc-comment "store must not be nil," not a code change. No change required.

## Missing Tests

- **[symmetry contract not pinned]** [`values_test.go:409-435`](https://github.com/gnolang/gno/blob/bac5ab8d1/gnovm/pkg/gnolang/values_test.go#L409-L435) · [↗](../../../../../.worktrees/gno-review-5710/gnovm/pkg/gnolang/values_test.go#L409-L435) — `TestComputeMapKey_GasViaStoreMeter` only lower-bounds `delta >= OpCPUComputeMapKey` for a single `int` key and checks the unmetered store doesn't panic. It never charges the *same* key on a write-shaped and a restore-shaped call and asserts equality — the actual "symmetric" contract. A composite (struct/array/byte) key would also exercise the slope and recursion that the scalar case skips. Add a case asserting write-path and restore-path deltas are equal for a multi-field struct key.
  <details><summary>details</summary>

  Symmetry holds structurally today (both call the same `ComputeMapKey(store, ...)`), and I verified it experimentally (identical 84-gas charge on both call shapes for one key). But nothing pins it: a future path-specific branch would not be caught. The slope term and per-child constant only matter for composite keys — exactly the DoS shape the slope was added to bound — yet the unit test uses a scalar; that path is covered only by the brittle absolute-pin txtar.
  </details>

- **[meter-sync untested]** [`machine.go:201-205`](https://github.com/gnolang/gno/blob/bac5ab8d1/gnovm/pkg/gnolang/machine.go#L201-L205) · [↗](../../../../../.worktrees/gno-review-5710/gnovm/pkg/gnolang/machine.go#L201-L205) — no test reaches the defensive sync; `TestComputeMapKey_GasViaStoreMeter` and `benchMachineWithGas` both set `ds.gasMeter` by hand and bypass it. A test that builds `NewMachineWithOptions` over a `transactionStore` with a nil meter and asserts `m.Store.GetMeter() != nil` afterward would fail today (the Warning) and lock in the fix. (See `../1-55d874f6/tests/transactionstore_meter_sync_test.go`.)

## Suggestions

- [`compute_map_key_restore_gas.txtar:35`](https://github.com/gnolang/gno/blob/bac5ab8d1/gno.land/pkg/integration/testdata/compute_map_key_restore_gas.txtar#L35) · [↗](../../../../../.worktrees/gno-review-5710/gno.land/pkg/integration/testdata/compute_map_key_restore_gas.txtar#L35), [`:47`](https://github.com/gnolang/gno/blob/bac5ab8d1/gno.land/pkg/integration/testdata/compute_map_key_restore_gas.txtar#L47) · [↗](../../../../../.worktrees/gno-review-5710/gno.land/pkg/integration/testdata/compute_map_key_restore_gas.txtar#L47) — the pins are full-tx `GAS USED` totals, so any unrelated gas-model drift on master reds this test even when the restore fix is intact (the same failure mode that left `compute_map_key_big_bytes/small_bytes` off by +113 on master). The pins prove "today's total equals today's total," not the asymmetry. A differential assertion (two txs differing by exactly one cold entry, asserting the per-entry delta) would machine-check the property the test documents instead of coupling it to the whole gas model.
  <details><summary>details</summary>

  The header's "removing the restore charge shifts Tx1 by ~3940 / Tx2 by ~4728" is reasoned in prose but only the two absolute totals are asserted; if `init()` seeded a different entry count the prose would silently go stale while re-pinned totals still "pass." The prose is already slightly off: measured, Tx2's full restore charge is ~5,516, not the header's 6 × 788 = 4,728 — the 3-field Insert key costs ~1,576, not 788, so "6 × per-entry" with a single per-entry value undercounts. Tie the claim to an assertion so a maintainer re-pinning after a benign drift can't mask a real regression, and so the prose can't drift from the gas it describes.
  </details>

## Open questions

- The realm-filetest restore phase ([`filetest.go:454`](https://github.com/gnolang/gno/blob/bac5ab8d1/gnovm/pkg/test/filetest.go#L454) · [↗](../../../../../.worktrees/gno-review-5710/gnovm/pkg/test/filetest.go#L454)) re-wraps the store with a nil meter, so map-heavy realm filetests undercharge `ComputeMapKey` — harmless today (no realm filetest pins gas), but it's the second instance of the same root cause as the Warning. Folds into the `SetMeter` fix. Not posted; no current consequence.
- `delete(m, k)` charges `ComputeMapKey` twice (via `GetValueForKey` then `DeleteForKey`); pre-existing on master, unchanged by this PR. Not posted.
