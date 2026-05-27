# PR #4707: feat: total supply invariant check

URL: https://github.com/gnolang/gno/pull/4707
Author: piux2 | Base: master | Files: 18 | +521 -39
Reviewed by: davd-gzl | Model: claude-opus-4-7
Local worktree: `git -C gno worktree add .worktrees/gno-review-4707 03a2bc7` (then `gh -R gnolang/gno pr checkout 4707` inside it)

**Verdict: REQUEST CHANGES** — the invariant is a silent no-op because writer and reader disagree on the storage key, the comparator (`inc.Sub(dec) != nil`) panics on burns instead of reporting "broken", state is never cleared between blocks (cumulative storage growth), `IsCheckTx`-only skip lets ReCheck/Simulate corrupt counters, and the branch is six months behind master so the diff reverts unrelated features (session accounts, hardfork mechanism, gas calibration, valoper coverage).

## Summary

The PR adds a bank-level "total supply" invariant: every block, the sum of balance increases across tracked denoms must equal the sum of decreases (no minting, no burning post-genesis). `SubtractCoins`/`AddCoins` call `trackBalanceChange`, which diffs old vs new coins, intersects with `Params.TotalSupply` denoms, and writes running `balanceIncrease`/`balanceDecrease` totals to the bank store. At `EndBlock`, `BalanceChangeInvariant` reads both keys back and panics if they differ. A separate `TotalSupplyInvariant` runs once at genesis (`BlockHeight == 0`) to verify `sum(balances) == Params.TotalSupply`.

The intent is sound but the mechanism is wired wrong end-to-end:

```
writer (addBalanceChanges)  ─▶  store key:  "balanceIncrease"     "balanceDecrease"
                                              │                      │
                                              ▼                      ▼
reader (BalanceChangeInvariant) ◀─  store key:  "/bk/balanceIncrease" "/bk/balanceDecrease"
                                              ▲                      ▲
                                              └── never the bytes the writer wrote ──┘
```

## Glossary

- `addBalanceChanges` — tracker that accumulates inc/dec totals in the bank store on every `AddCoins`/`SubtractCoins`.
- `BalanceChangeInvariant` — end-block check that reads inc/dec and panics on mismatch.
- `TotalSupplyInvariant` — genesis-only check; iterates all accounts and compares against `Params.TotalSupply`.
- `storeKey(s)` — helper that prepends `StoreKeyPrefix = "/bk/"`.
- `diffCoins(old, new, denomSet)` — splits a balance delta into per-denom inc/dec, restricted to denoms in `TotalSupply`.

## Fix

