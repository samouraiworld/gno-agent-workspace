# PR [#6089](https://github.com/gnolang/gno/pull/6089): refactor(gnovm): derive IsAssignable from NameSources, drop StaticBlock.UnassignableNames

URL: https://github.com/gnolang/gno/pull/6089
Author: ltzmaxwell | Base: master | Files: 5 | +112 -67
Reviewed by: davd-gzl | Model: claude-opus-5 | Commit: 687d9c51c (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6089 687d9c51c`
Overview: [visual overview](../overview.html)

## Overview

Gno refuses `f = nil` when `f` is a package-level `func`. Two fields on `StaticBlock` carried that fact. [`NameSources`](https://github.com/gnolang/gno/blob/687d9c51c/gnovm/pkg/gnolang/nodes.go#L1660) · [↗](../../../../../.worktrees/gno-review-6089/gnovm/pkg/gnolang/nodes.go#L1660) records how each name entered its block, one entry per name, aligned with `Names`. `UnassignableNames` was a second, flat list. One line in the preprocessor wrote both, on the same name, one immediately after the other, and no other line ever wrote the second.

This change deletes the flat list and answers the refusal from [`NameSources[idx].Type != NSFuncDecl`](https://github.com/gnolang/gno/blob/687d9c51c/gnovm/pkg/gnolang/nodes.go#L1934) · [↗](../../../../../.worktrees/gno-review-6089/gnovm/pkg/gnolang/nodes.go#L1934), indexed by the position [`GetLocalIndex`](https://github.com/gnolang/gno/blob/687d9c51c/gnovm/pkg/gnolang/nodes.go#L1981) · [↗](../../../../../.worktrees/gno-review-6089/gnovm/pkg/gnolang/nodes.go#L1981) already returns on the way. The amino slot the deleted field held is retired with a [blank `_ struct{}` field tagged `amino:"reserved"`](https://github.com/gnolang/gno/blob/687d9c51c/gnovm/pkg/gnolang/nodes.go#L1662) · [↗](../../../../../.worktrees/gno-review-6089/gnovm/pkg/gnolang/nodes.go#L1662), so `Consts` stays 9 and `Parent` stays 11. The rest of the diff is the regenerated schema and decoder, plus an ADR.

**Verdict: APPROVE** — the deleted list and the surviving one agree on every name in every block the filetest corpus builds, and the retired slot reproduces byte for byte from [`misc/genproto2`](https://github.com/gnolang/gno/blob/687d9c51c/misc/genproto2/Makefile#L1-L3) · [↗](../../../../../.worktrees/gno-review-6089/misc/genproto2/Makefile#L1) (1 Missing test, 2 Nits).

## Verify first

- [`gnovm/pkg/gnolang/nodes.go:1934`](https://github.com/gnolang/gno/blob/687d9c51c/gnovm/pkg/gnolang/nodes.go#L1934) · [↗](../../../../../.worktrees/gno-review-6089/gnovm/pkg/gnolang/nodes.go#L1934) — the read is unguarded, so it panics wherever `len(NameSources)` falls below `len(Names)`. Apply [`tests/equivalence-harness.patch`](tests/equivalence-harness.patch) at the merge base and run `go test ./gnovm/pkg/gnolang/ -run TestFiles`: it walks every block after every `Preprocess` and reports 0 length mismatches over 808021 blocks.
- [`gnovm/pkg/gnolang/nodes.go:1662`](https://github.com/gnolang/gno/blob/687d9c51c/gnovm/pkg/gnolang/nodes.go#L1662) · [↗](../../../../../.worktrees/gno-review-6089/gnovm/pkg/gnolang/nodes.go#L1662) — amino numbers fields by position, so the blank field has to sit between `HeapItems` and `Consts` or every later field shifts. Run `cd misc/genproto2 && make all`, then `git status`: it reports a clean tree at this head.
- [`gnovm/pkg/gnolang/preprocess.go:2731`](https://github.com/gnolang/gno/blob/687d9c51c/gnovm/pkg/gnolang/preprocess.go#L2731) · [↗](../../../../../.worktrees/gno-review-6089/gnovm/pkg/gnolang/preprocess.go#L2731) — this is the only caller, and nothing in `gnovm/tests/files/` exercises it. Drop [`tests/assign_func_decl_err.gno`](tests/assign_func_decl_err.gno) into that directory and run `go test -run 'TestFiles/assign_func_decl_err.gno$' ./gnovm/pkg/gnolang/`.

## Summary

The load-bearing claim is that `UnassignableNames` and `NameSources` never disagreed. Both were written by the non-method `*FuncDecl` case in [`initStaticBlocks2`, at `preprocess.go:481`](https://github.com/gnolang/gno/blob/687d9c51c/gnovm/pkg/gnolang/preprocess.go#L481) · [↗](../../../../../.worktrees/gno-review-6089/gnovm/pkg/gnolang/preprocess.go#L481), which called `Reserve(..., NSFuncDecl, -1)` and then appended the same name to the list. [`NSFuncDecl`](https://github.com/gnolang/gno/blob/687d9c51c/gnovm/pkg/gnolang/nodes.go#L1709) · [↗](../../../../../.worktrees/gno-review-6089/gnovm/pkg/gnolang/nodes.go#L1709) appears nowhere else in the package, and [`Define2`](https://github.com/gnolang/gno/blob/687d9c51c/gnovm/pkg/gnolang/nodes.go#L2339) · [↗](../../../../../.worktrees/gno-review-6089/gnovm/pkg/gnolang/nodes.go#L2339) is the only writer of either `Names` or `NameSources`, appending to both in the same branch, so the index alignment the new read depends on cannot drift.

The claim holds under measurement. A harness applied at the merge base add66f24b, where both representations still exist, compared them on every name in every block after every `Preprocess`, plus at each of the 24107 real call sites: 56860035 name comparisons, no disagreement, no length mismatch, and no redefinition that overwrote an `NSFuncDecl` entry with another kind. Nine hand-written programs produce identical output on the merge base and on this head.

What is missing is a test. The refusal the PR rewrites the predicate for has no filetest at all, so nothing in CI would have caught a change of behaviour here.

## Numbers

Harness at add66f24b, whole [`TestFiles`](https://github.com/gnolang/gno/blob/687d9c51c/gnovm/pkg/gnolang/files_test.go#L36) · [↗](../../../../../.worktrees/gno-review-6089/gnovm/pkg/gnolang/files_test.go#L36) corpus, 193s. Artifacts in [`tests/`](tests/).

| Probe | Result |
| --- | --- |
| Old answer against new answer at 24107 real call sites | 0 disagreements |
| 808021 blocks checked for `len(NameSources) == len(Names)` | 0 mismatches |
| 56860035 names compared, list membership against `Type == NSFuncDecl` | 0 disagreements |
| `Define2` redefinitions overwriting an `NSFuncDecl` entry | 0 |

Which branch of the rewritten function answers, counted at this head over the same corpus with the two filetests below in place, 189s.

| Outcome | Calls |
| --- | --- |
| name resolved, assignable | 23836 |
| name not resolved anywhere, assignable, all of them `_` | 693 |
| name resolved, `Type == NSFuncDecl`, **refused** | **1** |
| uverse fallback | 0 |

Remove the two filetests and the refusal count is 0 in 24527 calls.

## Missing Tests

- **[tests — the refusal has no coverage]** `gnovm/pkg/gnolang/preprocess.go:2731` — nothing under `gnovm/tests/files/` assigns to a package-level func name, so the predicate this diff rewrites has no regression guard.
  <details><summary>details</summary>

  `grep -rn "not assignable" gnovm/tests/ gno.land/pkg/integration/testdata/` returns three hits, all of them prose about type assignability in `types/cmp_primitive_1.gno`, `types/eql_0b2.gno` and `types/assign_range_j.gno`. None reaches [`preprocess.go:2732`](https://github.com/gnolang/gno/blob/687d9c51c/gnovm/pkg/gnolang/preprocess.go#L2732) · [↗](../../../../../.worktrees/gno-review-6089/gnovm/pkg/gnolang/preprocess.go#L2732). A counter on that branch agrees: 0 refusals in 24527 calls over the whole corpus. [PR 3198](https://github.com/gnolang/gno/pull/3198) introduced the field along with 36 `addressable_*.gno` filetests, none of which reaches the refusal.

  Two filetests close it, both passing at this head and at the merge base add66f24b: [`tests/assign_func_decl_err.gno`](tests/assign_func_decl_err.gno) asserts the refusal, and [`tests/assign_func_decl.gno`](tests/assign_func_decl.gno) asserts the two names it must not swallow, a package-level `var` holding a func value and a local shadowing a package-level func. Fix: add both to `gnovm/tests/files/`.
  </details>

## Nits

- **[dead code]** `gnovm/pkg/gnolang/nodes.go:1939-1940` — the uverse branch cannot be reached from the function's only caller.
  <details><summary>details</summary>

  [`preprocess.go:2727`](https://github.com/gnolang/gno/blob/687d9c51c/gnovm/pkg/gnolang/preprocess.go#L2727) · [↗](../../../../../.worktrees/gno-review-6089/gnovm/pkg/gnolang/preprocess.go#L2727) runs `AssertCompatible` first, and that reaches [`assertValidAssignLhs`](https://github.com/gnolang/gno/blob/687d9c51c/gnovm/pkg/gnolang/type_check.go#L1040) · [↗](../../../../../.worktrees/gno-review-6089/gnovm/pkg/gnolang/type_check.go#L1040) for every LHS through `evalAssignLhsType`, where a built-in panics with `cannot assign to uverse <n>`. So the name never arrives at [`nodes.go:1929`](https://github.com/gnolang/gno/blob/687d9c51c/gnovm/pkg/gnolang/nodes.go#L1929) · [↗](../../../../../.worktrees/gno-review-6089/gnovm/pkg/gnolang/nodes.go#L1929). `println = nil` reports `cannot assign to uverse println` at this head and at the merge base alike, never `not assignable`, and a counter on the branch fired 0 times across 24530 calls over the whole filetest corpus.

  The `return true` fallback under it is live, and blank identifiers are what reach it: `Define2` drops `_` without recording it, so `_ = 1` walks the chain to the end and lands there, 693 of those 24530 calls. Fix: none while the function keeps one caller; the uverse branch is the cost of the shape it borrows from `GetStaticTypeOf`.
  </details>

- **[comment accuracy]** `gnovm/pkg/gnolang/nodes.go:1927-1928` — the doc comment gives the wrong reason for constants and type names never reaching the check.
  <details><summary>details</summary>

  The comment reads "Constants and type names never reach this check: both are folded to const expressions during preprocessing." They do not reach it, but folding is not why. `assertValidAssignLhs` matches `case *NameExpr` and calls `last.GetIsConst(store, clx.Name)`, so the LHS is still a `*NameExpr` when the refusal fires: `const c = 1; c = 2` reports `cannot assign to const c` and `type T int; T = 1` reports `cannot assign to const T`, both from [`type_check.go:1042`](https://github.com/gnolang/gno/blob/687d9c51c/gnovm/pkg/gnolang/type_check.go#L1042) · [↗](../../../../../.worktrees/gno-review-6089/gnovm/pkg/gnolang/type_check.go#L1042), on this head and on the merge base alike. Fix: name `assertValidAssignLhs` as the check that refuses them.
  </details>

## Verified

- The two representations agree everywhere they could disagree. [`tests/equivalence-harness.patch`](tests/equivalence-harness.patch) at add66f24b, the merge base, instruments `IsAssignable`, `Preprocess` and `Define2`, then runs the `TestFiles` corpus: `calls=24107 blocks=808021 names=56860035 diffs=0`. Method in [`tests/equivalence-harness.md`](tests/equivalence-harness.md).
- The generated files are the generator's output. `cd misc/genproto2 && make all` at this head leaves `git status` clean, so `gnolang.proto` and `pb3_gen.go` are not hand-edited and the reserved slot numbering is amino's own.
- The schema change touches nothing on disk. [`SetBlockNode`, lines 941-946](https://github.com/gnolang/gno/blob/687d9c51c/gnovm/pkg/gnolang/store.go#L941-L946) · [↗](../../../../../.worktrees/gno-review-6089/gnovm/pkg/gnolang/store.go#L941-L946) has its backend write commented out behind a TODO and only fills `cacheNodes`, so no persisted bytes carry field 8.
- The struct shrink is not consensus-affecting. `StaticBlock` appears in neither the `_alloc*` table nor the `check(...)` list in [`alloc.go:152-164`](https://github.com/gnolang/gno/blob/687d9c51c/gnovm/pkg/gnolang/alloc.go#L152-L164) · [↗](../../../../../.worktrees/gno-review-6089/gnovm/pkg/gnolang/alloc.go#L152), which is where `unsafe.Sizeof` feeds allocation accounting.
- Round-tripping cannot shorten `NameSources`. [`pb3_gen.go:12419-12429`](https://github.com/gnolang/gno/blob/687d9c51c/gnovm/pkg/gnolang/pb3_gen.go#L12419-L12429) · [↗](../../../../../.worktrees/gno-review-6089/gnovm/pkg/gnolang/pb3_gen.go#L12419) emits a field-6 tag for every element, zero-valued ones included, so a decoded block keeps `len(NameSources) == len(Names)`.
- Nine programs produce identical output at this head and at the merge base: assignment to a func decl, to a local shadowing one, to a built-in, to a constant, to a type name, an `init` func beside `main`, a package-level `var` beside an unrelated func, and `var f` declared before and after `func f`. The last two are the one shape where the two representations could have diverged, since [`Reserve`](https://github.com/gnolang/gno/blob/687d9c51c/gnovm/pkg/gnolang/nodes.go#L2329) · [↗](../../../../../.worktrees/gno-review-6089/gnovm/pkg/gnolang/nodes.go#L2329) is a no-op on a name the block already holds while the deleted append was not; the go/types pass refuses both with `f redeclared in this block` before the preprocessor sees them. Programs and outputs in [`tests/probes/`](tests/probes/README.md).
- `go test -run 'TestFiles/assign_func_decl' ./gnovm/pkg/gnolang/` passes at this head with both new filetests in place.

## Open questions

- `gnokms / lint` is red at this head. The job failed fetching `golangci-lint.run/jsonschema/golangci.v2.11.jsonschema.json` with `context deadline exceeded`, which is the runner's network and not the diff. Not posted: nothing about CI belongs in the comment.
- The dead-branch and comment-accuracy nits are not posted. The first asks for no edit while the function keeps one caller, and the second is about a code comment's own wording.
- `IsAssignable` is exported and the rename carries no alias. `grep -rn "IsAssignable\b" .` at this head returns the single call site, so nothing in the monorepo breaks, and the third commit 5bc66c389 names the rename in its subject. Not posted: `gnovm/pkg/gnolang` makes no compatibility promise to out-of-tree importers.
