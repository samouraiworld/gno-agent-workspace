# PR #5556: fix(gnoland): detect validator changes via EndTxHook for same-block processing

**URL:** https://github.com/gnolang/gno/pull/5556
**Author:** thehowl | **Base:** master | **Files:** 9 | **+454 -402**
**Reviewed by:** davd-gzl | **Model:** glm-5.1

## Summary

This PR fixes a bug where validator set changes committed to `r/sys/validators/v2` are lost from the consensus set after a node restart. The root cause was the in-memory `collector[T]` that populated itself from `EventSwitch` events fired *after* commit (between block N-1 and block N). On restart the collector was empty, so `EndBlocker(N)` returned early without querying the VM — even though the realm had already recorded the change.

**The fix replaces the entire event-collector pipeline with `BaseApp.BlockEvents()`** — a new field on `BaseApp` that accumulates events from successful `DeliverTx` calls within the current block and is reset at `BeginBlock`. The `EndBlocker` now:

1. Checks `hasValidatorChangeEvent(app.BlockEvents())` — a simple scan of the block's tx events for `ValidatorAdded`/`ValidatorRemoved` from `r/sys/validators/v2`.
2. If found, calls `GetChanges(req.Height, req.Height)` — using the *current* block height instead of the previously-committed height.

This works because `DeliverTx` runs before `EndBlock`, so by the time the EndBlocker executes, the `deliverState` context already reflects all uncommitted writes from the current block's successful txs. The VM query sees changes written during those txs (`std.GetHeight() == req.Height`). Tendermint persists the `ValidatorUpdates` returned from `EndBlock(N)` in its state DB, so on restart it restores the correct validator set — no special recovery code needed.

**Key design improvements over PR #5469 (the `firstBlock` approach):**
- Same-block processing eliminates the inherent one-block delay of the old collector.
- No special-case restart logic needed; restarts are inherently safe because the validator update is persisted as part of the block that created it.
- The `collector[T]` type and `events.go` are deleted entirely, removing the `EventSwitch` dependency from the EndBlocker path.
- `BlockEvents()` is a general-purpose `BaseApp` addition usable by any EndBlocker.

**Files changed:**
- `tm2/pkg/sdk/baseapp.go` — adds `blockEvents []abci.Event` field, reset in `BeginBlock`, append in `DeliverTx`, exposed via `BlockEvents()`.
- `gno.land/pkg/gnoland/app.go` — removes collector setup, simplifies `EndBlocker` signature, queries `GetChanges(req.Height, req.Height)`.
- `gno.land/pkg/gnoland/validators.go` — replaces `validatorEventFilter` (operated on `EventSwitch` events) with `hasValidatorChangeEvent` (operates on `[]abci.Event`). Deletes `validatorUpdate` type.
- `gno.land/pkg/gnoland/events.go` — deleted entirely (`collector[T]` removed).
- `gno.land/pkg/gnoland/mock_test.go` — replaces `LastBlockHeight()` mock with `BlockEvents()` mock.
- `gno.land/pkg/gnoland/app_test.go` — rewrites `TestEndBlocker` subtests to use `BlockEvents` mock instead of `EventSwitch`/collector.
- `gno.land/pkg/integration/testscript_gnoland.go` — adds `gnorpc` testscript command with `validators -wait` polling subcommand; fixes `loadUserEnv` account accessor.
- `gno.land/pkg/integration/testdata/restart_validators.txtar` — integration test: add validator via GovDAO, restart, confirm persistence.
- `gno.land/adr/pr5556_restart_validator_changes.md` — ADR documenting context, decision, and 5 alternatives considered.

## Test Results
- **Existing tests:** PASS — all CI checks green (build, lint, test, e2e, analyze, docs).
- **Edge-case tests:** Skipped — unit tests and integration test cover the primary scenarios.

## Critical (must fix)

None

## Warnings (should fix)

- [ ] `gno.land/pkg/gnoland/app.go:42` — **`EventSwitch` field still marked "required" but is no longer validated.** The `validate()` method at line 67 no longer checks `c.EventSwitch == nil`, but the struct comment still says `// required`. The `EventSwitch` field *is* still used by `NewApp()` (line 251) and `node_inmemory.go` (line 112), but it's no longer required for EndBlocker correctness. The comment should be updated to reflect its actual role — it's still wired into the app but the EndBlocker no longer depends on it.

- [ ] `gno.land/pkg/gnoland/mock_test.go:26-53` — **`mockEventSwitch` is dead code.** No test uses it after the collector was deleted. The entire `mockEventSwitch` type, its delegate types (`fireEventDelegate`, `addListenerDelegate`, `removeListenerDelegate`), and all three method implementations are unused. Should be removed.

- [ ] `gno.land/pkg/gnoland/app.go:42` + `gno.land/pkg/gnoland/node_inmemory.go:98-112` — **`EventSwitch` is still wired into the app but no longer consumed by EndBlocker.** It's created in `node_inmemory.go:98`, passed to `AppOptions`, and eventually to `NewAppWithOptions`, but the only remaining consumer would be whatever else uses `EventSwitch` for other event types. This is not broken, but the wiring is now vestigial for the validator change path. If no other consumer needs it, it should be removed; if other consumers do, a comment should explain what still uses it.

