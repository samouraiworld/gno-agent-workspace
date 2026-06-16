# PR #5829: fix(gnovm): reject oversized fixed-size array length at preprocess time

URL: https://github.com/gnolang/gno/pull/5829
Author: ltzmaxwell | Base: master | Files: 4 | +66 -0
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `de74ab7` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5829 de74ab7`

**TL;DR:** A fixed-size array whose declared length is huge enough to overflow the allocator's byte accounting (`[9e18]int`) used to slip past the type checker and only blow up when the array was actually allocated. This PR catches it at compile time instead, the way Go does, by checking the length against the same overflow threshold the runtime allocator uses.

**Verdict: APPROVE** — the check is correct and its threshold exactly mirrors the runtime allocator (`AllocateListArray`/`AllocateDataArray`) for both the per-element and byte-array paths; the only gap is a pre-existing, recoverable edge in the `[...]T{idx: v}` form (`{MaxInt64: 1}`) that the new check is positioned too late to catch. Non-blocking, but it lives in the exact code this PR edits, so worth folding in.

## Summary

`go/types` accepts an oversized fixed-size array because the "larger than address space" rule is a gc-backend check, not a type-system one. So `var a [9223372036854775807]int` preprocessed clean and only failed when its zero value was constructed, at which point the allocator's `overflow.Mulp(allocArrayItem, len)` / `overflow.Addp(allocArray, ...)` would trip. This PR adds `checkArrayLenFits(elem, len)`, called at each of the three fixed-size-array forms, which panics a compile-time `type [N]T larger than address space` when `len` would overflow `int64` in the allocator's size computation. The threshold (`(MaxInt64 - allocArray) / per`, with `per = 40` for the per-element path and `1` for byte arrays) is the algebraic inverse of the allocator's overflow guard, so compile-time rejection and runtime overflow fire at the same length.

## Glossary

- `checkArrayLenFits`: new preprocess helper; panics if a fixed-size array's length would overflow the allocator's byte accounting.
- `allocArray` (232) / `allocArrayItem` (40): per-array and per-element byte costs the GnoVM allocator charges; `allocArrayItem == _allocTypedValue`.
- `AllocateListArray` / `AllocateDataArray`: runtime allocator entry points; byte arrays (`Uint8Kind` elements) route to the 1-byte/elem `Data` path, everything else to the 40-byte/elem `List` path (see `defaultArrayValue`).
- `[...]T{idx: v}` (ellipsis / variadic array): array whose length is inferred as `max keyed index + 1`, measured at preprocess.

## Fix

Before, the three fixed-size-array forms (`[N]T`, `[N]T{...}`, `[...]T{idx: v}`) reached the allocator with an unchecked length and overflowed there. After, [`checkArrayLenFits`](https://github.com/gnolang/gno/blob/de74ab7/gnovm/pkg/gnolang/preprocess.go#L4883-L4898) · [↗](../../../../../.worktrees/gno-review-5829/gnovm/pkg/gnolang/preprocess.go#L4883-L4898) is called at the `*ArrayTypeExpr` leave for `[N]T` / `[N]T{...}` ([`preprocess.go:2662`](https://github.com/gnolang/gno/blob/de74ab7/gnovm/pkg/gnolang/preprocess.go#L2662) · [↗](../../../../../.worktrees/gno-review-5829/gnovm/pkg/gnolang/preprocess.go#L2662)) and at the variadic-array measuring step for `[...]T{idx: v}` ([`preprocess.go:2441`](https://github.com/gnolang/gno/blob/de74ab7/gnovm/pkg/gnolang/preprocess.go#L2441) · [↗](../../../../../.worktrees/gno-review-5829/gnovm/pkg/gnolang/preprocess.go#L2441)). The load-bearing constraint is that the threshold must match the allocator exactly, or the two guards disagree on the boundary; it does.

## Benchmarks / Numbers

| Path | Element | `per` (compile check) | Runtime allocator | Threshold (`len >`) |
|------|---------|----------------------|-------------------|---------------------|
| List | non-byte | `allocArrayItem` = 40 | `allocArray + 40·len` | `(MaxInt64 − 232) / 40` ≈ 2.31e17 |
| Data | byte (`Uint8Kind`) | 1 | `allocArray + len` | `MaxInt64 − 232` ≈ 9.22e18 |

## Critical (must fix)

None.

## Warnings (should fix)

- **[largest ellipsis index escapes the compile-time check]** [`gnovm/pkg/gnolang/preprocess.go:2441`](https://github.com/gnolang/gno/blob/de74ab7/gnovm/pkg/gnolang/preprocess.go#L2441) · [↗](../../../../../.worktrees/gno-review-5829/gnovm/pkg/gnolang/preprocess.go#L2441) — `[...]T{MaxInt64: 1}` falls through to a runtime panic instead of the compile-time rejection this form is supposed to get.
  <details><summary>details</summary>

  The check runs on `idx` *after* `idx = k + 1` ([`preprocess.go:2429`](https://github.com/gnolang/gno/blob/de74ab7/gnovm/pkg/gnolang/preprocess.go#L2429) · [↗](../../../../../.worktrees/gno-review-5829/gnovm/pkg/gnolang/preprocess.go#L2429)). When the largest keyed index `k` is `MaxInt64`, `k + 1` overflows `int64` to `MinInt64`, so `checkArrayLenFits` sees a negative length, the `length <= 0` early-return skips it, and `at.Len` is set negative. At runtime `defaultArrayValue` → `NewListArray` hits its `n < 0` guard and panics a recoverable `len out of range`. Go rejects the same source at compile time (`array index 9223372036854775807 out of bounds`). `make22.gno` tests `{MaxInt64-1: 1}` (length exactly `MaxInt64`, the last value that does *not* overflow `idx`), so the boundary one step higher is untested.

  This is pre-existing (master behaves identically) and the panic is a recoverable `*Exception`, not the unrecoverable host panic that motivated #5723, so it is not a safety or determinism issue. But the PR's own comment claims the `[...]T{idx: v}` form is "rejected at compile time," and this one input in that exact form is not. Fix: validate the largest index before the `+ 1` (an index already past the threshold is "larger than address space" regardless of the `+1`), so the ellipsis form is rejected at preprocess time across its whole range.

  Observed on `de74ab7`: gno panics at runtime with `unexpected panic: len out of range`; Go rejects the same source at compile time (`array index 9223372036854775807 out of bounds`). [repro](comment_claude-opus-4-8.md)
  </details>

## Nits

- [`gnovm/pkg/gnolang/preprocess.go:4879`](https://github.com/gnolang/gno/blob/de74ab7/gnovm/pkg/gnolang/preprocess.go#L4879) · [↗](../../../../../.worktrees/gno-review-5829/gnovm/pkg/gnolang/preprocess.go#L4879) — the comment names an `allocMustFit` guard that does not exist anywhere in the tree (`grep allocMustFit` matches only this line). The real guard is the `overflow.Addp`/`overflow.Mulp` calls inside `AllocateListArray`/`AllocateDataArray`. Either drop the name or point at the actual functions.

## Missing Tests

- **[boundary one step past make22]** [`gnovm/tests/files/make22.gno`](https://github.com/gnolang/gno/blob/de74ab7/gnovm/tests/files/make22.gno) · [↗](../../../../../.worktrees/gno-review-5829/gnovm/tests/files/make22.gno) — the ellipsis tests stop at the largest length that does not overflow `idx`. Add a case for `[...]T{MaxInt64: 1}` (covered by the Warning above); it documents the boundary and would lock in the fix.

## Suggestions

- [`gnovm/pkg/gnolang/preprocess.go:2662`](https://github.com/gnolang/gno/blob/de74ab7/gnovm/pkg/gnolang/preprocess.go#L2662) · [↗](../../../../../.worktrees/gno-review-5829/gnovm/pkg/gnolang/preprocess.go#L2662) — `checkArrayLenFits(evalStaticType(store, last, n.Elt), cx.GetInt())` re-evaluates the element type via `evalStaticType`, which the immediately-following `evalStaticType(store, last, n)` recomputes as part of the whole array type. Cheap and memoized, so not worth changing on its own, but if a future edit wants the element `Type` it is already available from the array type evaluated one line down.

## Open questions

- `checkArrayLenFits` models every non-byte element as one `TypedValue` (40 bytes), which is exactly what the allocator charges, so the threshold is correct for the *allocator*. It is far more permissive than Go's actual address-space limit (e.g. `[1e15]int` passes the check and instead hits the runtime allocation cap). That is the intended scope here (catch only the `int64`-overflow case the allocator can't survive), not a gap. Not posted.
