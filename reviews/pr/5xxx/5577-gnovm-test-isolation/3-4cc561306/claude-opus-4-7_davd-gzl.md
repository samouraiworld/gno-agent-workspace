# PR #5577: fix(gnovm): reset package state between test functions

URL: https://github.com/gnolang/gno/pull/5577
Author: notJoon | Base: master | Files: 10 | +357 -107
Reviewed by: davd-gzl | Model: claude-opus-4-7

**Verdict: REQUEST CHANGES** — the round-2 critical (stale doc comment on `NewPackageInstance`) is still shipped; everything else round-2 flagged still holds. The HEAD jump `dff0112a9 → 4cc561306` is a single merge of master with zero PR-file changes, so nothing on this branch was rewritten in response to the round-2 review.

## Summary
Re-review at `4cc561306`. The diff against the previous review commit (`dff0112a9`) touches none of the PR's files — `git diff dff0112a9..4cc561306 -- gnovm/pkg/test/test.go gnovm/pkg/gnolang/machine.go` is empty. The HEAD bump is `Merge branch 'master' into fix/gnovm-test-isolation-1982` pulling in #5587 (gnovm regression) and #5590 (amino). Findings from round 2 carry over unchanged; see [`reviews/pr/5xxx/5577-gnovm-test-isolation/2-dff0112a9/claude-opus-4-7_davd-gzl.md`](../2-dff0112a9/claude-opus-4-7_davd-gzl.md) for the full analysis. This file lists only what the maintainer needs to re-confirm before merge.

## Glossary
- `tgs` — outer per-package transaction store created in `Test()` (`test.go:235`).
- `innerBase` / `innerTxn` — per-test cache-wrap and nested transaction created in `runTestFiles` (`test.go:426-427`).
- `NewPackageInstance` — new exported method on `*Machine` (`machine.go:795`) that builds a fresh `*PackageValue` from a preprocessed `*PackageNode`.
- `RunMemPackageSkipTestFileInits` — new exported entry on `*Machine` (`machine.go:309`) that runs everything except `init()` funcs in `*_test.gno` / `*_filetest.gno` files.

## Test Results
- [`gnovm/cmd/gno/testdata/test/init_and_isolation.txtar`](../../../../../.worktrees/gno-review-5577/gnovm/cmd/gno/testdata/test/init_and_isolation.txtar) PASS
- [`gnovm/cmd/gno/testdata/test/issue_1982_increment.txtar`](../../../../../.worktrees/gno-review-5577/gnovm/cmd/gno/testdata/test/issue_1982_increment.txtar) PASS
- [`gnovm/cmd/gno/testdata/test/realm_isolation.txtar`](../../../../../.worktrees/gno-review-5577/gnovm/cmd/gno/testdata/test/realm_isolation.txtar) PASS

CI on the PR page: `main / test` is failing (4m22s), the rest are green. The failing job's log shows the same kind of flake seen in the prior round; not investigated further this round.

