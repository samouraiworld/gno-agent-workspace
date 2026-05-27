# PR #4852: fix: respect storage deposit denom

URL: https://github.com/gnolang/gno/pull/4852
Author: julienrbrt | Base: master | Files: 3 | +135 -68
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `3bee18db8` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-4852 3bee18db8`

**Verdict: REQUEST CHANGES** — fix is direction-correct but the denom contract is incomplete: multi-coin `default_deposit` still chain-halts (pre-existing, but the PR is the right place to close it), the new "got %d%s" error misreports user-supplied deposit under denom mismatch, and `rlm.Deposit` remains a denom-less `uint64` so a governance-mediated denom swap can desynchronise locked funds from the refund denom.

## Summary

Replaces three hardcoded `ugnot.Denom` references in `processStorageDeposit` / `lockStorageDeposit` / `refundStorageDeposit` with the denom parsed from `params.DefaultDeposit`, and adds a `Validate` rule that `storage_price`'s denom set must be a subset of `default_deposit`'s. CI is green, the new param test passes, and the keeper builds. The reviewer/author disagreement (piux2: hardcode ugnot; julienrbrt: trust params) is design-level — but the implementation leaves three concrete sharp edges around (1) multi-coin `DefaultDeposit`, (2) the new error text, and (3) cross-time denom drift for already-locked deposits.

## Glossary

- `processStorageDeposit` — keeper hook called after deliver-tx to settle realm storage growth/release against the caller's `--max-deposit`.
- `rlm.Deposit` — per-realm `uint64` (no denom) tracking how much has been locked at the realm's storage-deposit address.
- `DenomsSubsetOf(B)` — `coins`'s denoms are all present in `B` (asymmetric; quantity-blind).
- `WillSetParam` — hook fired by `params.ParamsKeeper` before each individual key write; runs `Validate` on the **after-this-single-write** struct, not on a batched proposal.

## Fix

Pre-PR: keeper always pulled `ugnot.Denom`, so a GovDAO change to `default_deposit`/`storage_price` denom was visible in parameters but not honoured by the keeper. Post-PR: keeper parses `params.DefaultDeposit` once per call, uses that denom for `deposit.AmountOf`, for the bank coin built in `lockStorageDeposit`/`refundStorageDeposit`, for the `StorageDeposit/UnlockEvent`, and for the restricted-denom check. The new load-bearing gate is [`params.go:121`](https://github.com/gnolang/gno/blob/3bee18db8/gno.land/pkg/sdk/vm/params.go#L121) · [↗](../../../../../.worktrees/gno-review-4852/gno.land/pkg/sdk/vm/params.go#L121) — `storagePriceCoins.DenomsSubsetOf(defaultDepositCoins)` — which, together with the per-write `Validate` fired by `WillSetParam`, blocks any single-key governance proposal that would put the two fields into disagreeing denoms.

## Critical (must fix)

- **[denom drift desyncs already-locked deposits]** [`gno.land/pkg/sdk/vm/keeper.go:1597`](https://github.com/gnolang/gno/blob/3bee18db8/gno.land/pkg/sdk/vm/keeper.go#L1597) · [↗](../../../../../.worktrees/gno-review-4852/gno.land/pkg/sdk/vm/keeper.go#L1597) — refund denom is read from current `params.DefaultDeposit`, but `rlm.Deposit` (`uint64`, [`realm.go:159`](https://github.com/gnolang/gno/blob/3bee18db8/gnovm/pkg/gnolang/realm.go#L159) · [↗](../../../../../.worktrees/gno-review-4852/gnovm/pkg/gnolang/realm.go#L159)) records only an amount, not the denom it was locked under.
  <details><summary>details</summary>

  Lock path constructs `std.Coins{Denom: defaultDenom, Amount: requiredDeposit}` from the params snapshot at lock time. Refund path constructs `std.Coins{Denom: defaultDenom, Amount: depositUnlocked}` from the params snapshot at refund time. These two snapshots can differ: at genesis the chain may run with `default_deposit="1000ugnot"`; a later GovDAO proposal can atomically rewrite both `default_deposit` and `storage_price` to `barrt` (the per-write `Validate` blocks splitting these into two writes, but does not block coordinated rewrites that pass through valid intermediate states — and does not block a custom executor that calls `SetParams` directly, e.g. at genesis-style upgrades). After such a rewrite, every realm with a non-zero `rlm.Deposit` holds `Xugnot` at its storage-deposit address while `refundStorageDeposit` will try to `SendCoinsUnrestricted(..., Xbarrt)`. Either the bank refuses (deposit becomes permanently stuck), or the storage-deposit address happens to hold barrt and the wrong coin gets debited while the ugnot is orphaned. Fix: persist the locking denom alongside `rlm.Deposit` (either widen `Deposit` to `std.Coin`/`std.Coins`, or hard-pin a chain-wide deposit denom that `Validate` refuses to change once any realm holds a non-zero deposit). This is exactly piux2's [comment thread](https://github.com/gnolang/gno/pull/4852#discussion_r2466100451) reframed as a concrete failure mode rather than a "we agreed to hardcode" disagreement.
  </details>

## Warnings (should fix)

- **[multi-coin `DefaultDeposit` chain-halts on storage IO]** [`gno.land/pkg/sdk/vm/params.go:113`](https://github.com/gnolang/gno/blob/3bee18db8/gno.land/pkg/sdk/vm/params.go#L113) · [↗](../../../../../.worktrees/gno-review-4852/gno.land/pkg/sdk/vm/params.go#L113) — `Validate` accepts `default_deposit="1000ugnot,1barrt"` (uses `ParseCoins`, plural), but `processStorageDeposit` calls `std.MustParseCoin(params.DefaultDeposit)` (singular, regex `^<amount><denom>$`) which panics on the comma.
  <details><summary>details</summary>

  Pre-existing: master also has `std.MustParseCoin(params.DefaultDeposit).Amount` at the same site. The PR is the natural place to close it because it (a) is already rewriting this exact line and (b) adds the first cross-field denom-shape rule in `Validate`. The blast radius is total: any non-checkTx that grows or releases realm storage hits this path, so a single GovDAO proposal setting `default_deposit` to a comma-separated list freezes the VM module. Fix: in `Validate`, additionally require `len(defaultDepositCoins) == 1` and `len(storagePriceCoins) == 1` — the design only supports one deposit denom per chain anyway (the keeper picks the first/only one via `MustParseCoin`).
  </details>

- **[error message lies about what user supplied]** [`gno.land/pkg/sdk/vm/keeper.go:1550-1551`](https://github.com/gnolang/gno/blob/3bee18db8/gno.land/pkg/sdk/vm/keeper.go#L1550-L1551) · [↗](../../../../../.worktrees/gno-review-4852/gno.land/pkg/sdk/vm/keeper.go#L1550-L1551) — under denom mismatch the "got %d%s" tail prints the fallback default amount, not the user's actual `--max-deposit`.
  <details><summary>details</summary>

  Flow at [`keeper.go:1516-1519`](https://github.com/gnolang/gno/blob/3bee18db8/gno.land/pkg/sdk/vm/keeper.go#L1516-L1519) · [↗](../../../../../.worktrees/gno-review-4852/gno.land/pkg/sdk/vm/keeper.go#L1516-L1519): if `deposit.AmountOf(defaultDenom) == 0`, `depositAmt` falls back to `defaultDepositCoin.Amount`. A user who sent `--max-deposit 5000barrt` on an ugnot chain triggers the fallback, then sees `"... but got <default-amount>ugnot"` — the message attributes the chain's default to the user. The "got" half is genuinely useful when the user IS short of the right denom (the case the message was added for), but as written it cannot distinguish "user sent too little of the right denom" from "user sent zero of the right denom and we fell back". Fix: when `deposit.AmountOf(defaultDenom) == 0` and `deposit` is non-empty, emit a separate "no %s in --max-deposit (sent %s)" error; otherwise print the real `deposit.AmountOf(defaultDenom)`, not the fallback.
  </details>

- **[stale gnot-specific comment]** [`gno.land/pkg/sdk/vm/keeper.go:1592-1593`](https://github.com/gnolang/gno/blob/3bee18db8/gno.land/pkg/sdk/vm/keeper.go#L1592-L1593) · [↗](../../../../../.worktrees/gno-review-4852/gno.land/pkg/sdk/vm/keeper.go#L1592-L1593) — comment says "If gnot tokens are locked, sent them to the storageFeeCollector address // If unlocked, sent them to memory releaser" but the code is now generic in `defaultDenom`.
  <details><summary>details</summary>

  Decay risk: if the chain runs with `default_deposit="1000barrt"` and someone reads this comment to reason about the restricted-denom branch, they'll get the wrong mental model. The comment also contains two typos ("sent" should be "send"). Fix: replace with "If the deposit denom is restricted, route refunds to StorageFeeCollector instead of the caller."
  </details>

- **[`TestDefaultParams` skips `SysCLAPkgPath`]** [`gno.land/pkg/sdk/vm/params_test.go:14-21`](https://github.com/gnolang/gno/blob/3bee18db8/gno.land/pkg/sdk/vm/params_test.go#L14-L21) · [↗](../../../../../.worktrees/gno-review-4852/gno.land/pkg/sdk/vm/params_test.go#L14-L21) — new test asserts every default field except `sysCLAPkgDefault`.
  <details><summary>details</summary>

  Trivial gap but the test reads as "everything is checked"; readers won't spot the omission. Add `assert.Equal(t, sysCLAPkgDefault, params.SysCLAPkgPath)`.
  </details>

## Nits

- [`gno.land/pkg/sdk/vm/params.go:122`](https://github.com/gnolang/gno/blob/3bee18db8/gno.land/pkg/sdk/vm/params.go#L122) · [↗](../../../../../.worktrees/gno-review-4852/gno.land/pkg/sdk/vm/params.go#L122) — error string formats `storagePriceCoins` with `%q` via `Coins.String()` which renders as `"10uatom"`; readable but contrast with `defaultDepositCoins` which renders the same way; one quoted token instead of two would be tighter ("storage price denom set must be ⊆ default deposit denom set; got %s vs %s").
- [`gno.land/pkg/sdk/vm/params_test.go:53`](https://github.com/gnolang/gno/blob/3bee18db8/gno.land/pkg/sdk/vm/params_test.go#L53) · [↗](../../../../../.worktrees/gno-review-4852/gno.land/pkg/sdk/vm/params_test.go#L53) — `TestParamsValidate` was rewritten as a table; the previous (deleted) version covered the same cases plus a couple more (e.g. malformed default_deposit "garbage"). The rewrite is strictly better in structure but loses the `valid_default_params` check that the `nil`-modify path doesn't accidentally pass. Fine to keep as-is.
- [`gno.land/pkg/sdk/vm/keeper.go:1514-1515`](https://github.com/gnolang/gno/blob/3bee18db8/gno.land/pkg/sdk/vm/keeper.go#L1514-L1515) · [↗](../../../../../.worktrees/gno-review-4852/gno.land/pkg/sdk/vm/keeper.go#L1514-L1515) — `defaultDepositCoin := std.MustParseCoin(...)` followed by `defaultDenom := defaultDepositCoin.Denom` introduces two locals where one is used four times and the other once. Inline `defaultDepositCoin.Amount` at L1518 to drop the intermediate.

## Missing Tests

- **[no `Validate` test for `default_deposit` and `storage_price` swapped]** [`gno.land/pkg/sdk/vm/params_test.go:120`](https://github.com/gnolang/gno/blob/3bee18db8/gno.land/pkg/sdk/vm/params_test.go#L120) · [↗](../../../../../.worktrees/gno-review-4852/gno.land/pkg/sdk/vm/params_test.go#L120) — the single subset-check case covers "different denomination"; missing the symmetric case where `storage_price` has more denoms than `default_deposit`.
  <details><summary>details</summary>

  Add a case: `default_deposit="1000ugnot", storage_price="10ugnot,1barrt"` — expected error from the subset rule. Belt-and-braces against an asymmetric reading of `DenomsSubsetOf`.
  </details>

- **[no keeper-level test that switching `default_deposit` denom mid-realm-lifetime preserves refund correctness]** `gno.land/pkg/sdk/vm/params_deposit_test.go` — every existing `params_deposit_test.go` case operates with a single ugnot denom across lock and unlock.
  <details><summary>details</summary>

  Without an integration test that (1) locks deposit under denom A, (2) flips `default_deposit` / `storage_price` to denom B via the keeper's `WillSetParam` path, (3) triggers a release, the Critical denom-drift issue above will silently regress as soon as someone adds a second valid use-case for the subset rule.
  </details>

## Suggestions

- [`gno.land/pkg/sdk/vm/params.go:113-123`](https://github.com/gnolang/gno/blob/3bee18db8/gno.land/pkg/sdk/vm/params.go#L113-L123) · [↗](../../../../../.worktrees/gno-review-4852/gno.land/pkg/sdk/vm/params.go#L113-L123) — combine with the Warning above: enforce `len(coins) == 1` for both fields **and** require `defaultDepositCoins[0].Denom == storagePriceCoins[0].Denom`. The subset relation is weaker than the constraint the keeper actually relies on; this would replace one weak invariant with one tight, locally-checkable one and remove the multi-coin panic surface in the same commit.

## Questions for Author

- The PR title says "respect storage deposit denom" but the keeper only ever consults `default_deposit`'s denom — `storage_price`'s denom (after validation forces it to be a subset) is never re-read. Is the model "one chain, one storage denom, governance can swap it"? If yes, why expose two denoms at all instead of a single `storage_denom` param plus two amount-only fields? See [piux2's October 27 thread](https://github.com/gnolang/gno/pull/4852#discussion_r2466100451) — much of the back-and-forth seems to come from this redundancy.
- How is the already-locked-deposit / new-denom scenario meant to be handled operationally? Is there an implicit assumption that GovDAO would only switch denoms after all realms have unlocked all deposits, and if so, where is that assumption enforced or documented?
