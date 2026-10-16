# Claims: https://github.com/gnolang/gno/pull/5820 round 1, 8995e9424, claude-opus-5-5, solo review

Round shape: solo round, 2 finders and one judge and writer, every Critical and Warning run, the rest read

## Candidates

| # | State | Band | file:line | Check | Observed | Artifact | Tier | Mechanism |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Warning | gnovm/pkg/gnolang/types.go:907 | go test -run 'TestFiles/solo-finder-1-tiewalk.gno$' in gnovm/pkg/gnolang at head and with base types.go; Go twin under go test | head: 'unexpected panic: main/solo-finder-1-tiewalk.gno:44:6-15: main.T does not implement main.I (missing method M)'; base types.go: PASS, prints R.M; Go twin via go test: 'R.M' | tests/solo-finder-1-tiewalk.gno | hot | shared-visited-set-pruning |
| 2 | CONFIRMED | Warning | gnovm/pkg/gnolang/types.go:902 | go test -run 'TestFiles/solo-finder-1-wrongmethod.gno$' at head and base types.go; Go twin | head Output diff '-R.M outer -R.M outer +C.M +C.M', no TypeCheckError; base: 'unexpected panic: main/solo-finder-1-wrongmethod.gno:39:10-13: missing field M in main.T'; Go twin: 'R.M outer' twice | tests/solo-finder-1-wrongmethod.gno | hot | discarded-walk-prunes-shallower-embed |
| 3 | CONFIRMED | Nit | gnovm/pkg/gnolang/types.go:902 | go test -run 'TestFiles/solo-finder-1-falseaccept.gno$' at head and base types.go; Go twin | head: Output '+C.M' and TypeCheckError '+main/solo-finder-1-falseaccept.gno:39:12: ambiguous selector t.M'; base: 'unexpected panic: ... missing field M in main.T'; Go twin: 'ambiguous selector t.M' | tests/solo-finder-1-falseaccept.gno | hot | discarded-walk-prunes-shallower-embed |
| 4 | PLAUSIBLE | Nit | gnovm/pkg/gnolang/types.go:900 | still to run: testing.AllocsPerRun over StructType.FindEmbeddedFieldType for a struct embedding two types that both promote M, head against base | read: types.go:900 'cand := append([]ValuePath{vp}, trail2...)' precedes the switch at 901 |  | hot |  |
| 1001 | CONFIRMED | Warning | gnovm/pkg/gnolang/types.go:902 | go test -run 'TestFiles/solo-finder-2-shadowed-member-selected.gno$' at head and base types.go; Go twin | head Output diff '+iface: A +method: A +field: a' against Go's C / C / c, no TypeCheckError; base: 'unexpected panic: main/solo-finder-2-shadowed-member-selected.gno:40:6-13: main.T does not implement main.I (missing method M)'; Go twin: 'iface: C method: C field: c' | tests/solo-finder-2-shadowed-member-selected.gno | hot | discarded-walk-prunes-shallower-embed |
| 1002 | CONFIRMED | Warning | gnovm/pkg/gnolang/types.go:907 | go test -run 'TestFiles/solo-finder-2-valid-program-rejected.gno$' at head and base types.go; Go twin | head: 'unexpected panic: main/solo-finder-2-valid-program-rejected.gno:42:10-15: missing field M in main.T'; base types.go: PASS; Go twin via go test: 'C' | tests/solo-finder-2-valid-program-rejected.gno | hot | shared-visited-set-pruning |
| 1003 | CONFIRMED | Nit | gnovm/pkg/gnolang/types.go:896 | go test -run 'TestFiles/solo-finder-2-diamond-ambiguity-accepted.gno$' at head and base types.go; Go twin | head: output 'base', TypeCheckError '+main/solo-finder-2-diamond-ambiguity-accepted.gno:38:14: ambiguous selector T{}.M'; base: 'unexpected panic: ...:38:10-15: missing field M in main.T'; Go twin: 'ambiguous selector T{}.M' | tests/solo-finder-2-diamond-ambiguity-accepted.gno | hot |  |
| 1004 | CONFIRMED | Missing test | gnovm/tests/files/embed_method_shadow_internal_ambiguous.gno:24 | go test -run 'TestFiles/judge-1004-clean-own-type.gno$' at head and base types.go; Go twin | head: 'unexpected output: C' against the expected 'does not implement' Error; base types.go: same file FAIL; Go twin: 'T does not implement I (ambiguous selector T.M)' | tests/judge-1004-clean-own-type.gno | cold |  |
| 2001 | PLAUSIBLE | Warning | gnovm/pkg/gnolang/types.go:907 | still to run: txtar addpkg of the tiewalk shape on a base-built gnoland, restart on head, gnokey maketx call a func doing i.M() | read: values.go:1962 'tr, _, _, _, _ := findEmbeddedFieldType(callerPath, dtv.T, path.Name, nil)' then 'panic(fmt.Sprintf("method %s not found in type %s"' |  | hot | shared-visited-set-pruning |

