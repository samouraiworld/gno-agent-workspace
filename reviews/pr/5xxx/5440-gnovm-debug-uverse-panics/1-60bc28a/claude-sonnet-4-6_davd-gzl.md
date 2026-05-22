# PR #5440: fix(gnovm): fix debug mode panics during uverse initialization

**URL:** https://github.com/gnolang/gno/pull/5440
**Author:** omarsy | **Base:** master | **Files:** 4 | **+110 -10**
**Reviewed by:** davd-gzl | **Model:** claude-sonnet-4-6

## Summary

The PR fixes three distinct panics that occur when running GnoVM tests with `-tags debug`. All three bugs exist only in debug-gated code paths and do not affect production builds. The changes span `nodes.go`, `preprocess.go`, and `uverse.go`, with an ADR included.

**Bug 1 — `GetLocalIndex` nil dereference (`nodes.go:1931`)**
During uverse initialization, `UverseNode()` can be called re-entrantly (when `uverseInit == uverseInitializing`) and returns an empty `&PackageNode{}` stub. This stub's `StaticBlock.Block.Source` is nil. The debug-mode logging in `GetLocalIndex` calls `reflect.TypeOf(sb.Source).String()`, which panics on a nil interface. Fix: guard on `sb.Source == nil`, print an alternative log line, and assert via `sb.Location.PkgPath` that nil Source only appears for uverse.

**Bug 2 — `Define2` calls `TypeID()` on generic types (`nodes.go:2306`)**
`DefineNative` calls `Preprocess` on uverse native functions like `cap(x <X>{})`. During preprocessing, `initStaticBlocks` defines parameter names, then the `TRANS_BLOCK *FuncDecl` case redefines them with actual types. The re-definition path in `Define2` previously called `tv.T.TypeID()` unconditionally for a type-identity check, but `InterfaceType.TypeID()` panics when `Generic != ""` (generic types are uverse-only placeholders with no stable TypeID). Fix: `isGeneric(tv.T) || isGeneric(old.T)` skips the TypeID comparison, with a debug assertion verifying that any generic type is anchored to the uverse package path.

**Bug 3 — `DelAttribute` on uninitialized attribute map (`preprocess.go:661`)**
`Preprocess`'s deferred cleanup walks every node and calls `DelAttribute(ATTR_PREPROCESS_SKIPPED)` and `DelAttribute(ATTR_PREPROCESS_INCOMPLETE)`. `DelAttribute` has a debug assertion: `if debug && attr.data == nil { panic(...) }`. Because most nodes never have either attribute set, `attr.data` is nil, triggering the assertion. Fix: guard both calls with `HasAttribute(...)` checks — the assertion in `DelAttribute` is preserved, callers are now correct.

**Supporting change — `UverseNode()` stub gets a location**
The empty `&PackageNode{}` stub now has `SetLocation(PackageNodeLocation(uversePkgPath))` called on it. This enables the debug assertions added for bugs 1 and 2 to identify the stub as belonging to uverse via `sb.Location.PkgPath`.

All three fixes are strictly additive to debug builds; non-debug (production) code paths are unchanged.

## Test Results

- **Existing tests (no debug tag):** PASS for `gnovm/pkg/gnolang/...` (excluding pre-existing timeout on `TestFiles` and pre-existing `TestComputeMapKey` failure unrelated to this PR — both reproduce identically on the merge base `4dc22a442827`).
- **Existing tests (with `-tags debug`):** The three original panic-on-init failures are fixed; `TestComputeMapKey` still fails under `-tags debug` but also fails on the base branch — confirmed pre-existing and unrelated.
- **Edge-case tests:** skipped (no new test files written; see Missing Tests below).

## Critical (must fix)

None

## Warnings (should fix)

- [ ] `gnovm/pkg/gnolang/nodes.go:2326` — After `isGeneric` short-circuits the type comparison, the value check `if tv.V != old.V { panic(...) }` still executes. This is fine in practice because generic-typed parameters in uverse functions have nil values, but the invariant is implicit. If a future caller sets a non-nil `V` on a generic-typed `TypedValue` (e.g., a default value for a generic param), it will hit a false-positive panic with no explanation. A comment clarifying why `tv.V` and `old.V` are expected to be nil for generic params would prevent confusion.

