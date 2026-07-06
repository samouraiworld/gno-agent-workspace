# PR #5739: fix(gnovm): embedded type identity — struct field name + interface method-set flatten

URL: https://github.com/gnolang/gno/pull/5739
Author: ltzmaxwell | Base: master | Files: 29 | +1007 -92
Reviewed by: davd-gzl | Model: claude-opus-4-8 (xhigh) | Commit: `a00dde6b3` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5739 a00dde6b3`

Round 4 (head advanced `a3b14e207` → `a00dde6b3`, PR content changed). Delta since round 3: the legacy-persisted-interface story flipped from "serve unflattened state through recursion branches" to "hard-error on it" ([`880040d29`](https://github.com/gnolang/gno/commit/880040d29)), the unexported-selection gate in `FindEmbeddedFieldType` now keys on the method's origin package instead of the enclosing interface ([`0c7c6d322`](https://github.com/gnolang/gno/commit/0c7c6d322)), the hard-error check was hoisted to the `fillType` decode boundary with the interior sites `debugAssert`-gated, and new filetests pin cross-package unexported selection, builtin-alias embeds, and the runtime `doOpStructType` naming path. Re-review focused on those four things. Both round-3 open questions are now closed by the code.

**TL;DR:** When you embed a type by its name inside `struct{...}`, that name becomes the field's name; for `interface{...}`, embedding contributes only the embedded type's methods. This PR makes struct embeds keep the name exactly as written (so a type alias is its own field, distinct from the aliased type) and makes interface embeds collapse to their flattened method set (so `interface{ Stringer }` equals `interface{ Str() string }`), both matching the Go compiler. A per-method origin-package field keeps an unexported method from one package distinct from a same-named one elsewhere, so a type can't satisfy or select another package's sealed interface.

**Verdict: APPROVE** — the delta is a soundness fix (origin-package selection gate) plus a cleaner failure model for unsupported legacy state, both well-tested; CI is fully green. Reverting the selection gate reintroduces cross-package sealed-method access, and the hard-error path is anchored at the one decode boundary every stored type passes through.

## Summary
Round 3 gated unexported-method *selection* in [`FindEmbeddedFieldType`](https://github.com/gnolang/gno/blob/a00dde6b3/gnovm/pkg/gnolang/types.go#L1057) · [↗](../../../../../.worktrees/gno-review-5739/gnovm/pkg/gnolang/types.go#L1057) against the enclosing interface's package, while satisfaction (`VerifyImplementedBy`) already gated against the method's origin package. After flattening hoists a cross-package unexported method into a locally-spelled interface, those two disagree: a `main`-spelled `interface{ ifaceext.Sec }` would let `main` select `sec`. Round 4 aligns selection with satisfaction, gating both on [`im.originPkg(it.PkgPath)`](https://github.com/gnolang/gno/blob/a00dde6b3/gnovm/pkg/gnolang/types.go#L1071) · [↗](../../../../../.worktrees/gno-review-5739/gnovm/pkg/gnolang/types.go#L1071).

Round 3 kept embedded-interface branches in `FindEmbeddedFieldType`/`VerifyImplementedBy` to serve unflattened interfaces decoded from pre-flattening persisted state. Round 4 drops that: construction always flattens, so an `InterfaceKind` entry in `Methods` can only be pre-flattening state, whose identity already moved with flattening and is therefore unsupported. It now hard-errors via [`panicUnflattened`](https://github.com/gnolang/gno/blob/a00dde6b3/gnovm/pkg/gnolang/types.go#L1088) · [↗](../../../../../.worktrees/gno-review-5739/gnovm/pkg/gnolang/types.go#L1088). The one ungated production check sits at the decode boundary [`fillType`](https://github.com/gnolang/gno/blob/a00dde6b3/gnovm/pkg/gnolang/realm.go#L1786) · [↗](../../../../../.worktrees/gno-review-5739/gnovm/pkg/gnolang/realm.go#L1786), reached by every stored type via `GetType` ([`store.go:813`](https://github.com/gnolang/gno/blob/a00dde6b3/gnovm/pkg/gnolang/store.go#L813) · [↗](../../../../../.worktrees/gno-review-5739/gnovm/pkg/gnolang/store.go#L813)); the interior sites in `TypeID`, `FindEmbeddedFieldType`, and `VerifyImplementedBy` assert only under `-tags debugAssert`, so the hot paths stay branch-free in production. This is the `fix(gnovm)!` breaking change: old chain state holding unflattened interfaces requires regenesis or a re-flatten migration, as the ADR sequences.

## Glossary
- TypeID: a type's canonical string identity; type equality and persisted on-chain state both key off it. For interfaces it folds in each method entry.
- flatten (interface): replacing an embedded-interface entry with its individual methods, so identity is the method set, not the embed name.
- sealed interface: an interface containing an unexported method, satisfiable only from the method's defining package.
- stamp: setting `FieldType.PkgPath` on a method to record its origin package, used when the method has been hoisted out of another package's interface.
- decode boundary: `fillType`, which resolves stored type refs on load; every persisted type passes through it once.
- filetest: a VM-run `.gno` file asserted against `// Output:` / `// Error:` golden directives.

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
None.