Hit rate per tier, from the rows above: hot 6/8 confirmed over 1 files, warm 0/0 confirmed over 1 files, cold 1/1 confirmed over 6 files.

### Settled by the finder, no verifier

| Angle | file:line | Suspected | Settled by |
| --- | --- | --- | --- |
| claims | gnovm/pkg/gnolang/types.go:888 | comment 'trail length is the depth' might not hold through pointer or declared-type layers | sed -n 649,735p, 1037,1080p and 2185,2235p of types.go: StructType prepends one VPField per embedded step; PointerType only retypes trail[0] (VPDerefField, SetDepth(2)); DeclaredType only calls SetDepth(+1) or returns a single method path; InterfaceType returns a single VPInterface. Trail length is embedded steps + 1, uniformly |
| claims | gnovm/adr/pr5820_embedded_method_depth_shadowing.md:131 | ADR test claims: shadow0/1/2 panic at base, the other three pass at base | go test -run 'TestFiles/embed_method_shadow' with base types.go: 'unexpected panic: ... embed_method_shadow0.gno:28:6-13: main.T does not implement main.I (missing method M)', same for shadow1 and shadow2; '--- PASS' for _ambiguous, _fieldmethod and _internal_ambiguous; all six PASS at head |
| claims | gnovm/adr/pr5820_embedded_method_depth_shadowing.md:30 | on-chain addpkg type-checks with go/types before the VM preprocesses | grep TypeCheckMemPackage gno.land/pkg/sdk/vm/keeper.go: '647: _, err = gno.TypeCheckMemPackage(memPkg, opts)' in AddPackage (583) and '1030: _, err = gno.TypeCheckMemPackage(memPkg, ...' in Run (1003) |
| reach | gnovm/pkg/gnolang/types.go:902 | trail length might not equal promotion depth through a pointer, a named struct or an interface embed, so a same-depth tie could read as shadowing | Read types.go:688-722 (the PointerType rewrite changes trail[i].Type and trail[0].Depth in place and never adds an element), 2232 (DeclaredType does trail[0].SetDepth(Depth+1) without appending), 1060 (an interface method is one VPInterface element whatever the embedding depth): length is one element per embedded hop plus one, on every path |

## Completeness

- Catalog: determinism holds, `m` is a set and never iterated. Gas: the walk now continues past a tie, still bounded by `MaxEmbedDepth` (types.go:1602). VM semantics vs Go: candidates 1, 2, 3, 1001, 1002, 1003. Type-check and preprocess: 1004. Data an older binary wrote: 2001, PLAUSIBLE, a realm stored under the base faulting at values.go:1962 after the upgrade, its txtar still to run.
- Siblings swept by shape: `PointerType` and `DeclaredType` `FindEmbeddedFieldType` delegate to the changed function; `InterfaceType.FindEmbeddedFieldType` (types.go:1037) keeps first match with no depth, and Go interface method sets carry no depth, so no sibling repeats the edit.
- Tests the diff adds, run at the merge base's types.go: `embed_method_shadow0/1/2` panic `missing method M`, the three guard fixtures pass, as the ADR says; all six pass at head.
- Draft repros `tests/shadow_wrong_method.gno` and `tests/shadow_tie_walk.gno` run at head and base: head `-R.M outer +C.M c` and `main.T does not implement main.I (missing method M)`; base `missing method M` and `ok`. Go twin of the first prints `R.M outer`.
- The Missing test case in the draft is derived from `tests/judge-1004-clean-own-type.gno` with its header cut; its `Error` and `TypeCheckError` positions follow the existing fixture's and were not run under that name.
- ADR claims refuted: "zero-regression" by candidates 1 and 1002; "no extra allocation" by the read in candidate 4, magnitude not measured.

## Retro

- Failed: nothing died, no refusal, no turn on another model; one Warning (2001) left PLAUSIBLE, its txtar upgrade check not run, so it stays out of the draft.
- Worked: both finders reached the two Warnings independently (rows 1/1002, 2/1001), each confirmed against a Go twin and the merge base.
- Cost: 3 agents, 26 min, about $6.38 measured; finder 1 26 turns, finder 2 19, judge 20.
