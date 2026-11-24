## Findings

| Line | # | Band | State | Merge base | Draft |
| --- | --- | --- | --- | --- | --- |
| `preprocess.go:1057` | 1 | Nit | CONFIRMED | differs | posts |
| `nodes.go:2470` | 2 | Suggestion | REFUTED |  |  |

### 1 · Nit · CONFIRMED · `gnovm/pkg/gnolang/preprocess.go:1057`

defineSwitchVar writes the type switch variable at numFauxCopiedNames(), a slot `_` never gets, so a blank type switch variable now fails preprocess with an internal error where the merge base ran it

- Check: Add tests/solo-finder-blank-tsvar.gno as gnovm/tests/files/blank_tsvar.gno and run `go test ./gnovm/pkg/gnolang -run TestFiles/blank_tsvar.gno`: expect the Error `faux copy slot 0 holds x, not _` at head. Fix shape: skip defineSwitchVar when ss.VarName == blankIdentifier, or reject it in preprocess with Go's message
- Observed: go test -v -run TestJudgeBlankTSVar ./gnovm/pkg/repl/ (tests/judge-1-blank-tsvar-repl_test.go). Head a40c1fbf7: blank-body err=":1:31-73: faux copy slot 0 holds x, not _"; blank-empty err="runtime error: index out of range [0] with length 0"; blank-stmt err="runtime error: index out of range [0] with length 0"; named out="named 3 42". Base a6fa06145: blank-body out="blank-body 3"; blank-empty out="blank-empty ok"; blank-stmt the same index out of range; named the same. Filetest route (tests/solo-finder-blank-tsvar.gno, tests/judge-1-blank-tsvar-empty.gno): head and base both `unexpected panic: ...:9:9: no new variables on left side of :=` from the parser. With tests/judge-1-fix.patch: all four REPL shapes run; go test -run 'TestFiles/(typeswitch.*\|switch.*\|if9\|redefine.*)\.gno$' ok.
- Test: [`tests/judge-1-blank-tsvar-repl_test.go`](tests/judge-1-blank-tsvar-repl_test.go)

### 2 · Suggestion · REFUTED · `gnovm/pkg/gnolang/nodes.go:2470`

Both defineNew callers already refuse the duplicate the debugAssert check guards against

- Check: grep every defineNew caller and confirm each one returns before appending a name already at or past the boundary
- Refuted by: nodes.go:2313 `if idx >= sb.numFauxCopiedNames() { return }` ahead of the only Reserve append

## Cleared by the finder

No verifier ran these: the finder's own read or run settled each.

| Line | Angle | Suspected | Cleared by |
| --- | --- | --- | --- |
| `nodes.go:1908` | user topic: GetIsConstAt with a const shadow in a faux case block | A use or assignment of the outer name before, after, in a closure, in a nested if or block, and through fallthrough might resolve against the const shadow | gno run tests/solo-finder-const-shadow-paths.gno at head prints `closure-before 2 c / closure-after c / nested-if c / inner-before c / inner-after 7 / fell 2 / fell2 9`, the same as go run. The if/else form (`5 c`) matches too. |
| `nodes.go:2312` | removed | The removed NSTypeSwitch guard in Reserve might have been the only stop for a clause body redeclaring the type switch variable | grep NSTypeSwitch: the only setter is preprocess.go:641, on a clause block, after the copy loop at :637. Copies take the faux parent's NameSources (:591, :637), and a SwitchStmt or IfStmt never holds NSTypeSwitch, so the guard was unreachable for idx < numFauxCopiedNames(). The variable sits at numFauxCopiedNames(), which takes the `idx >=` return. |
| `nodes.go:2356` | user topic: defineFauxCopy at node boot | Node boot re-preprocess of stored packages might see a case block whose copied names do not line up with the parent, now a panic in every build | machine.go:343 `fset := m.ParseMemPackage(mpkg)` in PreprocessAllFilesAndSaveBlockNodes parses fresh nodes, so the copy loop always lays out the parent's names first in order. Only blank tsvar (candidate 1) breaks alignment, and typecheck rejects it. |
| `typeswitch11.gno:16` | reach: sibling shapes | A type switch variable shadowing an init name, or a clause shadow captured by closures (heap items) in a loop, might pick the wrong slot | gno run tests/solo-finder-typeswitch-shadow-closures.gno at head prints `5 t 42 / ts 7 / 0 0 / 2 10`, the same as go run |
| `nodes.go:2309` | user topic: master Reserve guards ahead of the faux-copy branch | The iota refusal or blank return placed before GetLocalIndex might skip or wrongly refuse a shadow of a faux-copied slot | Base nodes.go:2311 carries the same iota guard ahead of everything. Blank never gets a slot in the parent either: base Define2 ignores `_`, head nodes.go:2309 returns. A `_` therefore never occupies a copied slot, and only an iota declaration is refused, as on master. |
| `pr6058_faux_block_shadowing.md:168` | claims | ADR: on master the case-body const form is rejected with `StaticBlock.Define2(v) cannot change const status` | Base read: the copy loop reserves `v` non-const via Define2; base Reserve no-ops on the existing name for `const v`; base copyFromFauxBlock (preprocess.go:4275) calls bn.Define(n); the const decl's Define2(true, v) then hits nodes.go `isConst != sb.getLocalIsConst(n)` and panics with that message |
| `pr6058_faux_block_shadowing.md:118` | claims | ADR: the fallthrough truncation landed separately in #6056 | git -C <base> log --grep 6056 returns `0cf310707 fix(gnovm): drop clause-local names on switch fallthrough (#6056)`, which is in the merge base; head op_exec.go:767 `b.Values = b.Values[:ss.GetNumNames()]` |
| `nodes_test.go:325` | claims | ADR: putting buildNameIndex back to first-wins fails TestStaticBlock_GetLocalIndex_LastWinsPastThreshold at width 40 | Read: width 40 > nameIndexThreshold (32) builds nameIndex lazily in GetLocalIndex (nodes.go:1956). First-wins maps `v` to 0, and `assert.Equal(t, uint16(3), idx)` fails. Width 20 takes the reverse linear scan and pins that branch. |
| `switch52.gno:24` | reach: type-decl shadow before use | A `type T` shadow in a case body might be predefined ahead of a prior use of the outer `T`, which then resolves to the type | gno run at head fails the case form with `expected type expr but got *gnolang.SwitchClauseStmt`, and the plain nested-block form (`T := 1; { println(T); type T string }`) fails identically with `*gnolang.BlockStmt`. The behaviour predates the PR, is shared with ordinary blocks, and is outside this round's diff. |

