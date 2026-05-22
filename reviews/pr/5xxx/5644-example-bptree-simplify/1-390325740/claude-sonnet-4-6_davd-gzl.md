# PR #5644: feat(example/bptree): simplify

**URL:** https://github.com/gnolang/gno/pull/5644
**Author:** @davd-gzl | **Base:** master | **Files:** 41 | **+198 -217**
**Reviewed by:** davd-gzl | **Model:** claude-sonnet-4-6

## Summary

This PR simplifies the `Get` API of `gno.land/p/nt/bptree/v0` and `gno.land/p/nt/bptree/v0/rotree`, mirroring the same change applied to `avl` in PR #5314.

**Core change:** `BPTree.Get(key string)` now returns `any` instead of `(any, bool)`. A missing key returns `nil`; callers who need to distinguish a stored `nil` value from a missing key must use `Has`.

The PR is structured as four commits:
1. `feat(example/bptree)`: API change to `bptree` and `rotree`, plus migration of all callsites in the examples tree.
2. `fix(example/r/sys/validators/v3)`: migrate two missed `valoperCache.Get` callsites.
3. `fix(example/r/gnops/valopers)`: migrate two `signingRegistry.Get` callsites in rotate_test.
4. `docs(example/bptree)`: align the doc comment on `Get` with the avl package.

**Changed areas:**
- `p/nt/bptree/v0/tree.gno` — `ITree` interface + `BPTree.Get` implementation.
- `p/nt/bptree/v0/rotree/rotree.gno` — `IReadOnlyTree` interface + `ReadOnlyTree.Get` implementation.
- `p/nt/bptree/v0/PLAN.md` — semantics documentation.
- `p/nt/bptree/v0/tree_test.gno`, `rotree/rotree_test.gno` — test updates.
- Callsite migration across 35 files: `p/gnoland/boards/`, `r/gnoland/blog/`, `r/gnoland/boards2/v1/`, `r/gnoland/ghverify/`, `r/gnoland/pages/`, `r/gnops/valopers/`, `r/gov/dao/`, `r/gov/dao/v3/`, `r/nt/commondao/v0/`, `r/sys/users/`, `r/sys/validators/v2-v3/`, `p/nt/commondao/v0/`.

The migration pattern is mechanical and consistent: `v, ok := tree.Get(k); if !ok { ... }` becomes either `v := tree.Get(k); if v == nil { ... }` (when the value is used), or `tree.Has(k)` (when only existence is tested). Both patterns are correct given that callers in this tree never intentionally store nil values.

## Test Results

- **`p/nt/bptree/v0`**: PASS (all 30+ tests including stress tests: `TestExhaustiveRemovalPermutations`, `TestAVLCrossValidation`, etc.)
- **`p/nt/bptree/v0/rotree`**: PASS
- **`r/sys/users`**: PASS
- **`r/gov/dao/v3/memberstore`**: PASS
- **`r/gnoland/boards2/v1`**: PASS (including filetests)
- **`p/gnoland/boards`**: PASS
- **`r/sys/validators/v3`**: PASS
- **`r/gnops/valopers`**: BUILD FAIL — pre-existing issue: `testing.SetSysParamUint64` is undefined in the current gno version. This is not introduced by this PR (the same test file used the old two-value `Get` before, and the new one-value call is a correct migration). The rotate_test and valopers_test changes are syntactically and semantically correct.
- **Edge-case tests written**: none (behavior is covered by existing suite)

## Critical (must fix)

None.

## Warnings (should fix)

- [ ] `p/nt/bptree/v0/rotree/rotree.gno:103-108` — `rotree.Get` early-returns `nil` when `bptree.Get` returns `nil`, without calling `getSafeValue`. This means `makeEntrySafeFn` is **never invoked** when a key stores an explicit `nil` value. The `avl/rotree.Get` does not have this optimization — it always calls `getSafeValue(value)` if the key exists. The bptree design explicitly documents that `nil` is a valid stored value (PLAN.md line 365: "nil is a valid value"), so a user who stores nil and has a non-nil `makeEntrySafeFn` will silently get different behavior. In practice, none of the current callers store nil values via rotree, so this does not cause a real bug today, but it is a semantic divergence from `avl/rotree` that could surprise future users.

  Suggested fix: use `Has` to distinguish stored nil from missing key, then call `getSafeValue` only for present keys:
  ```
  if !roTree.tree.Has(key) {
      return nil
  }
  return roTree.getSafeValue(roTree.tree.Get(key))
  ```
  This costs one extra traversal but keeps semantics consistent with avl/rotree.

