# PR #5681: feat(gnovm): capture panic origin at construction via NewException

**URL:** https://github.com/gnolang/gno/pull/5681
**Author:** ltzmaxwell | **Base:** master | **Files:** 11 | **+244 -108**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4-7

## Summary

This PR is the author's proposed alternative to [#5670](https://github.com/gnolang/gno/pull/5670) for the same observability goal: surfacing the VM-internal Go call chain at the site where a gno panic is raised, so unhandled-panic reports point at (e.g.) `op_exec.go:165` rather than only the recovery-harness frames.

Design philosophy: capture the Go stack **at construction time** via a new `NewException(value TypedValue) *Exception` constructor that wraps `runtime.Callers` and stamps the result onto a new `*Exception.GoStack` field. This mirrors the Java/Python/Rust pattern of errors carrying their own stack from birth. PR #5670 instead rebuilds the chain after the fact inside `runOnce.func1` and gates on a per-Machine `goStackCaptured` flag so each chain only pays once.

Concrete changes:
- **New constructor & helpers** (`gnovm/pkg/gnolang/frame.go`): `NewException`, `captureExceptionStack` (walks `runtime.CallersFrames`, skips `runtime.*` frames, stops at `(*Machine).Run`), and `TrimOriginFile` (strips path prefixes up to `/gnovm/` or `/src/`).
- **`*Exception` now satisfies `error`**: `Error()` returns the cached `Descriptor` when `Abort` is set, else `Value.String()`. New fields: `Abort bool`, `Descriptor string`, `GoStack string`.
- **`UnhandledPanicError` removed**. Terminal-panic state is now expressed as `*Exception{Abort: true, Descriptor: ...}`. The former `makeUnhandledPanicError` is replaced by `markAbort()`, which **mutates `m.Exception` in place** to set `Abort=true` and populate `Descriptor`.
- **`Machine.Run` short-circuits Abort'd exceptions**: catches `*Exception` from `runOnce`, and if `caught.Abort` re-panics it directly to the outer recoverer; else routes through the cooperative `pushPanicException` path.
- **`pushPanic` now captures `GoStack` on every call** via `captureExceptionStack(1)`.
- **Recoverers updated** for the new shape: `gno.land/pkg/sdk/vm/{bounded,keeper}.go`, `gnovm/cmd/gno/run.go`, `gnovm/pkg/test/{filetest,imports}.go`. The `boundedString` arm for `*Exception` prefers the cached `Descriptor` when `Abort=true`.
- **Filetest output polish**: `stacktrace:`→`gno stack:`, `stack:`→`go stack:`, empty `output:` block suppressed, absolute paths trimmed via `TrimOriginFile`. A new `runResult.GoVMStack` carries the constructor-captured chain; the legacy `debug.Stack()` dump is now used only as a fallback for true Go runtime bugs that bypass `NewException`. `harnessAfterRun` slices the raw `debug.Stack` output past the `(*Machine).Run(` frame to avoid duplicating the harness chain.

## Test Results

- **Existing tests:**
  - `go test ./gno.land/pkg/sdk/vm/` — PASS (13.3s, covers the rewritten `bounded.go` arm and the updated `doRecoverInternal`).
  - `go test ./gno.land/pkg/sdk/vm/ -run Gas` — PASS (2.6s).
  - `go test ./gnovm/pkg/test/ -short` — PASS.
  - `go test ./gnovm/pkg/gnolang/ -run Files -test.short` — **2 pre-existing failures** in `types/eql_0f0.gno` and `types/or_f0.gno`. Both are `TypeCheckError` directive mismatches against go/types output (expected `interface{Error() string}`, actual `error`). The PR does not touch these files, the type checker, or the `error` interface in a way that should affect them; `git log` on those files shows no churn. Unrelated to this PR.
- **Edge-case tests:** skipped (review scope).

## Critical (must fix)

- None.

## Warnings (should fix)