## Nits

- [ ] `tm2/pkg/sdk/baseapp.go:555` — **Magic number 64 in capacity cap.** `app.blockEvents[:0:min(cap(app.blockEvents), 64)]` — the cap of 64 is undocumented. Why 64 specifically? A comment explaining the choice (e.g., "cap at 64 to limit per-block event buffer reuse; most blocks have far fewer validator events") would help future readers.

- [ ] `tm2/pkg/sdk/baseapp.go:597-599` — **`BlockEvents()` returns a reference to the internal slice.** This is fine for read-only consumers like `hasValidatorChangeEvent`, but since the ADR positions `BlockEvents()` as a "general-purpose addition to tm2 that any EndBlocker can use", future callers could accidentally mutate the internal state. Consider either documenting the read-only contract or returning a copy for safety (though the copy cost is unnecessary for the current use case).

- [ ] `gno.land/pkg/gnoland/validators.go:22-37` — **`hasValidatorChangeEvent` iterates all block events.** For blocks with many txs and events, this scans every event. The old `validatorEventFilter` had the same cost. Not a regression, but worth noting: the function short-circuits on the first validator event match, so the common case (no validator change) scans all events but the match case is fast.

## Missing Tests

- [ ] **No unit test for `BaseApp.BlockEvents()` accumulation logic.** Codecov reports 60% patch coverage for `baseapp.go` (2 missing lines). The append in `DeliverTx` (line 589-591) and the reset in `BeginBlock` (line 555) have no direct test coverage. A test should verify: (1) events from a successful DeliverTx are accumulated, (2) events from a failed DeliverTx are *not* accumulated, (3) the slice is reset at BeginBlock, (4) the capacity cap logic works correctly.

- [ ] **No test for `hasValidatorChangeEvent` with mixed event types.** The current tests use `valEvents()` which returns only a validator event. No test verifies that non-validator `chain.Event` types, or non-`chain.Event` types, are correctly skipped while a validator event is still detected among them. The function logic is simple and correct, but a test with mixed events would confirm it.

- [ ] **No test for the "failed tx events not accumulated" edge case.** If a tx that emits a `ValidatorAdded` event *fails*, its events should not appear in `BlockEvents()`, and the EndBlocker should not trigger a VM query. This is a correctness-critical scenario — a failed tx should never cause a validator update — but has no test coverage.

- [ ] **No test for `gnorpc validators` without `-wait`.** The code path at `testscript_gnoland.go:940-947` (no-wait immediate query) is untested.

## Suggestions

- Remove `mockEventSwitch` and its delegate types from `mock_test.go`. It's 30 lines of dead code that could mislead future test authors into thinking the EventSwitch is still relevant to EndBlocker tests.

- Update the `EventSwitch` comment in `AppOptions` from `// required` to something like `// used by node event broadcasting; no longer required for EndBlocker`. This prevents future readers from thinking a nil EventSwitch will break EndBlocker.

- Add a `TestBlockEvents` test in `tm2/pkg/sdk/` that exercises the full lifecycle: BeginBlock → DeliverTx (success, events accumulated) → DeliverTx (failure, events not accumulated) → EndBlock (read BlockEvents) → Commit → BeginBlock (reset). This would cover the 2 uncovered lines reported by Codecov.

- The `restart_validators.txtar` test sends a trigger tx after restart to produce a live block before querying. Consider adding a comment in the test explaining *why* the trigger is needed — is it a testscript infrastructure requirement (RPC not available until first block), or a Tendermint requirement (validator set not queryable until after first block commit)?

## Questions for Author

- The `blockEvents` capacity cap of 64 — is this based on observed event counts in production/testnet blocks, or is it an arbitrary reasonable limit? If a block legitimately has >64 events from successful txs, the backing array reallocates, which is fine but worth documenting.

- @omarsy suggested in the PR comments that a native `ValidatorKeeper` on a standard SDK `KVStore` (PR #5488 direction) would make the entire event→scan→QueryEval→regex-parse pipeline unnecessary. You responded that you don't understand the criticism. Is the intent for this PR to be a focused bug fix (correct approach) while the broader refactoring happens separately in #5488/#5485, or is there a concern that this PR's approach conflicts with the Keeper direction?

- Is `mockEventSwitch` in `mock_test.go` intentionally kept for future use, or is it leftover from the collector deletion?

## Verdict

**APPROVE** — The core design is sound: same-block processing via `BlockEvents()` is strictly better than the old `EventSwitch` collector (eliminates one-block delay, makes restarts inherently safe, removes shared mutable state). The ADR is thorough and the integration test validates the critical restart scenario. The warnings (stale `EventSwitch` "required" comment, dead `mockEventSwitch` code) are cleanup items that don't affect correctness. The missing test coverage for `BlockEvents()` accumulation in `baseapp.go` should be addressed before or shortly after merge — the "failed tx events not accumulated" edge case is particularly important to verify.
