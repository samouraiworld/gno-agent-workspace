# PR [#5991](https://github.com/gnolang/gno/pull/5991): refactor(gnovm): drop redundant TypeID recompute in DeclaredType.TypeID

URL: https://github.com/gnolang/gno/pull/5991
Author: Villaquiranm | Base: master | Files: 2 | +52 -5
Reviewed by: davd-gzl | Model: claude-opus-5 | Commit: 672650736 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5991 672650736`

**TL;DR:** Every named type in the VM remembers its own identity string the first time something asks for it. The old code rebuilt that string from scratch on every later call just to confirm it still matched, and crashed if it ever did not. This PR deletes the recheck and keeps the remembered value.

**Verdict: APPROVE** — the three fields the identity is built from are written once at construction and never touched again, so the memo cannot drift from a fresh computation (1 Nit, 1 Missing test, 1 Suggestion).

## Verify first

- [`gnovm/pkg/gnolang/types.go:1931-1936`](https://github.com/gnolang/gno/blob/672650736/gnovm/pkg/gnolang/types.go#L1931-L1936) · [↗](../../../../../.worktrees/gno-review-5991/gnovm/pkg/gnolang/types.go#L1931-L1936) — the memo is only sound if `PkgPath`, `ParentLoc` and `Name` are immutable after construction. Confirm with `grep -rn 'ParentLoc\|\.PkgPath =\|\.Name =' --include='*.go' gnovm/pkg/gnolang/`: the only writes are [`declareWith`'s literal](https://github.com/gnolang/gno/blob/672650736/gnovm/pkg/gnolang/types.go#L1485-L1491) · [↗](../../../../../.worktrees/gno-review-5991/gnovm/pkg/gnolang/types.go#L1485-L1491), the two persistence copy constructors, and amino decode into a fresh value.
- [`gnovm/pkg/gnolang/store.go:831-836`](https://github.com/gnolang/gno/blob/672650736/gnovm/pkg/gnolang/store.go#L831-L836) · [↗](../../../../../.worktrees/gno-review-5991/gnovm/pkg/gnolang/store.go#L831-L836) — after the merge nothing compares a declared type's memo against a recomputation, and the nearest surviving check compares a decoded type against the key it was loaded under, only under `-tags debugAssert`. Confirm that is acceptable coverage by building the package both with and without the tag.

## Summary

[`(*DeclaredType).TypeID()`](https://github.com/gnolang/gno/blob/672650736/gnovm/pkg/gnolang/types.go#L1931-L1936) · [↗](../../../../../.worktrees/gno-review-5991/gnovm/pkg/gnolang/types.go#L1931-L1936) memoizes a named type's identity string on first call. The cached branch then rebuilt the same string with [`DeclaredTypeID`](https://github.com/gnolang/gno/blob/672650736/gnovm/pkg/gnolang/types.go#L1957-L1963) · [↗](../../../../../.worktrees/gno-review-5991/gnovm/pkg/gnolang/types.go#L1957-L1963) on every later call and panicked `should not happen` on mismatch, a check the code itself marked `XXX delete this if tests pass`. That identity is read on the interpreter's hottest paths: [map-key hashing](https://github.com/gnolang/gno/blob/672650736/gnovm/pkg/gnolang/values.go#L1926) · [↗](../../../../../.worktrees/gno-review-5991/gnovm/pkg/gnolang/values.go#L1926), [typed `==`](https://github.com/gnolang/gno/blob/672650736/gnovm/pkg/gnolang/op_binary.go#L593) · [↗](../../../../../.worktrees/gno-review-5991/gnovm/pkg/gnolang/op_binary.go#L593), [type assertions](https://github.com/gnolang/gno/blob/672650736/gnovm/pkg/gnolang/op_expressions.go#L278) · [↗](../../../../../.worktrees/gno-review-5991/gnovm/pkg/gnolang/op_expressions.go#L278), [type switches](https://github.com/gnolang/gno/blob/672650736/gnovm/pkg/gnolang/op_exec.go#L857) · [↗](../../../../../.worktrees/gno-review-5991/gnovm/pkg/gnolang/op_exec.go#L857), and the per-commit [`assertTypeIsPublic`](https://github.com/gnolang/gno/blob/672650736/gnovm/pkg/gnolang/realm.go#L1266) · [↗](../../../../../.worktrees/gno-review-5991/gnovm/pkg/gnolang/realm.go#L1266) walk.

The premise holds. The three fields feeding the identity are set together in [`declareWith`](https://github.com/gnolang/gno/blob/672650736/gnovm/pkg/gnolang/types.go#L1485-L1491) · [↗](../../../../../.worktrees/gno-review-5991/gnovm/pkg/gnolang/types.go#L1485-L1491) and nothing in the tree writes them afterwards; the only other producers are the two persistence copy constructors and amino decode, each building a fresh value with a zero memo. `fillType` mutates a decoded declared type's `Base`, `Methods` and `sealed` but [reads the identity rather than changing it](https://github.com/gnolang/gno/blob/672650736/gnovm/pkg/gnolang/realm.go#L1846) · [↗](../../../../../.worktrees/gno-review-5991/gnovm/pkg/gnolang/realm.go#L1846). So the deleted comparison was a pure function of never-mutated inputs against its own cached output, and could not fire.

Gas is untouched. The recompute allocated on the Go heap, which the VM does not meter; the identity string reaches gas accounting only through its own length in [`ComputeMapKey`](https://github.com/gnolang/gno/blob/672650736/gnovm/pkg/gnolang/values.go#L1841-L1851) · [↗](../../../../../.worktrees/gno-review-5991/gnovm/pkg/gnolang/values.go#L1841-L1851), and the string is unchanged.

## Diagram

```
declareWith (types.go:1485)      TypeID()                         hot callers
   PkgPath  ─┐                    first call → memoize            ComputeMapKey
   Name     ─┤ written once ───►  dt.typeid ──────────────────►   == and !=
   ParentLoc┘  never rewritten    later calls → read the field    type assert
                                    was: rebuild the string and   type switch
                                    panic on mismatch             assertTypeIsPublic