- [ ] `gnovm/pkg/gnolang/frame.go:259-266` — `Exception` docstring is **incorrect and self-contradictory**. It claims "No code path panics with `*Exception` — recovery infrastructure is unused for VM signals, retaining its purpose only for true Go runtime bugs", yet this same PR adds 20+ `panic(NewException(...))` sites in `values.go`/`alloc.go`, `m.Panic` panics with `*Exception` (`machine.go:2676`), `markAbort`'s callers panic with `*Exception` (`op_call.go:389,599`), and `runOnce.func1` exists precisely to recover those panics (`machine.go:1550-1558`). Rewrite the doc to describe what actually happens: most VM-internal raise sites go-panic with an `*Exception` carrying `GoStack`; `runOnce` recovers it; `Run` either re-panics (Abort) or replays cooperatively (`pushPanicException`).
- [ ] `gnovm/pkg/gnolang/values.go:378-391` — `SliceValue.GetPointerAtIndexInt2` still does `panic(&Exception{Value: ...})` directly. These raise sites will have **empty `GoStack`**, defeating the PR's stated goal at slice-bounds checks (a common runtime panic). Route through `NewException`.
- [ ] `gnovm/pkg/gnolang/op_binary.go:935-937, 1034-1036` — `quoAssign` and `remAssign` construct `&Exception{Value: typedString("runtime error: division by zero")}` and return it to the caller, which then panics with it. Same `GoStack` gap as above. Either build via `NewException` (and live with the slightly off skip frame) or expose a `NewExceptionSkip(value, skip)` for the return-then-panic pattern.
- [ ] `gnovm/pkg/gnolang/machine.go:52, 89` — comments on `Machine.BoundedPanicRender` and `MachineOptions.BoundedPanicRender` still reference `makeUnhandledPanicError`. Function is now `markAbort`; update both.
- [ ] `gnovm/adr/op_handler_gas_audit.md:127` — ADR gas-audit row references `makeUnhandledPanicError` in the `doOpPanic2` cost analysis. Rename to `markAbort` (the cost shape is unchanged: `O(numExceptions × Sprint cost)` still, plus one `runtime.Callers` per panic on the cooperative path now).
- [ ] **No ADR included.** `AGENTS.md` requires "every non-trivial AI-assisted PR must include an ADR" and the commit was AI-generated. This PR materially changes the `*Exception` contract (now satisfies `error`, gains three fields, gains in-place mutation by `markAbort`) and proposes a design alternative to #5670 worth recording. Add `gnovm/adr/pr5681_capture_panic_origin.md` with: context (observability goal, relationship to #5439's iterative recovery), the construction-time vs reconstruction-time tradeoff vs #5670, alternatives considered, consequences (per-panic capture cost, GoStack absent from keeper-path output).

## Nits

