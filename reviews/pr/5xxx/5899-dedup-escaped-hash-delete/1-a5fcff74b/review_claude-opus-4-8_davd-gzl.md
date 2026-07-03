# PR [#5899](https://github.com/gnolang/gno/pull/5899): fix(gnovm): remove duplicated escaped-hash delete in DelObject

URL: https://github.com/gnolang/gno/pull/5899
Author: omarsy | Base: master | Files: 1 | +0 -5
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: a5fcff74b (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5899 a5fcff74b`

**TL;DR:** `DelObject` deleted the same escaped-object hash from the IAVL store twice with two back-to-back identical blocks. This PR removes the redundant second block, keeping the one that also emits the gas trace.

**Verdict: APPROVE** — pure removal of a proven-duplicate delete; state-identical, gas-neutral, no golden changes, no invariant touched.

## Summary
When an object is escaped, `DelObject` removes its hash from the IAVL store. Two consecutive blocks did this delete on the same key, so the second was pure dead node work. The redundant block was introduced by [#5415](https://github.com/gnolang/gno/pull/5415), which added a new copy carrying the `IAVL_DEL_ESCAPED` gas trace directly above the pre-existing untraced copy. The PR deletes the older untraced block and keeps the traced one, so the surviving behavior is unchanged and the trace still fires exactly once.

## Fix
Before: `DelObject` at `store.go:745-752` and `store.go` old-753..758 ran the escaped-hash delete twice. After: only the traced block at [`gnovm/pkg/gnolang/store.go:745-752`](https://github.com/gnolang/gno/blob/a5fcff74b/gnovm/pkg/gnolang/store.go#L745-L752) · [↗](../../../../../.worktrees/gno-review-5899/gnovm/pkg/gnolang/store.go#L745) remains. The load-bearing constraint is gas: a delete on an already-deleted key must not shift any gas golden, which holds because the cache store dedups repeated per-key writes.

Verified on a5fcff74b: reverting the deletion (restoring the second block from master) leaves `go test ./gno.land/pkg/sdk/vm/ -run Gas` and the storage integration txtars green with no golden diff, confirming the second delete was already gas-neutral and its removal changes nothing observable.

## Glossary
- IAVL — the versioned Merkle tree backing tm2 state; here it stores escaped-object hashes.
- gas — metered, consensus-relevant cost; any change to it is a behavior change.
- Store — the backing package/object store layered over a tm2 CommitStore/IAVL.

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
None.

## Missing Tests
None. The surviving path is exercised by the existing gas and storage integration suites; no new behavior is introduced, so there is nothing to assert that master did not already assert.

## Suggestions
None.

## Open questions
- The kept block emits `IAVL_DEL_ESCAPED` with a hardcoded `0` gas figure (`store.go:750`); the actual delete gas is charged inside the cache store's `Delete`, not here. That is a pre-existing trace-reporting choice untouched by this PR, so not posted.

## Verification notes (not posted)
- Duplicate confirmed byte-identical on master: the two blocks differ only in that the first (kept) carries the `trace.Store("IAVL_DEL_ESCAPED", ...)` call; the second (removed) did not. `git log -L` blame attributes the kept block to [#5415](https://github.com/gnolang/gno/pull/5415) (`4e1745ab4`) and the removed block to the earlier gas-model PR (`1ad092227`).
- Gas-neutrality mechanism: `cacheStore.Delete` (`tm2/pkg/store/cache/store.go:214-247`) refunds the prior per-key charge (`store.chargedGas[k]`) before re-charging the identical delete cost, so two deletes on one key total the same gas as one; `store.chargedGas` also gates by `gctx != nil`, and when `gctx == nil` neither delete charges. Both branches are gas-neutral.
- State idempotence: `setCacheValue(key, nil, true, true)` marks the key deleted; running it twice yields the same cache entry.
- Single caller `removeDeletedObjects` (`gnovm/pkg/gnolang/realm.go:1126`) uses only the returned `size`, which is computed once at the top of `DelObject` and is independent of the removed block.
- No test or golden asserts the `IAVL_DEL_ESCAPED` trace count, so removing one emission breaks nothing (grep across `gnovm/`, `gno.land/`, `tm2/`).
- CI: all code checks green. The only red gate is `Merge Requirements`, the routine pending-approval bot, not a code problem.
- Local `go test ./gnovm/pkg/gnolang/ -run Files -test.short` shows type-checker error-wording failures (`or_f0.gno`, `add_f0.gno`, redeclaration tests, ...). These reproduce identically with `store.go` reverted to master and stem from the local go1.26.4 go/types wording; unrelated to this PR, and CI (older Go) is green.
