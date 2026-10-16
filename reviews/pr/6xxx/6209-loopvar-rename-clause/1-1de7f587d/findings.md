# Findings in posting order, from round assemble: 4 to post, 0 SKIP, 3 refuted kept out

## Body

- Warning: gnovm/pkg/gnolang/preprocess.go:344 [gh](https://github.com/gnolang/gno/blob/1de7f587da469abad9ce4cf5fa945f3328aa0e40/gnovm/pkg/gnolang/preprocess.go#L344), outside the diff, so no anchor
  State: CONFIRMED, band: Warning, angle: removed
  TL;DR: Inside a loop declaring x, a func literal whose result is named x reads its result type [len(x)]int against the outer x and returns the wrong array length with no error
  Check: cp tests/solo-finder-2-funclit-type-mask.gno gnovm/tests/files/zz_funclit_type_mask.gno; go test -short -run 'TestFiles/zz_funclit_type_mask.gno$' ./gnovm/pkg/gnolang at head, at the base, and with tests/solo-finder-2-funclit-type-mask.patch applied
  Details: The FuncLitExpr frame is pushed at TRANS_ENTER with its params and results deleted, so the NameExprs in the FuncTypeExpr (walked before the body) no longer see the loop's x and keep the bare name, which binds to the plain x declared before the loop. Go resolves a signature's types in the enclosing scope. The param form `func(x [len(x)]int)` is refused with `name x not declared` at head and base. Predates the branch; in scope because the branch reworks exactly this timing (a construct's own names masked before the clauses Go resolves outside them) for ForStmt, RangeStmt and SwitchStmt and leaves FuncLitExpr on the early push. Not a disclosure: only a realm's own source triggers it, nothing an outside caller supplies. Moving the FuncLitExpr push to TRANS_BLOCK makes the fixture print 2, keeps the six new loopvar fixtures green and TestFiles/(loopvar|closure|heap|fun|range|typeswitch|for|var|block|shadow).* ok. The param variant was not rerun under the patch.
  Evidence: head: 'Output diff: @@ -1 +1 @@ -2 +3', FAIL; base (preprocess.go+transcribe.go at bbd9b2ffe): same '-2 +3'; head+patch: '--- PASS: TestFiles/zz_funclit_type_mask.gno', subset run 'ok github.com/gnolang/gno/gnovm/pkg/gnolang 1.814s'; param variant head and base: 'main/zz_judge_paramlen.gno:5:20-21: name x not declared'
  Artifact: tests/solo-finder-2-funclit-type-mask.gno
- Nit: gnovm/pkg/gnolang/preprocess.go:354 [gh](https://github.com/gnolang/gno/blob/1de7f587da469abad9ce4cf5fa945f3328aa0e40/gnovm/pkg/gnolang/preprocess.go#L354), outside the diff, so no anchor
  State: CONFIRMED, band: Nit, angle: claims
  TL;DR: The second initStaticBlocks1 pass still allocates a frame map for every FuncDecl and FuncLitExpr, about two allocations per function, where the ADR says it allocates none
  Check: copy tests/judge-2-second-pass-allocs_test.go into gnovm/pkg/gnolang and run go test -short -run 'TestZZJudgeSecondPassAllocs$' at head and with tests/judge-2-fastpath.patch applied
  Details: pushClone takes the fast path only for `modify == nil && len(parent) == 0`; FuncDecl and FuncLitExpr always pass a delete-only modify, so `f := map[Name]bool{}` runs for each even when the parent frame is empty. At head the only non-nil modify callers are these two and both only delete, so `if len(parent) == 0` is equivalent and removes the growth. Preprocessing allocations are not gas-metered; this is cost, not behaviour.
  Evidence: head: funcs=1 allocs=7, funcs=10 allocs=25, funcs=100 allocs=206; with `if len(parent) == 0`: 3, 3, 4; TestFiles/(loopvar|closure|heap|fun|range|typeswitch|for|var|block|shadow).* 'ok ... 1.924s' under the patch
  Artifact: tests/judge-2-second-pass-allocs_test.go

## gnovm/pkg/gnolang/preprocess.go:377 [gh](https://github.com/gnolang/gno/blob/1de7f587da469abad9ce4cf5fa945f3328aa0e40/gnovm/pkg/gnolang/preprocess.go#L377) · Suggestion
State: CONFIRMED, band: Suggestion, angle: two versions
TL;DR: Four loop shapes that preprocess and run on the base with a wrong binding run with Go's binding at head, so a stored package using one computes differently once a node re-preprocesses it on the new binary
Check: bash tests/solo-finder-1-formerly-compiling-shapes.sh bbd9b2ffe87c9a295ccfd6dfb45224bb02c82d8a, then the same at 1de7f587da469abad9ce4cf5fa945f3328aa0e40; sed -n '162,163p' gnovm/pkg/gnolang/store.go; sed -n '203p' gno.land/pkg/sdk/vm/keeper.go
Details: Where a plain variable of the loop variable's name is in scope, the base binds the clause to it instead of failing: range over make([]int, x) loops 0..99 twice instead of 0, 0 1; switch x := x.(type) takes the default branch with "outer"; for k = range writes the outer k; for x := x between two shadows reads the outer loop's x. Head gives Go's result for all four. BlockNodes are not persisted (store.go:162-163) and the keeper re-preprocesses every stored package at boot (keeper.go:203), so the change applies to code already on chain from the first block on the new binary. The new behaviour is the correct one; what is open is whether deployed packages carry these shapes.
Evidence: base: four '--- FAIL' with diffs '+0 +1 +2 +3 +4  5', '-int 1 +other outer', '@@ -1,3 +1,200 @@', '+0  1 --1'; head: four '--- PASS', 'ok github.com/gnolang/gno/gnovm/pkg/gnolang 0.038s'; store.go:162-163 'BlockNodes are not stored in the underlying DB and must be re-initialized using PreprocessAllFilesAndSaveBlockNodes'; keeper.go:203 'm2.PreprocessAllFilesAndSaveBlockNodes()'
Artifact: tests/solo-finder-1-formerly-compiling-shapes.sh

## gnovm/pkg/gnolang/preprocess.go:398 [gh](https://github.com/gnolang/gno/blob/1de7f587da469abad9ce4cf5fa945f3328aa0e40/gnovm/pkg/gnolang/preprocess.go#L398) · Suggestion
State: CONFIRMED, band: Suggestion, angle: removed
TL;DR: for x := x between a plain shadow and an outer loop x prints 0..6 on the base and 5 6 at head
Check: cp tests/solo-finder-2-shadow-between-output.gno gnovm/tests/files/zz_shadow_between.gno; go test -short -run 'TestFiles/zz_shadow_between.gno$' ./gnovm/pkg/gnolang at the base and at head
Details: Same program as the shadow_between case of candidate 1, which carries it; one section covers both.
Evidence: base: 'Output diff: @@ -1 +1,6 @@ +0 +1 +2 +3 +4  5'; head: '--- PASS: TestFiles/zz_shadow_between.gno'
Artifact: tests/solo-finder-2-shadow-between-output.gno
