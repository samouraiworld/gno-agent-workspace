# PR #5739: fix(gnovm): preserve embedded alias name as struct field name

URL: https://github.com/gnolang/gno/pull/5739
Author: ltzmaxwell | Base: master | Files: 7 | +145 -51
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 155f1a7 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5739 155f1a7`

**TL;DR:** When you embed a type by its name inside a `struct{...}` or `interface{...}`, that name becomes the field's name. This PR makes that name come from exactly what was written rather than from the underlying type, so a type alias keeps its own spelling as the field name instead of resolving away to the aliased type.

**Verdict: REQUEST CHANGES.** Struct, pointer, selector, and rune/byte embed naming now matches Go, but embedding an aliased interface inside an interface (`type SAlias = Stringer; interface{ SAlias }`) splits a type that Go and current master both treat as identical to `interface{ Stringer }`; fix that sub-case before this consensus-breaking identity change lands.

## Summary
A Gno embedded field takes its name from the embedded type, and that name is part of the type's [TypeID](#glossary), so `struct{ Int }` (where `type Int = int`) must be distinct from `struct{ int }`. Master derived the name from the resolved type, so the alias collapsed to `int` and the two structs shared a TypeID, which is wrong and against Go. The fix derives the name from the source AST expression instead, threaded into all three construction paths, fixing struct/pointer/selector embeds plus the `rune`/`byte` cases (previously misnamed `int32`/`uint8`). The same rewrite, applied to an interface embedded in an interface, now also feeds the written alias name into the interface's TypeID, so `interface{ SAlias }` and `interface{ Stringer }` diverge where Go keeps them identical.

```
                 struct{ Int }     interface{ SAlias }
                 vs struct{ int }   vs interface{ Stringer }
  Go             distinct  ✓        identical ✓
  master (pre)   identical ✗        identical ✓
  PR 155f1a7     distinct  ✓        distinct  ✗  <- new divergence
```

## Examples
How the embedded field is named under this PR, all matching the Go compiler:

| Written embed | Field name | Type identity |
|---|---|---|
| `type Int = int; struct{ Int }` | `Int` | distinct from `struct{ int }` |
| `struct{ rune }` / `struct{ byte }` | `rune` / `byte` | distinct from `struct{ int32 }` / `struct{ uint8 }` |
| `struct{ pkg.Int }` (qualified) | `Int` | same as `struct{ Int }` |
| `struct{ *Int }` (pointer) | `Int` | distinct from `struct{ *int }` |
| `type C = T; struct{ C }` (alias to a named type) | `C` | distinct from `struct{ T }` |

## Glossary
- TypeID: a type's canonical string identity; type equality and persisted on-chain state both key off it. For structs/interfaces it folds in each field/method entry's name.
- preprocess: the static pass that resolves names and types before execution; one of the three paths (`buildFieldTypesAST`) that names embedded fields.
- filetest: a VM-run `.gno` file asserted against `// Output:` golden directives.

