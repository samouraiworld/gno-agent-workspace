# PR #5670: feat(gnovm): capture VM-internal panic origin; deprecate UnhandledPanicError

**URL:** https://github.com/gnolang/gno/pull/5670
**Author:** ltzmaxwell | **Base:** master | **Files:** 9 | **+243 -58**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4-7

## Summary

This PR captures the VM-internal Go call chain at panic raise time so unhandled-panic reports point at the actual VM site (e.g. `op_exec.go:165` for `range` over a nil pointer, `values.go:1986` for a slice-bounds helper panic) rather than only the harness frames produced by `debug.Stack()`. The captured trace is stamped onto a new `Exception.GoStack` field; the existing `UnhandledPanicError` value type is folded into `*Exception` with `Abort=true` and a pre-rendered `Descriptor`, so `Run()` can short-circuit terminal panics directly to the outer recoverer instead of round-tripping through `runOnce.func1`.

It is the author's competing alternative to [#5681](https://github.com/gnolang/gno/pull/5681) for the same observability goal. The design difference is **where** the Go stack is captured. #5681 captures at *construction* time inside a new `NewException()` constructor, and therefore has to instrument every direct `panic(&Exception{...})` raise site (`values.go`, `alloc.go`, `op_binary.go`) — and missed several. #5670 captures at *recovery* time inside `runOnce.func1`, gated on a per-Machine `goStackCaptured` flag (reset by `Recover()`). Because `runtime.Callers` inside a deferred recover preserves the panicking call chain (verified with a standalone Go test against go1.25.9 — defers run during unwind while the panic state is still intact), this single capture point sees every `*Exception` panic regardless of where it was raised, and naturally pays at most O(1) per panic chain even under #5439's many-defer scenarios. No raise-site instrumentation needed.

