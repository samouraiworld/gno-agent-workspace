# PR #5188: feat(test): Support Example tests

**URL:** https://github.com/gnolang/gno/pull/5188
**Author:** jefft0 | **Base:** master | **Files:** 10 | **+372 -22**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.6

## Summary

This PR adds support for Example test functions in the Gno test framework, mirroring Go's `testing` package behavior. In Go, functions named `Example*` in `_test.go` files with a trailing `// Output:` comment block have their stdout compared against the expected output.

**How it works:**

1. **Comment extraction at parse time** (`go2gno.go`, `util_go2gno.go`): The `Go2Gno` function signature gains a `fileComments []*ast.CommentGroup` parameter. When converting a `FuncDecl` whose name starts with `Example`, it calls `exampleOutput()` (adapted from Go stdlib) to extract the `// Output:` comment from the function body. The expected output and unordered flag are stored as node attributes (`ATTR_EXAMPLE_OUTPUT`, `ATTR_OUTPUT_UNORDERED`) on the `FuncDecl`.

2. **File-level comment routing** (`go2gno.go:559-567`): The `*ast.File` case passes `gon.Comments` (the file's full comment list) through `toDecl` to `Go2Gno`, so nested `FuncDecl` conversions have access to all comments. External callers pass `nil` for `fileComments`, which correctly skips example output extraction.

3. **Test execution** (`test.go:542-597`): After running regular `Test*` functions, `loadExampleTestFuncs()` collects all `Example*` functions (non-method, no parameters). For each example with an `ATTR_EXAMPLE_OUTPUT` attribute, it captures stdout via `tee`, evaluates the function, and compares output using `processExampleResult()`.

4. **Result processing** (`util_example.go`): Adapted from Go's `testing/example.go`. Compares trimmed stdout against trimmed expected output. For unordered output, sorts lines before comparing. Reports PASS/FAIL with duration.

5. **All 13 `Go2Gno` call sites** were updated to pass the new `fileComments` parameter (mostly `nil` for non-file contexts).

**Files affected:** `gnovm/pkg/gnolang/` (parser/AST), `gnovm/pkg/test/` (test runner), `gnovm/stdlibs/math/rand/` (existing example updated), plus 4 new txtar tests.

## Test Results

- **CI checks:** All passing
- **Existing tests:** PASS (all 4 new txtar tests pass: `example_test_pass`, `example_test_pass_unordered`, `failing_example_test`, `panic_example_test`)
- **Edge-case tests:** skipped

## Critical (must fix)

None

## Warnings (should fix)

- [ ] `gnovm/pkg/test/test.go:542-597` — **Examples ignore `opts.RunFlag` filtering.** Regular tests pass `opts.RunFlag` to `testing.RunTest` (line 470), allowing `-run` to filter which tests execute. The example loop runs all examples unconditionally. In Go's stdlib, `-test.run` filters examples too. Users running `gno test -run SpecificTest` will unexpectedly run all examples. Should add `shouldRun(opts.RunFlag, fname)` or similar filtering before executing each example.

- [ ] `gnovm/pkg/test/test.go:552` — **`defer revert()` inside a `for` loop.** The `defer` schedules `revert()` at function exit, not loop iteration end. While functionally harmless here (each `revert` captures the same original writer and the explicit `revert()` at line 596 handles the normal path), this is a well-known Go anti-pattern that can confuse readers and wastes deferred calls (N defers for N examples all firing redundantly at function exit). The `defer` does serve as a panic safety net for the current iteration, but a cleaner pattern would be to wrap the loop body in a closure or use an explicit cleanup in the panic recovery path.

- [ ] `gnovm/pkg/test/util_example.go:17` — **`finished` and `recovered` parameters are dead code.** `processExampleResult` is always called with `finished=true, recovered=nil` (line 587). When an example panics, the recover at line 359-370 catches it and `processExampleResult` is never reached. The XXX comment at lines 41-47 acknowledges the commented-out panic propagation logic. These parameters should either be removed to avoid misleading callers, or the panic recovery should be integrated so examples report proper FAIL messages before the panic propagates.

## Nits

- [ ] `gnovm/pkg/gnolang/nodes.go:146` — `ATTR_OUTPUT_UNORDERED` comment says "the expected output for an Example test function is unordered" but this attribute is a `bool`, not an output string. Consider: "whether the expected output comparison is unordered."

- [ ] `gnovm/pkg/gnolang/go2gno.go:549` — The condition `gon.Body != nil && strings.HasPrefix(gon.Name.Name, "Example") && fileComments != nil` duplicates the `gon.Body != nil` check from line 539. Minor redundancy, not a bug.

- [ ] `gnovm/pkg/test/test.go:544` — Examples without `ATTR_EXAMPLE_OUTPUT` are silently skipped. This matches Go's behavior (examples without output comments are compiled but not verified), but there's no verbose logging to indicate an example was skipped, which could confuse users debugging test suites.

## Missing Tests

- [ ] No test for `-run` flag filtering of examples. A txtar test running `gno test -run ExampleFoo` with multiple examples (ExampleFoo, ExampleBar) would verify filtering works — and currently would demonstrate it doesn't.
- [ ] No test for an example function that produces no output when one is expected (empty stdout vs non-empty expected output).
- [ ] No test for an example with a multi-line `// Output:` comment containing blank lines.

## Suggestions

- Consider extracting the example execution loop body (lines 543-596) into a helper function to avoid the `defer`-in-loop pattern and improve readability. This would also naturally scope the `defer revert()` correctly.
- The `loadExampleTestFuncs` function (line 636) could validate that example function names follow Go's naming conventions (e.g., `ExampleFoo`, `ExampleFoo_bar`, `Example_suffix` are valid; `Examplefoo` with lowercase after "Example" is not, unless it's exactly "Example"). Go's stdlib has this validation in `cmd/go/internal/load`. This isn't blocking but would improve parity.

## Questions for Author

- Is the omission of `-run` flag filtering for examples intentional, or an oversight? Go's `go test -run` filters examples by name.
- For the `finished`/`recovered` parameters in `processExampleResult`: is there a plan to integrate example-specific panic recovery (so a panicking example reports FAIL rather than aborting the entire test run), or is the current behavior (panic propagates to the top-level recover) the intended design?

## Verdict

REQUEST CHANGES — The missing `-run` flag filtering for examples is a behavioral gap that breaks user expectations; the other findings are lower severity but the dead code parameters and defer-in-loop pattern should be cleaned up.