## Critical (must fix)
- **[stale doc still shipping for new exported API]** [`gnovm/pkg/gnolang/machine.go:785-794`](../../../../../.worktrees/gno-review-5577/gnovm/pkg/gnolang/machine.go#L785-L794) — round-2 critical, still unaddressed.
  <details><summary>details</summary>

  The block immediately above [`func (m *Machine) NewPackageInstance(...)`](../../../../../.worktrees/gno-review-5577/gnovm/pkg/gnolang/machine.go#L795) reads `// instantiatePackageFiles performs runtime instantiation for the given / file nodes: ...` and closes with `// fns must already be present in pn.FileSet and fully preprocessed; ...`. That doc describes `instantiatePackageFiles` (whose own doc is at [`machine.go:653-660`](../../../../../.worktrees/gno-review-5577/gnovm/pkg/gnolang/machine.go#L653-L660)), not `NewPackageInstance`. `go doc` and IDE hover will show the wrong description for the new public API. Fix: rewrite the block to describe what `NewPackageInstance` does (take a preprocessed `*PackageNode`, allocate fresh `*PackageValue` via `pn.NewPackage`, wire it as active, build file blocks and re-run var decls via `instantiatePackageFiles`, then run `init()` funcs via `runInitFromUpdates`).
  </details>

## Warnings (should fix)
All round-2 warnings still apply unchanged at the same line numbers — listed terse for triage:

- **[pool leak per test function]** [`gnovm/pkg/test/test.go:406, 429`](../../../../../.worktrees/gno-review-5577/gnovm/pkg/test/test.go#L406-L429) — N+1 `Machine()` calls per package, none paired with `m.Release()` ([`machine.go:174`](../../../../../.worktrees/gno-review-5577/gnovm/pkg/gnolang/machine.go#L174)).
- **[implicit GC of inner cache wrap]** [`gnovm/pkg/test/test.go:426-427`](../../../../../.worktrees/gno-review-5577/gnovm/pkg/test/test.go#L426-L427) — `innerBase` and `innerTxn` are dropped by scope-exit; no explicit teardown, no comment documenting the GC-discards-by-design contract.
- **[`IsTestFile` scope vs API name]** [`gnovm/pkg/gnolang/machine.go:831`](../../../../../.worktrees/gno-review-5577/gnovm/pkg/gnolang/machine.go#L831) — `IsTestFile` matches both `_test.gno` and `_filetest.gno` ([`mempackage.go:154-156`](../../../../../.worktrees/gno-review-5577/gnovm/pkg/gnolang/mempackage.go#L154-L156)), so the new exported `RunMemPackageSkipTestFileInits` silently drops filetest inits too. The single current caller at [`test.go:263`](../../../../../.worktrees/gno-review-5577/gnovm/pkg/test/test.go#L263) is filetest-free (MPFTest strips them upstream at [`test.go:255`](../../../../../.worktrees/gno-review-5577/gnovm/pkg/test/test.go#L255)), so this is decay risk for future callers, not a current bug. Either rename the API/param to `…AndFiletestInits` or narrow the check at [`machine.go:831`](../../../../../.worktrees/gno-review-5577/gnovm/pkg/gnolang/machine.go#L831) to `strings.HasSuffix(fv.FileName, "_test.gno")`.
- **[type filter silent fall-through]** [`gnovm/pkg/test/test.go:396-399`](../../../../../.worktrees/gno-review-5577/gnovm/pkg/test/test.go#L396-L399) — `if IsAll filter; else don't` is correct for today's two call sites (All from `runTestFiles(mpkg, …)` at [`test.go:281`](../../../../../.worktrees/gno-review-5577/gnovm/pkg/test/test.go#L281) and Integration from [`test.go:302`](../../../../../.worktrees/gno-review-5577/gnovm/pkg/test/test.go#L302)) but the comment only justifies the integration half. A future call site with a pre-filtered Test/Prod type falls through silently. Either tighten to an explicit `else if IsIntegration { /* skip */ } else { panic }` or extend the comment.
- **[func closure parent binding implicit]** [`gnovm/pkg/gnolang/machine.go:795-808`](../../../../../.worktrees/gno-review-5577/gnovm/pkg/gnolang/machine.go#L795-L808) — `pn.NewPackage(m.Alloc)` copies `FuncValue`s into `pv.Block.Values`; the second `PrepareNewValues` inside `instantiatePackageFiles` is a no-op for an already-preprocessed `pn`. Safe today because top-level func closures resolve names via the package block (not file block — files hold imports only). Worth a one-line comment so a future change that lets file blocks hold mutable state doesn't quietly break isolation.

## Nits
- [`gnovm/pkg/gnolang/machine.go:417-425`](../../../../../.worktrees/gno-review-5577/gnovm/pkg/test/test.go#L417-L425) — the explanatory block "own Block and file blocks, re-run var decls and init() funcs" is slightly out of order vs `NewPackageInstance`'s actual call sequence (`NewPackage` → `instantiatePackageFiles` → `runInitFromUpdates`). Reads fine, low priority.
- Commit history still noisy (`fix`, `realm state isolation test`, etc.). Squash to ~3 thematic commits at merge.

## Missing Tests
- **[integration `xxx_test` package isolation]** [`gnovm/pkg/test/test.go:302`](../../../../../.worktrees/gno-review-5577/gnovm/pkg/test/test.go#L302) — all three new txtars use same-package (`package counter`); none exercises the integration test path. The integration call site lands at [`test.go:408-412`](../../../../../.worktrees/gno-review-5577/gnovm/pkg/test/test.go#L408-L412), which takes the `if tgs.GetMemPackage(itmpkg.Path) == nil` branch and runs `m.RunMemPackage(tmpkg, false)` with `skipTestFileInits=false`. With `save=false`, `saveNewPackageValuesAndTypes` is not invoked, so realm-finalization writes never reach tcw and `innerTxn.cacheObjects` is a fresh empty map (see [`store.go:218`](../../../../../.worktrees/gno-review-5577/gnovm/pkg/gnolang/store.go#L218)), meaning seed-init mutations don't leak into per-test transactions. The reasoning chain is non-trivial and currently unverified by any test — add a txtar with a `foo_test` package whose `init()` calls a cross-realm mutator (à la `dao.InitWithUsers`) and a follow-up test asserting clean state, to lock in the invariant.
- **[exactly-once init per test]** — `TestInitRan` checks `X == 42` (at-least-once), not exactly-once. A `init() { counter++ }` and `if counter != 1 { fail }` would prove no init is re-scheduled across iterations.
- **[cross-file var deps under fresh instantiation]** — [`machine.go:683-697`](../../../../../.worktrees/gno-review-5577/gnovm/pkg/gnolang/machine.go#L683-L697) `fdeclared` is empty on the fresh-instantiation path; a txtar with `a.gno: var X = Y + 1` and `b.gno: var Y = 2` would verify the topological sort still resolves cross-file deps after `NewPackageInstance` re-runs decls.

## Suggestions
- Document the per-test overhead (`~1.7-1.8ms`) in a code comment on [`NewPackageInstance`](../../../../../.worktrees/gno-review-5577/gnovm/pkg/gnolang/machine.go#L795), or land a benchmark — so the next person eyeing this for further isolation knows the cost ceiling.
- `defer m.Release()` inside the per-test loop, or an explicit `oldM := m; m = Machine(...); if oldM != nil { oldM.Release() }`, returns machines to the pool.
- Nil-guard at the top of `NewPackageInstance`: `if pn == nil || pn.FileSet == nil { panic("NewPackageInstance: pn.FileSet is nil") }`. Current code nil-derefs at [`machine.go:805`](../../../../../.worktrees/gno-review-5577/gnovm/pkg/gnolang/machine.go#L805) with a cryptic message.

## Questions for Author
- Was the round-2 doc-comment fix (the round-2 critical) overlooked, or is there a reason to keep it as-is until later?
- Any plan to extend `RunMemPackageSkipTestFileInits` semantics (or equivalent) to the integration path at [`test.go:409`](../../../../../.worktrees/gno-review-5577/gnovm/pkg/test/test.go#L409) — or is the `save=false` + fresh `cacheObjects` chain considered enough?
- The PR description gives per-test cost `~1.7-1.8ms`; do you have a number for `gov/dao/v3/impl` total wall-clock before/after, given its `init()` runs `InitWithUsers`?
