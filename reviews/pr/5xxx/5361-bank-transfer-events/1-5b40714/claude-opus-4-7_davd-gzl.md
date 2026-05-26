# PR #5361: feat(tm2): add transfer event for bank ops

URL: https://github.com/gnolang/gno/pull/5361
Author: mvallenet | Base: master | Files: 11 | +321 -52
Reviewed by: davd-gzl | Model: claude-opus-4-7

**Verdict: APPROVE** — additive, well-scoped wiring of bank events on top of the typed-ABCI-event infrastructure (#4630); known gaps (ante-handler invisibility, missing mint/burn) are correctly documented in the ADR. One asymmetric zero-amount guard between `sendCoins` and `InputOutputCoins`, plus a stale proto field type, are minor.

## Summary

The bank module (ported from Cosmos SDK in 2021) had all event-emission code commented out because the original `EventManager`/`NewEvent` API was never ported to Gno. The Gno event system arrived later in two phases (`EventLogger`/`std.Emit` in #1653, typed ABCI events in #4630), but `tm2/pkg/sdk/bank` was never updated, so indexers had no way to track coin movements via events. This PR adds three typed events — `TransferEvent` for 1:1 sends, `CoinSpentEvent`/`CoinReceivedEvent` for N:M multi-sends — emitted from [`sendCoins`](../../../../../.worktrees/gno-review-5361/tm2/pkg/sdk/bank/keeper.go#L146-L172) and [`InputOutputCoins`](../../../../../.worktrees/gno-review-5361/tm2/pkg/sdk/bank/keeper.go#L75-L110). Partial fix for #5344.

## Glossary

- `sendCoins` — the unexported transfer primitive shared by `SendCoins` and `SendCoinsUnrestricted`.
- `SendCoinsUnrestricted` — same as `SendCoins` but bypasses `canSendCoins` restriction; used for gas fee deduction in `auth/ante.go` and for storage deposit lock/refund in the VM keeper.
- `InputOutputCoins` — the N:M multi-send primitive backing `MsgMultiSend`.
- `EventLogger` — per-message event sink. `runMsgs` resets it before processing messages, so ante-handler events do not survive into the tx result.
- `AssertABCIEvent` — marker method on the `abci.Event` interface; the three new types implement it.

## Fix

Three new types are added in [`tm2/pkg/sdk/bank/events.go`](../../../../../.worktrees/gno-review-5361/tm2/pkg/sdk/bank/events.go) and amino-registered in [`tm2/pkg/sdk/bank/package.go`](../../../../../.worktrees/gno-review-5361/tm2/pkg/sdk/bank/package.go#L16-L18). `sendCoins` emits a single `TransferEvent{From,To,Amount}` after the subtract+add succeeds, gated on `!amt.IsZero()` because the VM keeper calls `SendCoins` with empty coins on every `MsgCall`/`MsgRun` ([keeper.go:723](../../../../../.worktrees/gno-review-5361/gno.land/pkg/sdk/vm/keeper.go#L723), [keeper.go:879](../../../../../.worktrees/gno-review-5361/gno.land/pkg/sdk/vm/keeper.go#L879)) and a zero-amount transfer would be indexer noise. `InputOutputCoins` emits one `CoinSpentEvent` per input and one `CoinReceivedEvent` per output, rather than a single `TransferEvent` per pair, because an N:M send cannot be decomposed into independent 1:1 transfers without inventing a join. Stale comment blocks of the dead Cosmos SDK `EventManager` API are removed from `handler.go` and `keeper.go`. Dead-letter txtars that pinned events with `\[…\]` exact-match (`grc20_registry_emit`, `storage_deposit`, `storage_deposit_collector`) are relaxed to `.*` so the new bank events do not break them.

## Critical (must fix)

None.

## Warnings (should fix)

- **[asymmetric zero-amount guard]** [`tm2/pkg/sdk/bank/keeper.go:75-110`](../../../../../.worktrees/gno-review-5361/tm2/pkg/sdk/bank/keeper.go#L75-L110) — `InputOutputCoins` has no `IsZero` skip, unlike [`sendCoins`](../../../../../.worktrees/gno-review-5361/tm2/pkg/sdk/bank/keeper.go#L162-L169).
  <details><summary>details</summary>

  Today this is safe in practice: `InputOutputCoins` calls [`ValidateInputsOutputs`](../../../../../.worktrees/gno-review-5361/tm2/pkg/sdk/bank/msgs.go#L165) at the top, which requires `Coins.IsAllPositive()` per input and per output ([`msgs.go:121`](../../../../../.worktrees/gno-review-5361/tm2/pkg/sdk/bank/msgs.go#L121), [`msgs.go:149`](../../../../../.worktrees/gno-review-5361/tm2/pkg/sdk/bank/msgs.go#L149)), so zero-amount entries never reach the emit site. But `InputOutputCoins` is exported on `BankKeeperI` and `sendCoins` also runs through validation only via the public `SendCoins` wrapper; the zero-amount guard sits inside `sendCoins` precisely because callers (the VM keeper) hand it empty coins. The asymmetry means future callers of `InputOutputCoins` who skip the `MsgMultiSend` validation path would silently emit `CoinSpentEvent`/`CoinReceivedEvent` with empty `Amount`. Fix: either drop the `IsZero` guard in `sendCoins` (and have the VM keeper not call it with empty coins — preferable since the zero-call is conceptually a no-op anyway), or mirror the guard inside the two loops in `InputOutputCoins`. The first option is cleaner.
  </details>

- **[proto / Go type mismatch]** [`tm2/pkg/sdk/bank/bank.proto:22-36`](../../../../../.worktrees/gno-review-5361/tm2/pkg/sdk/bank/bank.proto#L22-L36) — proto declares `string amount`, Go declares `std.Coins`.
  <details><summary>details</summary>

  The proto entries for `TransferEvent`, `CoinSpentEvent`, `CoinReceivedEvent` use `string amount`, but the Go types in [`events.go`](../../../../../.worktrees/gno-review-5361/tm2/pkg/sdk/bank/events.go) use `std.Coins`. No `.pb.go` is generated under this directory, so the .proto is not load-bearing for runtime, but it is checked into the tree as the canonical spec for amino-over-the-wire encoding. The pre-existing `MsgSend` entry has the same inconsistency (`string amount` vs `std.Coins`), so this is consistent with prior style — but it perpetuates the drift. Fix: either align the proto fields to `repeated tm.Coin amount` (matching the actual Go shape) or drop the `.proto` if it is never regenerated. Out-of-scope cleanup; mentioning so it does not decay further.
  </details>

## Nits

- [`tm2/pkg/sdk/bank/bank.proto:36`](../../../../../.worktrees/gno-review-5361/tm2/pkg/sdk/bank/bank.proto#L36) — no trailing newline; minor lint sting.
- [`tm2/pkg/sdk/bank/keeper_test.go:166-189`](../../../../../.worktrees/gno-review-5361/tm2/pkg/sdk/bank/keeper_test.go#L166-L189) — `TestSendCoinsEmitsTransferEvent` only covers the non-zero path. A one-line `require.Len(events, 0)` after a `SendCoins(..., std.NewCoins())` would pin the zero-amount-no-event invariant that `sendCoins` deliberately introduces.
- [`tm2/pkg/sdk/bank/keeper.go:162`](../../../../../.worktrees/gno-review-5361/tm2/pkg/sdk/bank/keeper.go#L162) — comment `// Only emit a transfer event when coins are actually moved.` is correct but understates: the VM keeper passes empty `Coins` on every `MsgCall`/`MsgRun`, so without this guard every gno tx would carry a null-amount `TransferEvent`. A pointer-comment to that call site would help future readers.

## Missing Tests

- **[ante-handler invisibility]** [`tm2/pkg/sdk/auth/ante.go:326`](../../../../../.worktrees/gno-review-5361/tm2/pkg/sdk/auth/ante.go#L326) — no test asserts that gas-fee `TransferEvent`s are emitted by `DeductFees` but absent from `result.Events`.
  <details><summary>details</summary>

  The ADR's "Known limitations" calls this out: ante events are dropped because [`baseapp.runMsgs` resets the EventLogger](../../../../../.worktrees/gno-review-5361/tm2/pkg/sdk/baseapp.go#L645) before iterating messages. A txtar assertion that the gas-fee transfer event is NOT in `EVENTS:` would lock the current behavior down so a future refactor doesn't silently expose them (which would either be a feature or a privacy/perf regression depending on direction). Cheap to add to `bank_transfer_events.txtar`. Worth adding before merge so the invariant is testable, not just documented.
  </details>

- **[storage-deposit transfer surfaced]** [`gno.land/pkg/sdk/vm/keeper.go:1337,1352`](../../../../../.worktrees/gno-review-5361/gno.land/pkg/sdk/vm/keeper.go#L1337) — ADR claims storage-deposit `TransferEvent`s are visible in tx results; no assertion exercises this claim.
  <details><summary>details</summary>

  When running the modified `storage_deposit.txtar` locally, the new `TransferEvent` does appear between the existing storage event and the storage entry — but the txtar still only asserts the storage event (`bytes_delta`, `fee_delta`) via `.*` wildcard. Adding an explicit `stdout 'EVENTS:.*"from":".*","to":".*","amount":\[{"denom":"ugnot","amount":502500}'` next to the existing line would document that storage-deposit fee transfers are visible, pinning what the ADR currently only asserts in prose.
  </details>

## Suggestions

- [`tm2/pkg/sdk/bank/keeper.go:146-172`](../../../../../.worktrees/gno-review-5361/tm2/pkg/sdk/bank/keeper.go#L146-L172) — consider hoisting the zero-amount check to the VM-keeper call sites.
  <details><summary>details</summary>

  The reason `sendCoins` carries a zero-amount guard is that the VM keeper invokes `SendCoins(ctx, caller, pkgAddr, send)` even when `send` is empty ([keeper.go:723](../../../../../.worktrees/gno-review-5361/gno.land/pkg/sdk/vm/keeper.go#L723), [keeper.go:879](../../../../../.worktrees/gno-review-5361/gno.land/pkg/sdk/vm/keeper.go#L879)). Skipping the call when `send.IsZero()` would let `sendCoins` be unconditionally event-emitting, which is the cleaner invariant (the bank module says "if I subtracted and added, I emit"; the VM module decides what counts as a real transfer). Out of scope for this PR but worth a follow-up.
  </details>

- [`tm2/adr/pr5361_bank_transfer_events.md`](../../../../../.worktrees/gno-review-5361/tm2/adr/pr5361_bank_transfer_events.md) — solid context; consider adding the exact tx-result shape (one `TransferEvent` JSON snippet from `bank_transfer_events.txtar` output) so future readers see what indexers will receive without having to instrument a node.

## Questions for Author

- Storage-deposit calls emit `TransferEvent` AND `StorageDepositEvent` for the same fee movement (ADR §"Storage deposit transfers"). Is that intended dual-emission, or should `lockStorageDeposit` / `refundStorageDeposit` use a quiet path (`SendCoinsUnrestrictedSilent`-equivalent) so indexers don't double-count the same flow?
- Any plan to upstream the `baseapp.go` change so ante events survive into `result.Events`? "Tracked separately" in the ADR — is there a PR number to link?
