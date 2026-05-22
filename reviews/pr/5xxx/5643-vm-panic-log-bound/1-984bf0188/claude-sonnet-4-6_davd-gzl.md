# PR #5643: fix(vm): bound panic-Log rendering to prevent unmetered long running txs

**URL:** https://github.com/gnolang/gno/pull/5643
**Author:** @mvertes | **Base:** master | **Files:** 24 | **+3292 -319**
**Reviewed by:** davd-gzl | **Model:** claude-sonnet-4-6

## Summary

This PR is a large security/correctness fix that closes several denial-of-service vectors in the GnoVM validator path. It bundles four distinct but related workstreams:

**1. Bounded panic-log rendering (the headline fix)**

The VM's error-recovery path (`doRecoverInternal` in `keeper.go`) previously called `fmt.Errorf("%v", r)` and `m.Stacktrace().String()` on arbitrary panic values, which could trigger unbounded string allocation (e.g., a panic with a huge `StringValue` or a deeply-nested struct). The PR introduces:

- `gnovm/pkg/gnolang/bounded_strings.go` (829 lines): A complete bounded-rendering library for Gno values, exceptions, and stacktraces. Key API: `BoundedSprintTV`, `BoundedSprintException`, `BoundedStacktrace`, `BoundedExceptionStacktrace`. Internal helpers are deliberately machine-free (no user `.String()`/`.Error()` dispatch). User-defined methods are called only via `boundedUserSprint`, which tightens the allocator to 64 KB and recovers from any panic.
- `gno.land/pkg/sdk/vm/bounded.go` (115 lines): A whitelist-based `boundedString(v any, depth int) string` for the keeper's recovery path. Handles `UnhandledPanicError`, `*Exception`, `*PreprocessError`, `OutOfGasError`, `abci.Error`, and `cmnError` types with dedicated bounded renders; falls back to `<%T>` for anything else. Recursion via `errors.Unwrap` is capped at depth 8.
- All validator-side `Machine` constructions in `keeper.go` now set `BoundedPanicRender: true`. A new branch in `makeUnhandledPanicError()` in `op_call.go` uses `BoundedSprintException` instead of unbounded `Sprint` when the flag is set.
- `tm2/pkg/sdk/helpers.go`: `clipLog()` function clips ABCI log lines at 1 KB per line, 16 lines total (~17 KB net). Applied to `ABCIResultFromError`, `ABCIResponseQueryFromError`, and the `default` branch of `baseapp.go`'s inner recover.

**2. Per-tx preprocess allocator (closes const-fold DoS)**

Preprocess sub-Machines (spun up inside `evalStaticType`, `evalConst`, `tryEvalStatic`, `evalStaticTypeOfRaw` at preprocess.go lines ~3947, 4112, 4175, 4258) previously ran with `nil Alloc`, so adversarial constant folding (e.g., doubling-concatenated string constants growing 2^N bytes from O(N) source) could allocate unbounded heap. The fix:

- `SetPreprocessAllocator`/`GetPreprocessAllocator` added to `Store` interface and `defaultStore`. The allocator is inherited via `BeginTransaction`.
- `keeper.AddPackage` and `keeper.Run` install a `gno.NewAllocator(maxAllocTx)` (500 MB hard cap, `collect=nil` so no GC retry) on the store before `RunMemPackage`, and clear it via `defer`. The gas meter is shared with the outer tx machine.
- `NewMachineWithOptions` detects the preprocess allocator from the store (`isPreprocessing=true`) and skips `SetGCFn`/`SetGasMeter` overwrite, preserving the hard-cap semantics.
- The allocator's gas meter charges preprocess alloc-gas against the tx budget, so an attacker pays for the work done before the cap fires.

**3. Preprocessing O(N²) → O(N) algorithmic fixes**

Several preprocess algorithms had quadratic or higher complexity for adversarial inputs:

