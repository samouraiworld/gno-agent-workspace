# PR #5501: fix(gnovm): recoverable panic, runtime error prefix

**URL:** https://github.com/gnolang/gno/pull/5501
**Author:** ltzmaxwell | **Base:** master | **Files:** 37 | **+150 -89**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.6

## Summary

This PR makes two related changes to GnoVM's runtime error handling:

1. **"runtime error:" prefix**: Adds the standard `runtime error:` prefix to all runtime panic messages (nil pointer dereference, division by zero, slice index out of bounds, negative shift amount, etc.). This brings Gno closer to Go's error message convention where runtime panics are prefixed with `runtime error:`.

2. **Recoverability**: Converts several plain Go `panic(string)` and `panic(fmt.Sprintf(...))` calls into proper `*Exception` panics (via `panic(&Exception{...})` or `m.Panic(typedString(...))`) so they can be caught by Gno's `recover()`. The key changes are:
   - `shlAssign`/`shrAssign` in `op_binary.go`: `panic(fmt.Sprintf(...))` → `m.Panic(typedString(...))`
   - `doOpPrecall` in `op_call.go`: same pattern for negative shift during type conversion
   - `GetSlice2` in `values.go`: all plain `panic(fmt.Sprintf(...))` calls → `panic(&Exception{...})`
   - `GetPointerAtIndex` in `values.go`: `panic("nil slice index (out of bounds)")` → `panic(&Exception{...})`

The PR adds 4 new file tests (`recover21-24.gno`) covering negative shift, 3-index slice negative bounds, high>max, and capacity-exceeded scenarios. It updates 29 existing test files to reflect the new message prefix.

All CI checks pass. The two `TestFiles` failures (`types/slice_5.gno`, `types/varg_12.gno`) are pre-existing on master and unrelated to this PR.

## Test Results
- **Existing tests:** PASS (all modified tests pass; 2 pre-existing failures unrelated to PR)
- **CI:** All checks green
- **Edge-case tests:** 4 new tests added (recover21-24.gno)

## Critical (must fix)
None

## Warnings (should fix)

- [ ] `gnovm/pkg/gnolang/values.go:2174` — **Pre-existing copy-paste bug**: The `high < 0` check in `GetSlice` formats the error message using `low` instead of `high`. The code reads:
  ```go
  if high < 0 {
      panic(&Exception{Value: typedString(fmt.Sprintf(
          "runtime error: invalid slice index %d (index must be non-negative)",
          low))})  // BUG: should be `high`
  }
  ```
  This bug exists on master already, but since the PR is touching this exact line to add the prefix, it's a good opportunity to fix it.

- [ ] `gnovm/tests/files/recover21.gno:8` — **Style inconsistency**: Uses `fmt.Println("recovered: ", r)` while all other recover tests use `println("recover:", r)`. This produces a different prefix ("recovered:" with double space vs "recover:" with single space) and unnecessarily imports `fmt`. Should use `println("recover:", r)` for consistency with recover12-20, 22-24.

- [ ] **PR body is empty** — The PR has no description explaining the motivation, the two distinct changes (prefix + recoverability), or the breaking change to error message strings. A description would help reviewers and future searchers understand the intent.

## Nits

- [ ] `gnovm/tests/files/recover14.gno:10` — The inline comment was changed from `// Panics because of division by zero` to `// Panics because of runtime error: division by zero`, which reads awkwardly as prose. Consider keeping the original wording or something like `// Panics with runtime error: division by zero`.

## Missing Tests

- [ ] No test for recovering from a negative shift in a type conversion context (the `doOpPrecall` change at `op_call.go:71`). `recover21.gno` tests the `shlAssign` path but not the `doOpPrecall` path. Example: `_ = uint(negativeValue)` where the value triggers the shift-RHS check.
- [ ] No test for recovering from `FieldType` or `*SliceType` used as map key (`values.go:1621,1648`). While these were already `*Exception` panics, they are now prefixed and it would be good to verify they're recoverable.
- [ ] No test for 3-index slice on a nil slice (`values.go:2305`), which was changed from `panic("nil slice index out of range")` to `panic(&Exception{...})`.

## Suggestions

- **Consider fixing the `GetSlice` copy-paste bug** at `values.go:2174` since the line is already being modified. One-character fix: `low` → `high`.
- **Document the breaking change**: Any Gno code that does `recover()` and string-matches on error messages (e.g., `r.(string) == "division by zero"`) will break. While this is unlikely in practice and the prefix is the correct convention, it should be called out in the PR description.
- **Message format divergence from Go**: Several messages don't match Go's exact wording:
  - Gno: `"division by zero"` vs Go: `"integer divide by zero"`
  - Gno: `"nil pointer dereference"` vs Go: `"invalid memory address or nil pointer dereference"`
  - Gno: `"negative shift amount: (-1 int)"` vs Go: `"negative shift amount"`
  - Gno: `"uninitialized map index"` — no Go equivalent (Go panics with "assignment to entry in nil map")

  These are intentional Gno-isms and probably fine, but worth documenting that full Go message compatibility is a non-goal.

## Questions for Author

- Is there a linked issue for this work? The PR has no body or linked issues.
- Was full Go message compatibility considered and rejected, or is it planned for a follow-up?
- The `doOpPrecall` negative-shift check at `op_call.go:71` uses `m.Panic()` which captures a stacktrace, but the surrounding code path continues after the panic (via the `runOnce` recovery loop). Was the `m.Panic` vs `m.pushPanic` choice deliberate here? `m.Panic` seems correct since the function returns to `doOpPrecall`'s caller via Go's panic/recover, but confirming intent.

## Verdict
**APPROVE** — The changes are correct, well-scoped, and all tests pass. The conversion from plain `panic(string)` to `*Exception` panics is the right fix for recoverability. The prefix addition aligns with Go's convention. The copy-paste bug in `GetSlice` and the missing PR description are worth addressing but not blocking.
