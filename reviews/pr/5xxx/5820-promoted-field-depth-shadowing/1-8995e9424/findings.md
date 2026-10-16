# Findings in posting order, from round assemble: 4 to post, 1 SKIP, 0 refuted kept out

## gnovm/pkg/gnolang/types.go:902 [gh](https://github.com/gnolang/gno/blob/8995e9424b363e8fc70c1c285087300f9627f68a/gnovm/pkg/gnolang/types.go#L902) · Warning
State: CONFIRMED, band: Warning, angle: lines
TL;DR: A subtree's shallower candidate wins after its discarded deeper walk marked a type the parent embeds directly, so the selector silently runs a method Go does not pick
Check: go test -run 'TestFiles/solo-finder-1-wrongmethod.gno$' at head and base types.go; Go twin
Details: T{S; R}, S{C; X}, X{R}, C and R declaring M. Go runs R.M (depth 1). Inside S, line 902 keeps C.M and drops X.R.M as deeper, but R stays marked in m, so T's direct R returns nil and S.C.M at depth 2 is the only candidate. t.M() and i.M() both run C.M with no error, and go/types accepts the program. The base saw two matches in S and rejected.
Evidence: head Output diff '-R.M outer -R.M outer +C.M +C.M', no TypeCheckError; base: 'unexpected panic: main/solo-finder-1-wrongmethod.gno:39:10-13: missing field M in main.T'; Go twin: 'R.M outer' twice
Artifact: tests/solo-finder-1-wrongmethod.gno
State: CONFIRMED, band: Warning, angle: reach
TL;DR: Same mechanism as candidate 2 for fields too: T.V reads Sub.A.V where Go reads C.V
Check: go test -run 'TestFiles/solo-finder-2-shadowed-member-selected.gno$' at head and base types.go; Go twin
Details: T{Sub; C}, Sub{A; W}, W{C}. Go resolves T.M and T.V to C at depth 1. Head picks Sub.A at depth 2 for the method, the interface and the field. Same edit as candidate 2, anchored with it.
Evidence: head Output diff '+iface: A +method: A +field: a' against Go's C / C / c, no TypeCheckError; base: 'unexpected panic: main/solo-finder-2-shadowed-member-selected.gno:40:6-13: main.T does not implement main.I (missing method M)'; Go twin: 'iface: C method: C field: c'
Artifact: tests/solo-finder-2-shadowed-member-selected.gno
State: CONFIRMED, band: Nit, angle: catalog
TL;DR: Same pruning makes a selector Go rejects as ambiguous resolve to C.M; go/types rejects it first on chain
Check: go test -run 'TestFiles/solo-finder-1-falseaccept.gno$' at head and base types.go; Go twin
Details: T{A; B}, A{C; X}, X{D}, B{D}. Go: ambiguous selector t.M. Head runs C.M; base rejected at preprocess. Reached only where go/types does not run first.
Evidence: head: Output '+C.M' and TypeCheckError '+main/solo-finder-1-falseaccept.gno:39:12: ambiguous selector t.M'; base: 'unexpected panic: ... missing field M in main.T'; Go twin: 'ambiguous selector t.M'
Artifact: tests/solo-finder-1-falseaccept.gno

