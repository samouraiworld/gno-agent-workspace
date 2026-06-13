# PR #5681: feat(gnovm): capture panic origin at construction via NewException

URL: https://github.com/gnolang/gno/pull/5681
Author: ltzmaxwell | Base: master | Files: 16 | +801 -122
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `bdf44ca8f` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5681 bdf44ca8f`

Round 2 (round 1 reviewed `ec473b6a`, never posted). Since round 1 the author corrected the `Exception` docstring and the `BoundedPanicRender` comments, added the unit coverage round 1 asked for (`NewException` GoStack, `markAbort`/`Error`, filetest helpers), added the per-chain GoStack cap, and a nil-guard on `Error()`. Still open: the GoStack-gap raise sites round 1 flagged.

**TL;DR:** When a gno program panics and nothing recovers it, the VM aborts the transaction. This PR makes each panic also record the Go-internal call chain of the interpreter at the moment it was raised, so a failing filetest can print where inside the VM the panic actually came from, and replaces the old `UnhandledPanicError` wrapper with a flag on the existing `Exception` object.

**Verdict: REQUEST CHANGES** — the design is sound and well-tested, but two things should change before merge: the new `GoStack` is captured on every VM panic including the validator path, where nothing ever reads it (~6.7µs and 7.7KB/13 allocs per panic of unmetered work); and the capture is inconsistent: slice-index and integer-division panics still bypass `NewException`, so the feature's own `go stack:` output is blank for exactly those common cases.

## Summary
Captures the interpreter's Go call chain at panic origin: `NewException(value)` stamps `Exception.GoStack` via `runtime.Callers` at construction, and most `panic(&Exception{...})` sites in `values.go`/`alloc.go` now route through it. `UnhandledPanicError` is gone; the terminal unhandled-panic state is now `Exception{Abort:true}` with a pre-rendered `Descriptor`, set in place by `markAbort`. `Run` detects `Abort` and re-panics the `*Exception` straight to the outer recoverer instead of re-entering the cooperative `pushPanic` path. Filetest failures gain a `go stack:` section (project-relative paths), rename `stacktrace:` to `gno stack:`, and suppress an empty `output:` block. A per-chain cap (`MaxGoStackCaptures = 8`) bounds retained GoStack strings against adversarial deep defer-repanic chains.

## Glossary
- **Abort exception** — an unhandled panic whose defers are exhausted; `markAbort` flips `Exception.Abort=true` and `Run` surfaces it directly.
- **GoStack** — the interpreter's Go-side call chain at the raise site, captured by `runtime.Callers`. Distinct from the gno-level stacktrace.
- **BoundedPanicRender** — per-Machine flag, true on validators, that caps the cost of rendering panic *values* into the Descriptor.

## CI
Both red checks are merge-behind artifacts, not code problems:
- `gno-checks / lint` — fails on `r/test/sealviolation` (`foreignImpl ... missing method isSealed`). That realm and `p/test/seal` do not exist on this branch; the seal machinery (#5706) landed on master after this branch's merge-base (`7bea4699`).
- `main / test` — fails on `params_valset_rotation_throttle`. This branch carries the old height-counting version of that txtar; master has since rewritten it to use `gnoland wait-for-new-block`. The old test fails identically on the merge-base with zero exception changes (verified: 3/3 FAIL at `7bea4699`, 3/3 PASS at `origin/master`).

Merging master clears both.

## Warnings (should fix)

- **[validator pays for a stack only the test harness reads]** [`gnovm/pkg/gnolang/machine.go:2812`](https://github.com/gnolang/gno/blob/bdf44ca8f/gnovm/pkg/gnolang/machine.go#L2812) · [↗](../../../../../.worktrees/gno-review-5681/gnovm/pkg/gnolang/machine.go#L2812), [`gnovm/pkg/gnolang/frame.go:296-298`](https://github.com/gnolang/gno/blob/bdf44ca8f/gnovm/pkg/gnolang/frame.go#L296-L298) · [↗](../../../../../.worktrees/gno-review-5681/gnovm/pkg/gnolang/frame.go#L296-L298) — GoStack is captured on every VM panic, including the validator path, but the only non-test reader is the filetest harness.
  <details><summary>details</summary>

  `pushPanic` and `NewException` call `captureExceptionStack` unconditionally — no `BoundedPanicRender` gate. On a validator, the keeper renders the Descriptor and the gno stacktrace only ([`gno.land/pkg/sdk/vm/keeper.go:960-962`](https://github.com/gnolang/gno/blob/bdf44ca8f/gno.land/pkg/sdk/vm/keeper.go#L960-L962) · [↗](../../../../../.worktrees/gno-review-5681/gno.land/pkg/sdk/vm/keeper.go#L960-L962)); it never touches `GoStack`. A grep confirms the sole non-test reader of `.GoStack` is [`gnovm/pkg/test/filetest.go:375`](https://github.com/gnolang/gno/blob/bdf44ca8f/gnovm/pkg/test/filetest.go#L375) · [↗](../../../../../.worktrees/gno-review-5681/gnovm/pkg/test/filetest.go#L375). So every nil-deref, index-out-of-range, slice-bounds, map-key, and user `panic()` on chain runs a `runtime.Callers` walk plus per-frame string building whose result production discards. Measured cost vs the pre-PR bare `&Exception{}`: ~6.7µs and 7702 B / 13 allocs per panic. It compounds an existing per-panic `m.Stacktrace()` allocation rather than introducing an unbounded vector, and `MaxGoStackCaptures` bounds *retention* per chain, not *capture* across many recovered panics (the counter resets on `Recover`), so a tx that recovers panics in a loop pays this per panic. `BoundedPanicRender` exists precisely to bound validator-side panic cost; this new work sidesteps it. Fix: gate the capture to where GoStack is consumed (mirror `BoundedPanicRender`, or a debug/test flag), so validators don't pay for data they never render.
  </details>

- **[the new go-origin output is blank for slice-index and divide-by-zero panics]** [`gnovm/pkg/gnolang/values.go:384-395`](https://github.com/gnolang/gno/blob/bdf44ca8f/gnovm/pkg/gnolang/values.go#L384-L395) · [↗](../../../../../.worktrees/gno-review-5681/gnovm/pkg/gnolang/values.go#L384-L395), [`gnovm/pkg/gnolang/op_binary.go:935-937`](https://github.com/gnolang/gno/blob/bdf44ca8f/gnovm/pkg/gnolang/op_binary.go#L935-L937) · [↗](../../../../../.worktrees/gno-review-5681/gnovm/pkg/gnolang/op_binary.go#L935-L937), [`gnovm/pkg/gnolang/op_binary.go:1034-1036`](https://github.com/gnolang/gno/blob/bdf44ca8f/gnovm/pkg/gnolang/op_binary.go#L1034-L1036) · [↗](../../../../../.worktrees/gno-review-5681/gnovm/pkg/gnolang/op_binary.go#L1034-L1036) — three common runtime-panic sites still build `&Exception{}` directly, so their `GoStack` is empty and the feature's `go stack:` section falls back to the raw dump for exactly these cases.
  <details><summary>details</summary>

  `SliceValue.GetPointerAtIndexInt2` (slice index out of bounds) and `quoAssign`/`remAssign` (integer division by zero) bypass `NewException`. The slice site can panic directly; `quoAssign`/`remAssign` build the `*Exception` and return it for the caller to panic with, so a plain `NewException` would record the wrong frame — they need a skip-aware constructor (`NewExceptionSkip(value, skip)`) to capture the real raise site. As-is, an unhandled slice-OOB or div-by-zero produces an empty per-link GoStack, `renderGoVMChain` skips it, and `goOriginOrStack` falls back to the raw `debug.Stack` dump — the pre-PR behavior, not the new VM-origin output. Confirmed by the code path: these are the only `*Exception` raise sites in `gnovm/pkg/gnolang/` that don't carry a GoStack. Fix: route all three through a (skip-aware) `NewException`, or, paired with the gating above, capture consistently on the debug path only.
  </details>

## Nits
- [`gnovm/pkg/test/filetest.go:395`](https://github.com/gnolang/gno/blob/bdf44ca8f/gnovm/pkg/test/filetest.go#L395) · [↗](../../../../../.worktrees/gno-review-5681/gnovm/pkg/test/filetest.go#L395) — `label := "panic"` is dead; both branches of the following `if i == 0` reassign it before use. Drop the initializer (`var label string`).

## Missing Tests
- None blocking. Round 1's gaps are now covered ([`frame_test.go`](https://github.com/gnolang/gno/blob/bdf44ca8f/gnovm/pkg/gnolang/frame_test.go#L1) · [↗](../../../../../.worktrees/gno-review-5681/gnovm/pkg/gnolang/frame_test.go#L1), [`exception_chain_test.go`](https://github.com/gnolang/gno/blob/bdf44ca8f/gnovm/pkg/gnolang/exception_chain_test.go#L1) · [↗](../../../../../.worktrees/gno-review-5681/gnovm/pkg/gnolang/exception_chain_test.go#L1), [`filetest_helpers_test.go`](https://github.com/gnolang/gno/blob/bdf44ca8f/gnovm/pkg/test/filetest_helpers_test.go#L1) · [↗](../../../../../.worktrees/gno-review-5681/gnovm/pkg/test/filetest_helpers_test.go#L1)). The GoStack-gap sites above lack any coverage, but that is folded into the Warning rather than tracked separately.

## Open questions
- Determinism: `GoStack` carries VM source paths and Go line numbers, which differ across builds. It is currently kept out of every consensus-visible output (keeper uses Descriptor + gno stacktrace; `Error()` returns Descriptor/Value). Safe today; a latent footgun if anyone later plumbs GoStack into an on-chain error string. Not posted: no issue in this PR.
- `markAbort` flips `Abort=true` on `m.Exception` in place and relies on `Machine.Release()` zeroing the field for pooled reuse. Safe today. Not posted: no concrete defect.
