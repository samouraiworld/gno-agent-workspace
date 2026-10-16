# Claims: https://github.com/gnolang/gno/pull/6272 round 2, 03363f296, claude-opus-5-5, solo review

Round shape: solo round, one finder and one judge and writer, every Critical and Warning run, the rest read

## Candidates

| # | State | Band | file:line | Check | Observed | Artifact | Tier | Mechanism |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Suggestion | gnovm/pkg/gnolang/nodes.go:2085 | go test ./gnovm/pkg/gnolang -run TestZZJudgeNameSourceShapes -v with tests/judge-1-namesource-shapes_test.go copied in: four package-block shapes, at head and with nodes.go reverted to the merge base; then tests/judge-1-guard.patch applied | head: `emptyFileSet+Reserve, nil store: ok fnNil=true type=2`; `emptyFileSet+Define: PANIC runtime error: invalid memory address or nil pointer dereference`; `nilFileSet+Reserve: PANIC ...nil pointer dereference`; `uverse true: PANIC ...nil pointer dereference`. base nodes.go: the same three nil-pointer panics, and the Reserve shape panics `name outVar not defined in fileset with files []`. With judge-1-guard.patch: all four `ok fnNil=true`, `--- PASS: TestGetNameSourceForPathNoFilesetDecl`, `ok github.com/gnolang/gno/gnovm/pkg/repl 2.232s`. | tests/judge-1-namesource-shapes_test.go | warm |  |
| 2 | PLAUSIBLE | Suggestion | gnovm/pkg/gnolang/nodes.go:2085 | still to run: a caller of GetNameSourceForPath over file-built packages (none exists at head) under tests/judge-2-scoped-panic.patch, across the gnolang filetests, to show no file-built package-block name legitimately lacks a file decl | judge-2-scoped-panic.patch applied: `ok github.com/gnolang/gno/gnovm/pkg/repl 1.685s`; grep at head: GetNameSourceForPath has no caller outside nodes.go:1625 and getnamesource_test.go:45. | tests/judge-2-scoped-panic.patch | warm |  |
| 3 | CONFIRMED | Nit | gnovm/pkg/gnolang/getnamesource_test.go:20 | the PR's test shape with nil passed as the store and no TestStore, NewPackage, SetBlockNode or SetCachePackage (the first probe of tests/judge-1-namesource-shapes_test.go), run at head and with nodes.go reverted | head: `emptyFileSet+Reserve, nil store: ok fnNil=true type=2`; nodes.go reverted: `emptyFileSet+Reserve, nil store: PANIC name outVar not defined in fileset with files []`. | tests/judge-1-namesource-shapes_test.go | cold |  |

Hit rate per tier, from the rows above: hot 0/0 confirmed over 0 files, warm 1/2 confirmed over 1 files, cold 1/1 confirmed over 1 files.

### Settled by the finder, no verifier

| Angle | file:line | Suspected | Settled by |
| --- | --- | --- | --- |
| removed | gnovm/pkg/gnolang/nodes.go:2089 | Suspected: the else branch has the same class (it assumes the block's file is in the FileSet) and still panics for REPL blocks. | sed -n 1246,1253p nodes.go: GetFileByName loops over fs.Files and ends with `return nil`, so it returns nil on a miss and never panics. |
| removed | gnovm/pkg/gnolang/preprocess.go:191 | Suspected: lookupPredefineDecl's panicking fset.GetDeclFor is a sibling the swap missed. | sed -n 170,235p preprocess.go: lookupPredefineDecl and predefineDeclIndex.lookup run only while PredefineFileSet walks a file-built FileSet. A miss there is an undefined name, and the panic is intended (`a miss can never resolve a decl`). This is not the synthetic-package class. |
| lines | gnovm/pkg/gnolang/nodes.go:2085 | Suspected: hit-path results differ between GetDeclFor and GetDeclForSafe. | sed -n 1262,1269p nodes.go: `fn, decl, ok := fs.GetDeclForSafe(n); if !ok { panic(...) }; return fn, decl`. The hit path is identical. |
| catalog | gnovm/pkg/gnolang/nodes.go:2085 | Catalog walk: determinism, gas, realm state, caller/access, coin, storage deposit, global state, twin input paths, older data, VM-fault recoverability and VM semantics. None is touched: the helper is preprocess-only and has no caller on master. | grep -rn 'GetNameSourceForPath' --include='*.go' . at head: hits are only nodes.go:1625 (interface), nodes.go:2080 (impl) and getnamesource_test.go. Error & panic handling is covered by candidates 1 and 2. |

## Completeness

- Thin angle or bound: the reach angle came back with one candidate; the judge ran it over four shapes (empty FileSet with `Reserve`, empty FileSet with `Define`, nil FileSet, uverse) rather than the two the finder named, `tests/judge-1-namesource-shapes_test.go`.
- Sibling sweep by shape: every caller of the panicking `FileSet.GetDeclFor` and of `GetDeclForSafe` over the tree. `lookupPredefineDecl` (preprocess.go:191) and `predefineDeclIndex.lookup` (preprocess.go:235) resolve names of the FileSet being predefined, the index built from that FileSet's own decls (preprocess.go:203-217), so neither sees a synthetic package's block names. `GetTypeDeclForPath` asserts `Origin.(*TypeDecl)` unchecked, and its godoc names that panic as its contract. No candidate filed.
- Catalog: Error & panic handling is the class touched, covered by candidate 1. VM-fault recoverability: `GetNameSourceForPath` has no caller outside its interface (nodes.go:1625) and the new test at the head, so no user `.gno` reaches it. The other classes are untouched by a lookup helper's miss handling.
- PR body claims checked: the hit path is unchanged (`GetDeclFor` is `GetDeclForSafe` plus a panic on `!ok`, nodes.go:1262-1269); the new test is red with nodes.go reverted (`panic: name outVar not defined in fileset with files []`) and green at the head. The body's "REPL's exact package shape" holds for the assertions, but the store setup it lists is unread (candidate 3).
- Repro of candidate 1 run verbatim from the draft in the judge's worktree, the `gh pr checkout` line aside: `empty FileSet, Reserve: ok fnNil=true`, then the three `PANIC runtime error: invalid memory address or nil pointer dereference` lines, `FAIL`, as the draft's output block shows.

## Retro

- Failed: round 1 at the same sha stopped at the judge and wrote no draft; this round reran from scratch.
- Worked: 3 candidates, 2 confirmed with a run at head and merge base, 0 refuted.
- Cost: 2 agents, 22 min, about $3.92 measured against $5 projected.
