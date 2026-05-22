# PR #5577: Fix/gnovm test isolation 1982

**URL:** https://github.com/gnolang/gno/pull/5577
**Author:** notJoon | **Base:** master | **Files:** 4 | **+184 -29**
**Reviewed by:** davd-gzl | **Model:** glm-5.1

## Summary

This PR addresses issue #1982 ("Reset machine context & realm state after each Test function") by introducing per-test isolation in `gno test`. The core idea is correct and much-needed: each test function should run against a fresh package state, so mutations in one test don't leak into the next.

**How it works:**

1. **New method `Machine.NewPackageInstance(pn *PackageNode)`** (`machine.go:784-797`): Creates a fresh `PackageValue` from an already-preprocessed `PackageNode`. It calls `pn.NewPackage(alloc)` (which internally calls `PrepareNewValues`), sets the package in the store, then calls `instantiatePackageFiles(pn.FileSet.Files...)` to create new file blocks and re-run var declarations + init functions. Finally runs init funcs via `runInitFromUpdates(pv, pv.Block.Values)`.

2. **Refactored `runFileDecls` into two phases** (`machine.go:593-772`): Preprocessing + `pn.FileSet` mutation stays in `runFileDecls`; runtime instantiation (file block allocation, `PrepareNewValues`, var declaration execution) is extracted into `instantiatePackageFiles`. The `fdeclared` set (names from files not in `fns`, treated as pre-satisfied dependencies) computation is moved from before `AddFiles` to after it, using a `fnsSet` map to achieve the same semantics.

3. **Per-test nested transaction** (`test.go:417-421`): Each test now opens `innerTxn := tgs.BeginTransaction(...)`, creates a machine on `innerTxn`, and calls `NewPackageInstance(pn)`. The inner transaction's isolated cache ensures state changes are discarded when the txn is dropped (no `Write()` call).

4. **Two new txtar tests** verifying init runs per-test and state doesn't leak between tests.

**CI status:** FAILING — `gno-checks/test` reports 8 test errors. At minimum, `gno.land/r/tests/vm/map_delete` fails because `TestGetMapAfterDelete` relied on state from `TestDeleteMap` (now reset). `check` fails due to non-conventional PR title. `main/test` and `stdlibs/test` likely timeout.

## Test Results

- **Existing tests:** FAIL — CI reports 8 test errors; `map_delete` is confirmed broken by the behavior change. Other failures likely follow the same pattern (tests written assuming shared state).
- **Edge-case tests:** skipped

## Critical (must fix)

- [ ] `gnovm/pkg/gnolang/machine.go:774-783` — **Stale doc comment on `NewPackageInstance`**: The doc comment reads "instantiatePackageFiles performs runtime instantiation for the given file nodes..." which is a copy of `instantiatePackageFiles`'s doc (line 642). It does not describe what `NewPackageInstance` does. This is misleading and must be rewritten to accurately document `NewPackageInstance`.

- [ ] CI: 8 test errors — **Breaking change without updating existing tests**. The isolation change is semantically correct but breaks existing tests that relied on shared state between test functions (e.g., `gno.land/r/tests/vm/map_delete` where `TestGetMapAfterDelete` expected state from `TestDeleteMap`). All broken tests must be identified and updated before merge.

## Warnings (should fix)

- [ ] `gnovm/pkg/gnolang/machine.go:791` — **Function closures may not be properly bound to new file blocks**. `pn.NewPackage(m.Alloc)` calls `PrepareNewValues` which copies function values into `pv.Block.Values` and tries to set their `Parent` to file blocks — but `pv.fBlocksMap` is empty at that point, so function `Parent` fields are not set to the new file blocks. The subsequent `instantiatePackageFiles` call adds file blocks but its `PrepareNewValues` is a no-op (`pvl == pnl`). The copied function values inherit `Parent` from `pn.Values` (pointing to the original pv's file blocks from the `tgs`-level load). This works for name resolution *today* because file blocks hold immutable imports, but it's fragile: (1) if the VM ever stores mutable runtime state in file blocks, isolation breaks; (2) the old file blocks could be GC'd if the original pv is released, causing dangling pointers. Consider reordering so file blocks are created before `pn.NewPackage`, or adding a separate pass to update function `Parent` fields after file block creation.

- [ ] `gnovm/pkg/test/test.go:419-420` — **Machine leak**: `m` is overwritten each iteration without calling `m.Release()`. The pre-loop machine (line 399) is also leaked. This is pre-existing but worsened by this PR (one machine per test instead of one shared machine). Consider adding `defer m.Release()` or releasing before reassignment.

- [ ] `gnovm/pkg/test/test.go:417` — **`innerTxn` never explicitly cleaned up**. The transaction is opened but never committed (`Write()`) or rolled back. It relies entirely on GC to reclaim resources. While functionally correct (writes are discarded), explicit cleanup would be safer and more idiomatic for resource management.

## Nits

- [ ] PR body is "WIP" — PR is clearly not ready for final review.
- [ ] PR title `Fix/gnovm test isolation 1982` doesn't follow conventional commits format (causes CI `check` failure). Should be e.g. `fix(gnovm): reset package state between test functions`.
- [ ] Commit messages are sparse: "fix", "feat(gnovm): add `NewPackageInstance`...", "fix(gnovm): reset package state...". Consider squashing into a single well-described commit.

## Missing Tests

- [ ] No test for **realm state isolation** — issue #1982 specifically mentions realm state reset, and the core maintainer (thehowl) flagged this as the most important angle. The new tests only cover `p/` package globals. A txtar test under `r/` verifying that realm store changes from one test don't leak to the next is essential.
- [ ] No test for **multi-file package with cross-file dependencies** — `instantiatePackageFiles` has new `fnsSet`/`fdeclared` logic; a test with vars in one file depending on vars in another would verify the topological sort still works correctly in the fresh-instantiation path.
- [ ] No test for **`xxx_test` integration packages** — `runTestFiles` is also called for integration test packages; verify isolation works there too.

## Suggestions

- Add an ADR in `gnovm/adr/` per the project's convention for non-trivial changes. This PR introduces a new public method (`NewPackageInstance`) and changes test execution semantics — it's not a trivial bug fix.
- In `NewPackageInstance`, consider creating file blocks *before* calling `pn.NewPackage`, then passing the populated `pv` so `PrepareNewValues` inside `NewPackage` can correctly bind function closures. This avoids the two-phase binding issue described in the warning above.
- Run `grep -rn 'TestDelete\|TestAfter\|TestPrevious\|TestNext' examples/` to systematically find tests that depend on execution order.

## Questions for Author

- What is the intended behavior for **realm state** (on-chain store) between tests? `innerTxn` discards cache-layer changes, but if a test calls a realm function that persists state to the base store, that state would survive. Is this intentional, or should realm state also be fully reset?
- Have you identified all 8 failing CI tests? Are they all the same pattern (relying on shared state), or are there deeper issues?

## Verdict

REQUEST CHANGES — The approach is architecturally sound and addresses a real need, but the PR has two blockers: (1) stale doc comment on the new public method and (2) CI failures from existing tests broken by the behavior change. The function-closure binding concern also needs investigation before this can merge safely. Fix these, add realm isolation tests, and this will be a valuable addition.