- `initStaticBlocks1` (loopvar renaming): Replaced a nested O(K×M) walk (per-loopvar full-tree re-scan via `replaceAllLoopvar`) with a single O(N) `TranscribeB` pass using a stack-of-maps for scope tracking.
- `StaticBlock.GetLocalIndex`: O(1) `nameIndex` map lazily built above `nameIndexThreshold=32` names. Maintained by `Define2` and `SetStaticBlock`.
- `DeclaredType.lookupMethod` / `TryDefineMethod` / `FindEmbeddedFieldType` / `GetPathForName`: O(1) `methodIndex` map above `methodIndexThreshold=8` methods.
- `FuncLitExpr.heapCapturesIdx`: O(1) map for heap-capture deduplication in `addHeapCapture`.
- `addAttrHeapUse` / `hasAttrHeapUse`: Changed from `[]Name` + `slices.Contains` to `map[Name]struct{}`.
- `evalStaticType`: Fast path `staticTypeFromAST` constructs composite types directly from AST fields (avoiding Machine spin-up) for `SliceTypeExpr`, `ArrayTypeExpr`, `StarExpr`, `MapTypeExpr`, `ChanTypeExpr`, `FuncTypeExpr`, `StructTypeExpr`, `InterfaceTypeExpr`.
- `doOpStaticTypeOf`: Refactored via `staticTypeOfX()` helper that short-circuits on cached `ATTR_TYPEOF_VALUE`.

**4. Type-validation depth caps**

New functions in `types.go` validate at type-construction time:
- `validateTypeDepth`: Panics if composite type wrapper nesting exceeds `MaxTypeDepth=8`.
- `validateEmbedDepth`: Panics if struct/interface embed chain depth exceeds `MaxEmbedDepth=8`.
- `validateStructFields` / `validateInterfaceMethods`: Panic if effective accessible-name / method count exceeds `MaxStructFields=128` / `MaxInterfaceMethods=128`.
- New `effectiveFields`/`effectiveMethods` cache fields on `StructType`/`InterfaceType`.

**Files affected by area:**
- `gnovm/pkg/gnolang/`: `bounded_strings.go` (new), `bounded_strings_test.go` (new), `preprocess_alloc_test.go` (new), `alloc.go`, `debug.go`, `frame.go`, `machine.go`, `nodes.go`, `op_call.go`, `op_types.go`, `preprocess.go`, `store.go`, `types.go`
- `gno.land/pkg/sdk/vm/`: `bounded.go` (new), `bounded_test.go` (new), `gas_test.go`, `keeper.go`
- `tm2/pkg/sdk/`: `helpers.go`, `helpers_clip_log_test.go` (new), `baseapp.go`
- `gno.land/pkg/integration/testdata/`: 4 txtar files (1 new, 3 updated for gas changes)

## Test Results

- **`gno.land/pkg/sdk/vm` (all tests):** PASS (36.4s)
- **`gnovm/pkg/gnolang` (all tests, -short):** PASS (190.8s)
- **`tm2/pkg/sdk` (all tests):** PASS (0.05s)
- **`TestBoundedString_*` (15 tests):** PASS
- **`TestBoundedSprintTV_*`, `TestBoundedStacktrace_*`, `TestStacktraceFuncName_*`:** PASS (30 tests)
- **`TestPreprocessAlloc_*` (7 tests):** PASS
- **`TestAddPkgDeliverTx_PreprocessAllocLimit`:** PASS (5.8s)
- **`TestRunDeliverTx_AdversarialErrorOOG`:** PASS (0.2s)
- **`TestClipLog_*` (8 tests):** PASS
- **Edge-case tests:** All included adversarial tests pass; no additional tests written by reviewer.

## Critical (must fix)

- [ ] `gnovm/pkg/gnolang/bounded_strings.go:220-253` — `boundedUserSprint` restores `m.Alloc.bytes = savedBytes` in the `defer` unconditionally, **even on the success path**. This silently discards all allocations performed by `tv.Sprint(m)`, rolling back the machine's bytes counter regardless of whether a panic occurred. For the recovery path this is arguably intentional ("don't charge the panic-rendering against the main alloc budget"), but it means a user-defined `Error()` that allocates 60 KB of strings gets those 60 KB accounted for by gas (via `ConsumeGas` during alloc) but *not* by the `m.Alloc.bytes` counter. This creates a subtle inconsistency: the allocator's GC decision (`collect()` fires when `bytes` overflows `maxBytes`) will fire later (or not at all) if a large rendering happened. Concretely, if the tx then continues (after a non-fatal recovery path), the post-Sprint allocation budget looks artificially wide. The current callers only reach `boundedUserSprint` via `makeUnhandledPanicError` (which terminates the tx), so the risk is currently limited to re-entry scenarios. Still, silently resetting `bytes` on success is dangerous and should either be documented more prominently or the restore should only happen in the panic branch.

