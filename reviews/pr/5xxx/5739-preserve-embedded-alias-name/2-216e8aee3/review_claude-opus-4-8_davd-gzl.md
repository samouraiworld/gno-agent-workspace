# PR #5739: fix(gnovm): preserve embedded alias name as struct field name

URL: https://github.com/gnolang/gno/pull/5739
Author: ltzmaxwell | Base: master | Files: 24 | +801 -86
Reviewed by: davd-gzl | Model: claude-opus-4-8 (xhigh) | Commit: `216e8aee3` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5739 216e8aee3`

Round 2 (head advanced `155f1a7` → `216e8aee3`, PR content changed). The round-1 blocker is resolved: the approach shifted from naming embeds to flattening interface method sets, so embedded-interface identity no longer carries the embed/alias spelling. Re-review focused on the delta.

**TL;DR:** When you embed a type by its name inside `struct{...}`, that name becomes the field's name; for `interface{...}`, embedding contributes only the embedded type's methods. This PR makes struct embeds keep the name exactly as written (so a type alias is its own field, distinct from the aliased type) and makes interface embeds collapse to their flattened method set (so `interface{ Stringer }` equals `interface{ Str() string }`), both matching the Go compiler. A new per-method origin-package field keeps unexported methods from one package distinct from same-named ones elsewhere, so a type can't satisfy another package's sealed interface.

**Verdict: APPROVE** — embedded-field naming now matches Go across struct spelling, interface flattening, alias, multi-level, diamond, order, and cross-package exported/unexported method sets; the round-1 interface-alias split is gone and thehowl's provenance/panic/fast-path items are addressed. Only open item is a red `main / build`: `alias2.gno` is missing one `gno fmt` blank line; a routine reformat, not a code defect.

## Summary
A Gno anonymous interface derived its TypeID from a method list in which an embedded interface was a single named entry, so the embed/alias spelling leaked into identity and `interface{ Stringer }` diverged from `interface{ Str() string }`. The PR flattens embedded interfaces into their constituent methods at construction ([`flattenInterfaceMethods`](https://github.com/gnolang/gno/blob/216e8aee3/gnovm/pkg/gnolang/types.go#L2717) · [↗](../../../../../.worktrees/gno-review-5739/gnovm/pkg/gnolang/types.go#L2717), called from `doOpInterfaceType` and `staticTypeFromAST`), so identity is the method set, matching Go. Flattening exposed a latent gap: an unexported method's Go identity is `(pkgpath, name)`, but Gno encoded the package once on the enclosing interface, so hoisting a method out of another package re-qualified it, over-collapsing distinct interfaces and letting a local type satisfy another package's sealed interface. The fix records each method's origin package in a new `FieldType.PkgPath` (amino field 5, empty-guarded) and gates sort, TypeID emission, and satisfaction on it. Struct embeds are untouched and keep the written spelling. All consensus-breaking: anonymous-interface TypeIDs change and `FieldType` gains a serialized field; the PR flags a chain upgrade.

## Examples
How an embed is named/identified under this PR, all matching the Go compiler:

| Written embed | Result | Matches Go |
|---|---|---|
| `type Int = int; struct{ Int }` | field `Int`, distinct from `struct{ int }` | yes |
| `struct{ SAlias }` (`type SAlias = Stringer`) | field `SAlias`, distinct from `struct{ Stringer }` | yes |
| `interface{ SAlias }` | == `interface{ Stringer }` (method set) | yes |
| `interface{ Stringer }` | == `interface{ Str() string }` | yes |
| `interface{ p.Sec }` (unexported `p.sec`) | distinct from `interface{ sec() int }` in `q`; `q`-local type can't satisfy it | yes |

## Glossary
- TypeID: a type's canonical string identity; type equality and persisted on-chain state both key off it. For interfaces it folds in each method entry.
- flatten (interface): replacing an embedded-interface entry with its individual methods, so identity is the method set, not the embed name.
- sealed interface: an interface containing an unexported method, satisfiable only from the method's defining package.
- filetest: a VM-run `.gno` file asserted against `// Output:` golden directives.
- amino: the binary/JSON serialization used for persisted VM state.