## gnovm/pkg/gnolang/types.go:907 [gh](https://github.com/gnolang/gno/blob/8995e9424b363e8fc70c1c285087300f9627f68a/gnovm/pkg/gnolang/types.go#L907) · Warning
State: CONFIRMED, band: Warning, angle: lines
TL;DR: A same-depth tie keeps walking into later siblings, marks their types in the shared set, and prunes the parent's shallower embed: a Go-valid program the base ran is rejected at preprocess
Check: go test -run 'TestFiles/solo-finder-1-tiewalk.gno$' in gnovm/pkg/gnolang at head and with base types.go; Go twin under go test
Details: T{S; R}, S{A; B; Z}, Z{R}. Go resolves T.M to R.M at depth 1. At line 907 the A/B tie only sets ambiguous, so S goes on into Z and marks R in m; T's direct R embed then returns nil at the recursion guard and T has no M. The base returned at the A/B conflict before Z, so R stayed unmarked. go/types accepts, so a package of this shape passes the type-check on addpkg and fails at VM preprocess.
Evidence: head: 'unexpected panic: main/solo-finder-1-tiewalk.gno:44:6-15: main.T does not implement main.I (missing method M)'; base types.go: PASS, prints R.M; Go twin via go test: 'R.M'
Artifact: tests/solo-finder-1-tiewalk.gno
State: CONFIRMED, band: Warning, angle: removed
TL;DR: Second shape of candidate 1: Amb{A; B; W}, W{C} beside T's direct C; the removed early return at the conflict kept C unmarked
Check: go test -run 'TestFiles/solo-finder-2-valid-program-rejected.gno$' at head and base types.go; Go twin
Details: T{Amb; C}, Amb{A; B; W}, W{C}. Go resolves T.M to C.M at depth 1. The ambiguous case at 907-909 lets Amb walk W, which marks C, and T's own C is pruned. Same edit as candidate 1, anchored with it.
Evidence: head: 'unexpected panic: main/solo-finder-2-valid-program-rejected.gno:42:10-15: missing field M in main.T'; base types.go: PASS; Go twin via go test: 'C'
Artifact: tests/solo-finder-2-valid-program-rejected.gno
State: PLAUSIBLE, band: Warning, angle: catalog
TL;DR: A realm of candidate 1's shape stored under the base binary faults on i.M() after the upgrade, since values.go:1962 re-runs the lookup at call time
Check: still to run: txtar addpkg of the tiewalk shape on a base-built gnoland, restart on head, gnokey maketx call a func doing i.M()
Details: values.go:1962-1966 calls findEmbeddedFieldType on the dynamic type for each interface method call and panics when the trail is empty. Not run: needs a base-built and a head-built gnoland over one home.
Evidence: read: values.go:1962 'tr, _, _, _, _ := findEmbeddedFieldType(callerPath, dtv.T, path.Name, nil)' then 'panic(fmt.Sprintf("method %s not found in type %s"'

## gnovm/tests/files/embed_method_shadow_internal_ambiguous.gno:24 [gh](https://github.com/gnolang/gno/blob/8995e9424b363e8fc70c1c285087300f9627f68a/gnovm/tests/files/embed_method_shadow_internal_ambiguous.gno#L24) · Missing test
State: CONFIRMED, band: Missing test, angle: reach
TL;DR: The internal-ambiguity fixture rejects only because Clean embeds the A the shared set already pruned; Clean{C} with its own M is accepted and runs C.M where Go rejects
Check: go test -run 'TestFiles/judge-1004-clean-own-type.gno$' at head and base types.go; Go twin
Details: Amb's ambiguity returns nil with no depth, so T takes Clean's C.M as the only candidate. Base behaves the same, so the defect predates the diff; the fixture's header states the general rule and pins only the dedup of A.
Evidence: head: 'unexpected output: C' against the expected 'does not implement' Error; base types.go: same file FAIL; Go twin: 'T does not implement I (ambiguous selector T.M)'
Artifact: tests/judge-1004-clean-own-type.gno

## gnovm/pkg/gnolang/types.go:896 [gh](https://github.com/gnolang/gno/blob/8995e9424b363e8fc70c1c285087300f9627f68a/gnovm/pkg/gnolang/types.go#L896) · Nit
State: CONFIRMED, band: Nit, angle: reach
TL;DR: A pruned revisit returns nil like a miss, so a diamond's one surviving copy beats a deeper member and the VM runs a selector Go rejects
Check: go test -run 'TestFiles/solo-finder-2-diamond-ambiguity-accepted.gno$' at head and base types.go; Go twin
Details: T{A; B; C}, A{Base}, B{Base}, C{D}, D{E}. Go: Base.M twice at depth 2, ambiguous. Head: B's Base is pruned, trail2 == nil continues at line 896, Base.M at depth 2 beats E.M at depth 3, prints base. Base counted Base.M and E.M as a conflict and rejected. go/types rejects first on chain.
Evidence: head: output 'base', TypeCheckError '+main/solo-finder-2-diamond-ambiguity-accepted.gno:38:14: ambiguous selector T{}.M'; base: 'unexpected panic: ...:38:10-15: missing field M in main.T'; Go twin: 'ambiguous selector T{}.M'
Artifact: tests/solo-finder-2-diamond-ambiguity-accepted.gno

## SKIP gnovm/pkg/gnolang/types.go:900 [gh](https://github.com/gnolang/gno/blob/8995e9424b363e8fc70c1c285087300f9627f68a/gnovm/pkg/gnolang/types.go#L900) · Nit
State: PLAUSIBLE, band: Nit, angle: claims
TL;DR: cand is built for every matching embed, discarded ones included, against the ADR's 'no extra allocation'
Check: still to run: testing.AllocsPerRun over StructType.FindEmbeddedFieldType for a struct embedding two types that both promote M, head against base
Details: Line 900 runs append before the switch; the base built the slice only for the first match. Magnitude not measured.
Evidence: read: types.go:900 'cand := append([]ValuePath{vp}, trail2...)' precedes the switch at 901