`SubtractCoins`/`AddCoins` ([`tm2/pkg/sdk/bank/keeper.go:200-242`](https://github.com/gnolang/gno/blob/03a2bc7/tm2/pkg/sdk/bank/keeper.go#L200-L242) · [↗](../../../../../.worktrees/gno-review-4707/tm2/pkg/sdk/bank/keeper.go#L200-L242)) gain a `trackBalanceChange` call; `diffCoins` ([`tm2/pkg/sdk/bank/keeper.go:307-367`](https://github.com/gnolang/gno/blob/03a2bc7/tm2/pkg/sdk/bank/keeper.go#L307-L367) · [↗](../../../../../.worktrees/gno-review-4707/tm2/pkg/sdk/bank/keeper.go#L307-L367)) does a sorted merge restricted to tracked denoms. `BalanceChangeInvariant` ([`tm2/pkg/sdk/bank/invariants.go:81-104`](https://github.com/gnolang/gno/blob/03a2bc7/tm2/pkg/sdk/bank/invariants.go#L81-L104) · [↗](../../../../../.worktrees/gno-review-4707/tm2/pkg/sdk/bank/invariants.go#L81-L104)) reads the totals and panics in `EndBlocker` ([`tm2/pkg/sdk/bank/abci.go:10-17`](https://github.com/gnolang/gno/blob/03a2bc7/tm2/pkg/sdk/bank/abci.go#L10-L17) · [↗](../../../../../.worktrees/gno-review-4707/tm2/pkg/sdk/bank/abci.go#L10-L17)) on mismatch. `BankKeeper` now requires a `StoreKey` (was: keeper-less) so `ctx.Store(bank.key)` can read/write the tracker. `gno.land/pkg/gnoland/app.go` and the test plumbing thread the new `mainKey` through bank construction. Genesis sets `Bank.Params.TotalSupply = TotalBalance(balances)` in every test setup so `TotalSupplyInvariant` passes at block 0.

## Critical (must fix)

- **[invariant is a no-op — silently passes printed tokens]** [`tm2/pkg/sdk/bank/keeper.go:288,296`](https://github.com/gnolang/gno/blob/03a2bc7/tm2/pkg/sdk/bank/keeper.go#L288-L296) · [↗](../../../../../.worktrees/gno-review-4707/tm2/pkg/sdk/bank/keeper.go#L288-L296) vs [`tm2/pkg/sdk/bank/invariants.go:86,90`](https://github.com/gnolang/gno/blob/03a2bc7/tm2/pkg/sdk/bank/invariants.go#L86-L90) · [↗](../../../../../.worktrees/gno-review-4707/tm2/pkg/sdk/bank/invariants.go#L86-L90) — writer uses raw `[]byte(balanceIncKey)` / `[]byte(balanceDecKey)`; reader uses `storeKey(...)` which prepends `StoreKeyPrefix = "/bk/"`.
  <details><summary>details</summary>

  Reproduced as a unit test against the PR HEAD: one-sided `AddCoins(addr, 100foo)` writes under `"balanceIncrease"`; `BalanceChangeInvariant` reads under `"/bk/balanceIncrease"`, finds nothing, returns `broken=false`. The chain happily lets a tracked denom be minted out of thin air with no halt. The full feature is a no-op in production. Adversarial test: [`reviews/pr/4xxx/4707-total-supply-invariant/1-03a2bc7/tests/keyprefix_mismatch_test.go`](tests/keyprefix_mismatch_test.go). Fix: pick one convention and use it everywhere — either drop `storeKey(...)` in `invariants.go` and read raw keys, or wrap both reads and writes with `storeKey(...)`. The existing `TestTrackBalanceChange` only validates the writer side via `readCounters` using raw keys, which is why the bug slipped through.
  </details>

  Repro:

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 4707 -R gnolang/gno
  cat > tm2/pkg/sdk/bank/keyprefix_repro_test.go <<'EOF'
  package bank
  import (
      "testing"
      "github.com/gnolang/gno/tm2/pkg/crypto"
      "github.com/gnolang/gno/tm2/pkg/std"
  )
  func TestKeyPrefixMismatchRepro(t *testing.T) {
      env := setupTestEnv(); ctx := env.ctx
      p := env.bankk.GetParams(ctx)
      p.TotalSupply = std.NewCoins(std.NewCoin("foo", 1000))
      env.bankk.SetParams(ctx, p)
      a := crypto.AddressFromPreimage([]byte("a"))
      env.acck.SetAccount(ctx, env.acck.NewAccountWithAddress(ctx, a))
      env.bankk.AddCoins(ctx, a, std.NewCoins(std.NewCoin("foo", 100)))
      _, broken := BalanceChangeInvariant(env.bankk)(ctx)
      if broken { t.Fatal("repro fails: invariant flagged the inflation") }
      t.Log("BUG: invariant silently passed a one-sided AddCoins")
  }
  EOF
  go test -v -run TestKeyPrefixMismatchRepro ./tm2/pkg/sdk/bank/
  rm tm2/pkg/sdk/bank/keyprefix_repro_test.go
  ```

- **[invariant check panics on burn instead of reporting "broken"]** [@jaekwon](https://github.com/gnolang/gno/pull/4707#pullrequestreview-3158795310) [`tm2/pkg/sdk/bank/invariants.go:94-100`](https://github.com/gnolang/gno/blob/03a2bc7/tm2/pkg/sdk/bank/invariants.go#L94-L100) · [↗](../../../../../.worktrees/gno-review-4707/tm2/pkg/sdk/bank/invariants.go#L94-L100) — `balanceInc.Sub(balanceDec) != nil` calls `Coins.Sub`, which panics whenever the result has any negative coin.
  <details><summary>details</summary>

  `Coins.Sub` ([`tm2/pkg/std/coin.go:372-378`](https://github.com/gnolang/gno/blob/03a2bc7/tm2/pkg/std/coin.go#L372-L378) · [↗](../../../../../.worktrees/gno-review-4707/tm2/pkg/std/coin.go#L372-L378)) wraps `SubUnsafe` and `panic`s if the result is not valid. When tokens are burned (dec > inc), or when inc and dec touch disjoint denoms, `Sub` panics with `"invalid result: %v - %v = %v"`, not with the invariant-broken message. The chain halts either way, but the operator sees `panic: invalid result: 5foo - 10foo = -5foo` instead of `invariant broken: sum of balance increase ... != sum of balance decrease ...`. Verified by direct call: `std.NewCoins(std.NewCoin("foo",5)).Sub(std.NewCoins(std.NewCoin("foo",10)))` panics. Same for disjoint denoms (`5foo.Sub(5bar)` panics). Fix: replace `balanceInc.Sub(balanceDec) != nil` with `!balanceInc.IsEqual(balanceDec)`; the comparator does what the comment claims and never panics. Also, after detecting the mismatch, format the actual diff with `Sub` only if `inc >= dec` (or do per-denom diffs).
  </details>

- **[inc/dec counters are never cleared between blocks]** [`tm2/pkg/sdk/bank/keeper.go:264-266`](https://github.com/gnolang/gno/blob/03a2bc7/tm2/pkg/sdk/bank/keeper.go#L264-L266) · [↗](../../../../../.worktrees/gno-review-4707/tm2/pkg/sdk/bank/keeper.go#L264-L266) vs [`tm2/pkg/sdk/bank/abci.go:10-17`](https://github.com/gnolang/gno/blob/03a2bc7/tm2/pkg/sdk/bank/abci.go#L10-L17) · [↗](../../../../../.worktrees/gno-review-4707/tm2/pkg/sdk/bank/abci.go#L10-L17) — the comment says "the kv will be deleted if invariant check passed", but `EndBlocker` only runs the check; nothing deletes the keys.
  <details><summary>details</summary>

  Cumulative inc and dec grow unboundedly over the chain's lifetime. For a long-running chain that's a permanent, monotonically growing pair of `Coins` values in IAVL — every block re-marshals and re-writes both keys, the IAVL tree gains nodes proportional to the magnitude of total volume, and the comparison `inc == dec` stays mathematically equivalent only as long as no block ever broke (after which it stays broken forever, but the chain has already panicked). Functionally tolerable, but: (a) IAVL/PebbleDB bloat for what should be a per-block tally, (b) once an exploit breaks invariance the operator can't restart cleanly without manually editing state, (c) jaekwon's design comment [#4707-comment](https://github.com/gnolang/gno/pull/4707#issuecomment-3550127378) calls for `AssertBalanced` that resets between calls — adopt that or at minimum delete both keys at the end of `EndBlocker`.
  </details>

- **[`IsCheckTx`-only skip silently corrupts counters in ReCheck/Simulate]** [`tm2/pkg/sdk/bank/keeper.go:267-271`](https://github.com/gnolang/gno/blob/03a2bc7/tm2/pkg/sdk/bank/keeper.go#L267-L271) · [↗](../../../../../.worktrees/gno-review-4707/tm2/pkg/sdk/bank/keeper.go#L267-L271) — the tracker bails on `ctx.IsCheckTx()` but the SDK has more non-deliver modes.
  <details><summary>details</summary>

  `sdk.RunTxMode` has `RunTxModeCheck`, `RunTxModeReCheck`, `RunTxModeSimulate`, `RunTxModeDeliver`. `IsCheckTx()` returns true only for `Check`; `ReCheck` and `Simulate` execute the same keeper paths and would credit inc/dec into the persisted store, breaking the next `EndBlock`. Wallets calling `simulate` (gas estimation) and the consensus engine's recheck pass on mempool eviction both trigger this. Fix: skip whenever `ctx.Mode() != sdk.RunTxModeDeliver` (positive check), or at minimum `IsCheckTx() || ctx.Mode() == sdk.RunTxModeReCheck || ctx.Mode() == sdk.RunTxModeSimulate`.
  </details>

- **[PR reverts ~6 months of master in unrelated files]** [`gno.land/pkg/gnoland/app.go:1-263`](https://github.com/gnolang/gno/blob/03a2bc7/gno.land/pkg/gnoland/app.go#L1-L263) · [↗](../../../../../.worktrees/gno-review-4707/gno.land/pkg/gnoland/app.go#L1-L263), [`contribs/gnodev/setup_node.go:7-145`](https://github.com/gnolang/gno/blob/03a2bc7/contribs/gnodev/setup_node.go#L7-L145) · [↗](../../../../../.worktrees/gno-review-4707/contribs/gnodev/setup_node.go#L7-L145), [`tm2/pkg/std/coin_test.go`](https://github.com/gnolang/gno/blob/03a2bc7/tm2/pkg/std/coin_test.go) · [↗](../../../../../.worktrees/gno-review-4707/tm2/pkg/std/coin_test.go), [`gno.land/pkg/gnoland/app_test.go`](https://github.com/gnolang/gno/blob/03a2bc7/gno.land/pkg/gnoland/app_test.go) · [↗](../../../../../.worktrees/gno-review-4707/gno.land/pkg/gnoland/app_test.go) — the branch was opened 2025-09-01 and last touched 2025-09-01; rebasing against current master shows the diff stripping `ProtoGnoSessionAccount`, `nodeParamsKeeper`, `checkSessionRestrictions`, `checkNodeStartupParams`, `SkipUpgradeHeight`, VM gas-config application, `PruneSyncableStrategy` default, the entire `TestShouldAssertValoperCoverage`, `TestNewAppWithOptions_ErrNoLogger`/`ErrNoEventSwitch`, `TestCoinsValidate`, `TestValidate`, the `gnodev` deploy-key plumbing, the example-package helpers in `vm/common_test.go`, etc.
  <details><summary>details</summary>

  CI confirms: `Run Main (gnodev) / Go Test`, `Run Main (gnogenesis) / Go Test`, `Run gno.land suite / Go Test`, and `e2e-test` all fail on this branch. The reversions are not part of the feature; they're bit-rot. Anyone landing this would unintentionally remove session accounts (#5614), the chain hardfork mechanism (#5511), the per-native gas calibration / per-realm storage deposit (#5629), the valset-via-VM-params (#5485), the genesis valoper-coverage assertion (#5701), the `/genesis` RPC streaming (#5684), and many other features. Fix: rebase onto current master and resolve every conflict, restoring (not deleting) the master-side code; verify the diff stat shrinks to the bank module + a thin `gno.land` wiring change.
  </details>

## Warnings (should fix)

- **[performance: `denoms` rebuilt and `params` re-read on every coin op]** [@moul](https://github.com/gnolang/gno/pull/4707#discussion_r2353123010) [@jaekwon](https://github.com/gnolang/gno/pull/4707#discussion_r2553260017) [`tm2/pkg/sdk/bank/keeper.go:267-281`](https://github.com/gnolang/gno/blob/03a2bc7/tm2/pkg/sdk/bank/keeper.go#L267-L281) · [↗](../../../../../.worktrees/gno-review-4707/tm2/pkg/sdk/bank/keeper.go#L267-L281) — `trackBalanceChange` reads `Params` and constructs `denoms` + `stringSet` on every `AddCoins`/`SubtractCoins`.
  <details><summary>details</summary>

  `bank.GetParams(ctx)` walks the params store; the result is then reduced to a `[]string` and then to a `stringSet`. Both allocations happen for every coin op, including the most hot path on the chain (gas payment in `SendCoinsUnrestricted` plus every transfer). For a TotalSupply with N tracked denoms, the per-op overhead is O(N) allocs plus the `diffCoins` O(M+N) merge over old/new coins. Cache the denom set on `WillSetParam` (already a hook for restricted denoms) or memoize per-block in the context. Even better: pass tracked denoms in directly to `addBalanceChanges` from callers that know what denom they're moving (the suggestion in [@ltzmaxwell's comment](https://github.com/gnolang/gno/pull/4707#discussion_r2352855128)).
  </details>

- **[no path for new tokens or burn/mint operations]** [@moul](https://github.com/gnolang/gno/pull/4707#pullrequestreview-3239001384) [`tm2/pkg/sdk/bank/invariants.go:106`](https://github.com/gnolang/gno/blob/03a2bc7/tm2/pkg/sdk/bank/invariants.go#L106) · [↗](../../../../../.worktrees/gno-review-4707/tm2/pkg/sdk/bank/invariants.go#L106) — only denoms in `Params.TotalSupply` are tracked, and the invariant assumes fixed supply.
  <details><summary>details</summary>

  The PR description acknowledges this ("Tokens issued by gnoVM contracts are ignored") but the implementation hard-blocks any future `Mint`/`Burn` operation in the bank module without a parallel update to `Params.TotalSupply`. The PR's name (`TotalSupplyInvariant`, `BalanceChangeInvariant`) suggests universality; the behavior is "fixed-supply only". Either rename to `FixedSupplyInvariant`/`Params.FixedSupplyDenoms` (and document the policy that new tokens go through `Params` updates), or add `RegisterMint`/`RegisterBurn` hooks (as jaekwon proposes) so issuance/destruction can be authorized and accounted for.
  </details>

- **[`TotalSupplyInvariant` reads every account at genesis with no gas/iter cap]** [`tm2/pkg/sdk/bank/invariants.go:53-75`](https://github.com/gnolang/gno/blob/03a2bc7/tm2/pkg/sdk/bank/invariants.go#L53-L75) · [↗](../../../../../.worktrees/gno-review-4707/tm2/pkg/sdk/bank/invariants.go#L53-L75) — `auth.GetAllAccounts(ctx)` materializes every genesis account in one allocation.
  <details><summary>details</summary>

  Acceptable for current gno.land (~3k accounts) but the function is exported and may be called outside InitChainer. The early-return on `ctx.BlockHeight() != 0` makes it safe by convention only — anyone wiring it into a periodic check would scan every account every block. Document the intent ("genesis-only; do not call at runtime") or guard with a panic on `BlockHeight() != 0` instead of silently returning `false`.
  </details>

- **[`Validate` rejects zero-amount coins but `Params` allows TotalSupply per-coin zero check inconsistent]** [`tm2/pkg/sdk/bank/params.go:54-58`](https://github.com/gnolang/gno/blob/03a2bc7/tm2/pkg/sdk/bank/params.go#L54-L58) · [↗](../../../../../.worktrees/gno-review-4707/tm2/pkg/sdk/bank/params.go#L54-L58) — `coin.IsZero()` rejection contradicts `DefaultParams()` if a chain wants tracked denoms with no initial supply.
  <details><summary>details</summary>

  Zero-supply tracked denoms are useful for "will-be-issued-through-genesis-state" semantics. The validation hardcodes "no zero amounts", which makes denoms registered in `Params.TotalSupply` strictly equal the genesis sum. Combined with the genesis test pattern `Bank.Params.TotalSupply = TotalBalance(balances)`, this means TotalSupply is purely derivative and serves no purpose as a configurable parameter. Either drop `IsZero()` and let zero-supply denoms be tracked-but-unissued, or drop the parameter and compute it from balances at genesis.
  </details>

- **[redundant `ProcessConfig`-style state in `addBalanceChanges`: `IsZero()` skip never deletes]** [`tm2/pkg/sdk/bank/keeper.go:293-303`](https://github.com/gnolang/gno/blob/03a2bc7/tm2/pkg/sdk/bank/keeper.go#L293-L303) · [↗](../../../../../.worktrees/gno-review-4707/tm2/pkg/sdk/bank/keeper.go#L293-L303) — `if !balanceInc.IsZero() { store.Set(...) }`, no else.
  <details><summary>details</summary>

  Once `balanceInc` has been non-zero and is written to disk, a subsequent block where the accumulated inc would naturally become zero (impossible currently since inc only grows, but assume the model adds a clear step) cannot zero the store entry — the writer's guard skips the write. If you later adopt jaekwon's reset semantics, this guard becomes a bug. Either delete the entry on zero (`store.Delete(incKey)`) or unconditionally write.
  </details>

- **[testscript invariant test is shallow]** [@ltzmaxwell](https://github.com/gnolang/gno/pull/4707#discussion_r2353111301) [`gno.land/pkg/integration/testdata/total_supply.txtar`](https://github.com/gnolang/gno/blob/03a2bc7/gno.land/pkg/integration/testdata/total_supply.txtar) · [↗](../../../../../.worktrees/gno-review-4707/gno.land/pkg/integration/testdata/total_supply.txtar) — 7 lines, only checks that the `total_supply` parameter is queryable.
  <details><summary>details</summary>

  Does not exercise the actual invariant: no `gnokey maketx send` to verify the chain halts on a forced break (e.g. patched binary that skips `SubtractCoins`), no recovery path test, no multi-denom scenario, no test that genesis halts when balances don't sum to `Params.TotalSupply`. Given this is a critical security primitive, the integration coverage should be commensurate.
  </details>

- **[`@moul`'s suggestion to merge `trackBalanceChange`/`addBalanceChanges` was rejected for taste reasons, but the split is confusing]** [@moul](https://github.com/gnolang/gno/pull/4707#discussion_r2353128989) [`tm2/pkg/sdk/bank/keeper.go:267-305`](https://github.com/gnolang/gno/blob/03a2bc7/tm2/pkg/sdk/bank/keeper.go#L267-L305) · [↗](../../../../../.worktrees/gno-review-4707/tm2/pkg/sdk/bank/keeper.go#L267-L305) — `trackBalanceChange` is a thin wrapper that diffs then calls `addBalanceChanges`.
  <details><summary>details</summary>

  The two functions are private and called only from each other / from `AddCoins`/`SubtractCoins`. The split adds a function-call hop without separating concerns (both touch the same store, both filter by the same denom set). Suggested rename per jaekwon `addInc`/`addDec` if kept separate, or merge into one `tally` function.
  </details>

## Nits

- [`tm2/pkg/sdk/bank/invariants.go:98`](https://github.com/gnolang/gno/blob/03a2bc7/tm2/pkg/sdk/bank/invariants.go#L98) · [↗](../../../../../.worktrees/gno-review-4707/tm2/pkg/sdk/bank/invariants.go#L98) — copy-paste typo: `"sum of balance increase %v != sum of balance increase %v"` should read `"... != sum of balance decrease %v"` (already flagged by @moul, [comment](https://github.com/gnolang/gno/pull/4707#discussion_r2353159432)).
- [`tm2/pkg/sdk/bank/invariants.go:12-18`](https://github.com/gnolang/gno/blob/03a2bc7/tm2/pkg/sdk/bank/invariants.go#L12-L18) · [↗](../../../../../.worktrees/gno-review-4707/tm2/pkg/sdk/bank/invariants.go#L12-L18) — `balanceIncKey`/`balanceDecKey` constants are file-local in `invariants.go` but only used by `keeper.go`; co-locate next to the writer in `keeper.go` (per @moul).
- [`tm2/pkg/sdk/bank/params.go:16`](https://github.com/gnolang/gno/blob/03a2bc7/tm2/pkg/sdk/bank/params.go#L16) · [↗](../../../../../.worktrees/gno-review-4707/tm2/pkg/sdk/bank/params.go#L16) — `TotalSupply` const should be `DefaultTotalSupply` for clarity (per @moul, already a suggestion); the field is also misleadingly named — it's a default _placeholder_, not a real supply.
- [`tm2/pkg/sdk/bank/keeper.go:73-82`](https://github.com/gnolang/gno/blob/03a2bc7/tm2/pkg/sdk/bank/keeper.go#L73-L82) · [↗](../../../../../.worktrees/gno-review-4707/tm2/pkg/sdk/bank/keeper.go#L73-L82) — `stringSet`/`toSet` is duplicated logic; the `params.WillSetParam` registration already exists for `restricted_denoms`, so the same caching hook should apply for `total_supply`.
- [`tm2/pkg/sdk/bank/keeper.go:315`](https://github.com/gnolang/gno/blob/03a2bc7/tm2/pkg/sdk/bank/keeper.go#L315) · [↗](../../../../../.worktrees/gno-review-4707/tm2/pkg/sdk/bank/keeper.go#L315) — typo: `demons` should be `denoms`.
- [`gno.land/pkg/integration/testdata/total_supply.txtar:5`](https://github.com/gnolang/gno/blob/03a2bc7/gno.land/pkg/integration/testdata/total_supply.txtar#L5) · [↗](../../../../../.worktrees/gno-review-4707/gno.land/pkg/integration/testdata/total_supply.txtar#L5) — add a header comment explaining what the test validates (per @ltzmaxwell).
- `tm2/pkg/sdk/bank/keeper.go` import block: `amino` is now imported in the keeper; that's the first dependency of the bank keeper on amino — historically amino-stringification of params was localized to module boundaries. Consider whether `addBalanceChanges` should live behind a smaller abstraction.

## Missing Tests

- **[no test for the actual `BalanceChangeInvariant` end-to-end]** [`tm2/pkg/sdk/bank/keeper_test.go`](https://github.com/gnolang/gno/blob/03a2bc7/tm2/pkg/sdk/bank/keeper_test.go) · [↗](../../../../../.worktrees/gno-review-4707/tm2/pkg/sdk/bank/keeper_test.go) — every assertion in `TestTrackBalanceChange` is on the writer's counters via `readCounters` (raw keys); none exercise `BalanceChangeInvariant(env.bankk)(ctx)`.
  <details><summary>details</summary>

  Had a single test called `inv := BalanceChangeInvariant(env.bankk); _, broken := inv(ctx); require.False(t, broken)` been written, it would have caught both the prefix-mismatch bug AND the `inc.Sub(dec) != nil` semantic bug. The omission is the load-bearing reason the feature shipped non-functional. See [`reviews/pr/4xxx/4707-total-supply-invariant/1-03a2bc7/tests/keyprefix_mismatch_test.go`](tests/keyprefix_mismatch_test.go) for a minimal version.
  </details>

- **[no test for genesis-halt when balances don't sum to `Params.TotalSupply`]** [`gno.land/pkg/gnoland/app.go:438-443`](https://github.com/gnolang/gno/blob/03a2bc7/gno.land/pkg/gnoland/app.go#L438-L443) · [↗](../../../../../.worktrees/gno-review-4707/gno.land/pkg/gnoland/app.go#L438-L443) — `TotalSupplyInvariant` is wired into `loadAppState`, but nothing verifies that a genesis with mismatched balances actually panics.
- **[no test that Simulate/ReCheck don't corrupt the tracker]** — `TestTrackBalanceChange` only exercises `RunTxModeCheck` (correctly skipped) and `RunTxModeDeliver` (counted). Add modes Simulate and ReCheck.
- **[no test for ABCI EndBlocker panic path]** [`tm2/pkg/sdk/bank/abci.go:10-17`](https://github.com/gnolang/gno/blob/03a2bc7/tm2/pkg/sdk/bank/abci.go#L10-L17) · [↗](../../../../../.worktrees/gno-review-4707/tm2/pkg/sdk/bank/abci.go#L10-L17) — the panic on invariant break is the chain-halt mechanism but has no coverage.

## Suggestions

- `tm2/pkg/sdk/bank/keeper.go:267` — adopt jaekwon's `AssertBalanced`/`AssertAdded`/`AssertSubtracted` API ([comment](https://github.com/gnolang/gno/pull/4707#issuecomment-3550127378)) instead of per-op tally + EndBlock check. Lets contracts assert sub-tx invariants without needing global state.
- `tm2/pkg/sdk/bank/keeper.go:200-242` — for a single coin op, the writer reads params, sorts old & new coins, intersects denoms, walks two sorted slices, and writes two store entries. The hot path is fee payment on every tx. Even with the perf nit fixed, consider whether the tracker should run only on `SendCoins`/`InputOutputCoins` (user-facing transfers) and not on the low-level `AddCoins`/`SubtractCoins` paths the gas system uses — gas fees move tracked denoms but always within the system, so they cancel by construction.
- `gno.land/pkg/gnoland/genesis.go:255` — the default `totalSupply = "1000000000000000ugnot"` (1B GNOT) is hardcoded in `DefaultGenState`. If the param is to be the single source of truth, derive it from balances by default (`TotalBalance(state.Balances)`) and let operators override.

## Questions for Author

- Is this PR still alive? It's been stale since 2025-09-01 (single commit, no further touches) and the bot flagged it for closure in February 2026. If the author has moved on, recommend closing in favor of a fresh PR that builds on jaekwon's `AssertBalanced` design.
- Was the prefix mismatch (`storeKey(...)` vs raw bytes) intentional? If so, the design needs a comment explaining why — but the symmetric usage in the test (`readCounters` uses raw bytes) suggests it's just an oversight.
- Why does `BalanceChangeInvariant` take `BankKeeper` (not `BankKeeperI`)? It only needs `ctx.Store(bank.key)`. Tightening the type would expose the store-access dependency and make the helper testable independently of the full keeper.
