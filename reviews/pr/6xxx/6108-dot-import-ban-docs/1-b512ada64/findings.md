# Findings in posting order, from round assemble: 1 to post, 0 SKIP, 0 refuted kept out

## gnovm/pkg/gnolang/preprocess.go:5605 [gh](https://github.com/gnolang/gno/blob/b512ada646e132543f32d48631c8e1bf40a4d1d9/gnovm/pkg/gnolang/preprocess.go#L5605) · Nit
State: CONFIRMED, band: Nit, angle: removed
TL;DR: tryPredefine's dot-import branch is never reached, so its reworded message is untested
Check: tests/solo-trypredefine-unreachable.sh: sentinel at preprocess.go:5606 keeps TestFiles/import2.gno green, sentinel at :491 turns it red, base preprocess.go passes
Details: initStaticBlocks (preprocess.go:250) runs initStaticBlocks1 then initStaticBlocks2, and both callers of the predefine walk run it first: PredefineFileSet at :43 before predefining imports, Preprocess at :750 before the file's decls reach predefineRecursively (:876, :1220-1270). initStaticBlocks2 panics on nn == "." at :491. The branch is pre-existing; the diff edits its message and comment, which is why it is in scope. Keeping it as a guard is defensible; the loss is only that the aligned message is not pinned.
Evidence: sentinel at :5606 -> ok; sentinel at :491 -> +main/import2.gno:3:8-19: SENTINEL-isb2, FAIL; merge-base preprocess.go with the new filetest -> ok
Artifact: tests/solo-trypredefine-unreachable.sh
