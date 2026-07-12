# PR [#5721](https://github.com/gnolang/gno/pull/5721): fix(gnovm): shallowest-match wins for embedded field/method lookup (spec-compliant BFS)

URL: https://github.com/gnolang/gno/pull/5721
Author: ltzmaxwell | Base: master | Files: 13 | +660 -218
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 139e7d5e0 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5721 139e7d5e0`

Round 2. Head advanced 16d5227b9 → 139e7d5e0 (+6 PR commits over a master merge). PR code is byte-identical to round 1; only test files moved: `method47` renamed to `method43`, new `method45a` positive companion added, `method40` gains `bug485.go` / Go CL 94100045 provenance, empty `// Output:` directives dropped from `method40`/`method42`. All three prior findings carry unchanged (their anchors are untouched code); verdict unchanged.

**TL;DR:** Gno decides `x.f` on an embedded type by depth-first field order, so it can pick a deeper match, silently ignore a same-depth clash, or wrongly report a reachable name as missing. This rewrites the lookup so the shallowest match wins and a tie at that depth is an error, matching Go.

**Verdict: APPROVE** — spec-compliant rewrite; every Go-parity case checked matches the Go compiler; only minor diagnostic/test-coverage and comment-staleness follow-ups, no blockers.

## Summary
`findEmbeddedFieldType` resolves promoted field/method selectors in the GnoVM. The old code was a per-type depth-first search whose field-order decided winners, whose subtree conflicts collapsed to "not found", and whose shared recursion-guard could prune a type's shallower promotion (yielding a false "does not implement" that Go accepts, then a preprocess panic). The rewrite is a two-phase breadth-first walk: phase 1 (`lookupShallowestEmbedded`) walks the embedding graph level by level with a global visited set, returning the unique shallowest provider or ambiguity; phase 2 (`buildEmbeddedTrail`) re-walks only the winning path to build the `ValuePath` trail. Result: shallowest-depth match wins, a same-depth tie (two providers, a diamond to one type, or a method-vs-field clash) is `ambiguous selector`, and a deeper match can no longer rescue a shallower ambiguity. The walk is O(reachable types) via the global seen-set, bounded by `MaxEmbedDepth`=8 and `MaxStructFields`=128.

## Examples
Selector on the embedded graph, old gno vs new gno vs Go (`c.Foo` / `b.val()`):

| Shape | Old gno | New gno | Go |
|---|---|---|---|
| direct field at depth 1 over same name at depth 2 (`method40`) | field-order dependent | shallowest wins | shallowest wins |
| two siblings, same name, same depth (`method41`) | missing field | ambiguous | ambiguous |
| depth-1 field over depth-2 (`method42`) | field-order dependent | depth-1 wins | depth-1 wins |
| ambiguous at depth 2, unique match at depth 3 (`method45`) | resolves to depth 3 | ambiguous | ambiguous |
| ambiguous `o.Foo` does not poison unique deeper `o.Inner3` (`method45a`) | n/a | resolves | resolves |
| diamond: one type via two same-depth paths (`method46`) | resolves | ambiguous | ambiguous |
| promoted method vs promoted field, same depth (`method43`) | field-order dependent | ambiguous | ambiguous |

## Glossary
- promoted field/method: a field/method of an embedded type reachable on the embedding struct; Go's §Selectors resolves to the shallowest-depth provider, ties are ambiguous.
- selector: an `x.f` expression the preprocessor resolves to a `ValuePath` after walking embedding.
- ValuePath: the resolved access step (kind/`Depth`/`Index`) a selector compiles to; a promoted match is a trail of these.
- type-check: go/types validation (`TypeCheckMemPackage`) run before preprocess; the gate that already rejects ambiguous selectors on concrete types at deploy.
- flatten (interface) / sealed interface / stamp (origin package): the cross-package interface method-set machinery this PR carries forward unchanged.