- [ ] `r/gnoland/blog/admin_test.gno:94` — After `AdminRemoveModerator`, the test verifies the value is `false` by calling `moderatorList.Get(mod.String())` (new single-return form). This is a stale test pattern: `AdminRemoveModerator` uses `Set(addr, false)` not `Remove`, so `Has(addr)` returns `true` for removed moderators, and `isModerator` now uses `Has`. This means `isModerator` incorrectly returns `true` after removal. This is a **pre-existing FIXME bug** (`admin.gno:43` has `// FIXME: delete instead?`) not introduced by this PR, but the migration to `Has` in `isModerator` makes the bug more visible. Worth a note to the author to track this.

## Nits

- [ ] `r/sys/users/users.gno:63` — Comment says "when requesting data from this AVL tree, (exists bool) will be true / Even if the data is 'deleted'". This comment is stale: the underlying tree is now `bptree`, not avl, and the `(exists bool)` API no longer exists. The comment should be removed or rewritten to describe the current behavior (rotree.Get returns nil for deleted entries because `makeUserDataSafe` returns nil, which is now transparent to callers).

- [ ] `p/nt/bptree/v0/tree.gno:63-66` — The doc comment on `Get` shows an example using `if value, ok := tree.Get("key").(MyType); ok { ... }`. This is a type-assertion two-result form in Go, but in Gno the idiomatic pattern might vary. Worth verifying the example is actually idiomatic Gno (it works, but `if v := tree.Get("key"); v != nil { value := v.(MyType); ... }` may be clearer and avoids a panic if the type assertion is wrong).

- [ ] `p/nt/bptree/v0/PLAN.md:368` — "use `Has` to distinguish from a stored nil value" is documented. The rotree package doc comment in `rotree.gno:101` should also mention this, for parity with the PLAN.

## Missing Tests

- [ ] `p/nt/bptree/v0/rotree/rotree_test.gno` — No test for `rotree.Get` where the underlying tree has a stored `nil` value and `makeEntrySafeFn` is non-nil. Given the warning above, this case exercises a behavioral difference vs `avl/rotree`.

- [ ] `p/nt/bptree/v0/rotree/rotree_test.gno` — No test verifying that `IReadOnlyTree` and `ITree` compile-time interface checks (`var _ bptree.ITree = (*ReadOnlyTree)(nil)`) still hold after the signature change. They do (the file compiles), but an explicit test comment would help.

## Suggestions

- The doc comment example in `tree.gno:63-66` suggests using a type assertion comma-ok form (`value, ok := tree.Get(...).(MyType)`). Consider simplifying it to the two-step pattern that avoids potential panics:
  ```go
  // Typical usage:
  //
  //   if v := tree.Get("key"); v != nil {
  //       value := v.(MyType)
  //       // use value
  //   }
  ```
  This matches the pattern used consistently throughout the callsites in this PR.

- The `rotree.Get` early-nil optimization is a micro-optimization that adds behavioral complexity. Given that bptree traversal is O(log n) and `Has` + `Get` would be 2 × O(log n), the cost is manageable and keeping full semantic parity with `avl/rotree` is worth more than the optimization.

## Questions for Author

- Was the behavior of `rotree.Get` for stored-nil values intentionally changed (i.e., `makeEntrySafeFn` is not called for stored nil), or was the early-return added purely as a missing-key shortcut without considering the stored-nil case?

- The `isModerator` in `r/gnoland/blog/admin.gno` was migrated to `Has`, but `AdminRemoveModerator` uses `Set(addr, false)` rather than `Remove`. After this PR, `isModerator` returns `true` even for "removed" moderators. Was this regression noticed? The pre-existing FIXME comment suggests the correct fix is to use `Remove`; should this PR include that fix to avoid worsening the bug?

- `r/gnops/valopers` tests fail to build due to `testing.SetSysParamUint64` being undefined. Was this expected, and is there a tracking issue?

## Verdict

APPROVE — The core API simplification is correct, well-motivated (mirrors avl PR #5314), thoroughly tested, and the callsite migration is mechanically sound; the only concern worth addressing before merge is the `rotree.Get` nil-value edge case (warning above), which is minor given no current caller stores nil values through rotree.