- [ ] `gno.land/pkg/sdk/vm/keeper.go:964-983` — `doRecoverQueryNoMachine` still uses `fmt.Errorf("%v", r)` and `fmt.Sprintf(...r...)` on arbitrary panic values without bounding. Query paths that don't spin up a VM Machine (e.g., `QueryObjectJSON` at line 1444, `QueryObjectBinary` at line 1471) are not covered by `boundedString`. An adversarial contract that causes a panic in object serialization (e.g., a giant JSON-encoded object) can produce an unbounded log string. The `clipLog` applied in `baseapp.go` is a backstop but `doRecoverQueryNoMachine` still passes the unbounded result to `errors.Wrapf` which may buffer the full string before `clipLog` sees it. Should use `boundedString(r, 0)` and `clipLog` on the debug stack.

## Warnings (should fix)

- [ ] `gnovm/pkg/gnolang/bounded_strings.go:313-323` — `BoundedExceptionStacktrace` is called from `doRecoverInternal` when handling `UnhandledPanicError`. At that call site (`keeper.go:944`), `m` is the machine that panicked and `m.Exception` is still set (the Go panic propagated without running Gno defers). However, the function signature `BoundedExceptionStacktrace(m *Machine, lim int)` reads `m.Exception.Stacktrace` — which is the exception's own stacktrace captured at `panic()` time, not the current machine's Go-level call frames. This is correct only if the exception stacktrace was captured at the right moment. The existing `m.ExceptionStacktrace()` call that this replaces had the same dependency. The concern is: the `Stacktrace` field on `Exception` is set in `machine.go:2688-2693` — confirm via test that this is populated before `makeUnhandledPanicError()` fires. This looks correct today but is fragile; add a comment or assertion.

- [ ] `gnovm/pkg/gnolang/nodes.go:1631-1636` — `StaticBlock.SetStaticBlock` clears `nameIndex` on the *source* (`osb.nameIndex = nil`) before assigning `*sb = osb`. This means the receiving `sb` also gets `nameIndex = nil` (correctly), but the *caller's* `osb` variable (a copy) also has its `nameIndex` zeroed. Since `osb` is passed by value, this has no effect on the caller's original struct. However, the comment says "Reset on assign so the destination rebuilds lazily from its own Names instead of aliasing the source's map". The correct way to prevent aliasing is to nil-out the *destination's* index after the assignment: `sb.nameIndex = nil`. The current code happens to work because `osb.nameIndex = nil` before `*sb = osb` gives the same result (nil is assigned), but the intent and documentation are misleading. Should be `*sb = osb; sb.nameIndex = nil`.

- [ ] `gnovm/pkg/gnolang/types.go` — `MaxEmbedDepth=8`, `MaxTypeDepth=8`, `MaxStructFields=128`, `MaxInterfaceMethods=128` are new hard limits with no ADR and no reference in docs. These limits could break existing valid Gno programs that happen to use deep embedding (e.g., a library that embeds 5 interfaces each embedding 5 others = 10 levels). The PR comment says "deepest in stdlib + examples + tests is 3" but doesn't cite the measurement. An ADR documenting the threat model, the measured maximums in the current ecosystem, and the chosen headroom is strongly recommended given AGENTS.md requires ADRs for non-trivial AI-assisted PRs.

