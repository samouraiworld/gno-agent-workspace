# PR #5230: feat(bank): `TotalCoin` - track total supply of a denom

URL: https://github.com/gnolang/gno/pull/5230
Author: davd-gzl | Base: master | Files: 99 | +585 -279
Reviewed by: davd-gzl (self-review, [bot] AI agent) | Model: claude-opus-4-7
Local worktree: `git -C gno worktree add .worktrees/gno-review-5230 998bd9a16` (then `gh -R gnolang/gno pr checkout 5230` inside it)

**Verdict: APPROVE WITH NITS** — Implementation is correct, well-tested, and overflow-safe; CI is green; the supply index is correctly seeded for new chains via the genesis-balance `SetCoins` path. Worth flagging before merge: per-tx gas overhead is non-trivial (+30% on a minimal `Hello()` call), the test mock at [`testing_runtime.go:209`](https://github.com/gnolang/gno/blob/998bd9a16/gnovm/tests/stdlibs/chain/runtime/testing_runtime.go#L207-L213) · [↗](../../../../../.worktrees/gno-review-5230/gnovm/tests/stdlibs/chain/runtime/testing_runtime.go#L207-L213) silently wraps where the real keeper panics, and there is no ADR documenting the consensus-breaking decision.

## Summary

Implements the deferred `TotalCoin` builtin by maintaining a per-denomination supply index (`/s/<denom>`) on the main iavl store, updated as a delta on every `BankKeeper.SetCoins` call. Reads become O(1) instead of requiring an account-iteration scan. The trade-off is consensus-breaking: every coin-mutating tx now does an additional store read + write per affected denom (~83k gas/denom), and the multistore apphash shifts (pinned constant in [`apphash_crossrealm38_test.go:53`](https://github.com/gnolang/gno/blob/998bd9a16/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L53) · [↗](../../../../../.worktrees/gno-review-5230/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L53) re-rolled to `00e1bd88...`). The PR also bumps gas-wanted goldens across ~80 txtar tests to absorb the new per-SetCoins overhead.

## Glossary

- `SetCoins` — bank keeper's only mutator; both `AddCoins` and `SubtractCoins` funnel through it
- `/s/<denom>` — new top-level key on mainKey holding amino-marshaled `int64` total supply
- `updateSupply` — computes per-denom delta between old and new account balances, applies to index
- `overflow.Sub/Add` — checked int64 arithmetic that returns `(result, ok)` instead of wrapping

## Fix

Before: `TotalCoin` panicked with `"not yet implemented"` at both [`builtins.go:43`](https://github.com/gnolang/gno/blob/998bd9a16/gno.land/pkg/sdk/vm/builtins.go#L42-L44) · [↗](../../../../../.worktrees/gno-review-5230/gno.land/pkg/sdk/vm/builtins.go#L42-L44) and the test banker — calling `banker.NewBanker(...).TotalCoin(denom)` from a realm aborted the tx. After: every `BankKeeper.SetCoins` call computes the delta between `oldCoins` and `newCoins` for each affected denom and adjusts a single `int64` keyed at `/s/<denom>` ([`keeper.go:267-281`](https://github.com/gnolang/gno/blob/998bd9a16/tm2/pkg/sdk/bank/keeper.go#L260-L281) · [↗](../../../../../.worktrees/gno-review-5230/tm2/pkg/sdk/bank/keeper.go#L260-L281), [`keeper.go:283-314`](https://github.com/gnolang/gno/blob/998bd9a16/tm2/pkg/sdk/bank/keeper.go#L283-L314) · [↗](../../../../../.worktrees/gno-review-5230/tm2/pkg/sdk/bank/keeper.go#L283-L314)). `BankKeeper` now holds an additional `store.StoreKey` ([`keeper.go:43`](https://github.com/gnolang/gno/blob/998bd9a16/tm2/pkg/sdk/bank/keeper.go#L43) · [↗](../../../../../.worktrees/gno-review-5230/tm2/pkg/sdk/bank/keeper.go#L43)), and the constructor signature changes accordingly — all four call sites are updated.

Genesis-time correctness rests on the load-bearing fact that genesis balances are applied via `cfg.bankk.SetCoins(ctx, bal.Address, bal.Amount)` at [`app.go:508`](https://github.com/gnolang/gno/blob/998bd9a16/gno.land/pkg/gnoland/app.go#L508) · [↗](../../../../../.worktrees/gno-review-5230/gno.land/pkg/gnoland/app.go#L508), which is the same path that updates the supply index. New chains starting from genesis therefore boot with a correct index; chains that load a pre-existing iavl store without replaying genesis would observe `TotalCoin == 0` for every denom until the next `SetCoins` for that denom (see Warnings below).

## Benchmarks / Numbers

Per-tx gas impact, from the recalibrated txtar goldens:

| Tx kind | Before | After | Delta | Source |
|---|---|---|---|---|
| `addpkg` (hello) | 2,815,758 | 3,216,606 | +400,848 (+14%) | [`gnokey_gasfee.txtar`](https://github.com/gnolang/gno/blob/998bd9a16/gno.land/pkg/integration/testdata/gnokey_gasfee.txtar) · [↗](../../../../../.worktrees/gno-review-5230/gno.land/pkg/integration/testdata/gnokey_gasfee.txtar) |
| `Hello()` call | 1,271,083 | 1,671,931 | +400,848 (+32%) | [`gnokey_gasfee.txtar`](https://github.com/gnolang/gno/blob/998bd9a16/gno.land/pkg/integration/testdata/gnokey_gasfee.txtar) · [↗](../../../../../.worktrees/gno-review-5230/gno.land/pkg/integration/testdata/gnokey_gasfee.txtar) |
| `addpkg` (bar/foo/baz) | 2,823,085 | 3,223,933 | +400,848 (+14%) | [`restart_gas.txtar`](https://github.com/gnolang/gno/blob/998bd9a16/gno.land/pkg/integration/testdata/restart_gas.txtar) · [↗](../../../../../.worktrees/gno-review-5230/gno.land/pkg/integration/testdata/restart_gas.txtar) |
| addpkg restart sample | 2,235,727 | 2,636,575 | +400,848 (+18%) | [`stdlib_restart_compare.txtar`](https://github.com/gnolang/gno/blob/998bd9a16/gno.land/pkg/integration/testdata/stdlib_restart_compare.txtar) · [↗](../../../../../.worktrees/gno-review-5230/gno.land/pkg/integration/testdata/stdlib_restart_compare.txtar) |

The constant +400,848 delta across all single-denom (ugnot) transactions is exactly explained by store gas (`ReadCostFlat=59k + WriteCostFlat=24k` per `SetCoins` × ~5 fee/refund/storage-deposit-related SetCoins per tx, plus per-byte). It is a structural cost, not noise.

## Critical (must fix)

None.

## Warnings (should fix)

- **[test mock silently wraps where real keeper panics]** [`testing_runtime.go:207-213`](https://github.com/gnolang/gno/blob/998bd9a16/gnovm/tests/stdlibs/chain/runtime/testing_runtime.go#L207-L213) · [↗](../../../../../.worktrees/gno-review-5230/gnovm/tests/stdlibs/chain/runtime/testing_runtime.go#L207-L213) — `TestBanker.TotalCoin` uses raw `total += coins.AmountOf(denom)` while the real `BankKeeper.updateSupply` panics on overflow.
  <details><summary>details</summary>

  The two implementations now diverge on overflow semantics. A `.gno` test that legitimately stresses supply tracking near `int64` boundaries will silently wrap in `gno test` but panic on chain. This is exactly the asymmetry that breeds latent bugs — code shipped after passing `gno test` may still abort in production. Fix: mirror the real keeper's overflow check (e.g. `if _, ok := overflow.Add(total, coins.AmountOf(denom)); !ok { panic(...) }`) so the test mock behaves identically to the real banker.
  </details>

- **[no ADR for a consensus-breaking change]** repo-wide — Per [`gno/AGENTS.md:85`](https://github.com/gnolang/gno/blob/998bd9a16/AGENTS.md#L85) · [↗](../../../../../.worktrees/gno-review-5230/AGENTS.md#L85), every non-trivial AI-assisted PR requires an ADR.
  <details><summary>details</summary>

  This PR changes the multistore apphash, adds a new top-level keyspace on `mainKey`, and bumps ~80 gas-wanted constants. An ADR under `tm2/adr/pr5230_bank_supply_index.md` would document: (a) why a per-denom index instead of computing on demand, (b) the gas-cost trade-off the chain pays on every `SetCoins`, (c) the `/s/<denom>` prefix choice and non-collision with auth's nested `/a/<addr>/s/<session>` ([`auth/consts.go:25,50`](https://github.com/gnolang/gno/blob/998bd9a16/tm2/pkg/sdk/auth/consts.go#L25) · [↗](../../../../../.worktrees/gno-review-5230/tm2/pkg/sdk/auth/consts.go#L25)), (d) the migration path for chains that don't restart from genesis. Skipping the ADR forces every future maintainer to re-derive these from the diff.
  </details>

- **[no migration path for chains loading pre-existing state]** [`genesis.go:29-37`](https://github.com/gnolang/gno/blob/998bd9a16/tm2/pkg/sdk/bank/genesis.go#L29-L37) · [↗](../../../../../.worktrees/gno-review-5230/tm2/pkg/sdk/bank/genesis.go#L29-L37) — `InitGenesis` initializes only params; the supply index relies entirely on genesis-balance `SetCoins` being re-executed.
  <details><summary>details</summary>

  For a fresh chain this is fine — `app.go:505-512` iterates `state.Balances` and applies each via `cfg.bankk.SetCoins`, which seeds the index correctly. But a node that boots against a pre-existing iavl store (no genesis replay) will see `TotalCoin(denom) == 0` for every denom with existing balances, until the next mutation. If gno.land is still pre-mainnet this is moot; if any deployed chain is expected to upgrade in place, the InitGenesis needs an explicit "scan all accounts once, rebuild the index" path, or the keeper needs a one-time bootstrap. Worth confirming with the team which scenario applies.
  </details>

## Nits

- [`builtins.go:42-44`](https://github.com/gnolang/gno/blob/998bd9a16/gno.land/pkg/sdk/vm/builtins.go#L42-L44) · [↗](../../../../../.worktrees/gno-review-5230/gno.land/pkg/sdk/vm/builtins.go#L42-L44) — `SDKBanker.TotalCoin` has no docstring while every other method in the file is self-explanatory; one line clarifying that this is O(1) and reads from the bank supply index would help future readers understand the cost.
- [`keeper.go:286-294`](https://github.com/gnolang/gno/blob/998bd9a16/tm2/pkg/sdk/bank/keeper.go#L286-L294) · [↗](../../../../../.worktrees/gno-review-5230/tm2/pkg/sdk/bank/keeper.go#L286-L294) — `denoms := make(map[string]struct{})` iteration order is non-deterministic. Final state is identical regardless of order (each iteration writes to an independent key), but if you ever add a side effect inside the loop, this becomes a consensus hazard. A sorted slice or `slices.Sorted(maps.Keys(...))` would be free insurance.
- [`consts.go:7`](https://github.com/gnolang/gno/blob/998bd9a16/tm2/pkg/sdk/bank/consts.go#L7) · [↗](../../../../../.worktrees/gno-review-5230/tm2/pkg/sdk/bank/consts.go#L7) — `/s/` is a terse prefix that risks future collision. Auth already uses `/s/` as an infix under `/a/<addr>/s/<session>`. They don't actually collide (auth's `/s/` is nested), but a more descriptive prefix like `/supply/` would be unambiguous.
- [`keeper.go:317-328`](https://github.com/gnolang/gno/blob/998bd9a16/tm2/pkg/sdk/bank/keeper.go#L317-L328) · [↗](../../../../../.worktrees/gno-review-5230/tm2/pkg/sdk/bank/keeper.go#L317-L328) — `getSupply` panics on unmarshal failure of an internal-only amino payload that the keeper itself wrote. A panic is right for "corrupted store", but the error path is reachable only on genuine database corruption; a comment to that effect would prevent future "shouldn't we return an error here?" reviews.

## Missing Tests

- **[chain-upgrade migration]** none — No test exercises "load a chain with pre-existing balances, observe TotalCoin without re-running genesis".
  <details><summary>details</summary>

  If the answer is "we don't support that, all upgrades start from a new genesis", say so in the ADR. If migration is a real scenario, a `restart_supply_index.txtar` test that boots, sets balances, stops, restarts against the same DB without genesis replay, and asserts `TotalCoin == expected` would catch the regression. Today the test surface is "fresh chain only".
  </details>

- **[denomination with realm-issued long path]** [`totalcoin.txtar`](https://github.com/gnolang/gno/blob/998bd9a16/gno.land/pkg/integration/testdata/totalcoin.txtar) · [↗](../../../../../.worktrees/gno-review-5230/gno.land/pkg/integration/testdata/totalcoin.txtar) — Existing test uses `gno.land/r/test/supply_tracker:token` (~37 byte denom). [`realm_banker_issued_coin_denom.txtar:13`](https://github.com/gnolang/gno/blob/998bd9a16/gno.land/pkg/integration/testdata/realm_banker_issued_coin_denom.txtar#L13) · [↗](../../../../../.worktrees/gno-review-5230/gno.land/pkg/integration/testdata/realm_banker_issued_coin_denom.txtar#L13) demonstrates 120+ char denoms are valid.
  <details><summary>details</summary>

  Per-byte write cost (`WriteCostPerByte`) scales linearly with the key/value size. A 200-char denom path means the `/s/<denom>` key is 200+ bytes, and every `SetCoins` involving that denom pays write-per-byte gas on both old-value and new-value. Worth one test asserting TotalCoin works with a max-length denom (and noting the gas cost in the test) to make the cost shape explicit.
  </details>

## Suggestions

- [`keeper.go:286-294`](https://github.com/gnolang/gno/blob/998bd9a16/tm2/pkg/sdk/bank/keeper.go#L286-L294) · [↗](../../../../../.worktrees/gno-review-5230/tm2/pkg/sdk/bank/keeper.go#L286-L294) — Early-exit when `oldCoins.IsEqual(newCoins)` to avoid the per-denom set construction and map allocation entirely when SetCoins is called with the same value (e.g. session-spend bookkeeping that doesn't actually change balances).
  <details><summary>details</summary>

  Even the per-denom delta-zero shortcut (`if delta == 0 { continue }`) still pays the `oldCoins.AmountOf` + `newCoins.AmountOf` scan and the map insert. A top-level `if oldCoins.IsEqual(newCoins) { return }` skips all of that. Small win, but it's free.
  </details>

- The two `tm2/pkg/overflow` imports could be wrapped in a single helper `applyDelta(supply, delta int64) (int64, error)` that consolidates the overflow check and panic-format string. Optional.

## Questions for Author

- Does any in-flight upgrade plan rely on loading the existing iavl store without replaying genesis? If yes, this PR is incomplete (see migration warning). If no, an ADR note saying so closes the gap.
- The `bankerTotalCoin` native-gas calibration at [`native_gas.go:93`](https://github.com/gnolang/gno/blob/998bd9a16/gnovm/stdlibs/native_gas.go#L93) · [↗](../../../../../.worktrees/gno-review-5230/gnovm/stdlibs/native_gas.go#L93) is `Base: 89ns, flat`. That covers dispatch only — the actual store read is metered through `ctx.GasContext()` — but the calibration was captured when the function was `panic("not yet implemented")`. Worth confirming the 89ns figure still holds with a real store read, or re-running the calibration if not.