## Record

<details><summary>One row per candidate, the table the round tools parse</summary>

| # | State | Band | file:line | Check | Tier | Mechanism |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Nit | gnovm/pkg/gnolang/preprocess.go:1057 | Add tests/solo-finder-blank-tsvar.gno as gnovm/tests/files/blank_tsvar.gno and run `go test ./gnovm/pkg/gnolang -run TestFiles/blank_tsvar.gno`: expect the Error `faux copy slot 0 holds x, not _` at head. Fix shape: skip defineSwitchVar when ss.VarName == blankIdentifier, or reject it in preprocess with Go's message |  |  |
| 2 | REFUTED | Suggestion | gnovm/pkg/gnolang/nodes.go:2470 | grep every defineNew caller and confirm each one returns before appending a name already at or past the boundary |  |  |

</details>

## Completeness

- Catalog: the classes this diff touches are VM semantics vs Go and Type-check & preprocess; row 1 is the preprocess one. Determinism, gas, realm state, caller identity, coins, storage deposit, global state, twin inputs and stored data are not reached by these five files.
- VM-fault recoverability: the two panics row 1 reaches, a string panic and a Go index out of range during preprocess, are recovered by the REPL's `RunStatements` and by `doRecoverQuery` on `qeval` (keeper.go:2048, read, not run). `addpkg`, node boot and filetests never reach them, since `ParseFile` parses with `DeclarationErrors` and the resolver rejects `_ :=` (parser/resolver.go:183), at head and base alike.
- Sibling sweep by shape: `defineFauxCopy` has two callers, preprocess.go:1056 and :4269; the copy loop at :4269 indexes names the parent holds, and a parent never holds `_`. The other `debugAssert` guard on the duplicate-slot invariant, in `defineNew`, is row 2, refuted: both callers refuse the duplicate in every build.
- Fix run for row 1, not posted since its edit lands outside the diff: tests/judge-1-fix.patch maps `_` to an empty `VarName` in go2gno.go's TypeSwitchStmt case; all four REPL shapes then run, and `TestFiles/(typeswitch.*|switch.*|if9|redefine.*)` stays green.
- Round 1's seven findings: six fixed in a40c1fbf7 per the finder's dropped rows (typeswitch11.gno, the threshold test, the ADR paragraphs); the seventh moved to master with gnolang/gno#6056 and leaves this diff.

## Retro

- What failed: nothing died and no cap was hit. The parent's own re-run of row 1 at the head could not link (`no space left on device`, 118M free on `/`), so the head half rests on the judge's run and a read of `defineSwitchVar` and `Reserve`; the base half was re-run and matched.
- What worked: the three topics, each cleared by a run the finder made (`GetIsConstAt` paths, the merged `Reserve` guards, the unconditional `defineFauxCopy` check at node boot).
- Hit rate: 1 confirmed of 1 hot-tier row, 0 of 1 reflector row; the four cold files produced none.
- Upgrade: `review-setup.sh` read the PR number off the trailing digit of `--target 'PR 6058 round 2'` and set up against #2; a target text ending in a digit needs the number passed as `--target <n>`. Logged, not changed mid-task.
- Cost: 2 agents, 57 turns, 17 min, about $5.93 at the rates read 2026-09-24.