## Missing Tests
None. The delta lands the coverage for its own changes: [`iface_embed_xpkg_access.gno`](https://github.com/gnolang/gno/blob/a00dde6b3/gnovm/tests/files/iface_embed_xpkg_access.gno) · [↗](../../../../../.worktrees/gno-review-5739/gnovm/tests/files/iface_embed_xpkg_access.gno) pins the origin-package selection gate with paired VM `// Error:` and go/types `// TypeCheckError:` assertions, the extended [`iface_embed_xpkg.gno`](https://github.com/gnolang/gno/blob/a00dde6b3/gnovm/tests/files/iface_embed_xpkg.gno) · [↗](../../../../../.worktrees/gno-review-5739/gnovm/tests/files/iface_embed_xpkg.gno) covers same-name coexistence and the sealed-bypass rejection, [`TestInterfaceType_UnflattenedIsHardError`](https://github.com/gnolang/gno/blob/a00dde6b3/gnovm/pkg/gnolang/types_test.go#L162) · [↗](../../../../../.worktrees/gno-review-5739/gnovm/pkg/gnolang/types_test.go#L162) pins the hard-error, and [`TestDoOpStructType_EmbedNames`](https://github.com/gnolang/gno/blob/a00dde6b3/gnovm/pkg/gnolang/types_test.go#L244) · [↗](../../../../../.worktrees/gno-review-5739/gnovm/pkg/gnolang/types_test.go#L244) plus [`alias7.gno`](https://github.com/gnolang/gno/blob/a00dde6b3/gnovm/tests/files/alias7.gno) · [↗](../../../../../.worktrees/gno-review-5739/gnovm/tests/files/alias7.gno) drive the runtime struct-construction op, and [`alias5.gno`](https://github.com/gnolang/gno/blob/a00dde6b3/gnovm/tests/files/alias5.gno) · [↗](../../../../../.worktrees/gno-review-5739/gnovm/tests/files/alias5.gno) pins builtin-alias embeds (`rune`/`byte`) distinct from `int32`/`uint8`.

## Suggestions
None.

## Verified
- The origin-package selection gate is load-bearing. Reverting [`types.go:1071`](https://github.com/gnolang/gno/blob/a00dde6b3/gnovm/pkg/gnolang/types.go#L1071) · [↗](../../../../../.worktrees/gno-review-5739/gnovm/pkg/gnolang/types.go#L1071) to the round-3 enclosing-package form (`it.PkgPath != callerPath`) makes `iface_embed_xpkg_access` and `iface_embed_xpkg` fail: `main` selects `ifaceext`'s unexported `sec` through a locally-spelled interface. See repro below.
- `fillType` is the sole ungated production choke and every stored type reaches it. `GetType` calls `fillType(ds, tt)` on every return at [`store.go:813`](https://github.com/gnolang/gno/blob/a00dde6b3/gnovm/pkg/gnolang/store.go#L813) · [↗](../../../../../.worktrees/gno-review-5739/gnovm/pkg/gnolang/store.go#L813) (the amino cache holds unfilled copies, so the fill runs on every load, not once), and object loads reach it through `fillTypesOfValue`. A pre-flattening interface therefore panics on decode, before `TypeID` can emit a nonsense identity.
- Green at `a00dde6b3`: `TestInterfaceType_UnflattenedIsHardError`, `TestDoOpStructType_EmbedNames`, `TestFlattenInterfaceMethods`, and the filetests `alias5`, `alias7`, `iface_embed_xpkg`, `iface_embed_xpkg_access`, `iface_embed_conflict`, `iface_embed_provenance`, `iface_embed_id`, `iface_embed_more`, `alias2`, `alias4`, `alias6` all pass. CI: 86 pass, 4 skipping, 0 fail.

<details><summary>repro: origin-package selection gate is load-bearing</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5739 -R gnolang/gno
# revert the round-4 gate to the round-3 enclosing-package form
perl -0pi -e 's/if !isUpper\(string\(n\)\) && im\.originPkg\(it\.PkgPath\) != callerPath \{/if !isUpper(string(n)) && it.PkgPath != callerPath {/' gnovm/pkg/gnolang/types.go
go test -run 'TestFiles/(iface_embed_xpkg_access|iface_embed_xpkg)\.gno$' ./gnovm/pkg/gnolang/ 2>&1 | tail -3
git checkout gnovm/pkg/gnolang/types.go
```

```
--- FAIL: TestFiles (...)
    --- FAIL: TestFiles/iface_embed_xpkg_access.gno (...)
FAIL
```
</details>

## Open questions
None outstanding. Both round-3 open questions are resolved in this delta:
- Legacy persisted unflattened interfaces: no longer served by recursion branches. They hard-error at the `fillType` decode boundary and require regenesis or a re-flatten migration, the documented consensus break the `fix(gnovm)!` commit and ADR sequence.
- Flatten conflict reachability: unchanged from round 3, still confirmed reachable and surfaced as a positioned `PreprocessError` (`iface_embed_conflict.gno` `// Error:`), not an uncatchable crash.
