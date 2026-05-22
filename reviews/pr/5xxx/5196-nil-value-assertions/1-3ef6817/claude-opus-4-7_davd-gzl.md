# PR #5196: fix(gnovm): add nil checks for unsafe .V type assertions

**URL:** https://github.com/gnolang/gno/pull/5196
**Author:** davd-gzl | **Base:** master | **Files:** 9 | **+81 -24**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4-7

## Summary

Adds nil-V checks before three `.V` type assertions in the GnoVM that previously panicked with confusing Go-level errors instead of proper Gno-level panics, plus a defensive change to `Stacktrace()` for a (claimed) crash on negative `bodyStmt.NextBodyIndex`.

Concretely:

1. **`delete()` on nil map** — [gnovm/pkg/gnolang/uverse.go:700-702](gno/gnovm/pkg/gnolang/uverse.go#L700-L702). Adds `if arg0.TV.V == nil { return }` before the `arg0.TV.V.(*MapValue)` assertion. Matches Go's `delete(nilMap, key)` no-op semantics.

2. **`for range nilPtrToArray`** — [gnovm/pkg/gnolang/op_exec.go:155-228](gno/gnovm/pkg/gnolang/op_exec.go#L155-L228). The init phase (`case -2`) previously panicked unconditionally when `xv.V == nil`. The new logic:
   - When `V == nil`, derives the iteration length from the type (`baseOf(xv.T.Elem()).(*ArrayType).Len`), allowing `for range p` and `for i := range p` to iterate over the constant-length range with zero values, matching Go.
   - Defers the actual nil-pointer-dereference panic to the value-assignment phase (`case -1`, where `bs.Value != nil`), so only `for i, v := range p` panics. This mirrors Go where reading `v` requires dereferencing.

3. **`Stacktrace()` with negative `NextBodyIndex`** — [gnovm/pkg/gnolang/machine.go:478-485](gno/gnovm/pkg/gnolang/machine.go#L478-L485) and [gnovm/pkg/gnolang/nodes.go:1017-1027](gno/gnovm/pkg/gnolang/nodes.go#L1017-L1027). `Stacktrace()` calls `bs.LastStmt().GetLine()` which previously did `Body[NextBodyIndex-1]` — out of range when `NextBodyIndex` is `-2` or `-1` (range init / elem-assign phases). The PR returns `Body[0]` from `LastStmt()` in that range, and adds a nil guard at the call site. Also widens `bodyStmt.String()`'s `NextBodyIndex < 0` guard to `<= 0` (covers `NextBodyIndex == 0` with `Active != nil`, reachable via GOTO to body index 0).

Tests added:
- `map48.gno` — delete on nil maps (output match).
- `ptr_array4.gno` — refactored: was a `recover` demonstration on `_ = p[0:1]`, now a plain `// Error:` filetest.
- `ptr_array5.gno` — `for i := range p` (key-only) iterates 0,1,2 without panic.
- `ptr_array6.gno` — `for i, v := range p` panics with `runtime error: nil pointer dereference`.
- `ptr_array7.gno` — bare `for range p` iterates the constant length without panic.

Class of change is similar to #5195 (nil map read), #4452 (nil map range), #3856 (nil map len). The slice-of-nil-ptr fix originally in this PR (op_expressions.go) was already merged via #5166 and was removed after the merge conflict in commit `bd0f12be`.

## Test Results
- **Existing tests (PR-specific):** PASS — `TestFiles/map48.gno`, `TestFiles/ptr_array{0..7}.gno` all green.
- **CI:** all checks green.
- **Pre-existing failures unrelated to this PR:** `TestFiles/types/slice_5.gno` and `TestFiles/types/varg_12.gno` fail due to upstream type-checker error message changes that haven't been picked up in this branch's stale base. Not introduced by this PR.
- **Edge-case tests:** Wrote a reproducer that **does** trigger the stacktrace path the PR protects — see findings below.

## Critical (must fix)
- None.

## Warnings (should fix)
- [ ] [gnovm/pkg/gnolang/machine.go:478-485](gno/gnovm/pkg/gnolang/machine.go#L478-L485) and [gnovm/pkg/gnolang/nodes.go:1017-1027](gno/gnovm/pkg/gnolang/nodes.go#L1017-L1027) — **stacktrace fix is real and reachable, but lacks a regression test**. ltzmaxwell flagged this in [review 4240972032](https://github.com/gnolang/gno/pull/5196#pullrequestreview-4240972032). After deeper investigation, I found a concrete reproducer that the PR's nil-ptr-to-array tests don't cover:

  ```gno
  package main

  func main() {
      for k, v := range map[any]int{[]int{1, 2}: 10} {
          println(k, v)
      }
  }

  // Error:
  // runtime error: slice type cannot be used as map key
  ```

  With the stacktrace fix reverted, this test produces a Go-level crash:
  ```
  panic: runtime error: index out of range [-1]
  gnolang.(*bodyStmt).LastStmt nodes.go:1018
  gnolang.(*Machine).Stacktrace machine.go:483
  ```

  A second reproducer covers the for-loop variant:

  ```gno
  package main

  func main() {
      var s []int
      for ; s[0] == 0; {
          println("ok")
      }
  }

  // Error:
  // runtime error: nil slice index (out of bounds)
  ```

  Both crash without the fix with `panic: runtime error: index out of range [-3]` (because `NextBodyIndex == -2` and the old code did `Body[-3]`).

  Mechanism: panics that originate as `panic(&Exception{...})` directly (e.g. from [values.go](gno/gnovm/pkg/gnolang/values.go)) propagate up to [Run() at machine.go:1491-1494](gno/gnovm/pkg/gnolang/machine.go#L1491-L1494), which calls `m.Stacktrace()` to fill in the missing Stacktrace. At that moment the top stmt is a `bodyStmt` in its init phase (`NextBodyIndex == -2`) and `m.Lastline == 0`, so `bs.LastStmt()` is called and indexes `Body[-3]`.

  This differs from `pushPanic`-routed panics (used by the PR's own range-over-nil-ptr-to-array changes), where `pushPanic` captures the Stacktrace eagerly: by the time it runs, `m.Lastline` was set to the for-stmt's source line during the prior eval, so `Stacktrace()` short-circuits at [machine.go:474](gno/gnovm/pkg/gnolang/machine.go#L474) and never calls `LastStmt`. That's why the `ptr_array{4..7}` tests don't exercise the fix.

  **Recommendation:** add both reproducers above (or equivalents) under `gnovm/tests/files/` to lock the fix in. ltzmaxwell's concern is satisfied as soon as those tests land. **Note:** I noticed the fix also intentionally handles `NextBodyIndex == -1` (range elem-assign phase) and the `String()` change covers `NextBodyIndex == 0 with Active != nil` (GOTO into body index 0) — both of those are defensible defensive code I could not construct a reproducer for; consider documenting them with a code comment so a future reader knows what they protect.

## Nits
- [ ] [gnovm/tests/files/ptr_array4.gno](gno/gnovm/tests/files/ptr_array4.gno) and [ptr_array5.gno](gno/gnovm/tests/files/ptr_array5.gno) — both previously exercised `recover` over a nil-ptr panic. The PR drops the recover wrapping from both. General recover semantics are still covered elsewhere (`recover7.gno`, `recover9.gno`), but the recover-on-this-specific-panic interaction is no longer tested. Low-risk; consider keeping one of them as a recover variant (e.g. `ptr_array4_recover.gno`).
- [ ] [gnovm/pkg/gnolang/op_exec.go:170](gno/gnovm/pkg/gnolang/op_exec.go#L170) — the cast chain `baseOf(xv.T.Elem()).(*ArrayType).Len` will itself panic with a confusing type assertion error if `xv.T.Elem()` is somehow not an `*ArrayType`. In practice the preprocessor enforces this for `OpRangeIterArrayPtr`, so it should be unreachable, but a defensive type-switch with a clear panic message would match the spirit of this PR.
- [ ] [gnovm/pkg/gnolang/uverse.go:700-702](gno/gnovm/pkg/gnolang/uverse.go#L700-L702) — the early return on nil V skips the readonly check below. A readonly-tainted nil map (rare but possible via the N_Readonly attribute) will now silently no-op instead of returning "cannot delete from readonly tainted map". Matches Go semantics (no-op on nil), but worth a one-line comment noting the readonly check is intentionally skipped because no mutation occurs.
- [ ] [gnovm/pkg/gnolang/op_exec.go:185](gno/gnovm/pkg/gnolang/op_exec.go#L185) — for a nil `*[N]T`, `incrCPU(OpCPUSlopeRangeIterArray * N)` is now charged even though the iterations never dereference. Previously this code panicked immediately at zero gas cost. The new behavior is correct (iteration is real work) but is a behavioral change that should probably be called out in the PR body — for `*[10000]int{}`, that's 10000× the slope. Bounded by array-type length so unlikely to be abused, but flag for downstream gas accounting.

## Missing Tests
- [ ] **Regression tests for the stacktrace fix** — see Warning above. Two confirmed reproducers (both crash with `index out of range [-3]` when the fix is reverted, pass with the fix):
  - `range_init_panic.gno`: `for k, v := range map[any]int{[]int{1, 2}: 10} {}` — panic from `values.go` during range expression eval, bodyStmt at NextBodyIndex==-2.
  - `for_cond_panic.gno`: `var s []int; for ; s[0] == 0; {}` — panic from `values.go` during for-loop cond eval, bodyStmt at NextBodyIndex==-2.
  Adding both as `// Error:` filetests would close ltzmaxwell's concern.
- [ ] **`delete` on nil map within a realm** — `map48.gno` is a `package main` filetest. It would be worth a smoke test in a realm (`r/`) to confirm the early-return in `uverse.go` interacts correctly with `m.IsReadonly` / `Realm.DidUpdate` paths (it skips them, which is correct, but a realm-context test would lock that in).
- [ ] **`for k, v := range` on nil map** — already covered by #4452 per the PR description; no action here, just noting that the PR's class-of-fix list is correctly cross-referenced.

## Suggestions
- Move the `// In Go, ...` comments at [op_exec.go:165-169](gno/gnovm/pkg/gnolang/op_exec.go#L165-L169) and [op_exec.go:209-210](gno/gnovm/pkg/gnolang/op_exec.go#L209-L210) into a single doc-comment on the `OpRangeIterArrayPtr` case (or a short helper), so the rationale lives in one place rather than split across two phases of the same case.
- Consider adding an ADR per `gno/AGENTS.md` ("Every non-trivial AI-assisted PR must include an ADR"). The behavioral split between `for range p` / `for i := range p` (no panic) vs `for i, v := range p` (panics at value bind) is exactly the kind of subtle semantic alignment with Go that future maintainers will want documented. `gnovm/adr/pr5196_nil_ptr_array_range.md`.

## Questions for Author
- The `LastStmt()` change returns `Body[0]` during init phases. For stacktrace purposes, the line of the for-stmt itself (or the range-expr) seems more accurate than `Body[0]` (the first statement *inside* the loop body). Was returning `Body[0]` a deliberate choice over `nil` (which the call-site now handles), or a best-effort line-number heuristic?

## Verdict

**REQUEST CHANGES (small)** — the `delete()`, range-over-nil-ptr, and stacktrace fixes are all correct. The stacktrace fix is genuinely needed — the Go-level crash is reproducible with `for k, v := range map[any]int{[]int{1,2}: 10} {}` (and likely any `panic(&Exception{...})` from `values.go` that fires during a bodyStmt init phase). One small ask: add a regression filetest along those lines so the fix doesn't silently regress and to satisfy ltzmaxwell's "no test hit it" concern. With that test in place, this is good to merge.