- [ ] `gno.land/pkg/sdk/vm/keeper.go:744-747` — The `preAlloc` for `AddPackage` is installed *after* `m2` is constructed (lines 715-726) and *after* `defer doRecover(m2, &err)` is set. This means that if `GetParams(ctx)` at line 748 panics before `RunMemPackage`, `doRecover` runs with `preAlloc` still installed on `gnostore`. The `defer gnostore.SetPreprocessAllocator(nil)` at line 747 will still run (correct), but between the panic and that defer, the store carries the preAlloc pointer. This is benign today (GetParams shouldn't panic), but the preAlloc setup should ideally move to just before `RunMemPackage` to minimize the window. Same applies to `Run`.

## Nits

- [ ] `gnovm/pkg/gnolang/bounded_strings.go:87-92` — `newBoundedBuf(n int)` clamps `n < 0` to 0 silently. A negative `lim` argument could indicate a caller bug. `panic("negative lim")` or at least a log would be safer.
- [ ] `gno.land/pkg/sdk/vm/bounded.go:90-96` — The `cmnerrors.Error` branch peeks at `x.Data()` for `FmtError` and then calls `errors.Unwrap(x)` if not. The `errors.Unwrap(x)` call relies on `cmnError` implementing `Unwrap() error`. This is not tested directly; the test `TestBoundedString_CmnError_Wrapped` exercises only the data-is-wrapped-error path. A test where `x.Data()` is neither `FmtError` nor an `error` would exercise the `<error: %T>` fallback.
- [ ] `gnovm/pkg/gnolang/bounded_strings.go:374-387` — `boundedSprintBigDec` calls `bd.String()` unconditionally, then checks if `len(s) > rem*4` and falls back to a marker. The comment says "apd.Decimal doesn't expose a cheap pre-check". This unconditional allocation for big decimals is the only place where transient memory is not bounded before allocation. For a value with 1 million digits this could transiently allocate ~1 MB. Given the rendering is on the recovery path (not hot), this is acceptable, but should document the trade-off explicitly (the existing comment is incomplete).
- [ ] `gnovm/pkg/gnolang/preprocess.go` — Removing `ATTR_LOOPVAR_SKIP` from `nodes.go` (line 143) while the new `initStaticBlocks1` no longer uses it is clean, but a comment noting it was removed as part of the O(N²) → O(N) refactor would help readers searching for `ATTR_LOOPVAR_SKIP` in git history.

## Missing Tests

- [ ] `doRecoverQueryNoMachine` path: No test for the unbounded render issue at `keeper.go:978-981`. A test that simulates a panic with a huge string value in the no-Machine query recovery path would verify the clipLog backstop works and flag the absence of upstream bounding.
- [ ] `boundedUserSprint` success-path bytes rollback: No test verifying the allocator state after a successful call to `BoundedSprintTV` with a non-nil machine and composite value. A test would document the intentional rollback behavior and catch any future change to the restore semantics.
- [ ] `validateEmbedDepth` + `validateTypeDepth` + `validateStructFields` + `validateInterfaceMethods`: No standalone unit tests for these new validators. The integration tests in `gas_test.go` and the txtar file cover the happy path of the preprocess allocator, but there are no tests for type-construction violations (e.g., trying to define a struct with 129 effective fields, or a type chain of depth 9). These validators are called on the critical security path; their error messages and thresholds should be directly exercised.
- [ ] `initStaticBlocks1` refactor: The new O(N) loopvar renaming algorithm is not explicitly tested for the edge cases it must handle: (a) a `TypeDecl` with the same name as a loopvar, (b) a `SwitchStmt.VarName` shadowing a loopvar, (c) a `RangeStmt` key/value shadowing a loopvar. The old code handled these explicitly; the new code also does, but there are no targeted tests confirming the behavior.
- [ ] `PreprocessError.Stack()` bounded output: No test that `*PreprocessError` with a deeply-nested AST (many items in `p.stack`) produces output bounded by the new "first non-zero-span frame" rule, and that calling `x.Error()` returns a bounded string.

## Suggestions

- The two sets of render constants (`maxBoundedBytes=1024` in `bounded.go` and `BoundedRenderBytes=1024` in `bounded_strings.go`) are kept separate because they live in different packages. Consider exposing `BoundedRenderBytes` to `bounded.go` via a package constant or linking them with a comment to make it clear they must stay in sync.
- `boundedUserSprint` modifies `m.Alloc.maxBytes` and `m.Alloc.bytes` directly. This tightly couples the function to `Allocator`'s unexported fields. A cleaner API would be an `Allocator.WithTransientCap(cap int64, fn func()) error` method that handles snapshot/restore internally, making the invariant explicit and reducing risk of the defer-ordering bugs described in the critical section.
- `BoundedExceptionStacktrace(m, lim int)` takes a `*Machine` instead of a `Stacktrace` value. Since it only reads `m.Exception.Stacktrace`, it could take `*Exception` or `Stacktrace` directly, matching the style of `BoundedStacktrace(s Stacktrace, lim int)` and making the data dependency explicit.
- `clipLog` in `tm2/pkg/sdk/helpers.go` has a fast path (`len(s) <= maxLogLineBytes && !strings.ContainsRune(s, '\n')`) that returns the input unchanged for short single-line strings. However, `strings.ContainsRune` is O(N) for the no-newline case. For most short strings this is fine, but a comment explaining why `len(s) <= maxLogLineBytes` alone is insufficient (we need to also check for newlines) would help.
- The `preprocess.go` changes to `evalStaticType` / `staticTypeFromAST` change which Machines are spun up during preprocessing. The `BoundedPanicRender: true` flag is now set on all these machines (see `evalStaticTypeMachine`, `evalStaticTypeOfRaw`, `tryEvalStatic`, `evalConst`). The `NewMachine(pn.PkgPath, store)` call pattern in the old code was replaced with `NewMachineWithOptions(...)`. Confirm that the `applySpecifics` call at the bottom of `types.go` (which still existed) also sets `BoundedPanicRender: true` — it does (line ~2857), good.

## Questions for Author

1. **`boundedUserSprint` bytes rollback on success**: Is the `m.Alloc.bytes = savedBytes` restore intentional even on the non-panicking path? If `tv.Sprint(m)` succeeds and uses 50 KB of allocations, those 50 KB are charged to the gas meter (ConsumeGas was called per-alloc) but then erased from `m.Alloc.bytes`. This means the allocator's GC watermark is silently moved backward. Is this the intended behavior, or should the restore only occur inside the panic branch of the recover? It seems intentional for the recovery-path-only use, but it should be documented explicitly.

2. **`doRecoverQueryNoMachine` bounding**: The query paths at `keeper.go:1444` and `1471` use `doRecoverQueryNoMachine`, which still does `fmt.Errorf("%v", r)` on the panic value. Was the decision to leave this unbounded intentional (since `clipLog` backstops it in `baseapp`), or was it missed? The `%v` render on an arbitrary Gno value could still transiently allocate before `clipLog` sees the result.

3. **ADR for type-depth limits**: `MaxEmbedDepth`, `MaxTypeDepth`, `MaxStructFields`, `MaxInterfaceMethods` are new protocol-level limits. Are there existing Gno programs (in `examples/` or on testnet) that would fail validation with these caps? Was this surveyed? AGENTS.md requires ADRs for non-trivial AI-assisted changes — has one been written or is there a plan?

4. **`ATTR_LOOPVAR_SKIP` removal**: The attribute constant was removed from `nodes.go`. Is there any compatibility concern with serialized ASTs that may have this attribute set (e.g., amino-encoded packages already on chain)? Attributes are not serialized in the current amino schema, correct?

5. **Gas number changes in txtar files**: `gnokey_gasfee.txtar` and `restart_gas.txtar` have updated gas numbers (e.g., `2814892` → `2815758`). The delta is ~866 gas per `addpkg`. Is this the expected cost of the new preprocess alloc-gas charging? A brief note in the PR description or txtar header explaining what drives the change would help future readers distinguish intended gas-schedule changes from regressions.

## Verdict

REQUEST CHANGES — Two issues need addressing before merge: `doRecoverQueryNoMachine` still allows unbounded rendering on query paths (a partial fix for the stated security goal), and `boundedUserSprint`'s unconditional bytes rollback on success silently discards allocation tracking in a way that could cause allocator GC misfires if the code is reused beyond the current call sites.
