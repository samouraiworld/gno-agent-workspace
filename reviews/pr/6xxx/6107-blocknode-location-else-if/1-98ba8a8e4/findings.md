# Findings in posting order, from round assemble: 2 to post, 0 SKIP, 0 refuted kept out

## gnovm/pkg/gnolang/preprocess.go:43 [gh](https://github.com/gnolang/gno/blob/98ba8a8e4a1981b9aad23c5487387300d6eda15e/gnovm/pkg/gnolang/preprocess.go#L43) · Suggestion
State: CONFIRMED, band: Suggestion, angle: reach
TL;DR: the pkg-path and file-name branches of checkNodeLinesLocations never fire from either caller, and the call at line 43 re-checks what line 42 just built
Check: read preprocess.go:41-43 and 812-817 against 6436-6500: same arguments stamped then compared; plant panic in the two path branches and run go test ./gnovm/pkg/gnolang -run TestFiles -parallel 2: expect no hit
Details: preprocess.go:42-43 pass the same pn.PkgPath and fn.FileName to setNodeLocations and to the check. At preprocess.go:767-772 the FileNode branch passes packageOf(ctx).PkgPath and fn.FileName, the values preprocess1 stamped at preprocess.go:812-816; every other node passes the placeholders, which skip both branches. Sentinel panics at the top of both branches (6490, 6494) never fired over TestFiles.
Evidence: panic("SENTINEL-PKG") and panic("SENTINEL-FILE") planted in both branches: GOGC=50 go test -p 1 -parallel 1 -run TestFiles: ok 131.644s, no SENTINEL in output; a -parallel 2 run OOMed under ulimit -v (environment)

## gnovm/pkg/gnolang/preprocess.go:6458 [gh](https://github.com/gnolang/gno/blob/98ba8a8e4a1981b9aad23c5487387300d6eda15e/gnovm/pkg/gnolang/preprocess.go#L6458) · Suggestion
State: CONFIRMED, band: Suggestion, angle: lines
TL;DR: the Num=1 mark on the else-if wrapper is redundant: the setNodeLocations counter alone makes every Location unique, and without the mark the wrapper keeps its merge-base Location
Check: run tests/solo-finder-location-delta.go at head and merge base per its header; then revert go2gno.go:596-603 at head and rerun: expect duplicate_locations=0 still
Details: setNodeLocations (preprocess.go:6458) assigns max(nextNum[key], span.Num) and bumps nextNum per (Pos,End), so nodes sharing a span get distinct Nums whatever Go2Gno sets. With go2gno.go restored to the merge base inside the head tree: 0 duplicates over examples/, gnovm/stdlibs and gnovm/tests/files, the PR's two tests PASS, TestFiles ok. With the mark 254 wrappers 0->1, 254 IfStmts 0->2, 136 empty elses 1->3; without it the wrappers keep Num 0, 254 IfStmts 0->1 and 136 empty elses 1->2. No persisted state keys these Locations today: the backend node write is commented out (store.go:993-997), nodes are rebuilt at start by PreprocessAllFilesAndSaveBlockNodes (keeper.go:203), DeclaredType.ParentLoc comes only from FuncDecl/FuncLitExpr (types.go:1517-1520), closures carry Parent nil (op_expressions.go:711). The cost is a second mechanism, not a break.
Evidence: head: files=4253 blocknodes=66204 duplicate_locations=0; base: duplicate_locations=254; head with base go2gno.go: duplicate_locations=0, TestBlockNodeLocationUniqueElseIf and TestBlockNodeLocationUniqueEmptyElse PASS, go test -p 1 -parallel 1 -run TestFiles: ok 131.644s; base->no-mark Num moves: 254 IfStmt 0->1, 136 IfCaseStmt 1->2, wrapper unchanged
Artifact: tests/solo-finder-location-delta.go