- [ ] `gnovm/adr/prxxxx_fix_debug_mode_panics.md` — The file name uses `prxxxx_` despite the PR number being known (#5440). Per `AGENTS.md`, `prxxxx_` is for unknown PR numbers. After merge this should be `pr5440_fix_debug_mode_panics.md`. Low priority but worth a rename before merge.

## Nits

- [ ] `gnovm/pkg/gnolang/nodes.go:2307` — Comment says `cap(x <X>{})` as the example but `cap`'s generic param is `"X"`, not `"X{}"`. The curly braces are from the `InterfaceTypeExpr` syntax, not the type's string form. Consider using `cap(x <X>)` for clarity, or removing the parenthetical example entirely.

- [ ] `gnovm/pkg/gnolang/nodes.go:1935-1943` — The nil-Source debug branch checks `sb.Location.PkgPath != uversePkgPath` and panics if not uverse, but the panic message `"nil Source outside uverse"` gives no context (no location, no block pointer). Matching the style of the existing `"StaticBlock.Define2(%s) cannot change .T"` panics with at least `sb.Location` included would help.

## Missing Tests

- [ ] No test for the fixed `GetLocalIndex` nil-Source code path: there is no test that constructs a `StaticBlock` with `Source == nil` and `Location.PkgPath == uversePkgPath` and calls `GetLocalIndex` under the debug build tag. The fix is exercised implicitly when any test triggers uverse initialization with `-tags debug`, but an explicit unit test for `GetLocalIndex` with a nil-Source block would pin the invariant.

- [ ] No regression test for `Define2` with generic types under `-tags debug`. Since `TestComputeMapKey` already fails under `-tags debug` for an unrelated reason, the overall debug test run cannot serve as a regression guard. A targeted test — e.g., `TestDefine2GenericType` that calls `DefineNative` on a minimal `PackageNode` with a generic param and verifies no panic — would prevent regression.

- [ ] No test for the `DelAttribute` guard: there is no test that creates a node without setting any attributes and then calls `Preprocess` on it under `-tags debug`. Given the deferred cleanup in `Preprocess` visits every node, a small file-test that preprocesses a minimal function would cover this path.

## Suggestions

- The three fixes address symptoms (debug assertions in callers) rather than the underlying awkwardness: `UverseNode()` returning an un-initialized stub during reentrant calls. The stub bypasses `InitStaticBlock`, so it has no `Names` slice, no `Types`, and no `Block.Source`. While the current fix is minimal and correct, a future refactor could make the reentrant path return a sentinel (e.g., a package-level `uverseStub` pre-initialized with the location) rather than allocating a fresh `&PackageNode{}` each time. This would make the invariant explicit without relying on the caller to check `Location.PkgPath`. (`gnovm/pkg/gnolang/uverse.go:259`)

- The `isGeneric` guard in `Define2` silently allows redefining a name's type when either type is generic, including the case where `old.T` is generic but `tv.T` is not. In production code this is unreachable, but the guard is asymmetric: if somehow a non-generic `tv.T` replaces a generic `old.T`, it skips the TypeID check entirely. Changing the condition to `isGeneric(tv.T) && isGeneric(old.T)` (both must be generic, not just one) would be stricter and still cover the actual bug. (`gnovm/pkg/gnolang/nodes.go:2306`)

## Questions for Author

- Can `UverseNode()` be called re-entrantly outside of the `makeUverseNode()` path (i.e., from user code that imports gnolang)? If so, the stub is returned each time with a freshly allocated `PackageNode`, which callers may hold references to after initialization completes. Is that safe?

- The `isGeneric(old.T)` branch of the condition: when does `old.T` become generic while `tv.T` is not? Under what scenario would we re-define a name whose existing type is generic with a non-generic type? If this is unreachable, `isGeneric(tv.T) && isGeneric(old.T)` would be the safer condition.

- `TestComputeMapKey` panics under `-tags debug` on both the base branch and this branch. Is that a known pre-existing bug? It seems unrelated to this PR but blocks running the full debug test suite cleanly.

## Verdict

APPROVE — The fixes are correct, minimal, and well-reasoned. All three panics are in debug-only code paths, the production path is untouched, the debug assertions are preserved (callers are fixed, not assertions weakened), and the ADR documents the rationale clearly. The warnings and suggestions are minor polish items that can be addressed in follow-up.
