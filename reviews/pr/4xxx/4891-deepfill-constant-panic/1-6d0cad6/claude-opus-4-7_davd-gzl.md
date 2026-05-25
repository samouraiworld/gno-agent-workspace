# PR #4891: fix(gnovm): Add panic on `Deepfill` execution on constant type

URL: https://github.com/gnolang/gno/pull/4891
Author: davd-gzl | Base: master | Files: 3 | +56 -12
Reviewed by: davd-gzl | Model: claude-opus-4-7

Disclosure: PR author and reviewer share a GitHub account. Review run by an unattended agent on technical merits; conflict noted up front.

**Verdict: NEEDS DISCUSSION** — design has a substantive open objection from [@ltzmaxwell](https://github.com/gnolang/gno/pull/4891#issuecomment-2832168893) that the author has not responded to since 2026-04-28; the wrapper-guard pattern violates the interface contract and creates a load-bearing convention with no compile-time enforcement.

## Summary

The auditor flagged that `StringValue`, `BigintValue`, `BigdecValue` are constants-only types (never persisted, GC-skipped per [`garbage_collector.go:403-413`](../../../../../.worktrees/gno-review-4891/gnovm/pkg/gnolang/garbage_collector.go#L403-L413)), so their `DeepFill` methods returning self were "indicators of incorrect handling" — issue [#4777](https://github.com/gnolang/gno/issues/4777) recommended either removing the methods or adding **debug-only** assertions. This PR makes the three methods unconditionally panic, then adds a runtime type-switch guard at the [`(*TypedValue).DeepFill`](../../../../../.worktrees/gno-review-4891/gnovm/pkg/gnolang/values.go#L2327-L2336) wrapper that catches these three types before dispatch. Net effect: external callers (via the wrapper) never trigger the panic; direct interface dispatch (e.g. inside a future `MapValue.DeepFill`) would panic.

## Glossary

- `DeepFill` — synchronous recursive resolution of `RefValue` references to concrete values; used pre-`Gno2GoValue` in genstd-generated native bindings ([`values.go:25-29`](../../../../../.worktrees/gno-review-4891/gnovm/pkg/gnolang/values.go#L25-L29)).
- `StringValue` / `BigintValue` / `BigdecValue` — leaf value types used for constant expressions and untyped runtime values; never persisted as separate objects, no `ObjectInfo`.
- `VisitAssociated` — GC visitor for child references; returns `false` for these three types ([`garbage_collector.go:403-413`](../../../../../.worktrees/gno-review-4891/gnovm/pkg/gnolang/garbage_collector.go#L403-L413)), confirming the no-children property.

## Fix

Before: each of the three constant types' `DeepFill` returned the receiver; the wrapper called `tv.V = tv.V.DeepFill(store)` indiscriminately. After: the three methods panic; the wrapper short-circuits via a `switch tv.V.(type)` on the three concrete types and skips dispatch. The PR also refactors `ArrayValue.DeepFill`, `StructValue.DeepFill`, `HeapItemValue.DeepFill` to call `tv.DeepFill(store)` instead of inlining `if tv.V != nil { tv.V = tv.V.DeepFill(store) }` — genuine DRY win because the guard now lives in one place.

## Warnings (should fix)

- **[interface contract violation]** [@ltzmaxwell](https://github.com/gnolang/gno/pull/4891#issuecomment-2832168893) [`gnovm/pkg/gnolang/values_fill.go:7-17`](../../../../../.worktrees/gno-review-4891/gnovm/pkg/gnolang/values_fill.go#L7-L17) — panicking leaves break the documented `DeepFill` contract.
  <details><summary>details</summary>

  The `Value.DeepFill` interface ([`values.go:23-33`](../../../../../.worktrees/gno-review-4891/gnovm/pkg/gnolang/values.go#L23-L33)) says "DeepFill returns the same value, filled." For a leaf with no references, returning self IS the correct identity case — not an error condition. Panicking turns a valid base case into "shouldn't happen," which is inverted: the leaves are exactly where the recursion terminates.

  Concrete consequence: the wrapper-guard pattern becomes load-bearing for correctness, not just an ergonomic helper. `MapValue.DeepFill` is currently [`panic("not yet implemented")`](../../../../../.worktrees/gno-review-4891/gnovm/pkg/gnolang/values_fill.go#L65); when implemented, the author MUST route map element `TypedValue`s through `tv.DeepFill(store)`, not `tv.V.DeepFill(store)` directly. Nothing in the type system enforces this — same for any future implementer of `FuncValue.DeepFill`, `BoundMethodValue.DeepFill`, `PackageValue.DeepFill`, `Block.DeepFill`. The current refactor of `ArrayValue` / `StructValue` / `HeapItemValue` did remember; future implementers must too.

  Fix: revert the panics to `return sv` / `return biv` / `return bdv` (or keep them gated behind `if debug { panic(...) }` per the auditor's original recommendation in [#4956](https://github.com/gnolang/gno/issues/4956), which says "Panics happen only in debug mode"). Then drop the type-switch in `(*TypedValue).DeepFill` — the wrapper goes back to its trivial form. Keep the in-package refactor (`tv.DeepFill(store)` instead of inlined nil-check + reassign), which is the genuine improvement in this PR.
  </details>

- **[diverges from auditor recommendation]** [`gnovm/pkg/gnolang/values_fill.go:7-17`](../../../../../.worktrees/gno-review-4891/gnovm/pkg/gnolang/values_fill.go#L7-L17) — unconditional panic vs. debug-only panic.
  <details><summary>details</summary>

  Issue [#4956](https://github.com/gnolang/gno/issues/4956) cites the auditor's follow-up: "Panics happen only in debug mode. Those functions should never be reached even in non debug mode." The PR makes the panics unconditional and prevents them from triggering via a wrapper guard. The net runtime behavior matches the auditor's intent (the panic never fires in correct code), but the implementation differs from the literal suggestion — there's no `if debug { panic(...) }` form using the existing `debug debugging` build flag at [`debug_true.go:5`](../../../../../.worktrees/gno-review-4891/gnovm/pkg/gnolang/debug_true.go#L5).

  The package already has the `debug` constant pattern available. Using `if debug { panic(...) }` would let the leaves still return self in production (preserving interface contract semantics), eliminate the need for the wrapper type-switch, and match the auditor recommendation verbatim. Fix: gate the panics behind `debug`, drop the wrapper guard.
  </details>

- **[merge conflict]** PR-level — `mergeable: CONFLICTING` per `gh pr view`.
  <details><summary>details</summary>

  The branch needs a rebase or merge with master before it can be merged. Last commit on the branch is from 2025-12-10; the merge from master is from 2026-04-28; substantial drift since.
  </details>

## Nits

- [`gnovm/pkg/gnolang/values.go:2331`](../../../../../.worktrees/gno-review-4891/gnovm/pkg/gnolang/values.go#L2331) — comment "Do nothing - these are constant values" is correct but the surrounding code shape is the real "huh?" trigger. If the wrapper guard stays, prefer a single-line `// Skip leaf types that panic on DeepFill (see values_fill.go).` so the reader knows where to look for the invariant.

- [`gnovm/pkg/gnolang/values_fill_test.go:11`](../../../../../.worktrees/gno-review-4891/gnovm/pkg/gnolang/values_fill_test.go#L11) — test name is "verifies that TypedValue.DeepFill correctly handles constant value types ... by not calling their DeepFill methods (which panic)." The test asserts the no-panic behavior of the wrapper, but the wrapper's correctness depends on the leaves continuing to panic. If the panic is removed, the test still passes — it's not actually pinning the contract. A `defer recover()` assertion on direct calls (e.g. `StringValue("x").DeepFill(nil)` should panic) would lock the current shape.

## Missing Tests

- **[direct-call panic not asserted]** [`gnovm/pkg/gnolang/values_fill_test.go`](../../../../../.worktrees/gno-review-4891/gnovm/pkg/gnolang/values_fill_test.go) — the PR's added tests only cover the happy path through the wrapper. They never verify that calling `StringValue.DeepFill(nil)` directly panics, which is the load-bearing change introduced by this PR. Adding a `recover`-based assertion would make the PR self-documenting and catch a future contributor who reverts the panics without re-evaluating the wrapper guard.

## Questions for Author

- [@ltzmaxwell raised a substantive design objection on 2026-04-28](https://github.com/gnolang/gno/pull/4891#issuecomment-2832168893); the PR has not been updated since. Do you accept the alternative (revert leaves to return-self, drop the wrapper guard, keep only the in-package refactor)? If not, what's the counter-argument for keeping the panics unconditional rather than `debug`-gated?
- Issue [#4956](https://github.com/gnolang/gno/issues/4956) explicitly asks for panics in debug mode only. Why was the unconditional-panic-plus-wrapper-guard approach chosen over `if debug { panic(...) }`? The package already exposes a `debug` build flag.
- The branch is in `CONFLICTING` state; please rebase on master before any further review pass.

## Suggestions

- [`gnovm/pkg/gnolang/values_fill.go:64-69`](../../../../../.worktrees/gno-review-4891/gnovm/pkg/gnolang/values_fill.go#L64-L69) — the unimplemented `FuncValue`, `MapValue`, `BoundMethodValue`, `TypeValue`, `PackageValue`, `Block` `DeepFill` methods all panic with `"not yet implemented"`. The wrapper-guard approach in this PR doesn't generalize to them (they need real implementations, not skipping). Worth a tracking issue if not already covered, since the same auditor concern (interface implementations that "shouldn't be called") still applies here.

## Critical (must fix)

None.
