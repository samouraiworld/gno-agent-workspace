# PR #5577: fix(gnovm): reset package state between test functions

**URL:** https://github.com/gnolang/gno/pull/5577
**Author:** notJoon | **Base:** master | **Files:** 10 | **+357 -107**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4-7

## Summary

Round 2 review of a rebased PR at `dff0112a9`. Since round 1 (`7b18f676b`), the author added five commits that address the main blockers flagged previously: failing CI, missing realm-state isolation, and broken example tests that relied on shared state.

**What's new since round 1:**

1. **`77cb0f28b` — realm state isolation test** (`realm_isolation.txtar`): a new txtar test under `r/demo/counter` verifies that `TestCrossingMutates_A` sees init-state (count=7), mutates it via a crossing call to count=9, and `TestCrossingMutates_B` sees count=7 again. Directly exercises realm finalization isolation.
2. **`df8103d70` — fix TestDeleteMap**: collapses two order-dependent tests (`TestDeleteMap` + `TestGetMapAfterDelete`) into a single self-contained test with a precondition check.
3. **`84d6e92f4` — isolate imported-realm state across per-test transactions** (`test/test.go:426-427`): `runTestFiles` now takes a `baseStore storetypes.Store` param, `CacheWrap`s it into `innerBase` per test, and opens `innerTxn := tgs.BeginTransaction(innerBase, innerBase, nil, nil)` — so realm finalization writes (`SetObject → baseStore.Set`) hit only the per-test cache and are discarded when the txn is dropped. Closes the gap where the round-1 code's inner txn only isolated gno-store cache-objects but not realm-persisted state.
4. **`dff0112a9` — fold test-file init skip into `runInitFromUpdates`**: adds `skipTestFileInits` parameter to `runInitFromUpdates` and a new exported `RunMemPackageSkipTestFileInits`. Called once in `Test()` (test.go:263) to seed `tgs` without running *_test.gno inits (e.g. `gov/dao.InitWithUsers`), which would otherwise pollute imported-realm state shared across all per-test transactions.
5. **Non-PR fixups (`4620381c8 fix`, examples changes)**: `todolist_test`, `govdao_test`, `hof_test`, `evaluation_test`, `map_delete_test` were rewritten to not rely on shared state. The changes are idiomatic — each test now sets up its own fixtures via helper functions (e.g. `setupTodoList`, `registerItem`, `createMemberProposal`).

**How it works (current):**

- `Test()` runs the mempackage in a common `tgs` (outer transaction) with test-file inits skipped, so `tgs` holds only the preprocessed package plus production-file state.
- `runTestFiles()` uses `tgs.GetBlockNode(PackageNodeLocation(pkgPath))` to get the already-preprocessed `*PackageNode`. For each test function, it `CacheWrap`s the base store and begins an inner txn, constructs a machine on that txn, and calls `m.NewPackageInstance(pn)` to build a fresh `PackageValue` with its own file blocks, re-run var decls, and re-run init() funcs.
- Each test's mutations to package globals (via `pv.Block.Values`) and realm state (via `innerBase` writes) vanish when the test returns, because neither the inner txn nor `innerBase` is `Write()`-ed.

**CI status:** Per the PR page, all checks now pass (gnovm ×16, gnoland ×9, examples ×12, contribs ×69, plus meta/codeql/codecov). The two blockers from round 1 (`map_delete` failure, non-conventional PR title) are resolved.

## Test Results

- **Existing tests:** PASS
  - `go test -run Test_Scripts/test/init_and_isolation ./gnovm/cmd/gno/` → PASS (1.68s)
  - `go test -run Test_Scripts/test/issue_1982_increment ./gnovm/cmd/gno/` → PASS (1.82s)
  - `go test -run Test_Scripts/test/realm_isolation ./gnovm/cmd/gno/` → PASS (1.86s)
  - `go test -run TestFiles -test.short ./gnovm/pkg/gnolang/` → PASS (82s)
- **Edge-case tests:** skipped (core flows already covered by the three new txtars)

## Critical (must fix)

- [ ] `gnovm/pkg/gnolang/machine.go:785-794` — **Stale doc comment on `NewPackageInstance` is still there.** Round 1 flagged this as critical. The comment block immediately preceding `func (m *Machine) NewPackageInstance(pn *PackageNode) *PackageValue` begins `// instantiatePackageFiles performs runtime instantiation for the given / file nodes: ...` and closes with `// fns must already be present in pn.FileSet and fully preprocessed; ...`. That text describes `instantiatePackageFiles` (whose own correct doc is at 653-660), not `NewPackageInstance`. Readers opening go doc or IDE hover will see the wrong description for a new exported API. Rewrite to describe what `NewPackageInstance` does: take a fully-preprocessed `*PackageNode`, allocate a fresh `*PackageValue` via `pn.NewPackage`, wire it into the active store, create file blocks + re-run var decls via `instantiatePackageFiles`, then run init() funcs via `runInitFromUpdates`.

