# PR #5708: fix(gnovm): implement zerobase semantics for zero-sized types

URL: https://github.com/gnolang/gno/pull/5708
Author: ltzmaxwell | Base: master | Files: 6 | +210 -29
Reviewed by: davd-gzl | Model: claude-opus-4-7

**Verdict: REQUEST CHANGES** — the stated parity with Go's `runtime.zerobase` is broken by an incomplete `isZeroSizeType` (misses `[N]T` where `T` is zero-sized but `N>0`), and the per-machine cache silently breaks the same pointer-identity guarantee across realm persistence.

## Summary

The PR makes `new(T)` and `&var` for zero-sized types share a single `HeapItemValue` per machine, so `new(struct{}) == new(struct{})` and `&x == &y` evaluate to true — matching Go's `runtime.zerobase` shape. The mechanism is a `map[TypeID]*HeapItemValue` on `Machine`, gated by a new `isZeroSizeType` predicate ([`types.go:1554-1567`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/types.go#L1554-L1567)), with two callsites: the `new` builtin ([`uverse.go:1168`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/uverse.go#L1168)) and `doOpRef` ([`op_expressions.go:224`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/op_expressions.go#L224)). The Go spec is explicit that both "may equal" and "may not equal" outcomes are conformant for distinct zero-size variables, so the choice is defensible.

Two gaps undercut the parity claim. First, `isZeroSizeType` for `*ArrayType` only returns true when `Len == 0`, so it misses `[10]struct{}` and `[5][0]int` — both genuinely zero-sized in Go (`unsafe.Sizeof` returns 0) — and `new` on those types still returns distinct pointers. Second, the shared `HeapItemValue` is cached on the live `Machine` and stamped with the *first* realm's `PkgID` at creation; this leaks across realm boundaries within a tx and breaks pointer identity across txs once one of these pointers reaches realm storage.

## Glossary

