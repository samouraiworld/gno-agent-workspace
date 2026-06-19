# PR #5641: refactor(gnovm): stream print/println & panic formatting through a buffered metered writer

URL: https://github.com/gnolang/gno/pull/5641
Author: omarsy | Base: master | Files: 9 | +1622 -302
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 8e90f07e (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5641 8e90f07e`

**TL;DR:** `print`/`println` and panic formatting used to build the whole output string in memory (with `fmt.Sprintf` + `make([]string,N)` + `strings.Join` intermediates) and only then charge a flat gas cost. Printing a 1M-element slice allocated ~887 MB of native memory, none of it metered. This rewrites formatting to stream bytes into a small reused buffer and charges gas per flush, so output is now bounded by the per-transaction gas budget. Output bytes are unchanged.

**Verdict: APPROVE** — code is correct and the output is byte-identical to master (verified against master's old implementation, not just the PR's own goldens); the one decision left for a maintainer is accepting the deterministic gas-schedule change this introduces (print output ~0.1 → ~0.24 gas/byte, plus panic output now metered). morgan already reviewed the mechanism; omarsy addressed every comment.

## Summary
The pre-refactor `ProtectedString`/`Sprint` family builds output bottom-up: one Go string per leaf, `make([]string,N)`+`strings.Join` per aggregate, then a single after-the-fact gas charge on the joined result. For a wide value that is a ~100× native-allocation amplification over the value's own GnoVM footprint, and the output bytes themselves never pass the gas meter. This replaces the build-then-emit model with a `bufio.Writer`-style buffer (`meteredWriter`, 1 KB): formatters write incrementally, and each flush charges `allocGas(n)` once against the gas meter directly. The public `String`/`Sprint`/`ProtectedString`/`ProtectedSprint` become thin wrappers over the new `WriteProtected` path, so existing callers and output are unchanged. The headline effect is that `print`/`println` output is now gas-metered: a wide print trips `OutOfGasError` mid-traversal instead of completing unmetered.

## Fix
The accounting boundary is `meteredWriter` in [`values_string_stream.go:48-98`](https://github.com/gnolang/gno/blob/8e90f07e/gnovm/pkg/gnolang/values_string_stream.go#L48) · [↗](../../../../../.worktrees/gno-review-5641/gnovm/pkg/gnolang/values_string_stream.go#L48): it holds a `store.GasMeter` (never an `*Allocator`) and never allocates, because print output is a transient sink the GC never owns and must not count against the per-tx allocator budget. Numeric helpers `reserve(worst-case)` then `strconv.Append*` into the buffer tail, so no scratch slice escapes. `uversePrint` ([`uverse.go:1585-1609`](https://github.com/gnolang/gno/blob/8e90f07e/gnovm/pkg/gnolang/uverse.go#L1585) · [↗](../../../../../.worktrees/gno-review-5641/gnovm/pkg/gnolang/uverse.go#L1585)) and the panic-path callers stream through `SprintTo`. In production the unhandled-panic path is gated by `BoundedPanicRender=true` and still routes through the pre-existing bounded printer, so the new streaming panic path runs only in filetests/REPL.

## Glossary
- alloc gas — the calibrated per-allocation gas cost (`allocGas`, table in `alloc.go`), reused here as the per-byte output cost amount.
- `MaxAllocBytes` — the ~500 MB per-tx allocator/GC budget; print output deliberately does not count against it.

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
- `gnovm/pkg/gnolang/uverse.go:1592-1593` — comment says `formatUverseOutput` is "still used by benchmark instrumentation at lines ~920 and ~942"; the actual callers are [`uverse.go:1205`](https://github.com/gnolang/gno/blob/8e90f07e/gnovm/pkg/gnolang/uverse.go#L1205) · [↗](../../../../../.worktrees/gno-review-5641/gnovm/pkg/gnolang/uverse.go#L1205) and `1227`. Stale line numbers, off by ~280.

## Missing Tests
None blocking. See the Suggestion on the regression test's assertion.

## Suggestions
- **[test passes for the wrong reason]** `gno.land/pkg/integration/testdata/print_wide_value_gas_metering.txtar:14` — the assertion `stderr '(out of gas|allocation limit exceeded)'` does not pin the behavior the test documents.
  <details><summary>details</summary>

  The test's own comment says the signature of the print path is `location: stream output`, but the assertion accepts a bare "out of gas" or "allocation limit exceeded" anywhere. The slice make (8 MB) cannot trip `MaxAllocBytes` (~500 MB) here, so today the test does exercise the print path. But if a future change ever made the make trip the gas/alloc limit first, this assertion would still pass while never exercising print metering at all. Asserting `location: stream output` (or at minimum dropping the `allocation limit exceeded` alternative) makes the test actually pin the print-metering signature. Confirmed behaviorally: the tx aborts with `out of gas, gasWanted: 9000000, gasUsed: 9000164 location: stream output` (see [repro](comment_claude-opus-4-8.md)).
  Fix: assert `stderr 'location: stream output'`.
  </details>

## Open questions
- Gas-schedule change is consensus-relevant and disclosed by the author (print output rate, newly-metered panic output). Pre-launch and net-positive (closes an unmetered amplification path), so it reads as a deliberate maintainer sign-off item rather than a blocker; surfaced in the comment Body, not as an inline finding.

---

### Verification (CI-invisible)
- **Output byte-identical to master, not just to the PR's own goldens.** Ported the PR's 52-fixture corpus + golden map into a checkout of `origin/master` and ran them against master's pre-refactor `ProtectedString`; all 52 pass. The in-repo `TestSprintMatchesGolden` only asserts the new path against hand-written literals, so this independently confirms the literals equal master's real output. Cross-checked the bare (`writeProtectedSprint`) and wrapped (`WriteProtected`) dispatch against master's `ProtectedSprint`/`ProtectedString` case-for-case, including the `RefValue`-base slice path that the fixtures don't cover.
- **Print path is what trips, not the make.** `println(make([]int,1_000_000))` at `-gas-wanted 9000000` aborts with `gasUsed: 9000164 location: stream output`.
- **Test suites pass on the PR head 8e90f07e:** `TestSprintMatchesGolden`, `./gno.land/pkg/sdk/vm/ -run Gas`, `./gno.land/pkg/integration/ -run TestTestdata/print_wide_value_gas_metering`.
- **Filetests:** `go test ./gnovm/pkg/gnolang/ -run Files -test.short` fails on 10 cases (`types/{add,and,or}_f0`, `eql_0b4`, `eql_0f0`, `redeclaration3/4`, `redeclaration_global1`, `switch13`, `type41`), all of them `go/types` error-wording diffs in the typecheck phase (`cannot convert` vs `invalid operation`, `is not a type` vs `(local variable) is not a type`) — none touch the print/Sprint path. All fail identically on `origin/master`; a local go1.26 toolchain artifact, unrelated to this diff (CI uses the pinned toolchain and is green).
