# PR #5708 (round 2): fix(gnovm): implement zerobase semantics for zero-sized types

URL: https://github.com/gnolang/gno/pull/5708
Author: ltzmaxwell | Base: master | Files: 8 | +210 -13
Reviewed by: davd-gzl | Model: claude-opus-4-7
Prior review (for history; this review stands alone): [1-350d630e3/](../1-350d630e3/claude-opus-4-7_davd-gzl.md)

**Verdict: NEEDS DISCUSSION** — `isZeroSizeType` now recurses into array elements, closing the parity gap flagged in round 1; ptr12b/ptr12c filetests pin the fix. Two architectural warnings carry over unaddressed: pointer identity for shared zero-size HIVs breaks across tx boundaries (per-machine cache evaporates), and HIV ownership leaks across realms within a single tx (PkgID stamped at first allocation). Neither is a bug in the *new* commits, but both shape the contract worth pinning before merge. CI is red for reasons unrelated to this PR.

## Summary

The PR makes `new(T)` and `&var` for zero-sized types share a single `HeapItemValue` (HIV) per machine, so `new(struct{}) == new(struct{})` and `&x == &y` evaluate to true — matching Go's `runtime.zerobase` shape. The mechanism is a `map[TypeID]*HeapItemValue` on `Machine` (`zerobaseAllocs`), gated by an `isZeroSizeType` predicate ([`types.go:1554-1568`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/types.go#L1554-L1568)), with two callsites: the `new` builtin ([`uverse.go:1168`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/uverse.go#L1168)) and `doOpRef` ([`op_expressions.go:219`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/op_expressions.go#L219)). The Go spec allows both "may equal" and "may not equal" outcomes for distinct zero-size variables; the PR picks determinism.

## Glossary

