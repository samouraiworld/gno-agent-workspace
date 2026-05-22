# PR #5469: fix(gnoland): recover validator changes after node restart

**URL:** https://github.com/gnolang/gno/pull/5469
**Author:** omarsy | **Base:** master | **Files:** 5 | **+288 -9**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.6

## Summary

This PR fixes a bug where validator set changes committed to the `r/sys/validators/v2` realm are lost from consensus after a node restart. The root cause: on restart, the in-memory event collector is empty (it only accumulates events during runtime). The existing EndBlocker logic checked `len(collector.getEvents()) == 0` and returned early, never querying the VM for pending changes — even though the realm had already recorded them.

**Core fix (`gno.land/pkg/gnoland/app.go:462-489`):** A `firstBlock` closure variable is introduced in the `EndBlocker` factory function. On the first invocation after startup:
1. `firstBlock` is set to `false` (line 481)
2. The collector is drained via `collector.getEvents()` (line 482) — discards any stale events
3. If `app.LastBlockHeight() == 0` (genesis), returns early — no changes to recover (line 484-486)
4. Otherwise, falls through to the VM query, bypassing the collector check

On subsequent blocks, the original logic applies: if the collector has no events, return early.

The VM query `GetChanges(lastHeight, lastHeight)` at line 496 correctly returns changes at the specific height. The `r/sys/validators/v2` realm's `GetChanges(from, to)` uses AVL tree iteration with range `[getBlockID(from), getBlockID(to+1))` — start-inclusive, end-exclusive — so `GetChanges(N, N)` returns exactly the changes at height N.

**Duplicate application is safe:** If the validator change was already applied before shutdown, re-applying the same update is idempotent. The `verifyUpdates` function (in Tendermint2's state execution) computes a power diff between old and new validator sets. If the validator was already applied, the diff is zero and nothing changes. `computeNewPriorities` preserves existing priority for validators already in the set.

**Additional changes:**
- `gnorpc validators` testscript command (`testscript_gnoland.go:880-919`): queries the node's RPC for the current validator set, printing address and voting power.
- `sleep` testscript command (`testscript_gnoland.go:944-959`): pauses for a given duration.
- `restart_validators.txtar` integration test: exercises the full flow — add validator via GovDAO, restart node, verify the validator persists.
- ADR at `gno.land/adr/pr5469_restart_validator_changes.md` documenting the design decision.

## Test Results
- **Existing tests:** PASS — `go test -v -run 'TestEndBlocker' ./gno.land/pkg/gnoland/...` all 8 subtests pass in the worktree.
- **CI status:** Only failure is `Merge Requirements` (waiting for reviewer approval). All code checks pass.
- **Edge-case tests:** Skipped — the EndBlocker unit tests cover the relevant scenarios including the new `firstBlock` path.

## Critical (must fix)

None

## Warnings (should fix)

- [ ] `gno.land/pkg/gnoland/app_test.go:536` — **"no collector events" test relies on zero-value `mockEndBlockerApp`.** The `&mockEndBlockerApp{}` has a nil `lastBlockHeightFn`, which means `LastBlockHeight()` returns 0. This makes `firstBlock` trigger the `LastBlockHeight() == 0` early return (genesis path at `app.go:484`), masking the actual "no collector events" behavior. The test passes for the right reason (empty response) but via the wrong code path. If someone changes the genesis check, this test would break despite "no collector events" logic being correct. Fix: set `lastBlockHeightFn: func() int64 { return 1 }` like the other subtests do, and add a second EndBlocker invocation (where `firstBlock` is false) to test the actual "no events" early return at `app.go:487-488`.

- [ ] `gno.land/pkg/integration/testscript_gnoland.go:958` — **`time.Sleep` with no context cancellation.** The `sleep` command does an unconditional `time.Sleep(d)` without checking the test context. If the test times out or is cancelled, this sleep will block until completion. For short durations this is fine, but a malicious or buggy `.txtar` file could `sleep 1h`. Consider using `select { case <-time.After(d): case <-ctx.Done(): }` or at minimum capping the maximum duration.

## Nits

- [ ] `gno.land/pkg/integration/testscript_gnoland.go:907` — `rpcClient.Validators(context.Background(), nil)` creates a fresh background context. Consider using the testscript's context (if available through the node manager) for proper cancellation propagation.

- [ ] `gno.land/pkg/gnoland/app.go:482` — The comment-less `collector.getEvents()` call (draining stale events) could benefit from an inline comment explaining why: `// Drain stale events from collector — they predate the restart.` The block comment above (lines 462-464) explains the high-level rationale, but this specific call's purpose isn't immediately obvious.

## Missing Tests

- [ ] No EndBlocker unit test for the scenario: first block after restart with pending changes that were NOT yet applied before shutdown. The current `firstBlock` tests (`app_test.go` "first block after restart" subtest around line 830+) verify the happy path, but there's no test confirming that `GetChanges(N, N)` actually returns the pending change and that EndBlocker propagates it correctly — the mock VM keeper returns a canned response regardless.

- [ ] No test for the `sleep` command with an invalid or very large duration — `testscript_gnoland.go:946-959`. The `time.ParseDuration` call handles malformed input, but there's no cap on absurdly large values.

## Suggestions

- Add context awareness to the `sleep` command. A simple implementation:
  ```go
  select {
  case <-time.After(d):
  case <-ctx.Done():
      ts.Fatalf("sleep interrupted: %v", ctx.Err())
  }
  ```
  This requires threading the context through, which the `sleepCmd` closure doesn't currently have. An alternative is capping the duration to something reasonable (e.g., 60s) for safety.

- The ADR (`gno.land/adr/pr5469_restart_validator_changes.md`) is a good addition. Consider referencing the specific code locations (app.go EndBlocker closure) in the ADR to make it easier for future readers to find the implementation.

## Questions for Author

- Is there a scenario where `firstBlock` could be triggered more than once (e.g., if EndBlocker is somehow called before the node is fully initialized)? The code sets `firstBlock = false` on first call, so it's one-shot, but is there any initialization ordering guarantee that ensures EndBlocker isn't called spuriously?

- For the `sleep` testscript command: is this intended to be a general-purpose command for other tests, or specifically for the restart test? If general-purpose, the lack of context cancellation becomes a more significant concern.

## Verdict

**APPROVE** — The core fix is correct, well-reasoned, and the idempotency analysis holds. The `firstBlock` approach is simple and effective. The test fragility in the "no collector events" case (warning #1) is worth addressing but doesn't block merging. The `sleep` command's lack of context awareness (warning #2) is a minor concern for a test utility.