## Fix
Old lookup lived as per-type `FindEmbeddedFieldType` methods on `*StructType`/`*DeclaredType`/`*PointerType`, all deleted; the free `findEmbeddedFieldType` at [`types.go:2910`](https://github.com/gnolang/gno/blob/139e7d5e0/gnovm/pkg/gnolang/types.go#L2910) · [↗](../../../../../.worktrees/gno-review-5721/gnovm/pkg/gnolang/types.go#L2910) now drives the two phases and returns an `embedLookupStatus` enum instead of a bare `accessError bool`. `preprocess.go` maps that enum: `embedLookupAmbiguous` → `ambiguous selector` at [`preprocess.go:2529-2530`](https://github.com/gnolang/gno/blob/139e7d5e0/gnovm/pkg/gnolang/preprocess.go#L2529-L2530) · [↗](../../../../../.worktrees/gno-review-5721/gnovm/pkg/gnolang/preprocess.go#L2529), `embedLookupAccessError` → `cannot access`, else the prior `missing field`. Interface roots and embedded-interface nodes still delegate to the retained `InterfaceType.FindEmbeddedFieldType`, so flat interface method-set scans and the `originPkg` gating from #5739 are unchanged.

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
- **[stale comment names a deleted symbol]** [`types.go:2114`](https://github.com/gnolang/gno/blob/139e7d5e0/gnovm/pkg/gnolang/types.go#L2114) · [↗](../../../../../.worktrees/gno-review-5721/gnovm/pkg/gnolang/types.go#L2114) — the `GetValueAt` doc still says the preprocessor uses `*DT.FindEmbeddedFieldType()`, a method this PR deletes; same at [`uverse.go:1658`](https://github.com/gnolang/gno/blob/139e7d5e0/gnovm/pkg/gnolang/uverse.go#L1658) · [↗](../../../../../.worktrees/gno-review-5721/gnovm/pkg/gnolang/uverse.go#L1658). Both should name the free `findEmbeddedFieldType`.

## Missing Tests
- **[untested interface-satisfaction ambiguity path]** [`types.go:1026`](https://github.com/gnolang/gno/blob/139e7d5e0/gnovm/pkg/gnolang/types.go#L1026) · [↗](../../../../../.worktrees/gno-review-5721/gnovm/pkg/gnolang/types.go#L1026) — no test exercises `VerifyImplementedBy` when the concrete type's method is ambiguous.
  <details><summary>details</summary>

  The new tests all resolve selectors on values; none run `var i I = T{}` where `T`'s method is ambiguous. That path takes `findEmbeddedFieldType` through `VerifyImplementedBy`, which drops the status with `_` and treats `embedLookupAmbiguous` (trail `nil`) as not-found. The rejection is correct and matches Go: a type with an ambiguous method does not satisfy the interface. But the VM message is `main.C does not implement main.I (missing method Foo)` where go/types and this PR's own direct-selector path both say ambiguous. The author flagged this angle as uncovered. Fix: add the coverage filetest below (green today, guarding the correct rejection on this path).

  Ready to add, verified green at 139e7d5e0 — full file with exact goldens at [`tests/embed_iface_ambiguous.gno`](tests/embed_iface_ambiguous.gno):

  ```go
  package main

  type I interface{ Foo() string }
  type A struct{}
  func (A) Foo() string { return "a" }
  type B struct{}
  func (B) Foo() string { return "b" }
  type C struct {
  	A
  	B
  }
  func main() {
  	var i I = C{}
  	_ = i
  }

  // Error:
  // main/embed_iface_ambiguous.gno:<n>:6-15: main.C does not implement main.I (missing method Foo)
  ```
  </details>

## Suggestions
- **[diagnostic weaker than the direct-selector path]** [`types.go:1026`](https://github.com/gnolang/gno/blob/139e7d5e0/gnovm/pkg/gnolang/types.go#L1026) · [↗](../../../../../.worktrees/gno-review-5721/gnovm/pkg/gnolang/types.go#L1026) — `VerifyImplementedBy` now has the `embedLookupStatus` in hand but discards it; branching on `embedLookupAmbiguous` would let the interface-satisfaction error name ambiguity instead of "missing method", matching preprocess.go's improved selector message and go/types.
- **[redundant type-load call]** [`types.go:3194`](https://github.com/gnolang/gno/blob/139e7d5e0/gnovm/pkg/gnolang/types.go#L3194-L3204) · [↗](../../../../../.worktrees/gno-review-5721/gnovm/pkg/gnolang/types.go#L3194) — `buildEmbeddedTrail` calls `fv.GetType(nil)` twice (for `Params[0]` and `BoundType`) and repeats the two-line NOTE comment verbatim; the deleted `DeclaredType.FindEmbeddedFieldType` cached it once as `mt := fv.GetType(nil)`. Fold back to one call and one comment.

## Verified
- Go parity, every case matches the Go compiler (`go build`/`go run` of the equivalent Go program): the four ambiguous shipped cases (`method41/45/46/43`) and `method44` compile-error identically (`ambiguous selector` / `cannot refer to unexported field`); the valid ones (`method40` → `b`, `method42` → `direct`) print the same; the new `method45a` positive companion prints `deep`/`deep`/`m1` in both while its bare `o.Foo` is an ambiguous selector in both, confirming the ambiguity does not poison the unique deeper `o.Inner3` or the anchored `o.DeepBox.Foo` / `o.AmbBox.M1.Foo`. Five extra probes match too: method-over-field and field-over-method at different depths resolve to the shallower, a foreign gated name at depth 1 with a local same-name field at depth 2 resolves to the local, a shallow unique field shadows a deeper diamond, and a type reached at two depths resolves at the shallower. See [repro](comment_claude-opus-4-8.md).
- Behavior change is gated by type-check, so no deployed realm can regress: `AddPackage` runs `gno.TypeCheckMemPackage` (go/types) before the VM preprocesses ([`keeper.go:647`](https://github.com/gnolang/gno/blob/139e7d5e0/gno.land/pkg/sdk/vm/keeper.go#L647) · [↗](../../../../../.worktrees/gno-review-5721/gno.land/pkg/sdk/vm/keeper.go#L647)), and go/types already enforces the same shallowest-match rules on concrete types. The old VM was merely more permissive than the gate it sits behind (ambiguous selectors it resolved were already rejected at deploy) or more restrictive in the false-reject direction (go/types accepts, old VM panicked at preprocess), which this PR fixes.
- Reverting the fix reproduces the old behavior: with `method45` copied onto the merge-base, the VM resolves the depth-3 selector and emits no error, versus the new `ambiguous selector Foo in main.Outer`.
- Interface satisfaction with an ambiguous method rejects (does not implement) matching go/types; a coverage filetest is shipped under Missing Tests, verified green at 139e7d5e0.
- Green at 139e7d5e0: the eight method filetests (`method40/41/42/43/44/45/45a/46`); PR code is byte-identical to the round-1 head, and master did not touch `types.go`/`preprocess.go`/`uverse.go` between the two merge-bases, so the round-1 whole-package and integration runs still hold. `gofmt`/`go vet` clean on the changed files.

## Open questions
- Migration risk for third-party deployed realms: answered no, because the type-check gate above rejects any ambiguous selector at deploy, so a realm that reached the VM already passed the same rule; not posted, no action for the author.
- Runtime allocation: the dynamic interface path ([`values.go:1999`](https://github.com/gnolang/gno/blob/139e7d5e0/gnovm/pkg/gnolang/values.go#L1999) · [↗](../../../../../.worktrees/gno-review-5721/gnovm/pkg/gnolang/values.go#L1999)) allocates BFS state only for deeply-embedded lookups; depth-0 hits allocate nothing and the old code allocated a guard map too, so no gas-relevant change (Go allocations are unmetered). Not posted; noted for context.