## Fix
Round 1 named embedded interfaces from the resolved type (equalized alias-vs-target only). Round 2 replaces that: `staticTypeFromAST` and `doOpInterfaceType` build methods with `embed=false`, then call `flattenInterfaceMethods` to expand embeds and drop the name from identity, at [`preprocess.go:4139-4146`](https://github.com/gnolang/gno/blob/216e8aee3/gnovm/pkg/gnolang/preprocess.go#L4139-L4146) · [↗](../../../../../.worktrees/gno-review-5739/gnovm/pkg/gnolang/preprocess.go#L4139) and [`op_types.go:147-149`](https://github.com/gnolang/gno/blob/216e8aee3/gnovm/pkg/gnolang/op_types.go#L147-L149) · [↗](../../../../../.worktrees/gno-review-5739/gnovm/pkg/gnolang/op_types.go#L147). Unexported-method identity is carried per-method in `FieldType.PkgPath` and consumed by [`idName`](https://github.com/gnolang/gno/blob/216e8aee3/gnovm/pkg/gnolang/types.go#L418-L427) · [↗](../../../../../.worktrees/gno-review-5739/gnovm/pkg/gnolang/types.go#L418) (sort + TypeID emission key) and [`VerifyImplementedBy`](https://github.com/gnolang/gno/blob/216e8aee3/gnovm/pkg/gnolang/types.go#L1106-L1114) · [↗](../../../../../.worktrees/gno-review-5739/gnovm/pkg/gnolang/types.go#L1106) (satisfaction gating). The load-bearing constraint: the sort key and the emitted id use the same `idName(pkgPath)` fallback, so a method whose `PkgPath` is stamped (fresh construction) and one left empty (legacy-decoded or direct-declared) sort identically and yield one TypeID.

## Critical (must fix)
None.

## Warnings (should fix)
None.

The round-1 Warning (interface-alias embed split) is resolved: embedding an aliased interface now flattens to the same method set as embedding its target, so `interface{ SAlias } == interface{ Stringer }` again. Verified at `216e8aee3` against a side-by-side Go run; the round-1 repro now passes.

## Nits
- [`gnovm/tests/files/alias2.gno:30-31`](https://github.com/gnolang/gno/blob/216e8aee3/gnovm/tests/files/alias2.gno#L30-L31) · [↗](../../../../../.worktrees/gno-review-5739/gnovm/tests/files/alias2.gno#L30) — missing blank line between `func mustNot()` and `const badMsg`; `gno fmt` inserts it, and the missing line is what fails `main / build`. Confirmed behaviorally: `gofmt -d` on the file at `216e8aee3` shows exactly this one-line insertion, and it is the only PR `.gno` file `gofmt -l` flags.

## Missing Tests
None. Coverage is thorough: `alias2-4` (struct spelling: unqualified, selector, pointer, alias-to-defined), `alias5` (interface-alias flatten), `alias6` (struct embedding an interface alias keeps spelling), `iface_embed_id` (embed-vs-explicit, alias-vs-target, diamond), `iface_embed_more` (multi-level, order, mixed, superset, empty, dispatch), `iface_embed_xpkg` (cross-package exported flatten + unexported sealed-interface guard), plus `types_test.go` unit tests for flatten origin-pkgpath, diamond dedup, same-name-different-package coexistence, the conflict panic, and the stamped-vs-empty provenance TypeID stability.

## Suggestions
None.

## Open questions
- Legacy persisted interfaces that embed another interface keep their unflattened representation, so their TypeID differs from a freshly-flattened identical interface; the embedded-interface branches in `FindEmbeddedFieldType`/`VerifyImplementedBy` exist only to serve that decoded state. This is the documented consensus break and is gated behind the required chain upgrade. Not posted: author documented it in the PR body and ADR.
- The flatten conflict `panic` at [`types.go:2744`](https://github.com/gnolang/gno/blob/216e8aee3/gnovm/pkg/gnolang/types.go#L2744) · [↗](../../../../../.worktrees/gno-review-5739/gnovm/pkg/gnolang/types.go#L2744) is a bare Go panic, but a same-name/conflicting-type embed (`interface{ A; B }` with `M() int` vs `M() string`) is rejected by the gno typechecker as `duplicate method M` before flatten runs, so it is a should-not-happen guard, not a user-reachable uncatchable crash. Not posted: no valid trigger; matches the code comment.
