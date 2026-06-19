# PR #5812: perf(gnovm): avoid heap-boxed byte access in copy, range and index reads

URL: https://github.com/gnolang/gno/pull/5812
Author: thehowl | Base: master | Files: 7 | +144 -43
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: c845152ad (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5812 c845152ad`

**TL;DR:** Reading a byte out of a `[]byte`, `[N]byte`, or string (via `b[i]`, `for i, c := range b`, or `copy`) used to allocate a throwaway heap object per byte just to read it back. This adds a read-only fast path that returns the byte directly, and lets `copy` move bytes in bulk when both sides are byte-backed. No behavior change; it is purely fewer allocations.

**Verdict: APPROVE** — correct and behavior-preserving; out-of-range byte reads stay `recover()`-able (revert-proven), output matches Go across index/range/copy. Only a stale comment nit and one optional regression test.

## Summary
The per-element byte pointer getter (`ArrayValue.GetElementPointer`, the renamed `GetPointerAtIndexInt2`) materializes a heap `*TypedValue` + boxed `DataByteValue` per byte, which the read callers immediately `Deref` and discard: the PR measured 67% of all heap objects in the bytes stdlib suite (694M) coming from this. The fix adds `TypedValue.GetByteAtIndexInt`, a read-only fast path for strings and Data-backed (byte) arrays/slices that returns the element value directly, wired into `b[i]` reads (`doOpIndex1`) and the range loop; `copy` gets direct `copy()` of raw bytes when both sides are Data-backed (or the source is a string). The boxes were raw Go allocations never charged to the VM allocator, so gas and `MAXALLOC` goldens are unchanged. The rest of the diff renames the index-accessor family (`GetPointerAtIndexInt2` → `GetElementPointer`, `GetValueAtIntIndex` → `GetByteAtIndexInt`) and adds `copy5e.gno`.

The one real hazard in this class is recoverability: a Data-backed byte read that indexes `av.Data[ii]` directly raises a Go-native panic, which `runOnce` re-raises past gno `recover()` (the #5738 bug). The fast path mirrors `GetElementPointer`'s explicit `*Exception` bounds checks for all three kinds, so OOB byte reads stay catchable. Verified by reverting them (below).

## Glossary
- Exception: GnoVM's Go-level panic value; `runOnce` (`machine.go`) catches `*Exception` and re-raises anything else, so a bare Go panic escapes gno `recover()`.
- filetest: a `.gno` file run by the VM and asserted against `// Output:` / `// Error:` goldens.
- realm: stateful on-chain package under `r/`; `DidUpdate` marks its objects dirty and enforces cross-realm write permission.

## Fix
Before: `b[i]`, byte-range, and `copy` each routed through `GetPointerAtIndex` → `GetElementPointer`, which heap-allocates a `DataByteValue` box per byte (`copy` allocated two per byte). After: `doOpIndex1` ([op_expressions.go:29](https://github.com/gnolang/gno/blob/c845152ad/gnovm/pkg/gnolang/op_expressions.go#L29) · [↗](../../../../../.worktrees/gno-review-5812/gnovm/pkg/gnolang/op_expressions.go#L29)) and the range body ([op_exec.go:219](https://github.com/gnolang/gno/blob/c845152ad/gnovm/pkg/gnolang/op_exec.go#L219) · [↗](../../../../../.worktrees/gno-review-5812/gnovm/pkg/gnolang/op_exec.go#L219)) try `GetByteAtIndexInt` first and fall back to the boxed path when it returns `ok == false` (maps, List-backed arrays/slices, pointer-to-array). `copy` writes raw bytes directly into the Data backing ([uverse.go:907](https://github.com/gnolang/gno/blob/c845152ad/gnovm/pkg/gnolang/uverse.go#L907) · [↗](../../../../../.worktrees/gno-review-5812/gnovm/pkg/gnolang/uverse.go#L907), [uverse.go:952](https://github.com/gnolang/gno/blob/c845152ad/gnovm/pkg/gnolang/uverse.go#L952) · [↗](../../../../../.worktrees/gno-review-5812/gnovm/pkg/gnolang/uverse.go#L952)), bypassing `Assign2`; the pre-existing top-level `m.Realm.DidUpdate(m, dstBase, nil, nil)` is what marks the array dirty and enforces cross-realm write permission, and Go's `copy` is overlap-safe so the manual backward-copy is only kept for the List fallback.

## Verification
Checks CI does not surface:

- **Recoverability is load-bearing.** Deleting the two `ii < 0` / `ii >= len(av.Data)` bounds checks from `GetByteAtIndexInt`'s array branch makes `recover25b.gno`'s `a[7]` (len 4) escape as a Go-native `index out of range` that crashes the test runner (`machine.go` stack, not a recovered gno panic) — exactly the #5738 regression. With the checks, `recover()` catches `runtime error: index out of range [7] with length 4`. String and byte-slice OOB through the fast path are likewise recoverable (`slice index out of bounds: 5 (len=3)`, `index out of range [5] with length 3`), matching the slow-path messages verbatim.
- **Go parity.** A side-by-side gno-vs-Go run of `b[i]` reads (slice/array/string), `for i, c := range` over all three, `copy(string→[]byte)`, `copy([]byte→[]byte)`, an overlapping `copy(o[2:], o[:6])` (`12345678` → `12123456`), a partial copy, and a named `MyByte` element type produced byte-identical output. The overlap case confirms the direct `copy()` handles aliasing the same as Go.
- **Result-type parity.** `Deref` of a `DataByteType` box sets `tv.T = dbv.ElemType` ([values.go:244](https://github.com/gnolang/gno/blob/c845152ad/gnovm/pkg/gnolang/values.go#L244) · [↗](../../../../../.worktrees/gno-review-5812/gnovm/pkg/gnolang/values.go#L244)); the fast path sets `res.T = bt.Elt` (the same element type) and `SetUint8`, so named byte types survive.
- **Gas/alloc unchanged.** `go test ./gno.land/pkg/sdk/vm/ -run Gas` passes. The local `alloc_0/3/4` filetest MemStats mismatches are pre-existing: they fail identically on pristine `origin/master` (`bytes:6918` golden vs `7526` observed), so they are an environment-calibration artifact, not PR-induced.

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
- **[stale comment after rename]** `gnovm/tests/files/recover25b.gno:2` — comment names `ArrayValue.GetPointerAtIndexInt2`, which this PR renames to `GetElementPointer`; and `a[i]` on a byte array now routes through the new `GetByteAtIndexInt` fast path, not that function at all. Point it at `GetByteAtIndexInt` so the regression test names the path it actually guards. [`recover25b.gno:2`](https://github.com/gnolang/gno/blob/c845152ad/gnovm/tests/files/recover25b.gno#L2) · [↗](../../../../../.worktrees/gno-review-5812/gnovm/tests/files/recover25b.gno#L2)

## Missing Tests
- **[only 1 of 3 fast-path branches has a recover guard]** `gnovm/pkg/gnolang/values.go:2010` — `recover25b` guards the array branch (the one that was actually buggy, #5738). The string and byte-slice branches of `GetByteAtIndexInt` carry the same `*Exception` bounds checks but no filetest exercises their OOB recoverability, so a future edit dropping them would go unnoticed. Two small `recover()` filetests (`s[i]` and `b[i]` on a byte slice, both OOB) would close the gap; confirmed behaviorally that both currently recover with the correct messages.
  <details><summary>details</summary>

  The three branches now share one "must stay recoverable" invariant in a single function; coverage is asymmetric to risk only by history, not by structure. Low priority — the code is correct today. [`values.go:2010`](https://github.com/gnolang/gno/blob/c845152ad/gnovm/pkg/gnolang/values.go#L2010) · [↗](../../../../../.worktrees/gno-review-5812/gnovm/pkg/gnolang/values.go#L2010)
  </details>

## Suggestions
None.

## Open questions
- `GetByteAtIndexInt`'s slice branch calls `sv.GetBase(store)` and dereferences `base.Data` *before* the `ii < 0` / `sv.Length <= ii` bounds checks ([values.go:2042](https://github.com/gnolang/gno/blob/c845152ad/gnovm/pkg/gnolang/values.go#L2042) · [↗](../../../../../.worktrees/gno-review-5812/gnovm/pkg/gnolang/values.go#L2042)), where `SliceValue.GetElementPointer` bounds-checks first and only then resolves the base. Not currently reachable as a nil deref: a zero/nil slice is `tv.V == nil` (the zero value of a `SliceType` is `V: nil`, [values.go:2768](https://github.com/gnolang/gno/blob/c845152ad/gnovm/pkg/gnolang/values.go#L2768) · [↗](../../../../../.worktrees/gno-review-5812/gnovm/pkg/gnolang/values.go#L2768)), caught by the `tv.V == nil` guard, so any non-nil `*SliceValue` has a non-nil `Base`. Flagging only because the ordering diverges from the slow path and would become a non-recoverable nil deref if a nil-`Base` `*SliceValue` were ever produced; not worth a code change in this PR. Not posted.
