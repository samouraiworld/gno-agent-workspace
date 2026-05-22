# PR #5188: feat(test): Support Example tests

**URL:** https://github.com/gnolang/gno/pull/5188
**Author:** jefft0 | **Base:** master | **Files:** 14 | **+516 -26**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.7

This is a follow-up review focused on the most recent commit `68fdcce` which addresses the panic-recovery gap surfaced in round 1 (`reviews/pr/5xxx/5188-support-example-tests/1-b9935be/`). The earlier round-1 findings (missing `-run` filter, dead `finished`/`recovered` params on `processExampleResult`, defer-in-loop, attribute comment, redundant body-nil check) are not re-litigated here unless this commit changes them.

## Summary

Commit `68fdcce` ("fix: Recover panic in example test") closes the bug where one panicking example aborted all remaining examples in the same package. It does so by introducing a Gno-side wrapper, `testing.RunExampleTest`, that:

1. Defers `recoverWithStacktrace()` around `exampleTestFunc()`.
2. On non-nil recovered value, prints `panic: <v>\nStacktrace:\n<st>\n` to `os.Stderr` and sets the named return `panicked = true`.
3. Returns false on normal completion.

On the Go side, `runTestFiles` in `gnovm/pkg/test/test.go` is reorganised:

- `testingpv` / `testingtv` / `testingcx` are hoisted out of the per-test inner loop to function scope (lines 391–393), so both the test loop and the new example loop can reuse them.
- Inside the example loop, `runExampleTestX` / `runExampleTest` / `runExampleTestCX` are evaluated per iteration (lines 564–566) on the per-iter machine — same pattern as `runTestX`/`runTestCX` for `RunTest`.
- The naked `m.Eval(gno.Call(gno.Nx(fname)))` is replaced by a call routed through `RunExampleTest`: `m.Eval(gno.Call(runExampleTestCX, gno.Nx(fname)))`.
- The boolean return is read via `eval[0].GetBool()`. On `true`, the iteration prints `--- FAIL: <name> (dur)` to `opts.Error` and skips the output-comparison branch. On `false`, the existing path runs (`--- GAS:`, capture stdout, `processExampleResult`).

A new txtar test `gnovm/cmd/gno/testdata/test/example_test_panic_recovery.txtar` covers the regression: two examples, the first panics, the second is asserted to still PASS.

Plumbing: `os.Stderr` inside the gno VM routes through `proxyWriter.StderrWrite` → `errW`, which during the example loop is teed into both `opts.Error` and `opts.filetestBuffer` (see `tee()` at `gnovm/pkg/test/test.go:197-204`). That means the `panic: …` line and stacktrace land on the host stderr (matching the txtar `stderr 'panic: …'` assertion) and *also* in `filetestBuffer` — but the panicked branch in `runTestFiles` skips the buffer comparison entirely, so this doesn't cause a false PASS.

## Test Results

- **CI checks:** All required checks PASS (`build`, `analyze`, `check`, `codecov/patch`, etc.). Only failing item is `Merge Requirements` (pending tech-staff review approval — bot gate, not code).
- **Existing tests:** PASS — `Test_Scripts/test/(example_test_pass|example_test_pass_unordered|failing_example_test|failing_example_no_output_test|panic_example_test|example_test_panic_recovery)` all pass against `68fdcce` in the worktree.
- **Edge-case tests:** 3 written, all PASS:
  - `example_multi_panic_test.txtar` — interleaved panicking/passing examples (A panics, B passes, C panics, D passes); confirms each example runs independently.
  - `example_panic_after_output_test.txtar` — example prints to stdout *then* panics; confirms the printed stdout doesn't accidentally pass output comparison and the next example is unaffected.
  - `example_internal_recover_test.txtar` — example uses `defer recover()` to swallow its own panic; confirms `RunExampleTest` does NOT mark it as panicked and the captured stdout is still compared (PASS).

  Saved under `reviews/pr/5xxx/5188-support-example-tests/2-68fdcce/tests/`.

## Critical (must fix)

None.

## Warnings (should fix)

- [ ] `gnovm/pkg/test/test.go:610` — **`processExampleResult` called only on the non-panic branch with `finished=true, recovered=nil`.** Round-1 already flagged that `finished` and `recovered` are dead. With this commit they are *more* clearly dead: the panic path no longer reaches `processExampleResult` at all (it FAILs directly at line 600). Either drop those two parameters from `processExampleResult` (and the matching block in `gnovm/pkg/test/util_example.go:17`), or move the panic-FAIL formatting into `processExampleResult` so all output reporting stays in one place.

- [ ] `gnovm/pkg/test/test.go:557-558,620` — **Defer-in-loop pattern still present** (round-1 finding). With the new control flow this is unchanged, but worth re-confirming: each iteration calls `revert()` explicitly at line 620 and *also* registers a `defer revert()` that fires only at function exit. The defer is the safety net for native (Go-level) panics from `m.Eval` that escape both `RunExampleTest`'s gno-level recover and the per-iter explicit revert. The outer `recover()` at `runTestFiles:359-369` will then trip and the deferred reverts will restore `proxyWriter.{w,errW}` correctly. Functionally fine but unidiomatic; an extracted helper would scope the defer naturally.

