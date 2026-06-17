# PR #5198: fix(gnovm): use proportional refund for storage deposit to prevent fund lock on storage price change

URL: https://github.com/gnolang/gno/pull/5198
Author: mvallenet | Base: master | Files: 3 | +363 -1
Reviewed by: davd-gzl | Model: claude-opus-4-7[1m] | Commit: `ce6881e1f` (stale)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5198 ce6881e1f`

Verdict: REQUEST CHANGES — fix logic is correct and well-tested, but the integration test [`storage_deposit_price_change.txtar:33`](https://github.com/gnolang/gno/blob/ce6881e1f/gno.land/pkg/integration/testdata/storage_deposit_price_change.txtar#L33) · [↗](../../../../../.worktrees/gno-review-5198/gno.land/pkg/integration/testdata/storage_deposit_price_change.txtar#L33) fails on PR head; hardcoded balance assertion has already drifted from actual gas cost.

## Summary

Closes HackenProof report N°128: changing `StoragePrice` via GovDAO governance permanently locks (price up) or orphans (price down) every realm deposit chain-wide, because [`processStorageDeposit`](https://github.com/gnolang/gno/blob/ce6881e1f/gno.land/pkg/sdk/vm/keeper.go#L1245) · [↗](../../../../../.worktrees/gno-review-5198/gno.land/pkg/sdk/vm/keeper.go#L1245) released deposits using `released * currentPrice` while realms only store totals `Deposit` and `Storage` with no historical price track. The fix replaces the formula with `deposit * released / storage` (full-release shortcut returns `Deposit` exactly to avoid truncation), uses `math/big` to dodge int64 overflow on the intermediate product, and is path-independent: lifetime refunds always equal lifetime deposits regardless of price moves. Lock side still uses current price (unchanged) — the asymmetry is intentional and documented in [PR body](https://github.com/gnolang/gno/pull/5198) under "Considered approaches".

## Glossary

- `processStorageDeposit` — keeper hook called after every `AddPkg`/`Call`/`Run`; settles storage deltas vs deposit.
- `rlm.Deposit` / `rlm.Storage` — per-realm totals (`uint64`) tracked in [`gnovm/pkg/gnolang/realm.go:128-129`](https://github.com/gnolang/gno/blob/ce6881e1f/gnovm/pkg/gnolang/realm.go#L128-L129) · [↗](../../../../../.worktrees/gno-review-5198/gnovm/pkg/gnolang/realm.go#L128-L129).
- `StoragePrice` — global `vm` param (default `100ugnot`/byte), settable via GovDAO.

## Fix

Before: [`keeper.go:1296` (master)](https://github.com/gnolang/gno/blob/ce6881e1f/gno.land/pkg/sdk/vm/keeper.go) · [↗](../../../../../.worktrees/gno-review-5198/gno.land/pkg/sdk/vm/keeper.go) computed `depositUnlocked := overflow.Mulp(released, price.Amount)`, so a price hike after a lock made `depositUnlocked > rlm.Deposit` and the panic at [`keeper.go:1316-1319`](https://github.com/gnolang/gno/blob/ce6881e1f/gno.land/pkg/sdk/vm/keeper.go#L1316-L1319) · [↗](../../../../../.worktrees/gno-review-5198/gno.land/pkg/sdk/vm/keeper.go#L1316-L1319) reverted the tx — locking the deposit permanently. Symmetric loss on a price cut: too small a refund, residual coins stuck in the storage-deposit derived address forever.

After: [`keeper.go:1300-1315`](https://github.com/gnolang/gno/blob/ce6881e1f/gno.land/pkg/sdk/vm/keeper.go#L1300-L1315) · [↗](../../../../../.worktrees/gno-review-5198/gno.land/pkg/sdk/vm/keeper.go#L1300-L1315) decouples refund from `price` entirely. Full release (`rlm.Storage == released`) shortcuts to `int64(rlm.Deposit)`; partial release uses `big.Int(deposit) * big.Int(released) / big.Int(storage)` then `.Int64()`. The existing `rlm.Deposit < uint64(depositUnlocked)` panic at [L1316](https://github.com/gnolang/gno/blob/ce6881e1f/gno.land/pkg/sdk/vm/keeper.go#L1316) · [↗](../../../../../.worktrees/gno-review-5198/gno.land/pkg/sdk/vm/keeper.go#L1316) becomes mathematically unreachable since `released ≤ storage ⇒ depositUnlocked ≤ deposit`, but kept as defense-in-depth.

## Critical (must fix)

- **[txtar test fails on head]** [`gno.land/pkg/integration/testdata/storage_deposit_price_change.txtar:33`](https://github.com/gnolang/gno/blob/ce6881e1f/gno.land/pkg/integration/testdata/storage_deposit_price_change.txtar#L33) · [↗](../../../../../.worktrees/gno-review-5198/gno.land/pkg/integration/testdata/storage_deposit_price_change.txtar#L33) — `TestTestdata/storage_deposit_price_change` currently fails: expected `"coins": "9999782` but actual is `"coins": "9999781628900ugnot"`.
  <details><summary>details</summary>

  Reproduces 100% on PR head `ce6881e1f`. Drift of ~360k ugnot (~0.0036%) from when the assertion was written — almost certainly an unrelated gas-cost change in master that the PR picked up through one of its 30+ merge commits. The deeper issue is that the test hardcodes the user balance prefix to 7 digits (`9999782`), so even a single ugnot drift over the boundary breaks the assertion. ltzmaxwell raised this exact concern in [the PR comments](https://github.com/gnolang/gno/pull/5198#discussion_r2851810075) and it has now materialized.

  Fix: drop the exact-prefix assertion or rewrite it as a delta check. Two options: (a) replace the `stdout '"coins": "9999782'` line with `stdout '"coins": "9999[0-9]{6}'` to match any plausible 13-digit balance under 10G ugnot; (b) better, capture the balance before/after `Allocate`/`Free` and assert `balance_after_free > balance_before_free` (the actual invariant — refund occurred). Same fix applies to the post-free assertion at [line 66](https://github.com/gnolang/gno/blob/ce6881e1f/gno.land/pkg/integration/testdata/storage_deposit_price_change.txtar#L66) · [↗](../../../../../.worktrees/gno-review-5198/gno.land/pkg/integration/testdata/storage_deposit_price_change.txtar#L66) (`stdout '"coins": "9999827'`).

  Repro:
  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5198 -R gnolang/gno
  go test -v -timeout 5m -run 'TestTestdata/storage_deposit_price_change' ./gno.land/pkg/integration/...
  # FAIL: testdata/storage_deposit_price_change.txtar:33: no match for `"coins": "9999782` found in stdout
  ```
  </details>

## Warnings (should fix)

- **[weak unit-test assertion]** [@ltzmaxwell](https://github.com/gnolang/gno/pull/5198#discussion_r2940095049) [`gno.land/pkg/sdk/vm/keeper_test.go:1562`](https://github.com/gnolang/gno/blob/ce6881e1f/gno.land/pkg/sdk/vm/keeper_test.go#L1562) · [↗](../../../../../.worktrees/gno-review-5198/gno.land/pkg/sdk/vm/keeper_test.go#L1562) — `require.True(t, refund > 0, ...)` passes even on a 1-ugnot refund instead of ~50M.
  <details><summary>details</summary>

  Two assertions later ([L1571](https://github.com/gnolang/gno/blob/ce6881e1f/gno.land/pkg/sdk/vm/keeper_test.go#L1571) · [↗](../../../../../.worktrees/gno-review-5198/gno.land/pkg/sdk/vm/keeper_test.go#L1571)) the test checks `diff < 1000` between `refund` and `depositForData`, which is the real correctness gate — so the `refund > 0` line is dead weight relative to the stronger check. But ltzmaxwell's reading suggests they wanted the exact value asserted up front, not buried in a tolerance check. Either drop the redundant `refund > 0` line, or replace with `require.InDelta(t, depositForData, refund, 1000)` so a single line carries the assertion.

  Same shape at [L1640](https://github.com/gnolang/gno/blob/ce6881e1f/gno.land/pkg/sdk/vm/keeper_test.go#L1640) · [↗](../../../../../.worktrees/gno-review-5198/gno.land/pkg/sdk/vm/keeper_test.go#L1640) in `TestStorageDepositPriceDecrease`.
  </details>

- **[asymmetric lock/release semantics undocumented in user-facing docs]** [`docs/resources/storage-deposit.md`](https://github.com/gnolang/gno/blob/ce6881e1f/docs/resources/storage-deposit.md) · [↗](../../../../../.worktrees/gno-review-5198/docs/resources/storage-deposit.md) — explains "Storing data → GNOT locked / Deleting data → GNOT refunded" but never says price changes affect lock cost vs refund formula.
  <details><summary>details</summary>

  Under this PR a user who locks 100 bytes at price=100 (deposit=10000), then sees price rise to 1000 with no other realm activity, gets back exactly 10000 ugnot on free — correct. But if the realm has multiple lock events at different prices, the refund is the proportional share, not the per-event price. Combined with the "Anyone Can Free Storage" semantics already documented at [line 36](https://github.com/gnolang/gno/blob/ce6881e1f/docs/resources/storage-deposit.md#L36) · [↗](../../../../../.worktrees/gno-review-5198/docs/resources/storage-deposit.md#L36), a deleter can over- or under-collect relative to their personal deposits. This is pre-existing behavior under the old formula too, but the new formula makes the share calculation explicit. One paragraph in the doc would close the gap. Fix: append a "Refund Calculation" subsection naming the `deposit * released / storage` formula and noting refund is path-independent over a realm's lifetime, not per-byte.

  ltzmaxwell raised the concern in [in #5198](https://github.com/gnolang/gno/pull/5198#discussion_r2940089841) about adding pre-`Allocate` storage/deposit queries to the txtar — they're optional but would make the doc story tighter.
  </details>

## Nits

- [`gno.land/pkg/sdk/vm/keeper.go:1300-1301`](https://github.com/gnolang/gno/blob/ce6881e1f/gno.land/pkg/sdk/vm/keeper.go#L1300-L1301) · [↗](../../../../../.worktrees/gno-review-5198/gno.land/pkg/sdk/vm/keeper.go#L1300-L1301) — comment says "Proportional refund based on actual deposit ratio" but never names the formula. A one-liner `refund = deposit * released / storage` would let the next reader skip the math.
- [`gno.land/pkg/sdk/vm/keeper_test.go:1570`](https://github.com/gnolang/gno/blob/ce6881e1f/gno.land/pkg/sdk/vm/keeper_test.go#L1570) · [↗](../../../../../.worktrees/gno-review-5198/gno.land/pkg/sdk/vm/keeper_test.go#L1570) — comment says "The diff comes from ~5 bytes of variable reference overhead" — useful but the magic threshold `1000` (Increase test) vs `10000` (Decrease test) deserves the same explanation; the 10x gap mirrors the 10x price ratio but a casual reader will miss that.
- [`gno.land/pkg/sdk/vm/keeper.go:1311-1313`](https://github.com/gnolang/gno/blob/ce6881e1f/gno.land/pkg/sdk/vm/keeper.go#L1311-L1313) · [↗](../../../../../.worktrees/gno-review-5198/gno.land/pkg/sdk/vm/keeper.go#L1311-L1313) — three `big.Int` allocations per partial-free hot-path; not measurable now, but if storage activity scales these are GC pressure. Could compute via `(deposit / storage) * released + ((deposit % storage) * released) / storage` in pure int64 with one overflow check — leaving for a later perf pass.

## Missing Tests

- **[no partial-free coverage]** [`gno.land/pkg/sdk/vm/keeper_test.go:1497`](https://github.com/gnolang/gno/blob/ce6881e1f/gno.land/pkg/sdk/vm/keeper_test.go#L1497) · [↗](../../../../../.worktrees/gno-review-5198/gno.land/pkg/sdk/vm/keeper_test.go#L1497) — both new tests `Allocate` then `Free` the full 500KB. The `else` branch at [keeper.go:1306-1314](https://github.com/gnolang/gno/blob/ce6881e1f/gno.land/pkg/sdk/vm/keeper.go#L1306-L1314) · [↗](../../../../../.worktrees/gno-review-5198/gno.land/pkg/sdk/vm/keeper.go#L1306-L1314) (proportional `big.Int` math) is exercised only against the small base-package overhead, not a real partial release.
  <details><summary>details</summary>

  The bug class this PR fixes is exactly the partial-release path: a realm that owns 1000 bytes of state, releases 100 across multiple txs, and survives 3 price changes in between. A test that does: lock 1000 bytes at price 100 → change price to 1000 → free 500 → change price to 200 → free 300 → free 200 — verifying total refund equals 100000 (initial deposit) within rounding — would lock in the path-independence claim from the PR body. Without it, a future regression that breaks the partial branch can pass CI because both new tests use the full-release shortcut.
  </details>

- **[no overflow-edge test]** [`gno.land/pkg/sdk/vm/keeper_test.go`](https://github.com/gnolang/gno/blob/ce6881e1f/gno.land/pkg/sdk/vm/keeper_test.go) · [↗](../../../../../.worktrees/gno-review-5198/gno.land/pkg/sdk/vm/keeper_test.go) — the PR body cites `math/big` "to prevent int64 overflow on large realms" but no test validates the overflow path.
  <details><summary>details</summary>

  Not blocking — overflow requires deposits near `MaxInt64` ugnot, infeasible in practice. But a unit test directly calling `processStorageDeposit` with a hand-constructed `rlm.Deposit = math.MaxUint64 - 1` and a small `released` would verify the big.Int math (and could be a regression sentinel if anyone "optimizes" back to int64). Acknowledged by author in [the comment thread](https://github.com/gnolang/gno/pull/5198#discussion_r2853931304) as "very unlikely / impossible" — fine to skip if the call-site invariants are documented in the code comment.
  </details>

## Suggestions

- [`gno.land/pkg/sdk/vm/keeper.go:1306-1314`](https://github.com/gnolang/gno/blob/ce6881e1f/gno.land/pkg/sdk/vm/keeper.go#L1306-L1314) · [↗](../../../../../.worktrees/gno-review-5198/gno.land/pkg/sdk/vm/keeper.go#L1306-L1314) — the truncated-dust acknowledgment in the comment is right, but you could *zero out* the realm dust by setting `depositUnlocked = int64(rlm.Deposit)` whenever `released ≥ storage - epsilon` for some tiny epsilon. Probably not worth the complexity — the current "drains on last free" property is clean.
  <details><summary>details</summary>

  Current behavior: dust accumulates in `rlm.Deposit` over partial frees; the full-release shortcut at [L1303-L1305](https://github.com/gnolang/gno/blob/ce6881e1f/gno.land/pkg/sdk/vm/keeper.go#L1303-L1305) · [↗](../../../../../.worktrees/gno-review-5198/gno.land/pkg/sdk/vm/keeper.go#L1303-L1305) drains everything when the realm hits zero storage. So total refunds equal total deposits over a realm's lifetime — which is the exact invariant the PR body claims. Don't change this.
  </details>

## Questions for Author

- Has the `9999781628900` vs `9999782xxxxx` drift been confirmed against a recent master tip, or is it possibly machine-specific? If reproducible on CI you'll see it after the next rebase.
- Any reason the full-release shortcut uses `int64(rlm.Deposit)` directly without checking `rlm.Deposit <= MaxInt64`? Same overflow note applies to [L1305](https://github.com/gnolang/gno/blob/ce6881e1f/gno.land/pkg/sdk/vm/keeper.go#L1305) · [↗](../../../../../.worktrees/gno-review-5198/gno.land/pkg/sdk/vm/keeper.go#L1305) as to the `big.Int.Int64()` at [L1314](https://github.com/gnolang/gno/blob/ce6881e1f/gno.land/pkg/sdk/vm/keeper.go#L1314) · [↗](../../../../../.worktrees/gno-review-5198/gno.land/pkg/sdk/vm/keeper.go#L1314), and the L1316 defensive panic catches it either way — but worth a one-line comment to acknowledge.