## Warnings (should fix)

- [ ] `gnovm/pkg/test/test.go:406, 429` — **Machine leak worsens per-test.** `Machine()` pulls from `machinePool` (`machine.go:100, 137`); `m.Release()` (`machine.go:174`) returns it. Neither the pre-loop machine at 406 nor the inner-loop `m` at 429 is released before reassignment. With N test functions per package, N+1 machines are dropped on the floor per package-run (GC-reclaimed, not pool-reused). The `defer` at 366-377 only handles panic. Add `if m != nil { m.Release() }` before each reassignment, or restructure the loop so each iteration's machine is released in its own scope.

- [ ] `gnovm/pkg/test/test.go:426-427` — **`innerBase` and `innerTxn` never explicitly discarded.** The comment on 422-425 correctly observes that "dropping innerBase without calling Write() discards all persisted mutations" — but "drop" here is implicit GC. `storetypes.Store` cache wraps often have `Discard()`/`Close()` sibling APIs; relying on GC means resources (maps of dirty keys, op logs) live until the next cycle. With 100-test packages this accumulates. Recommend an explicit teardown per iteration — even a comment `// intentionally not Write'd; discarded on scope exit` would document the intent if explicit APIs don't exist.

- [ ] `gnovm/pkg/gnolang/machine.go:831` — **`IsTestFile` matches `_test.gno` AND `_filetest.gno`.** `skipTestFileInits` was introduced to skip *_test.gno inits during tgs seeding, but `IsTestFile` (`mempackage.go:154`) also matches `_filetest.gno`. For the single current caller (`Test()` seeding, test.go:263) this is fine because filetest files are split out by `parseMemPackageTests` before. But the new exported `RunMemPackageSkipTestFileInits` is a public API — any future caller that passes a mempackage containing filetest inits will silently drop them. Either (a) rename the parameter / API to `skipTestAndFiletestInits` for accuracy, or (b) narrow `IsTestFile` here to `HasSuffix(fv.FileName, "_test.gno")` only.

- [ ] `gnovm/pkg/test/test.go:397-399` — **Silent skip of filter for non-`IsAll` non-integration types.** The branch `if mptype, ok := mpkg.Type.(gno.MemPackageType); ok && mptype.IsAll()` filters when the type is All, and leaves tmpkg = mpkg otherwise. The comment explains integration mempkgs must not be MPFTest-filtered, but says nothing about the case where `mpkg.Type` is, say, `MPUserTest` directly (pre-filtered upstream). The two current call sites pass `mpkg` (All) and `itmpkg` (Integration), so this is fine today — but the condition is "if All, filter; else don't" while the comment only documents the integration half. A future call site passing a not-All-not-Integration type will silently fall through. Either tighten to `if mptype.IsAll() { filter } else if mptype.IsIntegration() { /* skip */ } else { panic("unexpected type") }`, or expand the comment.

- [ ] `gnovm/pkg/gnolang/machine.go:795-808` — **Function closures may still bind to original pv's file blocks.** Round 1 warning: `pn.NewPackage(m.Alloc)` at 802 calls `pn.PrepareNewValues` internally — this copies `FuncValue`s into `pv.Block.Values`. The `instantiatePackageFiles(pn.FileSet.Files...)` call at 805 creates fresh file blocks on the new `pv`, but the `PrepareNewValues` inside it is a no-op (`pvl == pnl`, per the author's own comment at 798-801). If `FuncValue.Parent` on those copied closures still points at the original seeding-machine's file blocks, then any closure variable read would resolve against stale blocks. This is masked today because (a) top-level func closures resolve names via the package block, not the file block (file block holds imports only), and (b) the file blocks themselves are immutable after preprocessing. If either invariant breaks in future (e.g. mutable package-level file imports, or function-scoped file captures), isolation quietly breaks. Worth a short code comment documenting the dependency, or an explicit `rebindFuncParents` pass for defense in depth.

## Nits

- [ ] `gnovm/pkg/gnolang/machine.go:795` — function signature is `NewPackageInstance(pn *PackageNode) *PackageValue`. There is no doc on this new exported method (the block at 785-794 is the wrong doc, see Critical). A correct 2-line Go doc comment starting with "NewPackageInstance returns a fresh *PackageValue from the already-preprocessed pn..." is the minimum.

- [ ] Commit history is still noisy: `4620381c8 fix`, `77cb0f28b realm state isolation test` (no type prefix), `7b18f676b fix`. Squashing the 8 commits into 2-3 thematic commits (core fix, example test updates, realm isolation fix) would make the merge-commit changelog readable.