## Nits

- [ ] `gnovm/cmd/gno/testdata/test/example_test_panic_recovery.txtar:1-13` — **Stale comments describe the pre-fix state.** The header says "Currently FAILS: the second example is never executed …" and the assertion comment says "This assertion will FAIL with the current implementation." After this commit those statements are no longer true — they describe the bug, not the test contract. Replace with something like:
  ```txtar
  # Regression: a panicking example must not prevent subsequent examples from running.
  # Verifies that testing.RunExampleTest recovers panics in-process, so each example
  # gets its own independent run.
  ```

- [ ] `gnovm/tests/stdlibs/testing/testing.gno:357-370` — **`switch err.(type) { case nil: default: … }` for a single non-nil branch.** Functionally identical to `if err != nil { … }`; the switch-type form is justified in `tRunner`/`tRunner_cur` because they also handle `case SkipErr`, but `RunExampleTest` doesn't honour `Skip`. Either:
  1. Decide examples should support `t.Skip`-equivalent semantics and add the `SkipErr` case to match `tRunner`.
  2. Simplify to `if err != nil { … }` since the switch buys nothing here.

- [ ] `gnovm/tests/stdlibs/testing/testing.gno:357` — **Doc comment says "the example test function of no arguments."** Slightly awkward phrasing; consider "Runs an example test function (no arguments) and recovers any panic, returning whether the function panicked."

- [ ] `gnovm/pkg/test/test.go:564-566` — **Per-iteration evaluation of `runExampleTestX`/`runExampleTestCX`.** Mirrors the `RunTest` pattern in the test loop, so consistency is fine, but unlike `RunTest` (where the choice between `RunTest` and `runTest_cur` depends on per-test `IsCrossing()`), `RunExampleTest` is invariant across iterations. Hoisting these three lines out of the loop (alongside the `testingcx` hoist this commit already does) would be a cheap follow-up.

## Missing Tests

- [ ] **Multiple panicking examples in the same package.** `example_test_panic_recovery.txtar` has *one* panicking example and one passing one. Add a case where two examples panic (with a passing one between them) to lock in independence. Adversarial test `example_multi_panic_test.txtar` (in this review's `tests/` dir) is a candidate for upstreaming.
- [ ] **Example that prints to stdout *before* panicking.** Confirms (a) the panic path doesn't accidentally compare partial stdout against expected output, and (b) `filetestBuffer.Reset()` between iterations correctly clears mixed stderr/stdout from the panicked iteration. Adversarial `example_panic_after_output_test.txtar` covers this.
- [ ] **Example that recovers its own panic via `defer recover()`.** Should NOT be reported as panicked, and stdout (including text printed inside the deferred handler) should still be compared against the `// Output:` block. Adversarial `example_internal_recover_test.txtar` covers this.

## Suggestions

- After deciding on the dead-param question (Warning above), keep `RunExampleTest` symmetric with `tRunner`: print `--- FAIL: %s (%s)` from the gno side rather than from Go (currently the FAIL line is printed by Go at `test.go:600` while the panic header is printed by Gno at `testing.gno:364`). Putting both on the same side would make the output stream less interleaved with Go-side bookkeeping.
- Consider whether `RunExampleTest` should respect `failfast`. `runTestFiles` checks `opts.FailfastFlag` for regular tests at line 615 — already present in this commit's example loop. Good. But there is no in-loop early-abort for `failfast` *during* the test loop on a panic that was recovered by `RunTest`-equivalent paths; behaviour parity with `go test -failfast` for examples should be confirmed in a follow-up.

## Questions for Author

- Why hoist `testingcx`/`testingpv`/`testingtv` to function scope (line 391) but evaluate `runExampleTestX`/`runExampleTestCX` per-iteration? The latter doesn't depend on iteration state — was that purely to mirror the `RunTest` block's structure?
- The stacktrace currently includes the `RunExampleTest(<fn>)` frame and a `<VPBlock(...)>` annotation on the function name. Is that the intended UX, or would you want to strip the `RunExampleTest` frame the way `tRunner` frames are typically stripped from user-facing test output?
- Should examples that produce no output but have an `// Output:` comment (already covered by `failing_example_no_output_test.txtar`) and examples that *panic* (this commit) share a single FAIL formatter? Today they live in two places (`util_example.go` for the former, `test.go:600` for the latter).

## Verdict

APPROVE (with minor follow-ups) — the panic-recovery fix is correct, well-scoped, and verified by the new txtar plus three adversarial cases. The dead `finished`/`recovered` params and the stale txtar header comments should be cleaned up before merge; everything else is non-blocking.
