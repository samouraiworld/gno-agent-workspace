# PR #5723: fix(gnovm): convert allocator size-overflow to recoverable panic

URL: https://github.com/gnolang/gno/pull/5723
Author: ltzmaxwell | Base: master | Files: 2 | +42 -2
Reviewed by: davd-gzl | Model: claude-opus-4-7
Local worktree: `git -C gno worktree add .worktrees/gno-review-5723 1817e52` (then `gh -R gnolang/gno pr checkout 5723` inside it)

**Verdict: REQUEST CHANGES** — fix is correct for the two paths it patches, but the same unrecoverable-host-panic shape still exists in [`AllocateMap`](https://github.com/gnolang/gno/blob/1817e52/gnovm/pkg/gnolang/alloc.go#L381-L383) · [↗](../../../../../.worktrees/gno-review-5723/gnovm/pkg/gnolang/alloc.go#L381-L383), [`AllocateString`](https://github.com/gnolang/gno/blob/1817e52/gnovm/pkg/gnolang/alloc.go#L336-L338) · [↗](../../../../../.worktrees/gno-review-5723/gnovm/pkg/gnolang/alloc.go#L336-L338), [`AllocateBlock`](https://github.com/gnolang/gno/blob/1817e52/gnovm/pkg/gnolang/alloc.go#L397-L399) · [↗](../../../../../.worktrees/gno-review-5723/gnovm/pkg/gnolang/alloc.go#L397-L399), [`AllocateBlockItems`](https://github.com/gnolang/gno/blob/1817e52/gnovm/pkg/gnolang/alloc.go#L401-L403) · [↗](../../../../../.worktrees/gno-review-5723/gnovm/pkg/gnolang/alloc.go#L401-L403), [`AllocateStructFields`](https://github.com/gnolang/gno/blob/1817e52/gnovm/pkg/gnolang/alloc.go#L373-L375) · [↗](../../../../../.worktrees/gno-review-5723/gnovm/pkg/gnolang/alloc.go#L373-L375), and [`Allocate`](https://github.com/gnolang/gno/blob/1817e52/gnovm/pkg/gnolang/alloc.go#L303-L304) · [↗](../../../../../.worktrees/gno-review-5723/gnovm/pkg/gnolang/alloc.go#L303-L304) itself. `make(map[int]int, MaxInt)` reproduces the same Go-host panic the PR claims to fix.

## Summary

Before this PR, `make([]T, n)` with `n` close to `MaxInt` triggered a Go-host `panic("multiplication overflow")` from `overflow.Mulp` inside [`AllocateListArray`](https://github.com/gnolang/gno/blob/1817e52/gnovm/pkg/gnolang/alloc.go#L352-L362) · [↗](../../../../../.worktrees/gno-review-5723/gnovm/pkg/gnolang/alloc.go#L352-L362), which Gno code could not `recover()` because it escapes the GnoVM `recover` filter that only catches `*Exception`. The fix replaces the panicking `Addp`/`Mulp` in `AllocateDataArray` and `AllocateListArray` with the boolean-returning `Add`/`Mul`, panicking a Gno-side `*Exception` with `"runtime error: makeslice: len out of range"` (matching the existing message at [`uverse.go:1046,1081`](https://github.com/gnolang/gno/blob/1817e52/gnovm/pkg/gnolang/uverse.go#L1046) · [↗](../../../../../.worktrees/gno-review-5723/gnovm/pkg/gnolang/uverse.go#L1046)). The filetest `make19.gno` confirms the new panic is `recover()`-able. The fix is correct but narrow — five other allocator entry points still use `Addp`/`Mulp` with user-controllable sizes.

## Glossary

- `Addp`/`Mulp`: overflow-panicking helpers in `tm2/pkg/overflow` — panic with a plain Go string (`"multiplication overflow"`), not catchable by Gno `recover`.
- `Add`/`Mul`: same package, return `(N, bool)` — the OK form used post-fix.
- `*Exception`: GnoVM panic envelope; only this type is surfaced to Gno-level `recover()`.
- `AllocateListArray(items)` / `AllocateDataArray(size)`: per-element vs per-byte allocator entry points; `make([]struct{}, n)` flows through the former (`_allocTypedValue=40` bytes/item), `make([]byte, n)` through the latter (1 byte/item).
- `currentRealmID` / `stampPkgID`: unrelated to this PR; mentioned only because they coexist in `alloc.go`.

## Fix

Before: [`AllocateDataArray`](https://github.com/gnolang/gno/blob/1817e52/gnovm/pkg/gnolang/alloc.go#L344-L345) · [↗](../../../../../.worktrees/gno-review-5723/gnovm/pkg/gnolang/alloc.go#L344-L345) and [`AllocateListArray`](https://github.com/gnolang/gno/blob/1817e52/gnovm/pkg/gnolang/alloc.go#L347-L349) · [↗](../../../../../.worktrees/gno-review-5723/gnovm/pkg/gnolang/alloc.go#L347-L349) used `overflow.Addp` / `overflow.Mulp` — when arithmetic overflowed `int64`, the helper panicked with a plain Go string that bypassed `m.runOnce`'s `*Exception` filter, aborting the entire transaction. After: both functions now use the `(value, ok)` variants and, on overflow, panic with `&Exception{Value: typedString("runtime error: makeslice: len out of range")}` — the same Gno-style panic that the `li < 0` guards in [`uverse.go:1045-1046`](https://github.com/gnolang/gno/blob/1817e52/gnovm/pkg/gnolang/uverse.go#L1045-L1046) · [↗](../../../../../.worktrees/gno-review-5723/gnovm/pkg/gnolang/uverse.go#L1045-L1046) already emit. The new test `gnovm/tests/files/make19.gno` confirms the path is `recover()`-able from Gno.

## Critical (must fix)

- **[same bug, other paths]** [`gnovm/pkg/gnolang/alloc.go:381-383`](https://github.com/gnolang/gno/blob/1817e52/gnovm/pkg/gnolang/alloc.go#L381-L383) · [↗](../../../../../.worktrees/gno-review-5723/gnovm/pkg/gnolang/alloc.go#L381-L383) — `AllocateMap` still uses `Addp`/`Mulp`; `make(map[K]V, MaxInt)` reproduces the same unrecoverable host panic this PR claims to fix.
  <details><summary>details</summary>

  Verified via a probe filetest with `make(map[int]int, int(^uint(0) >> 1))`: the test fails with `unexpected panic: multiplication overflow` originating at [`overflow.Mulp[...] overflow.go:71`](../../../../../.worktrees/gno-review-5723/tm2/pkg/overflow/overflow.go#L71) called from [`AllocateMap alloc.go:382`](https://github.com/gnolang/gno/blob/1817e52/gnovm/pkg/gnolang/alloc.go#L382) · [↗](../../../../../.worktrees/gno-review-5723/gnovm/pkg/gnolang/alloc.go#L382), called from [`uverse.go:1144`](https://github.com/gnolang/gno/blob/1817e52/gnovm/pkg/gnolang/uverse.go#L1144) · [↗](../../../../../.worktrees/gno-review-5723/gnovm/pkg/gnolang/uverse.go#L1144) (the `make(map, n)` native). Same shape as the `make([]T, n)` bug, same Go-host escape, same DoS — but the fix as shipped doesn't address it. If the goal is "every user-callable `make(T, n)` should produce a recoverable Gno panic on overflow", the patch is incomplete.

  Fix: apply the same `(value, ok)` pattern to `AllocateMap`, and add `make_map_overflow.gno` filetest covering it. Consider a helper (e.g. `safeAllocSize(base, factor, n int64) int64`) so the four/five call sites share one overflow-checked path with one canonical error message, rather than five copies that can drift.

  Repro:

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5723 -R gnolang/gno
  cat > gnovm/tests/files/zz_probe_map.gno <<'EOF'
  package main

  func shouldPanic(f func()) {
  	defer func() {
  		r := recover()
  		if r == nil {
  			panic("not panicking")
  		}
  		println("recovered:", r)
  	}()
  	f()
  }

  func main() {
  	shouldPanic(func() {
  		length := int(^uint(0) >> 1)
  		_ = make(map[int]int, length)
  	})
  	println("ok")
  }

  // Output:
  // recovered: runtime error: makeslice: len out of range
  // ok
  EOF
  go test -v -run 'TestFiles/zz_probe_map' ./gnovm/pkg/gnolang/
  rm gnovm/tests/files/zz_probe_map.gno
  ```

  Expected after a complete fix: PASS. Observed today: FAIL with `unexpected panic: multiplication overflow`.
  </details>

## Warnings (should fix)

- **[overflow shape elsewhere]** [`gnovm/pkg/gnolang/alloc.go:336-338`](https://github.com/gnolang/gno/blob/1817e52/gnovm/pkg/gnolang/alloc.go#L336-L338) · [↗](../../../../../.worktrees/gno-review-5723/gnovm/pkg/gnolang/alloc.go#L336-L338) — `AllocateString` uses `Addp(allocString, Mulp(allocStringByte, size))`.
  <details><summary>details</summary>

  `allocStringByte == 1` makes the multiplication trivially safe, but `Addp(48, size)` overflows when `size > MaxInt64 - 48`. Reachable in practice only if a caller passes a near-`MaxInt64` size, which a malicious actor could engineer via string-conversion paths (`string([]byte{...})` of an attacker-constructed slice) or repeated concatenation building up size before truncation. Lower attack surface than `make`, but the shape is the same and the fix is identical — switch to `(value, ok)` and panic `*Exception`.

  Fix: same treatment as `AllocateDataArray`. If you accept the Critical's helper suggestion, this becomes a one-liner.
  </details>

- **[overflow shape elsewhere]** [`gnovm/pkg/gnolang/alloc.go:397-403`](https://github.com/gnolang/gno/blob/1817e52/gnovm/pkg/gnolang/alloc.go#L397-L403) · [↗](../../../../../.worktrees/gno-review-5723/gnovm/pkg/gnolang/alloc.go#L397-L403) — `AllocateBlock` and `AllocateBlockItems` use `Addp`/`Mulp` over `items`.
  <details><summary>details</summary>

  `items` comes from `source.GetNumNames()` (compile-time) at [`alloc.go:605, 629`](https://github.com/gnolang/gno/blob/1817e52/gnovm/pkg/gnolang/alloc.go#L605) · [↗](../../../../../.worktrees/gno-review-5723/gnovm/pkg/gnolang/alloc.go#L605) and `len(nvs)` at [`nodes.go:1458`](https://github.com/gnolang/gno/blob/1817e52/gnovm/pkg/gnolang/nodes.go#L1458) · [↗](../../../../../.worktrees/gno-review-5723/gnovm/pkg/gnolang/nodes.go#L1458) — all bounded by parsed source size, so the practical attack surface is low (a multi-GB source file would be needed). But the same shape means a future caller that exposes user-controlled `items` re-introduces the unrecoverable panic. Defensive consistency: switch these too.

  Fix: same `(value, ok)` pattern. If the helper from the Critical exists, drop these in.
  </details>

- **[overflow shape elsewhere]** [`gnovm/pkg/gnolang/alloc.go:373-375`](https://github.com/gnolang/gno/blob/1817e52/gnovm/pkg/gnolang/alloc.go#L373-L375) · [↗](../../../../../.worktrees/gno-review-5723/gnovm/pkg/gnolang/alloc.go#L373-L375) — `AllocateStructFields` uses `Mulp(allocStructField, fields)`.
  <details><summary>details</summary>

  `fields` comes from `len(st.Fields)` (struct type definition, compile-time bounded), so realistically unreachable. Listed for completeness — if the helper exists, fold it in; if not, leave with a comment noting "compile-time bounded" so a future reviewer doesn't waste time relitigating.
  </details>

- **[bookkeeping overflow]** [`gnovm/pkg/gnolang/alloc.go:303-304`](https://github.com/gnolang/gno/blob/1817e52/gnovm/pkg/gnolang/alloc.go#L303-L304) · [↗](../../../../../.worktrees/gno-review-5723/gnovm/pkg/gnolang/alloc.go#L303-L304) — `Allocate` itself uses `Addp(alloc.bytes, size)`.
  <details><summary>details</summary>

  `size` is now bounded by the post-fix overflow check, but `alloc.bytes` accumulates across the tx. If a tx is configured with `maxBytes` close to `MaxInt64` (the `fallbackAllocator` at [`alloc.go:45`](https://github.com/gnolang/gno/blob/1817e52/gnovm/pkg/gnolang/alloc.go#L45) · [↗](../../../../../.worktrees/gno-review-5723/gnovm/pkg/gnolang/alloc.go#L45) uses exactly `math.MaxInt64`), `bytes + size > MaxInt64` overflows before the `> maxBytes` check fires. The `fallbackAllocator` is only used for pure-fn paths and won't see oversized allocations, but the pattern is fragile.

  Fix: rewrite as `if size > alloc.maxBytes - alloc.bytes { ... }` to avoid the addition entirely. One operation, no overflow path.
  </details>

## Nits

- [`gnovm/pkg/gnolang/alloc.go:347,355,359`](https://github.com/gnolang/gno/blob/1817e52/gnovm/pkg/gnolang/alloc.go#L347) · [↗](../../../../../.worktrees/gno-review-5723/gnovm/pkg/gnolang/alloc.go#L347) — same panic message string is repeated three times.
  <details><summary>details</summary>

  Extract a constant `errMakeSliceOOR = "runtime error: makeslice: len out of range"` (or, with the suggested helper, hide the message inside it). Saves drift risk and one line of cognitive load per call site.
  </details>

- [`gnovm/pkg/gnolang/alloc.go:467,479,496`](https://github.com/gnolang/gno/blob/1817e52/gnovm/pkg/gnolang/alloc.go#L467) · [↗](../../../../../.worktrees/gno-review-5723/gnovm/pkg/gnolang/alloc.go#L467) — pre-existing inconsistency between `"len out of range"` (these three) and `"runtime error: makeslice: len out of range"` (uverse and now this PR).
  <details><summary>details</summary>

  Go's runtime uses the `"runtime error: makeslice: ..."` form. The terser messages predate uverse-level validation. Out of scope for this PR but worth normalising in a follow-up so users always see the same string regardless of which guard fires first.
  </details>

- [`gnovm/tests/files/make19.gno:14-19`](https://github.com/gnolang/gno/blob/1817e52/gnovm/tests/files/make19.gno#L14-L19) · [↗](../../../../../.worktrees/gno-review-5723/gnovm/tests/files/make19.gno#L14-L19) — the test is named/shaped around `append`, but the panic fires inside the first `make([]struct{}, length)` at line 16 (verified by reverting the fix and inspecting the stack). The second `make` and the `append` are unreachable.
  <details><summary>details</summary>

  Cosmetic, but a future reader will think the bug is about `append`. Drop the second `make` and the `append` line, rename the test something like `make_overflow.gno` or `make_struct_overflow.gno`. If you intentionally want to keep an append-overflow scenario as a second case, fine — but add a separate `f2()` and split them.
  </details>

## Missing Tests

- **[direct callers untested]** [`gnovm/tests/files/make19.gno`](https://github.com/gnolang/gno/blob/1817e52/gnovm/tests/files/make19.gno) · [↗](../../../../../.worktrees/gno-review-5723/gnovm/tests/files/make19.gno) covers only `make([]struct{}, MaxInt)` indirectly (via the first `make`).
  <details><summary>details</summary>

  Add explicit filetests for: `make([]byte, MaxInt)` (`AllocateDataArray` path), `make([]int, MaxInt)` (the `_allocTypedValue * items` overflow path, distinct from `struct{}`), and `make(map[int]int, MaxInt)` (the still-broken path, blocked by the Critical above). A regression suite of three two-line `shouldPanic` cases gives you confidence the next refactor doesn't lose coverage.
  </details>

## Suggestions

- [`gnovm/pkg/gnolang/alloc.go:344-362`](https://github.com/gnolang/gno/blob/1817e52/gnovm/pkg/gnolang/alloc.go#L344-L362) · [↗](../../../../../.worktrees/gno-review-5723/gnovm/pkg/gnolang/alloc.go#L344-L362) — fold the duplicated `bytes, ok := overflow.Mul(...); if !ok { panic(...) }; total, ok := overflow.Add(...); if !ok { panic(...) }` shape into one helper.
  <details><summary>details</summary>

  Sketch:

  ```go
  // sizeOrPanic returns base+factor*n, panicking a recoverable Gno *Exception
  // on int64 overflow. Used by allocator entry points whose size argument
  // is user-controllable (via make/append/string conversion).
  func sizeOrPanic(base, factor, n int64) int64 {
      prod, ok := overflow.Mul(factor, n)
      if !ok {
          panic(&Exception{Value: typedString("runtime error: makeslice: len out of range")})
      }
      total, ok := overflow.Add(base, prod)
      if !ok {
          panic(&Exception{Value: typedString("runtime error: makeslice: len out of range")})
      }
      return total
  }
  ```

  Then `AllocateDataArray`, `AllocateListArray`, `AllocateString`, `AllocateMap`, `AllocateBlock` reduce to one line each, share one canonical error message, and the Critical's gap closes automatically.
  </details>

## Questions for Author

- Was the recoverability of this panic intentional, or is the goal "fail the tx cleanly"? A recoverable host-OOM-style panic lets a contract `defer recover()` and continue, which could be exploited as a fast no-op gas-burn loop. If "fail the tx" is the intent, use `m.Panic(typedString(...))` (which preempts further execution) rather than `panic(&Exception{...})` (which is `recover`-able).
- Why patch only `AllocateDataArray` and `AllocateListArray`? Was the broader scan (Map/String/Block/StructFields/Allocate) considered and deferred, or is there a reason to leave them as Go-host panics?
- Did you consider linking the issue/report that motivated this fix in the PR body? The body is empty; reviewers and future debuggers benefit from the original repro context.