- [ ] `gnovm/pkg/gnolang/frame.go:289-293` — `NewException` and `captureExceptionStack` are exported; only `NewException` needs to be (it's called from the same package). `captureExceptionStack` is also called from `machine.go:2687` (same package), so it can stay unexported — it already is. `TrimOriginFile` is exported because `gnovm/pkg/test/filetest.go:679` references `gno.TrimOriginFile`. Justified.
- [ ] `gnovm/pkg/gnolang/frame.go:312` — `strings.HasSuffix(f.Function, ".(*Machine).Run")` works but `(*Machine).Run` is also a common substring in stack lines; an exact match against the full qualified function name (e.g. `github.com/gnolang/gno/gnovm/pkg/gnolang.(*Machine).Run`) would be more defensive. Probably overengineering — current code is fine in practice.
- [ ] `gnovm/pkg/test/filetest.go:649-666` — `harnessAfterRun` walks string offsets twice (`strings.Index` thrice) and uses index arithmetic that's hard to follow. A regexp or `strings.SplitN(s, "\n", N)` based version would be clearer; the current code is correct but brittle.
- [ ] `gnovm/pkg/gnolang/frame.go:286-288` — `NewException` doc says "Helpers without *Machine access (values.go, alloc.go) use this in place of `panic(&Exception{...})`" — accurate, but `m.Panic` (which *does* have `*Machine`) also uses it now (`machine.go:2673`). Tighten the description.

## Missing Tests

- [ ] No unit test verifies `NewException` actually populates `GoStack` with the expected frame chain (a `runtime.Caller`-style check that the constructor's caller appears in the output and that `(*Machine).Run` terminates it).
- [ ] No test for `markAbort` populating `Descriptor` correctly under both `BoundedPanicRender=true` and `false` — only the consumer-side `boundedString` arm is covered (`gno.land/pkg/sdk/vm/bounded_test.go`).
- [ ] No filetest exercising the new `go stack:` directive — the formatter is only hit on filetest failure, not on the directive-match path, so unless tests fail in CI, the new output goes unobserved. Consider a meta-test in `gnovm/pkg/test/` that drives `goOriginOrStack` directly with a synthetic `runResult`.
- [ ] No benchmark comparing this PR's per-panic capture cost vs #5670's per-chain capture for #5439's many-panicking-defers scenario. The PR description claims "Same end-user output; the difference is design philosophy" — the cost difference (N captures vs 1) is also a tradeoff worth quantifying before choosing between the two PRs.

## Suggestions

- The `GoStack` field is captured at construction but **never reaches the production-keeper error output**: `keeper.go:949-955` calls `boundedString(ex, 0)` which returns `ex.Descriptor` (no GoStack), then wraps with `BoundedExceptionStacktrace(m, ...)` for the gno trace. The captured chain only surfaces in `gnovm/pkg/test/filetest.go`'s failure path. If incident triage from production logs is part of the observability goal, plumb `GoStack` into the keeper's wrapped error too (with bounded length).
- `markAbort` mutates `m.Exception` in place, flipping `Abort=true` permanently on that object. Safe today because `Machine.Release()` zeros `m.Exception` (`machine.go:253`), but a one-line comment at `op_call.go:573` noting "we rely on Release() to reset" would protect against future regressions in the pool-recycle path.
- Consider naming `pushPanicException` something less collision-prone with `pushPanic` — e.g. `pushExistingPanic` or `repushPanic`. Currently the two function names differ only by suffix; readers may not notice which entry point a call site uses.
- The `captureExceptionStack` skip parameter differs by raise site (`NewException` passes `2`, `pushPanic` passes `1`). The values are correct but unobvious; a sentence in the doc comment explaining how to compute `skip` would help future raise-site additions.

## Questions for Author

- Why were `values.go:378-391`, `op_binary.go:935-937`, and `op_binary.go:1034-1036` left as direct `&Exception{}` constructions? Oversight, or are these intentionally exempt? If the latter, please document the carve-out.
- Did you benchmark per-panic capture cost vs #5670's gated approach on the #5439 many-defer-panic scenario? The PR description frames the tradeoff as purely "design philosophy" — cost data would help reviewers choose.
- Will the captured `GoStack` ever be surfaced through the keeper's error path for production incident triage, or is observability deliberately scoped to filetest debug output only?
- Is the new `*Exception.Error()` fallback (returning `Value.String()` when `!Abort`) intentional? Legacy `UnhandledPanicError.Error()` returned the descriptor only; the new fallback changes behavior for callers that treat `*Exception` as an `error` outside the Abort path.

## Verdict

**REQUEST CHANGES** — design is sound and tests on touched code pass, but the PR doesn't finish what it sets out to do: three raise sites (`values.go:378-391`, `op_binary.go:935`, `op_binary.go:1034`) bypass `NewException` and lose `GoStack`, the `Exception` docstring is actively wrong about how recovery works, two existing comments and one ADR reference the renamed function, and the required ADR is missing. The cooperative design is fine — once the missed sites are routed through `NewException`, the docs corrected, and an ADR added (also addressing the per-panic capture cost vs #5670), this is approvable.