- [ ] `gnovm/pkg/test/test.go:417-425` — the big comment block is helpful, but the phrase "own Block and file blocks, re-run var decls and init() funcs" is slightly out of order relative to the call sequence in `NewPackageInstance`. Matches current behavior but reads awkwardly.

- [ ] `gnovm/pkg/gnolang/machine.go:653-660` — `instantiatePackageFiles` doc says "runs top-level non-Func declarations via Kahn's topological sort" — matches the code. Good.

## Missing Tests

- [ ] **Integration / `xxx_test` package isolation.** The three new txtars (`init_and_isolation`, `issue_1982_increment`, `realm_isolation`) all use `package counter` directly. None uses `package counter_test` (the `pkg_test` / integration form). `runTestFiles` is invoked twice in `Test()` — once for `tset` (same-package) and once for `itset` (integration, at test.go:302). A txtar with a `foo_test.gno` in `package foo_test` that mutates a variable in `foo` would verify the integration path also resets state.

- [ ] **Multi-file package with cross-file var dependencies.** `instantiatePackageFiles` now has new `fnsSet`/`fdeclared` logic for the "external dep" set (machine.go:683-697). On the fresh-instantiation path (fns == all FileSet files), `fdeclared` is empty — but the Kahn sort still needs to resolve cross-file deps correctly. A txtar with `a.gno: var X = Y + 1` and `b.gno: var Y = 2` tests that init order is preserved across re-instantiation.

- [ ] **Test that inits run exactly once per test.** The new `TestInitRan` checks that init ran at least once (X == 42) on the first test and that state is reset on subsequent tests — but doesn't verify init didn't run *zero* times on tests 2+ (what if the sort in `instantiatePackageFiles` skipped inits the second time? then `X` would be zero, which also fails). A counter-like `init() { counter++ }` where the test checks `counter == 1` on every call would prove exactly-once.

- [ ] **Package-level import side effects.** If test A causes package P to be imported (via `m.Store.GetPackage`), does test B see P with fresh state too, or only the test's own package? The realm isolation txtar only covers the tested package itself; cross-package realm state isolation is not verified.

## Suggestions

- **Document the performance cost.** The PR description claims "1.7-1.8ms per test function overhead". This should be in a code comment on `NewPackageInstance` and/or the per-test loop, so future maintainers understand the tradeoff. A benchmark in `gnovm/pkg/test/` that calls `NewPackageInstance` in a loop would also pin the number.

- **Release machine to pool.** `defer m.Release()` inside the loop, or an explicit `oldM := m; m = Machine(...); if oldM != nil { oldM.Release() }`. See Warnings #1.

- **Add assertion that `pn.FileSet` is non-nil** at the top of `NewPackageInstance`. If called with a `PackageNode` whose `FileSet == nil` (e.g. from a mis-constructed node), the call to `pn.FileSet.Files` at 805 panics with a cryptic nil deref. A clear error is better.

- **Consider `defer innerTxn` discard semantics**. If the test panics, the inner txn/base are still dropped on GC — but making the scope explicit with a `func(){ ... }()` wrapper would let the `defer recover` in the outer scope catch test-function panics without the parent transaction being left in a half-observed state.

- **Rename `skipTestFileInits` → `skipTestAndFiletestInits`** to match what `IsTestFile` actually filters. Or narrow `IsTestFile` to only `_test.gno` at the call site in `runInitFromUpdates`. Either clarifies the semantics for future callers.

## Questions for Author

- Do you have numbers on the before/after performance for a real package (e.g. `gov/dao/v3/impl` govdao_test — with ~12 tests now each doing `NewPackageInstance`)? The 1.7-1.8ms figure is useful; per-package total would confirm it scales linearly.
- Why `tgs.GetBlockNode(PackageNodeLocation(...))` at test.go:413 rather than caching `pn` from `m.RunMemPackage`'s return value? The refactor discards the return; intentional?
- Is there a plan for a follow-up that addresses the pre-existing `// XXX delete?` `RunFiles` (machine.go:504)? It still uses the old single-machine pattern and would leak state if ever called again for tests.
- The `CacheWrap`'d baseStore per test: does this interact with `SetLogStoreOps`? If a test enables op-logging, the ops are captured on `innerBase`, which is then discarded — confirming that filetest op-logging (which needs observable ops) indeed uses the outer path, not `runTestFiles`.

## Verdict

**APPROVE WITH NITS** — The PR now delivers on the #1982 objective and the round-1 blockers are fully addressed (CI green, realm isolation test + fix, example tests de-coupled from shared state). The one *must*-fix is the stale doc comment on `NewPackageInstance`, which was flagged in round 1 and still ships the wrong description for a new public API. The warnings (machine leak, explicit cleanup, `IsTestFile` scope) are real but non-blocking; they can land in follow-up. Merging after the doc fix is reasonable; the suggested isolation test for `xxx_test` integration packages would be a nice-to-have in a follow-up PR.