- `zerobase` — Go's single shared heap address used as the backing for all zero-sized allocations (`runtime.zerobase`).
- `HeapItemValue` (HIV) — Gno's heap cell; persists as its own object with an `ObjectID` and a `PkgID` for storage attribution.
- `GetZerobase` — new helper in [`machine.go:296`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/machine.go#L296) that lazily allocates and caches one HIV per `TypeID`.
- `stampPkgID` — at HIV creation, [`alloc.go:629`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/alloc.go#L629) writes `currentRealmID` onto the HIV. That stamp determines storage rent and `touchForeignRealm` routing at persist time.

## Fix

Before: `new(struct{})` and `&var` for any type — zero-sized or not — went through `m.Alloc.NewHeapItem`, returning a fresh HIV each call. So `new(struct{}) == new(struct{})` was always false. After: `isZeroSizeType` short-circuits both paths through `m.GetZerobase`, which looks up (or lazily creates) a per-`TypeID` HIV in `m.zerobaseAllocs`. Within a single machine run, repeat calls for the same `TypeID` return the same `Base` HIV and pointer equality holds. The load-bearing constraint is that `m.zerobaseAllocs` is a `map[TypeID]*HeapItemValue` on the `Machine` struct ([`machine.go:38`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/machine.go#L38)), keyed by `t.TypeID()` (not `baseOf(t).TypeID()`), so a `DeclaredType` and its unnamed base resolve to different cache entries.

## Critical (must fix)

- **[parity gap: `[N]T` with N>0 and zero-sized T not detected]** [`types.go:1556-1557`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/types.go#L1556-L1557) — `isZeroSizeType` only returns true for `*ArrayType` when `Len == 0`, missing every array of zero-sized elements (`[10]struct{}`, `[5][0]int`, etc.). These are zero-sized in Go (`unsafe.Sizeof` = 0) and Go's runtime returns the shared zerobase for `new` on them.
  <details><summary>details</summary>

  Adversarial tests [`ptr12b_zerosize_array_of_zsized.gno`](tests/ptr12b_zerosize_array_of_zsized.gno) and [`ptr12c_zerosize_nested_array.gno`](tests/ptr12c_zerosize_nested_array.gno) both `// run` with expected `// Output: ok` but fail on this branch (`new([10]struct{})` and `new([5][0]int)` return distinct pointers). The Go reference at [`tests/go_zerosize_reference_test.go`](tests/go_zerosize_reference_test.go) confirms `unsafe.Sizeof([10]struct{}) == 0` and observes Go's heap shares the zerobase for these (both `new` calls returned `0x6f4b80` in my run).

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5708 -R gnolang/gno
  cat > gnovm/tests/files/ptr12b_zerosize_array_of_zsized.gno <<'EOF'
  // run
  package main
  func main() {
      p := new([10]struct{})
      q := new([10]struct{})
      if p != q { println("FAIL"); return }
      println("ok")
  }
  // Output:
  // ok
  EOF
  cat > gnovm/tests/files/ptr12c_zerosize_nested_array.gno <<'EOF'
  // run
  package main
  func main() {
      p := new([5][0]int)
      q := new([5][0]int)
      if p != q { println("FAIL"); return }
      println("ok")
  }
  // Output:
  // ok
  EOF
  go test -v -run 'TestFiles/ptr12[bc]' -test.short -timeout 60s ./gnovm/pkg/gnolang/
  rm gnovm/tests/files/ptr12b_zerosize_array_of_zsized.gno gnovm/tests/files/ptr12c_zerosize_nested_array.gno
  ```

  Fix: recurse into the element type, e.g.
  ```go
  case *ArrayType:
      return ct.Len == 0 || isZeroSizeType(ct.Elt)
  ```
  Add `ptr12b`/`ptr12c`-style cases to the filetest corpus.
  </details>

## Warnings (should fix)

- **[cross-tx pointer-identity break once persisted]** [`machine.go:296-308`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/machine.go#L296-L308) — the shared HIV lives only as long as the `Machine`. Once `p := new(struct{})` is reachable from realm state, the HIV gets a persistent `ObjectID` ([`realm.go:1306-1320`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/realm.go#L1306-L1320)). In the next tx, a fresh `Machine` starts with `m.zerobaseAllocs == nil` and the next `new(struct{})` mints a different HIV with a different `ObjectID` — so `globalP == new(struct{})` flips from true (in the originating tx) to false on reload.
  <details><summary>details</summary>

  Pointer equality in Gno compares `Base` ObjectIDs, so the persisted `globalP` (Base = HIV-O1, stored in stage 1) and a fresh `q := new(struct{})` (Base = HIV-O2 in a later tx) are no longer equal. Within a single tx the PR's invariant holds; across the persistence boundary it doesn't. The PR doc on `GetZerobase` ([`machine.go:276-295`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/machine.go#L276-L295)) hedges with "within a machine execution," but realm storage is the natural user mental model and the asymmetry is surprising.

  Fix options, roughly in order of effort: (a) document the within-machine scoping explicitly in the PR description and add a filetest pairing `new(struct{})` against `pe` (the global from `ptr12a.gno`) reloaded after a save round-trip to lock in the surprising behavior; (b) refuse to share when the pointer can escape to realm state (preprocess-time detect zero-sized pointer-to-global and skip the cache); (c) use a stable, well-known sentinel HIV per `TypeID` baked into the realm — significantly more work.
  </details>

- **[cross-realm HIV ownership leak in one tx]** [`machine.go:303-304`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/machine.go#L303-L304) — `NewHeapItem` stamps `PkgID = m.Alloc.currentRealmID` ([`alloc.go:629`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/alloc.go#L629)). The *first* realm to call `new(zero-sized)` owns the cached HIV. When a second realm later in the same tx allocates the same type and persists the resulting pointer, `assignNewObjectID` ([`realm.go:1937-1964`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/realm.go#L1937-L1964)) keeps the first realm's `PkgID` and routes via `touchForeignRealm`. Storage rent and bookkeeping for `r/B`'s value attribute to `r/A`.
  <details><summary>details</summary>

  For zero-sized types the persisted payload is empty so the practical rent impact is tiny, but the routing is still surprising — `r/B` cannot make its own `new(struct{})` purely local once `r/A` has touched the cache earlier in the same tx. None of the existing test plan exercises this (the PR runs `Gas`, `integration`, and `TestFiles` but never cross-realm). I did not write a txtar reproducer because the integration scaffolding is non-trivial, but the mechanism is mechanical from the stamp at HIV-creation time.

  Fix: stamp the HIV with a sentinel-zero `PkgID` (skip `alloc.stampPkgID` inside `GetZerobase`) so `assignNewObjectID`'s "PkgID.IsZero() → adopt by finalizing realm" branch ([`realm.go:1928-1936`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/realm.go#L1928-L1936)) takes over and each realm that persists a shared zerobase pointer adopts its own copy. Re-uses the existing "no authority stamp" code path designed for stdlib-block-allocated heap items.
  </details>

## Nits

- [`op_expressions.go:211-226`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/op_expressions.go#L211-L226) — for `&CompositeLit{}` where the literal is zero-sized, `PopAsPointer2` already allocated a fresh HIV at [`machine.go:2767`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/machine.go#L2767) (allocator-charged) before `doOpRef` discards it for the shared zerobase. Net cost for `p := &struct{}{}` is two `AllocateHeapItem` charges where one suffices. Hoist the `isZeroSizeType(elt)` check above `PopAsPointer2` for `CompositeLitExpr` rx, or special-case in `PopAsPointer2`.
- `uverse.go:1168-1191` — the two branches duplicate the `PushValue(TypedValue{T: ..., V: ...})` shape. Hoisting the `PointerType` construction above the branch and just choosing `V` inside would cut ~6 lines without changing behavior.
- [`machine.go:38`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/machine.go#L38) — `zerobaseAllocs` reformats the `Machine` field alignment block from a tight comment-aligned list to a wider one purely so the new field's type fits. Consider keeping the field unexported on a sub-struct or moving it next to `Alloc` so the canonical field block stays readable.
- [`machine.go:276-308`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/machine.go#L276-L308) — the doc comment cites the Go spec twice (also at [`op_expressions.go:216-223`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/op_expressions.go#L216-L223) and across both new filetests). One canonical paragraph on `GetZerobase` plus single-line "see GetZerobase doc" pointers elsewhere would cut ~30 lines of repeated spec citation.

## Missing Tests

- **[parity gap not covered]** `gnovm/tests/files/ptr12*.gno` — see Critical finding; the existing `ptr12.gno`/`ptr12a.gno` only exercise the cases that the partial `isZeroSizeType` already handles (`[0]int`, `struct{}`, `struct{_ [0]int}`). Add the failing cases.
- **[cross-tx persistence]** no test rounds-trips a zero-sized pointer through realm save/load and asserts equality against a fresh `new(T)` in a follow-up tx. Without it the invariant from `GetZerobase`'s doc is asserted only in the steady state, not at the boundary that breaks it.
- **[cross-realm reuse]** no test exercises `r/A.f()` (touches the cache) → `r/B.g()` (allocates same type, persists the pointer). Even if the Warning is accepted as-is, a test pinning the storage-rent attribution would prevent silent regression if `assignNewObjectID` ever changes the foreign-realm routing.

## Suggestions

- [`machine.go:38`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/machine.go#L38) — consider whether `zerobaseAllocs` should be allocated lazily inside `GetZerobase` (as it is now) versus initialized in the machine pool's `Release` path so the zero value of `Machine` is sufficient. As written, the first call per machine pays a `make(map)` charge; not on a hot path but worth a glance.
- [`uverse.go:1166-1167`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/uverse.go#L1166-L1167) — `m.Alloc.checkConstructionTime(tt)` runs unconditionally before the `isZeroSizeType` branch. For zero-sized types declared in a foreign realm (`type Z = [0]int` in `/r/foo`), this still panics on cross-realm `new(Z)`. That's probably the intended behavior, but worth confirming explicitly that "zerobase parity" doesn't extend to bypassing the realm-construction gate.

## Questions for Author

- The PR body says "Gno chooses the 'may' side (same address ⇒ equal)" — was the cross-tx asymmetry (within-tx: equal; cross-tx after persistence: unequal) considered? The Go spec text quoted is per-execution; Gno's realm persistence is a stronger guarantee surface than Go's program lifetime. Is the within-machine scoping intentional, or an artifact of the per-`Machine` cache choice?
- Did the test plan include cross-realm scenarios (`/r/a` and `/r/b` both calling `new(struct{})` in the same tx, persisting the result)? The `gno.land/pkg/integration/` line in the plan implies yes, but I couldn't find a specific txtar that pins the storage-rent attribution.
- `isZeroSizeType` doesn't recurse into `*ArrayType.Elt`. Was that an oversight or a deliberate scoping to "directly zero-length"?
