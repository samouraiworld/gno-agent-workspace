# PR [#5894](https://github.com/gnolang/gno/pull/5894): fix(gnovm): persist function-local declared types referenced by saved values

URL: https://github.com/gnolang/gno/pull/5894
Author: ltzmaxwell | Base: master | Files: 10 | +929 -4
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: d0bdb5049 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5894 d0bdb5049`

**TL;DR:** In gno, a type declared inside a function body (`type S struct{...}`) is created at runtime and never written to the type store. Storing a value of such a type in realm state saved a pointer to a type record that was never written, so after a node restart every read of that value crashed and the realm state was permanently unreadable. This PR walks each saved object and writes those function-local types on the save path, closing the hole.

**Verdict: APPROVE** — targeted fix at the single persistence choke point, backed by restart tests that genuinely reproduce on master and a save-time invariant that fails loudly if a future value shape reintroduces the hole. State and gas change, author-flagged as consensus-affecting.

## Summary
Package-level declared types are `SetType`'d at addpkg; function-local types are declared at runtime and were never persisted. Any saved `TypedValue` whose type is function-local serialized as `RefType{"pkg[loc].Name"}`, a dangling pointer into the type store `/t/`. The live process never noticed (the type sits in `cacheTypes`); after restart, loading the object hits `GetType` → miss → `panic("unexpected type with id ...")`. The fix adds a `localTypeSaver` walk in `saveObject` that `SetType`s every reachable function-local `DeclaredType` just before `SetObject`, makes `copyTypeWithRefs` preserve `ParentLoc` (part of the local type's TypeID, so the record lands under the ID values reference), and gates a `debugAssert` invariant in `SetObject` that panics on any dangling function-local ref in the persist-copy.

```
save path (FinalizeRealmTransaction → saveObject):

  before:  saveObject → SetObject → copyValueWithRefs mints RefType{pkg[loc].S}
                                     └─ no /t/<pkg[loc].S> record written  ✗ dangling
  after:   saveObject → localTypeSaver.saveObjectTypes(oo)  → SetType(S)   ✓ /t/ written
                      → SetObject → copyValueWithRefs mints RefType{pkg[loc].S} → resolves on reload
