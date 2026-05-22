# PR #5605: fix(gnovm/store): body-first AddMemPackage ordering + fail-fast IterMemPackage

**URL:** https://github.com/gnolang/gno/pull/5605
**Author:** moul | **Base:** master | **Files:** 2 | **+208 -40**
**Reviewed by:** davd-gzl | **Model:** claude-sonnet-4-6

## Summary

Fixes a store consistency bug: `AddMemPackage` previously wrote counter→index→body. A SIGKILL between counter bump and body write left a pointer to a missing body; on restart `IterMemPackage→ParseMemPackage(nil)` caused a SIGSEGV crash-loop. Fix 1: new write order body→index→counter (any partial write is invisible or safely overwritable). Fix 2: `IterMemPackage` converted from goroutine-based lazy iterator to eager synchronous validator that panics with operator-actionable message on corruption instead of feeding nil to consumers.

## Test Results
- **Existing tests:** PASS (gnolang, vm/keeper, Files -short)
- **New tests:** PASS (`TestAddMemPackage_WriteOrderIsBodyFirst`, `TestIterMemPackage_InconsistentBaseStorePanics`, `TestIterMemPackage_MissingIndexPanics`)
- **Edge-case tests:** skipped

## Critical (must fix)
None.

## Warnings (should fix)

- [ ] `gnovm/pkg/gnolang/store.go:882-884` — **Stale comment.** Says "the consumer-side nil skip must be retained as belt-and-braces" and references "the defensive consumer in machine.go". Both wrong: commit 99e762bb4 replaced nil-yielding with panic, and `machine.go` has no nil guard. Actively misleads reviewers. Replace with note: corrupt state is unrecoverable, panic is intentional.
- [ ] `gnovm/pkg/gnolang/store.go:882` — **Phantom commit reference.** `"see commit b15ffde6e"` does not exist in repo history. Remove or replace with descriptive note.

## Nits

- [ ] `store_test.go:331` — `_ = strconv.Itoa` is awkward; `strconv` is unused after `incGetPackageIndexCounter` was deleted. Remove import.
- [ ] `store_test.go:283` — Comment claims "snapshotting" but test only checks postconditions, not intermediate state. Drop snapshotting language.
- [ ] `store.go:1011-1046` — `IterMemPackage` builds a `[]*std.MemPackage` slice then fills a channel from it — transient double-allocation. Minor; could fill channel directly in single pass.

## Missing Tests

- [ ] Idempotent retry after index-before-counter crash: write slot, crash before counter bump, retry `AddMemPackage` — slot overwritten cleanly, counter = 1 (not 2).
- [ ] `NumMemPackages()` vs `len(IterMemPackage())` agreement after multiple adds.

## Suggestions

- `FormatUint` (write) vs `Atoi` (read) asymmetry for counter: use consistent signed/unsigned variants throughout.
- Document in `IterMemPackage` that eager loading means panics unwind on caller's goroutine (recoverable), unlike the old goroutine design.

## Questions for Author

- The "12 min → 36 sec" replay speedup: from ordering fix only, or also from eliminating goroutine scheduling in `IterMemPackage`?
- Was the nil-guard in `machine.go` intentionally omitted (trusting panic path entirely) or overlooked when design shifted from nil-yield to panic?

## Verdict

APPROVE — Write-ordering fix is correct and sound. Only substantive issue is a stale comment at line 882-884 that should be corrected. All tests pass, minimal blast radius.
