# PR #5738: fix(gnovm): bounds-check array index so recover() catches out-of-range panic

URL: https://github.com/gnolang/gno/pull/5738
Author: ltzmaxwell | Base: master | Files: 2 | +32 -0
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: 6b091a0 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5738 6b091a0`

**Verdict: APPROVE** — small targeted fix at the right convergence point, behaviour confirmed by filetest and three independent adversarial tests; only nits and a missing-coverage observation remain.

## Summary

Indexing a Gno array with an out-of-range index produced a native Go `runtime error: index out of range`, which escaped as a host panic rather than a Gno `*Exception`. Deferred `recover()` therefore could not catch it — unlike the equivalent slice/string OOB and division-by-zero cases (`recover14`). The PR replaces the raw `av.List[ii]` / `av.Data[ii]` indexing in [`ArrayValue.GetPointerAtIndexInt2`](https://github.com/gnolang/gno/blob/6b091a0/gnovm/pkg/gnolang/values.go#L294) · [↗](../../../../../.worktrees/gno-review-5738/gnovm/pkg/gnolang/values.go#L294) with explicit `ii < 0` and `ii >= len(...)` checks that panic with a recoverable `&Exception{...}`, plus filetest `recover25.gno`. Placement is the convergence point: every array access (direct index, slice-over-array via `SliceValue.GetPointerAtIndexInt2`, pointer rebind, `copy`/`==` builtins) routes through this one method.

## Glossary

- `ArrayValue.List` / `ArrayValue.Data` — two storage backends for an `[N]T`: `List` holds `TypedValue` elements for arbitrary types; `Data` is a raw byte buffer for `[N]byte`.
- `&Exception{...}` — Gno's recoverable panic wrapper; a native Go panic with a non-`*Exception` payload escapes `recover()`.
- `GetPointerAtIndexInt2` — value-layer method that returns a `PointerValue` for `av[ii]`; convergence point for every array indexing path.

## Fix

Before: [values.go:302](https://github.com/gnolang/gno/blob/6b091a0/gnovm/pkg/gnolang/values.go#L302) · [↗](../../../../../.worktrees/gno-review-5738/gnovm/pkg/gnolang/values.go#L302) did `fillValueTV(store, &av.List[ii])` with no Gno-level bounds check, so any OOB raised a native Go runtime panic that crossed the VM/host boundary. After: both branches (`Data == nil` and `Data != nil`) gain identical `ii < 0` and `ii >= len(...)` guards that wrap the same message Go's runtime uses (`"index out of range [%d] with length %d"`) in an `&Exception{}`, restoring parity with the existing slice/string OOB paths. The negative-index variant (`"index out of range [%d]"`, no length) matches Go's native format for negative indices.

## Critical (must fix)

None.

## Warnings (should fix)

None — the change is one well-scoped guard at a convergence point. Codecov's "50% patch coverage" flags the three missing-branch cases as a Missing Test, not a correctness Warning.

## Nits

- [`gnovm/tests/files/recover25.gno:1`](https://github.com/gnolang/gno/blob/6b091a0/gnovm/tests/files/recover25.gno#L1) · [↗](../../../../../.worktrees/gno-review-5738/gnovm/tests/files/recover25.gno#L1) — `// Issue 4353.` points to an unrelated closed issue (gnogenesis `xform2` packages bug). Either replace with the correct issue number or drop the reference.
- [`gnovm/pkg/gnolang/values.go:294-329`](https://github.com/gnolang/gno/blob/6b091a0/gnovm/pkg/gnolang/values.go#L294-L329) · [↗](../../../../../.worktrees/gno-review-5738/gnovm/pkg/gnolang/values.go#L294-L329) — the four `panic(&Exception{...})` lines are identical between the `Data == nil` and `Data != nil` arms apart from `len(av.List)` vs `len(av.Data)`. Optional: collapse to one block at the top of the function using `av.GetLength()` (defined at [values.go:286](https://github.com/gnolang/gno/blob/6b091a0/gnovm/pkg/gnolang/values.go#L286) · [↗](../../../../../.worktrees/gno-review-5738/gnovm/pkg/gnolang/values.go#L286), which already abstracts the List/Data split). Six lines instead of twelve, same behaviour.

## Missing Tests

- **[3 of 4 new branches uncovered]** [`gnovm/tests/files/recover25.gno`](https://github.com/gnolang/gno/blob/6b091a0/gnovm/tests/files/recover25.gno) · [↗](../../../../../.worktrees/gno-review-5738/gnovm/tests/files/recover25.gno) — the supplied filetest exercises only `Data == nil && ii >= len(List)`. Codecov reports 2 lines missing, 2 partials; the unexercised branches are: `Data == nil && ii < 0`, `Data != nil && ii >= len(Data)` (byte-array OOB), and `Data != nil && ii < 0` (byte-array negative). Without coverage these are correct today but invisible to future refactors.
  <details><summary>details</summary>

  Three minimal adversarial filetests confirm all three branches recover correctly under the fix:
  - [`recover_array_neg_int.gno`](tests/recover_array_neg_int.gno) — negative index on `[4]int`.
  - [`recover_byte_array_oob.gno`](tests/recover_byte_array_oob.gno) — positive OOB on `[8]byte`.
  - [`recover_array_assign_oob.gno`](tests/recover_array_assign_oob.gno) — OOB on assignment LHS (`a[i] = 1`), confirming the convergence-point claim holds for both read and write.

  All three pass. Worth folding at least the negative and `Data != nil` cases into `recover25.gno` (or as `recover25a.gno` / `recover25b.gno`) so a future change to `GetPointerAtIndexInt2` that drops one branch is caught.

  Repro:

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5738 -R gnolang/gno
  cat > gnovm/tests/files/recover25_neg.gno <<'EOF'
  package main

  var a [4]int

  func main() {
  	defer func() { println("recover:", recover()) }()
  	i := -1
  	_ = a[i]
  }

  // Output:
  // recover: runtime error: index out of range [-1]
  EOF
  go test -v -run 'TestFiles/recover25_neg.gno$' ./gnovm/pkg/gnolang/
  rm gnovm/tests/files/recover25_neg.gno
  ```

  ```
  === RUN   TestFiles/recover25_neg.gno
  --- PASS: TestFiles (0.05s)
      --- PASS: TestFiles/recover25_neg.gno (0.00s)
  PASS
  ```
  </details>

## Suggestions

- [`gnovm/pkg/gnolang/values.go:393-409`](https://github.com/gnolang/gno/blob/6b091a0/gnovm/pkg/gnolang/values.go#L393-L409) · [↗](../../../../../.worktrees/gno-review-5738/gnovm/pkg/gnolang/values.go#L393-L409) — out of scope here, but worth noting for a follow-up: the existing `SliceValue.GetPointerAtIndexInt2` emits `"runtime error: slice index out of bounds: %d (len=%d)"`, which neither matches Go's runtime nor the new array message (`"index out of range [%d] with length %d"`) nor the string-OOB path at [values.go:2033](https://github.com/gnolang/gno/blob/6b091a0/gnovm/pkg/gnolang/values.go#L2033) · [↗](../../../../../.worktrees/gno-review-5738/gnovm/pkg/gnolang/values.go#L2033). Three different OOB messages for three container types — would be cleaner to unify on Go's wording in a separate PR.

## Questions for Author

- What's the intended source for `// Issue 4353` in the test file? gnolang/gno#4353 is the closed gnogenesis `xform2` packages bug — likely a stale copy from another change.