Concrete changes:
- **`Machine.Run` short-circuits Abort'd exceptions** (`machine.go:1542-1544`): catches `*Exception` from `runOnce`, re-panics if `caught.Abort`, else replays cooperatively via `pushPanicException`.
- **`runOnce.func1` is the single capture point** (`machine.go:1554-1559`): on any caught `*Exception` with no GoStack, calls `attachGoStack`. The flag in `attachGoStack` makes the second-and-later raises in a chain free.
- **`captureGoStack` walker** (`machine.go:2672-2701`): walks `runtime.CallersFrames(pcs[:n])` with `runtime.Callers(2, ...)`, filters via `goStackIgnore` (drops `runtime.*` and our own `attachGoStack` / `captureGoStack` / `pushPanic` / `pushPanicException` / `(*Machine).Panic` / `(*Machine).PanicString` / `runOnce.func1`), stops at the first `(*Machine).Run` suffix. Trims paths via `TrimOriginFile`.
- **`TrimOriginFile`** (`machine.go:2722-2741`): strips absolute prefixes via `/gnovm/` and `/src/` markers (latter catches Go stdlib), falls back to basename.
- **`*Exception` gains `Abort`, `Descriptor`, `GoStack` fields and an `Error() string` method** (`frame.go:263-295`), with the `//nolint:errname` carve-out documented.
- **`UnhandledPanicError` removed**; `makeUnhandledPanicError` renamed to `markAbort` (`op_call.go:570-588`), which **mutates `m.Exception` in place** to set `Abort=true` and populate `Descriptor`.
- **`pushPanic` / `Panic` / `PanicString` instrument GoStack themselves** (`machine.go:2749-2769`) so the first-raise capture happens with the cleanest possible skip count. (The runOnce fallback catches the rest.)
- **Recoverers updated** for the new shape: `gno.land/pkg/sdk/vm/{bounded,keeper}.go`, `gnovm/cmd/gno/run.go`, `gnovm/pkg/test/{filetest,imports}.go`. The `boundedString` arm for `*Exception` prefers the cached `Descriptor` when `Abort=true`; `doRecoverInternal` matches `*Exception` and checks `.Abort`.
- **Filetest output polish** (`filetest.go:343-399`): `stacktrace:`→`gno stack:`, raw `stack:` replaced by spliced `go stack:` (VM chain via `GoStack` joined with `debug.Stack()` truncated past the `(*Machine).Run` frame so harness frames don't duplicate), empty `output:` block suppressed, all dump paths trimmed via `TrimOriginFile`.

## Test Results

- **Existing tests:**
  - `go test ./gno.land/pkg/sdk/vm/` — PASS (18.0s, covers the rewritten `bounded.go` arm and updated `doRecoverInternal`).
  - `go test ./gno.land/pkg/sdk/vm/ -run Gas` — PASS (3.0s).
  - `go test ./gnovm/pkg/test/ -short` — PASS.
  - `go test ./gnovm/pkg/gnolang/ -run Files -test.short` — **5 pre-existing failures** in `types/add_f0.gno`, `types/and_f0.gno`, `types/eql_0b4.gno`, `types/eql_0f0.gno`, `types/or_f0.gno`. All are `TypeCheckError` directive mismatches against `go/types` output (expected `interface{Error() string}`, actual `error` plus arithmetic-operator wording). The PR does not touch these files, the type checker, or `*Exception.Error()` in a way that should affect typecheck-directive matching; these track a recent `go/types` behavior change, not this PR. Unrelated.
  - CI on the PR head is fully green per `gh pr checks 5670` (CodeQL, analyze, build docker images, e2e, generate, docs, codecov/patch). Codecov patch coverage 79.46% with the misses concentrated in `filetest.go`'s `harnessAfterRun`/`goOriginOrStack` paths (only exercised on filetest *failure*).
- **Edge-case tests:** skipped (review scope).
- **`runtime.Callers` semantics check:** independently verified against go1.25.9 (`/tmp/test_callers.go`) that calling `runtime.Callers(2, pcs[:])` from a deferred recover **does** capture the original panic origin frames (`runtime.gopanic` → `panicking_func` → `main`). This corroborates the PR's design and the smoke-test claim that `s[10]` panics from `values.go:1986` show `GetPointerAtIndex` → `doOpIndex1` → `runOnce` → `Run` in `GoStack`. The non-instrumented raise sites (`values.go`, `alloc.go`, `op_binary.go:935/1034`) are correctly handled by the runOnce fallback, in contrast to #5681's same-line instrumentation that misses them.

## Critical (must fix)

- None.

## Warnings (should fix)

- [ ] `gnovm/pkg/gnolang/machine.go:57, 94` — comments on `Machine.BoundedPanicRender` (field) and `MachineOptions.BoundedPanicRender` still say "BoundedPanicRender gates `makeUnhandledPanicError` to use the bounded printer". Function is now `markAbort`; update both. Same stale name was flagged in my review of #5681.
- [ ] `gnovm/adr/op_handler_gas_audit.md:127` — ADR gas-audit table for `doOpPanic2` still references `makeUnhandledPanicError`. Rename to `markAbort`. The cost shape is unchanged: still `O(numExceptions × Sprint cost)` when `BoundedPanicRender=false`, plus on the recovery path now one `runtime.Callers` walk per panic chain (gated by `goStackCaptured`), which is O(stack depth) once per chain.
- [ ] `gnovm/pkg/gnolang/machine.go:2746` — `Panic` docstring still says "Some code in realm.go and values.go will `panic(&Exception{...})` directly. Keep this code in sync with those calls." The "keep in sync" caveat is now obsolete — the runOnce fallback covers direct constructions automatically, which is one of the PR's main wins. Rewrite to reflect the new invariant ("direct `panic(&Exception{...})` sites in values.go/alloc.go work without instrumentation because runOnce.func1 attaches GoStack on first recover").
- [ ] **No ADR included.** AGENTS.md: *"Every non-trivial AI-assisted PR must include an ADR."* This PR materially changes the `*Exception` contract (now satisfies `error`, gains three fields, gains in-place mutation by `markAbort`, removes `UnhandledPanicError`), introduces a new VM mechanism (`runOnce`'s recover→`attachGoStack`→`goStackCaptured` gate), and competes with #5681 on design. Add `gnovm/adr/pr5670_capture_vm_panic_origin.md` with context (observability goal, relationship to #5439's iterative recovery), the recovery-time vs construction-time tradeoff vs #5681 (no raise-site instrumentation needed, single capture point, naturally O(1) per chain), alternatives considered (instrument every raise site → see #5681), and consequences (per-panic `runtime.Callers` cost on the cooperative path, GoStack absent from production-keeper error output by default).

## Nits

- [ ] `gnovm/pkg/gnolang/machine.go:44-48` — the `goStackCaptured` comment references "machine.go ~2132" for `PushFrameCall` clearing `m.Exception`. Line-number references in source comments rot quickly; either drop the line ref ("see `PushFrameCall`'s `cfr.LastException = m.Exception; m.Exception = nil`") or name the field/operation that breaks the obvious "gate via `m.Exception != nil`" approach.
- [ ] `gnovm/pkg/gnolang/machine.go:2710-2718` — `goStackIgnore` lists `pushPanic`, `pushPanicException`, `(*Machine).Panic`, `(*Machine).PanicString` — these are correct for the *first-raise* path where Panic/pushPanic call `attachGoStack` themselves. But on the *recovery* path through `runOnce.func1`, those helpers will already have returned (the panic has unwound past them), so listing them is harmless overlap — worth a brief comment that the list intentionally covers both capture entry points.
- [ ] `gnovm/pkg/gnolang/machine.go:2738-2741` — `projectFileMarkers` is a package-level `var` listing two slashes. Could be a `const` array or moved next to `TrimOriginFile` for locality. Minor.
- [ ] `gnovm/pkg/gnolang/machine.go:2692` — `strings.HasSuffix(f.Function, ".(*Machine).Run")` is loose: any future helper that ends with that suffix in some other package would also terminate the walk. Anchoring on the full path (`github.com/gnolang/gno/gnovm/pkg/gnolang.(*Machine).Run`) is more defensive; current code is fine in practice. Same nit applies to the other `HasSuffix`-based filters in `goStackIgnore`.
- [ ] `gnovm/pkg/test/filetest.go:360-378` — `harnessAfterRun` does three sequential `strings.Index` walks with offset arithmetic; correct but brittle. A `strings.SplitN(s, "\n", N)` or regexp version would be clearer. Same nit appeared in my #5681 review.
- [ ] `gno.land/pkg/sdk/vm/keeper.go:951` — comment "Common unhandled panic error, skip machine state." is cryptic. What's actually happening: the recoverer can rely on `ex.Descriptor` being pre-rendered (Abort=true means `markAbort` populated it) and on `ex.Stacktrace` being set, so the recoverer doesn't need to call `m.Stacktrace()` for the gno trace — but the very next line calls `gno.BoundedExceptionStacktrace(m, ...)` which *does* still walk machine state. Either the comment is wrong, or the code is doing redundant work; please clarify.

## Missing Tests

- [ ] No unit test verifies `captureGoStack` populates with the expected frame chain — a `runtime.Caller`-style assertion that the caller's function name appears and that `(*Machine).Run` terminates the walk.
- [ ] No test for `markAbort` populating `Descriptor` correctly under both `BoundedPanicRender=true` and `false`; only the consumer-side `boundedString` arm is covered (`bounded_test.go`).
- [ ] No filetest exercising the new `go stack:` directive — the formatter is only hit on filetest *failure*, so unless tests fail in CI, the new output goes unobserved. Codecov-patch coverage confirms: `harnessAfterRun`, `goOriginOrStack`, `trimStackPaths` are uncovered. Consider a meta-test in `gnovm/pkg/test/` that drives `goOriginOrStack` directly with a synthetic `runResult`.
- [ ] No test for the `goStackCaptured` gate behavior under #5439's many-defer-panic scenario — the PR claims O(1) cost per chain but doesn't demonstrate it.
- [ ] No test for `trimStackPaths` against an actual `debug.Stack()` dump — the parsing assumes the `"\t/abs/path:LINE +0xOFF"` shape, which holds today but is undocumented Go runtime output.

## Suggestions

- The `GoStack` field is captured but **never reaches the production-keeper error output**: `keeper.go:949-960` calls `boundedString(ex, 0)` which returns `ex.Descriptor` (no GoStack), then wraps with `BoundedExceptionStacktrace(m, ...)` for the gno trace. The captured chain only surfaces in `gnovm/pkg/test/filetest.go`'s failure path. If incident triage from production logs is part of the observability goal, plumb `GoStack` into the keeper's wrapped error too (with a bounded length, perhaps `gno.BoundedRenderBytes/2`).
- `markAbort` mutates `m.Exception` in place, flipping `Abort=true` permanently on that object. Safe today because `Machine.Release()` zeros `m.Exception` (via `*m = Machine{...}` at `machine.go:258`), but a one-line comment at `op_call.go:573` noting "we rely on Release() to reset" would protect against future regressions in the pool-recycle path. Same suggestion applies to #5681.
- The recovery-time capture is elegant, but `goStackCaptured` resets only on `Recover()` — not on `Run` re-entry, not on `Machine.Release`. If a Machine is re-used across transactions without going through `Recover()` (e.g. an aborted transaction recycled in the pool), the next panic in the new transaction would skip the capture. `Release()` zeros the field via `*m = Machine{...}`, so pool-recycling is fine — but please add a sentence to the field's doc comment confirming the reset semantics so a reader doesn't need to re-derive it.
- The `captureGoStack` skip parameter is hardcoded to `2` (assumes the caller is `attachGoStack` → callsite). Add a sentence explaining how to compute `skip` for future raise-site additions (or accept a `skip int` parameter so callers can use it directly without an extra wrapper).
- `goStackIgnore`'s suffix list is duplicate-leaning: `runOnce.func1`, `attachGoStack`, `captureGoStack` are all part of the same wrapping chain. Consider grouping by purpose in a single comment block and asserting at init time that all listed names are exact functions in this package (so a rename doesn't silently break the filter).
- Per `simplify` reflex: there's room to deduplicate the four-way comment + code split across `Panic`, `pushPanic`, `attachGoStack`, `runOnce.func1`. Today each path independently calls `attachGoStack`. A cleaner factoring: only the runOnce path attaches, since it's the universal recover point. The Panic/pushPanic eager calls are an optimization for one less stack frame in the trace — worth keeping only if measured. (The #5681 author's per-site capture is the inverse extreme; #5670 is closer to the right answer but still does it twice.)

## Questions for Author

- Did you benchmark the recovery-time vs construction-time capture cost vs #5681 on the #5439 many-defer-panic scenario? The PR design is structurally cheaper (single capture point + flag) but a number would help reviewers choose between the two competing PRs.
- Will the captured `GoStack` ever be surfaced through the keeper's error path for production incident triage, or is observability deliberately scoped to filetest debug output only? If the latter, the `GoStack` field's value proposition is much smaller and worth saying so explicitly in the field doc.
- Why eagerly attach in `Panic`/`pushPanic` when `runOnce.func1` would catch them anyway? Saving 1-2 frames in the trace, or some other reason?
- Is the new `*Exception.Error()` fallback (returning `Value.String()` when `!Abort`) intentional? Legacy `UnhandledPanicError.Error()` returned the descriptor only; the new fallback changes behavior for callers that treat `*Exception` as an `error` outside the Abort path (and `imports.go:356` now returns `err = v` for non-Abort exceptions). Please confirm.
- Why does `keeper.go:951`'s comment say "skip machine state" when the next line calls `BoundedExceptionStacktrace(m, ...)` which walks machine state?

## Verdict

**APPROVE WITH NITS** — design is sound and substantively better than #5681 for this goal: capturing in `runOnce.func1` exploits Go's defer/recover semantics so no raise-site instrumentation is needed (a class of bug #5681 hits at three sites), the `goStackCaptured` flag gives O(1) per chain naturally, and CI is green with broad existing-test coverage. The blocking issues are doc hygiene (two stale `makeUnhandledPanicError` comments, one ADR reference, the obsolete `Panic` "keep in sync" caveat) and the missing ADR for an AI-generated, design-level PR that's competing with another open PR. None of these are correctness; the cleanup is small. With the doc fixes and an ADR (which can frame the design choice vs #5681), this is ready.