```

## Examples
| gno in a realm | before (after restart) | after |
|---|---|---|
| `type S struct{T}; X = S{...}` (interface var) | reload panics `unexpected type with id ...S` | reads back |
| `type S struct{T}; s:=S{...}; G = func(){ s.Get() }` (closure capture) | reload panics | reads back |
| `type K struct{k int}; m := map[K]string{...}` (local map key) | reload panics on lookup | key re-mints, lookup hits |
| `type S struct{T}` declared in a `maketx run` script, escaped to a realm | rejected already | still rejected: `cannot persist object of type defined in the private realm` |

## Glossary
- function-local type: `DeclaredType` declared in a function body; TypeID is `pkg[loc].Name`, not `SetType`'d at addpkg.
- RefType: placeholder a persist-copy stores for a type; `RefType{ID}` → `/t/<ID>` record, resolved on reload.
- persist-copy: ref-collapsed object copy (`copyValueWithRefs`) amino-marshaled on `SetObject`.
- addpkg: the `maketx addpkg` upload transaction, where package-level types are `SetType`'d.
- TypeID: canonical type identity; changing an already-persisted TypeID is consensus-breaking.

## Fix
`saveObject` gains one line before `SetObject`: `(&localTypeSaver{store: store}).saveObjectTypes(oo)` at [`realm.go:1079`](https://github.com/gnolang/gno/blob/d0bdb5049/gnovm/pkg/gnolang/realm.go#L1079) · [↗](../../../../../.worktrees/gno-review-5894/gnovm/pkg/gnolang/realm.go#L1079). The saver ([`realm.go:1105`](https://github.com/gnolang/gno/blob/d0bdb5049/gnovm/pkg/gnolang/realm.go#L1105) · [↗](../../../../../.worktrees/gno-review-5894/gnovm/pkg/gnolang/realm.go#L1105)) walks the object's typed slots, and for each `DeclaredType` with a non-zero `ParentLoc` calls `store.SetType` then recurses through `Base` under a per-object visited-guard for recursive types. `copyTypeWithRefs` now copies `ParentLoc` ([`realm.go:1716`](https://github.com/gnolang/gno/blob/d0bdb5049/gnovm/pkg/gnolang/realm.go#L1716) · [↗](../../../../../.worktrees/gno-review-5894/gnovm/pkg/gnolang/realm.go#L1716)) so the record's TypeID matches the referencing `RefType`. `IsFuncLocal` ([`types.go:2026`](https://github.com/gnolang/gno/blob/d0bdb5049/gnovm/pkg/gnolang/types.go#L2026) · [↗](../../../../../.worktrees/gno-review-5894/gnovm/pkg/gnolang/types.go#L2026)) is `!ParentLoc.IsZero()`, and `declareWith` sets `ParentLoc` non-zero exactly for `FuncDecl`/`FuncLitExpr` parents.

## Verification
Verified on d0bdb5049 (checks the test suite does not run):

- **Revert-proof:** the committed `restart_local_type.txtar` fails on master (dfe49509f) at the lt2 (interface-var) case with `unexpected type with id gno.land/r/test/lt2[gno.land/r/test/lt2/lt2.gno:11:1-14:2].S`, the permanently-unreadable-state panic; passes on the PR. The lt1 (bound-method) case does not fault on master, matching the ADR's eager-bind note.
- **Guard is effective:** neutralizing the `saveObjectTypes(oo)` call and running `zrealm_localtype0.gno` under `-tags debugAssert` (not in CI) panics `dangling function-local type ref gno.land/r/test[...].S in persisted value`; the clean PR passes the same run.
- **Choke point holds:** `store.SetObject` has one non-test caller repo-wide, [`realm.go:1086`](https://github.com/gnolang/gno/blob/d0bdb5049/gnovm/pkg/gnolang/realm.go#L1086) · [↗](../../../../../.worktrees/gno-review-5894/gnovm/pkg/gnolang/realm.go#L1086), immediately after the saver, so every persistence route is covered.
- **Empty containers:** a probe storing `map[string]LV{}`, `[]LV{}`, and a nil `*LV` for a function-local `LV` persists with no dangling ref under `-tags debugAssert` — the element type is reached through the slot's `tv.T`, not only through entries.

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
- [`realm.go:1225`](https://github.com/gnolang/gno/blob/d0bdb5049/gnovm/pkg/gnolang/realm.go#L1225) · [↗](../../../../../.worktrees/gno-review-5894/gnovm/pkg/gnolang/realm.go#L1225) — the production `saveType` switch omits `*ChanType`, so it would hit this `default` panic rather than walk the channel element. Unreachable today: gno's type-checker rejects channel types outright (`channels are not permitted`, confirmed by probing `type S struct{ c chan int }` in a realm), so no persisted value carries a `ChanType`. The `default`-panic is a deliberate fail-loud for unknown kinds, so this is consistent, just noted so a future channel-enabling change knows to add the `Elem()` walk here (and in the `debugAssert` walker, which benign-defaults on it).

## Missing Tests
None material. Coverage spans interface var, closure capture, bound-method value, cross-realm foreign owner, `p/`-package local type, local map key with TypeID stability, recursive type, primitive/byte base, local func type, nested types, TypeValue-only escape, closure-declared type, same-name-different-function, double-bind idempotency, and the negative `maketx run` guard.

## Suggestions
- [`realm.go:1079`](https://github.com/gnolang/gno/blob/d0bdb5049/gnovm/pkg/gnolang/realm.go#L1079) · [↗](../../../../../.worktrees/gno-review-5894/gnovm/pkg/gnolang/realm.go#L1079) — `&localTypeSaver{store: store}` is allocated per `saveObject` even when the object references no local type (the common case), and the walk runs unconditionally. The ADR reasons this is noise beside the amino marshal in `SetObject`, and the `visited` map stays nil unless a local type is found, so the per-save cost is one small struct plus a prune-early slot walk. No change needed; flagging only for whoever profiles the save path next.

## Open questions
- Type storage (`/t/` records, written via `SetType` → `baseStore.Set`) is charged encode gas but does not flow through the realm storage-deposit accounting (`sumDiff` tracks only `SetObject` deltas). This matches how package-level type records at addpkg have always been handled, so the new local-type records are consistent, not a regression. Not posted: no action, existing behavior.