- `HeapItemValue` (HIV) — Gno's heap cell. Persists with an `ObjectID` (assigned at persist time) and a `PkgID` (stamped at creation, drives storage rent and `touchForeignRealm` routing).
- `GetZerobase` — helper at [`machine.go:300-312`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/machine.go#L300-L312); lazily allocates and caches one HIV per `TypeID`.

## What changed since round 1 (350d630e3 → e94b9079)

| status | item |
|---|---|
| fixed | `isZeroSizeType` recurses into array elements ([`types.go:1557`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/types.go#L1557)); ptr12b/ptr12c filetests added |
| fixed | `zerobaseAllocs` moved out of `Machine` State block ([`machine.go:55-60`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/machine.go#L55-L60)) |
| fixed | spec-citation duplication collapsed to `// see GetZerobase` ([`op_expressions.go:216`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/op_expressions.go#L216)) |
| unaddressed | cross-tx pointer-identity break (Warning below) |
| unaddressed | cross-realm HIV ownership leak (Warning below) |
| unaddressed | `&CompositeLit{}` double-allocate in `doOpRef`; `uverse.go new` two-branch duplication (Nits below) |
| no response | cross-tx asymmetry question to the author |

## Verification

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5708 -R gnolang/gno
go test -v -run 'TestFiles/ptr12' -test.short -timeout 60s ./gnovm/pkg/gnolang/
# all four ptr12* filetests pass
```

CI is red but **unrelated**: `gno-checks / lint` and `params_valset_rotation_throttle` both fail on master HEAD too (master runs `26301907221`, `26290866204`).

## Critical (must fix)

None.

## Warnings (should fix)

- **[cross-tx pointer-identity break once persisted]** [`machine.go:300-312`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/machine.go#L300-L312) — `zerobaseAllocs` is a field on `Machine`, so the cache lives only as long as the live machine. Once a `new(struct{})` pointer reaches realm state, the underlying HIV gets a stable `ObjectID` and persists. The *next* tx starts with a fresh `Machine` (`zerobaseAllocs == nil`), so the next `new(struct{})` mints a different HIV with a different `ObjectID`. Pointer equality in Gno compares `Base.ObjectID`, so `globalP == new(struct{})` flips from true (originating tx) to false (reload). `GetZerobase`'s new doc hedges "within a machine execution" — accurate, but worth pinning explicitly since realm-persisted pointers naturally outlive their machine.

- **[cross-realm HIV ownership leak in one tx]** [`machine.go:300-312`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/machine.go#L300-L312) — `GetZerobase` calls `m.Alloc.NewHeapItem`, which stamps `PkgID = currentRealmID` ([`alloc.go:629`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/alloc.go#L629)). The *first* realm to call `new(zero-sized)` owns the cached HIV. A second realm in the same tx that allocates the same type and persists the result keeps the first realm's `PkgID`; `assignNewObjectID` ([`realm.go:1937-1964`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/realm.go#L1937-L1964)) routes via `touchForeignRealm` and storage rent for `r/B` attributes to `r/A`.
  <details><summary>reproducer</summary>

  [`tests/zerobase_cross_realm.txtar`](tests/zerobase_cross_realm.txtar) loads two realms, calls `r/zerobase_a.Demo` which (1) does `pa := new(struct{})` (cache populated, HIV stamped `PkgID = r/zerobase_a`), (2) cross-calls `r/zerobase_b.AllocAndStore` which does `Stored = new(struct{})` (shared HIV reused, persisted under `r/zerobase_b`'s state). On the new HEAD: `"pa==pb=true"` — pointers from two different realms compare equal. The tx's storage `EVENTS` charge both realms (516 bytes to `r/zerobase_a`, 139 to `r/zerobase_b`); a cleaner attribution signal would require separating realm-init from cache-exercise into two txs, but the shared identity is the necessary precondition.

  Minimal fix: skip `stampPkgID` inside `GetZerobase` so `assignNewObjectID`'s `PkgID.IsZero()` branch adopts the HIV into the persisting realm — same code path used for stdlib-block-allocated heap items. A more invasive fix (sentinel HIV per `TypeID` with a stable, well-known `ObjectID` baked into the realm) retires both warnings cleanly: same identity across txs *and* no per-realm ownership question.
  </details>

A targeted fix for one warning does not automatically fix the other: skipping `stampPkgID` retires only W2; scoping the cache to "skip when escapable" retires W2 incidentally but weakens the within-tx guarantee for W1. The sentinel-per-`TypeID` design is the only path that fixes both.

## Nits

- [`ptr12.gno:16-23`](../../../../../.worktrees/gno-review-5708/gnovm/tests/files/ptr12.gno#L16-L23) — the `if &x != &y { ... }` block has only commented-out body. The predicate is evaluated and the body is empty either way. A plain comment, or just deleting the empty `if`, reads more clearly. Pre-existing.
- [`op_expressions.go:211-226`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/op_expressions.go#L211-L226) (unaddressed) — for `&struct{}{}`, `PopAsPointer2` allocates a fresh HIV at [`machine.go:2767`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/machine.go#L2767) (allocator-charged) before `doOpRef` discards it for the shared zerobase. Two `AllocateHeapItem` charges where one suffices.
- [`uverse.go:1168-1191`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/uverse.go#L1168-L1191) (unaddressed) — the two branches of `defNative("new", ...)` duplicate the `PushValue(TypedValue{T: ..., V: ...})` shape; hoisting the `PointerType` construction would cut ~6 lines.

## Missing Tests

- No filetest in the PR rounds a zero-sized pointer through realm save/load and asserts equality against a fresh `new(T)` in a follow-up tx — the doc hedge is the only thing documenting the within-machine scoping. [`tests/zerobase_cross_realm.txtar`](tests/zerobase_cross_realm.txtar) (this review) demonstrates the shape; adopting it (or a variant) into the PR would lock in whichever side of W1 the author commits to.
- No test pinning cross-realm cache sharing; same txtar covers it.

## Questions for Author

- Is the within-machine scoping intentional, or an artifact of the per-`Machine` cache choice? Round-1 question still open. The new doc says "within a machine execution" but stops short of confirming intent.
- Was cross-realm storage-rent attribution considered? The minimal fix is small (skip `stampPkgID` in `GetZerobase`); a structural sentinel-per-`TypeID` fix would retire both warnings at once.