## Fix
Master's `fillEmbeddedName` read the field's resolved `ft.Type` (so an alias had already collapsed to its underlying type) and mapped it back to a name through a primitive-by-primitive switch. The PR passes the source expression alongside `ft` and unwraps its const and pointer layers down to the `NameExpr`/`SelectorExpr` that holds the name as written, at [`types.go:2640-2669`](https://github.com/gnolang/gno/blob/155f1a7/gnovm/pkg/gnolang/types.go#L2640-L2669) · [↗](../../../../../.worktrees/gno-review-5739/gnovm/pkg/gnolang/types.go#L2640). The three callers each thread their per-field AST expr: [`op_types.go:121`](https://github.com/gnolang/gno/blob/155f1a7/gnovm/pkg/gnolang/op_types.go#L121) · [↗](../../../../../.worktrees/gno-review-5739/gnovm/pkg/gnolang/op_types.go#L121) (struct), [`op_types.go:145`](https://github.com/gnolang/gno/blob/155f1a7/gnovm/pkg/gnolang/op_types.go#L145) · [↗](../../../../../.worktrees/gno-review-5739/gnovm/pkg/gnolang/op_types.go#L145) (interface), [`preprocess.go:4167`](https://github.com/gnolang/gno/blob/155f1a7/gnovm/pkg/gnolang/preprocess.go#L4167) · [↗](../../../../../.worktrees/gno-review-5739/gnovm/pkg/gnolang/preprocess.go#L4167) (const-type build). The load-bearing constraint: an embedded field's source is always a (possibly pointer/qualified) type name, so unwrapping always lands on a name; anything else hits the new panic.

## Critical (must fix)
None.

## Warnings (should fix)
- **[two interfaces Go calls identical now split apart]** [`gnovm/pkg/gnolang/types.go:2640-2669`](https://github.com/gnolang/gno/blob/155f1a7/gnovm/pkg/gnolang/types.go#L2640-L2669) · [↗](../../../../../.worktrees/gno-review-5739/gnovm/pkg/gnolang/types.go#L2640): embedding an aliased interface inside an interface derives its identity from the written name, so `interface{ SAlias }` no longer equals `interface{ Stringer }` (Go and master treat them as one type).
  <details><summary>details</summary>

  An interface embedded in an interface contributes only its method set; in Go its written name is not part of the resulting interface's identity, so `interface{ SAlias }` and `interface{ Stringer }` (with `type SAlias = Stringer`) are the same type. Gno keeps each embed as a method-list entry whose `Name` is folded into the interface TypeID at [`types.go:442-451`](https://github.com/gnolang/gno/blob/155f1a7/gnovm/pkg/gnolang/types.go#L442-L451) · [↗](../../../../../.worktrees/gno-review-5739/gnovm/pkg/gnolang/types.go#L442). Master derived that `Name` from the resolved type (`Stringer` either way) so the two matched; the PR derives it from the source spelling, so they now produce different TypeIDs. This is a regression: master agrees with Go here, 155f1a7 does not, and once shipped the divergent identity is baked into consensus. The struct direction is unaffected and correct: `struct{ SAlias }` vs `struct{ Stringer }` should differ (struct field names), and it does. Fix: for an embedded interface, derive identity from the resolved type, not the source spelling, so structurally identical interfaces keep one TypeID; the struct-embed renaming is correct and stays. Repro: [iface_alias_id.gno](tests/iface_alias_id.gno).
  </details>

## Nits
None.

## Missing Tests
- **[the regressing case has no coverage]** [`gnovm/tests/files/alias3.gno:41-46`](https://github.com/gnolang/gno/blob/155f1a7/gnovm/tests/files/alias3.gno#L41-L46) · [↗](../../../../../.worktrees/gno-review-5739/gnovm/tests/files/alias3.gno#L41): interface embeds are tested for method-set flattening and field access, never for type identity.
  <details><summary>details</summary>

  `alias3.gno` asserts that `T` implements `interface{ SAlias }` and that the embedded field is reachable, but never compares the *identity* of two interfaces that embed the same interface under different spellings. That is exactly the gap that hides the Warning above. A `// Output:` filetest pairing `interface{ SAlias } == interface{ Stringer }` (Go: true) with the struct counterpart (Go: distinct) would have caught it. See [iface_alias_id.gno](tests/iface_alias_id.gno) and its Go parity [iface_alias_id_go.go](tests/iface_alias_id_go.go).
  </details>

## Suggestions
None.

## Open questions
- The new bare `panic("cannot derive embedded name ...")` at [`types.go:2665-2666`](https://github.com/gnolang/gno/blob/155f1a7/gnovm/pkg/gnolang/types.go#L2665-L2666) · [↗](../../../../../.worktrees/gno-review-5739/gnovm/pkg/gnolang/types.go#L2665) is a Go panic, not a VM `Exception`, so it would escape `recover()` uncatchable if a user `.gno` program could reach it. I found no valid embed form that does (the full `TestFiles` corpus produces no new panic, and malformed embeds are rejected at type-check first), and master's `fillEmbeddedName` panicked the same way, so this is no regression. Flagging only so whoever revisits embed handling keeps the unreachability assumption in mind. Not posted: no demonstrated trigger.
- State compatibility is already called out in the PR description (persisted structs embedding aliases or `rune`/`byte` change TypeID, needs a chain upgrade). In-repo blast radius is nil: no package under `examples/` embeds a type alias or a bare `rune`/`byte`. Not posted: author already documented it.
