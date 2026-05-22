# PR #5439: feat(gnovm): implement iterative exception recovery to prevent stack overflow

**URL:** https://github.com/gnolang/gno/pull/5439
**Author:** davd-gzl | **Base:** master | **Files:** 3 | **+140 -11**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.6

## Summary

This PR fixes a node-crashing vulnerability (NEWTENDG-182) in GnoVM's `Machine.Run()`. The old code used a recursive `defer/recover` pattern: when a Go-level `*Exception` panic was caught, it called `m.pushPanic(r.Value)` then recursively called `m.Run(st)`. Each panicking deferred function added one Go stack frame. An attacker could deploy a realm with thousands of deferred closures that each trigger nil pointer dereferences (Go-level `panic(&Exception{...})`), causing unbounded Go stack growth that exceeds the 1GB goroutine limit. The resulting `runtime.throw("stack overflow")` is fatal and bypasses all `recover()` handlers — including the VM keeper's `doRecover` and `BaseApp.runTx()` — killing the node.

The fix splits `Run()` into two methods: `Run()` now contains an iterative `for` loop that calls `runOnce()`. The `runOnce()` method has its own `defer/recover`, runs the op loop, and returns the caught `*Exception` (or nil on normal completion). Non-Exception panics are re-raised. The outer loop calls `pushPanic` and loops back, converting the Go-level exception into the cooperative op-stack path (OpReturnCallDefers + OpPanic2) without adding Go stack frames. This is O(1) Go stack depth regardless of the number of panicking defers.

Files affected: `gnovm/pkg/gnolang/machine.go` (core fix), `gno.land/pkg/integration/testdata/recursive_run_overflow.txtar` (regression test), `gnovm/adr/pr5439_iterative_exception_recovery.md` (ADR).

## Test Results
- **Existing tests:** PASS — all panic, defer, and recover file tests pass (21 panic tests, 12 defer tests, 28+ recover tests).
- **Edge-case tests:** The PR includes a txtar integration test that deploys a realm with 1000 panicking defers, verifies the error is reported (not a crash), then confirms the node is still alive.

## Critical (must fix)
None

## Warnings (should fix)
- [ ] `gnovm/adr/pr5439_iterative_exception_recovery.md:59` — The ADR references `gnovm/pkg/gnolang/recursive_run_overflow_test.go` as a "Regression test: 50K panicking defers", but this file does not exist in the PR. The actual regression test is the txtar file at `gno.land/pkg/integration/testdata/recursive_run_overflow.txtar`. The ADR also says line 1309 for `runOnce()` but the actual line is 1300. Fix the ADR to reference the correct file and line number.
- [ ] `gnovm/adr/pr5439_iterative_exception_recovery.md:1` — The title says `PRxxxx` instead of `PR5439`. Update to match the actual PR number.

## Nits
- [ ] `gnovm/pkg/gnolang/machine.go:1300` — The `runOnce` method comment says "Returns the caught exception, or nil if the loop completed normally." Consider adding a note that non-Exception panics are re-raised (not returned), since this is a critical behavior detail for anyone reading just the method signature.
- [ ] `gno.land/pkg/integration/testdata/recursive_run_overflow.txtar:33` — The test uses 1000 panicking defers. While sufficient for a regression test, the ADR describes the attack vector as needing ~50,000. Consider adding a comment explaining why 1000 is sufficient (e.g., "1000 is enough to validate the fix; the old code would stack-overflow at ~50K on default goroutine stack limits").

## Missing Tests
- [ ] No Go unit test directly exercises the iterative recovery path with a large number of exceptions. The txtar test is good for integration-level validation but a Go unit test (e.g., creating a Machine, pushing N panicking defers, and asserting it doesn't stack overflow) would provide faster feedback and be more robust.

## Suggestions
- Consider whether the 19 `panic(&Exception{...})` call sites across `values.go`, `alloc.go`, `realm.go`, and `machine.go` could eventually be migrated to the cooperative `pushPanic` path. This would eliminate the need for the `defer/recover` in `runOnce()` entirely. The current PR is the right incremental fix, but a tracking issue for the longer-term cleanup would be valuable.

## Questions for Author
- Is the integration test stable under CI timeouts? Starting a full `gnoland` node and deploying two realms is heavyweight. Have you observed any flakiness?

## Verdict
APPROVE — Clean, minimal fix for a critical node-crash vulnerability. The iterative `Run()`/`runOnce()` split correctly preserves all existing panic/defer/recover semantics while eliminating unbounded Go stack growth. The ADR has minor inaccuracies that should be fixed before merge.