```

## Fix

The `else` arm of the cached branch is gone, so `TypeID()` computes on a zero memo and returns the field otherwise. The load-bearing constraint is that the identity stays a pure function of `PkgPath`, `ParentLoc` and `Name`: any future code that rewrites one of those on a live `*DeclaredType` now silently keeps the stale identity instead of aborting, and identity drift on a named type is a consensus fault rather than a local one.

## Benchmarks / Numbers

Per cached `TypeID()` call, measured against merge base d1a33f574 and 672650736. Allocation counts are machine-independent; wall time is not quoted because the sandbox runs a bare `fmt.Sprintf` at roughly 500 ns, five to eight times slower than a normal host.

| cached call on | `fmt.Sprintf` calls | allocations | bytes |
|---|---|---|---|
| package or file-level type, before | 1 | 3 | 64 |
| function-level type, before | 5 | 13 | 277 |
| either, after | 0 | 0 | 0 |

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- **[a permanent comment naming the wrong cost]** [`gnovm/pkg/gnolang/declaredtype_typeid_bench_test.go:30-32`](https://github.com/gnolang/gno/blob/672650736/gnovm/pkg/gnolang/declaredtype_typeid_bench_test.go#L30-L32) · [↗](../../../../../.worktrees/gno-review-5991/gnovm/pkg/gnolang/declaredtype_typeid_bench_test.go#L30-L32) — says the removed assertion cost three `fmt.Sprintf` for a function-level type, but the fixture's span runs from line 42 to line 50, which takes `Span.String`'s multi-line branch and makes it five.
  <details><summary>details</summary>

  [`Span.String`'s multi-line branch](https://github.com/gnolang/gno/blob/672650736/gnovm/pkg/gnolang/nodes_location.go#L192-L196) · [↗](../../../../../.worktrees/gno-review-5991/gnovm/pkg/gnolang/nodes_location.go#L192-L196) calls `Pos.String` on both ends, two `fmt.Sprintf` the same-line branches never pay. Counted by instrumenting `typeidf`, `Location.String`, `Span.String` and `Pos.String`: 1 for a zero `ParentLoc`, 3 for a same-line span, 5 for [this fixture's span](https://github.com/gnolang/gno/blob/672650736/gnovm/pkg/gnolang/declaredtype_typeid_bench_test.go#L40-L43) · [↗](../../../../../.worktrees/gno-review-5991/gnovm/pkg/gnolang/declaredtype_typeid_bench_test.go#L40-L43). A `ParentLoc` is a function or function-literal location, so multi-line is the ordinary case and five is the number a reader wants. The [same comments](https://github.com/gnolang/gno/blob/672650736/gnovm/pkg/gnolang/declaredtype_typeid_bench_test.go#L5-L16) · [↗](../../../../../.worktrees/gno-review-5991/gnovm/pkg/gnolang/declaredtype_typeid_bench_test.go#L5-L16) frame both benchmarks against a baseline that stops existing at merge, leaving two benchmarks of a single field read annotated with costs they no longer have. Fix: give the count for a multi-line span and describe what the benchmarks measure now rather than against the deleted branch.
  </details>

## Missing Tests

- **[the written form of a consensus-relevant identity is unpinned]** [`gnovm/pkg/gnolang/types.go:1957-1963`](https://github.com/gnolang/gno/blob/672650736/gnovm/pkg/gnolang/types.go#L1957-L1963) · [↗](../../../../../.worktrees/gno-review-5991/gnovm/pkg/gnolang/types.go#L1957-L1963) — no test asserts either written form of a declared type's identity, and the deleted branch was the only in-tree check tying `dt.typeid` to `DeclaredTypeID`.
  <details><summary>details</summary>

  The identity keys stored type definitions in the backend through [`backendTypeKey`](https://github.com/gnolang/gno/blob/672650736/gnovm/pkg/gnolang/store.go#L1456-L1458) · [↗](../../../../../.worktrees/gno-review-5991/gnovm/pkg/gnolang/store.go#L1456-L1458) and decides typed equality and map-key identity, so both forms are consensus-relevant. The `pkgPath[loc].Name` form does appear in filetest goldens, but only through [`DeclaredType.String`](https://github.com/gnolang/gno/blob/672650736/gnovm/pkg/gnolang/types.go#L1965-L1971) · [↗](../../../../../.worktrees/gno-review-5991/gnovm/pkg/gnolang/types.go#L1965-L1971), which rebuilds the same string through separate code, so a change to `DeclaredTypeID` alone breaks no golden. [`tests/declaredtype_typeid_format_test.go`](tests/declaredtype_typeid_format_test.go) pins both forms, re-checks the memo against a fresh computation the way the deleted branch did, and asserts `String()` and `TypeID()` still agree; it passes at 672650736 in the same style as the neighbouring [`TestInterfaceTypeID_PkgPathProvenance`](https://github.com/gnolang/gno/blob/672650736/gnovm/pkg/gnolang/types_test.go#L137-L152) · [↗](../../../../../.worktrees/gno-review-5991/gnovm/pkg/gnolang/types_test.go#L137-L152). Fix: add the test alongside the two benchmarks.
  </details>

## Suggestions

- **[an invariant this file keeps under a build tag rather than deleting]** [`gnovm/pkg/gnolang/types.go:1931-1936`](https://github.com/gnolang/gno/blob/672650736/gnovm/pkg/gnolang/types.go#L1931-L1936) · [↗](../../../../../.worktrees/gno-review-5991/gnovm/pkg/gnolang/types.go#L1931-L1936) — wrapping the comparison in `if debugAssert` costs nothing in production builds and keeps the tripwire for a future field rewrite.
  <details><summary>details</summary>

  An identity check of the same class already survives one layer out, [gated on `debugAssert` and comparing a decoded type against the key it was loaded under](https://github.com/gnolang/gno/blob/672650736/gnovm/pkg/gnolang/store.go#L831-L836) · [↗](../../../../../.worktrees/gno-review-5991/gnovm/pkg/gnolang/store.go#L831-L836), and [`debug.go`](https://github.com/gnolang/gno/blob/672650736/gnovm/pkg/gnolang/debug.go#L24-L28) · [↗](../../../../../.worktrees/gno-review-5991/gnovm/pkg/gnolang/debug.go#L24-L28) names the remaining `if debug { panic }` sites in other files as migration candidates for exactly this tag rather than removal. The deleted check was unconditional rather than `debug`-gated, so it is a step past that note, not an instance of it. `types_test.go` documents the same treatment for a sibling invariant: [the interior `TypeID` sites assume it and assert only under the tag](https://github.com/gnolang/gno/blob/672650736/gnovm/pkg/gnolang/types_test.go#L154-L160) · [↗](../../../../../.worktrees/gno-review-5991/gnovm/pkg/gnolang/types_test.go#L154-L160), with [a test that exercises them only when it is set](https://github.com/gnolang/gno/blob/672650736/gnovm/pkg/gnolang/types_test.go#L173-L177) · [↗](../../../../../.worktrees/gno-review-5991/gnovm/pkg/gnolang/types_test.go#L173-L177). The counter-argument is that the check is vacuous by construction, which the immutability of the three fields supports today; the choice is whether a future rewrite of one of them should abort or silently keep a stale identity. Fix: keep the comparison under `debugAssert` instead of deleting it, or say in the commit that the immutability of the three fields is the replacement for it.
  </details>

## Verified

- The cached branch agrees with a fresh computation on every execution: reinstated the recompute as a counter instead of a panic and ran the `gnovm/pkg/gnolang` suite, which executed the cached branch more than 24 million times with zero mismatches.
- The allocation delta is measured, not inferred: the PR's own two benchmarks run against merge base d1a33f574 and 672650736 go from 3 allocations and 64 bytes (package-level) and 13 allocations and 277 bytes (function-level) per cached call to zero.
- `fmt.Sprintf` counts per identity computation, from instrumenting `typeidf`, `Location.String`, `Span.String` and `Pos.String`: 1 for a zero `ParentLoc`, 3 for a same-line span, 5 for a multi-line one.
- [`tests/declaredtype_typeid_format_test.go`](tests/declaredtype_typeid_format_test.go) passes at 672650736.

## Open questions

- Both benchmarks reduce to the same single field read once this merges, so the pair may be worth dropping rather than keeping in the tree. Not posted: the author may want them as a regression tripwire on the memo, and either choice is defensible.
- [`sealUverseTypes`](https://github.com/gnolang/gno/blob/672650736/gnovm/pkg/gnolang/uverse.go#L1919-L1932) · [↗](../../../../../.worktrees/gno-review-5991/gnovm/pkg/gnolang/uverse.go#L1919-L1932) pre-fills these memos at package init so the process-global type singletons are read-only afterwards; this diff makes the cached path on them a plain field read, which strengthens that guarantee. Not posted: no change needed.
