# Findings in posting order, from round assemble: 4 to post, 0 SKIP, 0 refuted kept out

## tm2/pkg/amino/deep_copy.go:111 [gh](https://github.com/gnolang/gno/blob/a742126999443305c6bfcf6fd6364ea57fba51d4/tm2/pkg/amino/deep_copy.go#L111) · Nit
State: CONFIRMED, band: Nit, angle: reach
TL;DR: slice elements now go through the Struct case, which skips unexported fields, so a copied []U{{y: 2}} reads y=0
Check: go test ./tm2/pkg/bft/consensus/types -run 'TestSoloSiblings|TestSoloTypeWalk' -v -count=1 at head and at 7916d1dd6; compare 'unexported field in slice element kept' and look for any UNEXP-UNDER-SLICE line | go test ./tm2/pkg/bft/consensus/types -run 'TestSoloSiblings|TestSoloTypeWalk' -v -count=1 at head and at 7916d1dd6; compare 'unexported field in slice element kept' and look for any UNEXP-UNDER-SLICE line
Details: Real delta: the merge base returned the source slice, the head returns element copies built by the Struct case, whose continue at deep_copy.go:123 skips unexported fields, as it always has for any struct DeepCopy reaches outside a slice and as amino encoding does. No caller type has a struct with unexported fields under a slice (type walk: no UNEXP-UNDER-SLICE line). The only fix is a sentence on DeepCopy's doc comment, so it ships SKIP.
Evidence: finder probe at head: 'unexported field in slice element kept: y=0'; at 7916d1dd6: 'y=2'; TestSoloTypeWalk at head: no UNEXP-UNDER-SLICE line
Artifact: tests/solo-finder-deepcopy-probe.go

## tm2/pkg/amino/deep_copy_test.go:172 [gh](https://github.com/gnolang/gno/blob/a742126999443305c6bfcf6fd6364ea57fba51d4/tm2/pkg/amino/deep_copy_test.go#L172) · Nit
State: UNVERIFIED, band: Nit, angle: lines, on the finder's read only
TL;DR: assert.Nil(t, amino.DeepCopy(nilInts)) never reaches the new guard and passes with it deleted
Check: delete lines 87-90 of deep_copy.go, go test ./tm2/pkg/amino -run TestDeepCopySlice: the nilInts assertion stays green, only the *cpyPtr assertion fails
Evidence: not run: on the finder's read

## tm2/pkg/amino/deep_copy.go:87 [gh](https://github.com/gnolang/gno/blob/a742126999443305c6bfcf6fd6364ea57fba51d4/tm2/pkg/amino/deep_copy.go#L87) · Suggestion
State: CONFIRMED, band: Suggestion, angle: removed
TL;DR: the nil guard covers only a nil slice behind a pointer; a nil map, pointer or interface behind one still copies wrong or panics
Check: copy tests/solo-finder-deepcopy-probe.go to tm2/pkg/bft/consensus/types/zz_solofinder_test.go; go test ./tm2/pkg/bft/consensus/types -run TestSoloSiblings -v -count=1; expect every nil to copy as nil, observe the map line and the two PANIC lines | copy tests/solo-finder-deepcopy-probe.go to tm2/pkg/bft/consensus/types/zz_solofinder_test.go; go test ./tm2/pkg/bft/consensus/types -run TestSoloSiblings -v -count=1; expect every nil to copy as nil, observe the map line and the two PANIC lines
Details: The Pointer case at deep_copy.go:57 calls _deepCopy(src.Elem(), cpy.Elem()) and so skips deepCopy's isNil check at line 41, which is the bypass the second commit names. The diff guards the Slice kind only (lines 87-90). A nil map behind a pointer comes back as an empty non-nil map; a nil *int behind a pointer recurses into the Pointer case, takes Elem() of a nil pointer and hits the default case with Kind Invalid; a nil interface behind a pointer calls src.Elem().Type() on a zero Value. Predates the diff, in scope because the diff patches one kind of this path. No production caller hands a nil header to BlockHeader, so the panic is latent. One isNil(src) return at the top of _deepCopy, with the Slice guard removed, copies all four as nil: tm2/pkg/amino/... all ok, and the tm2/pkg/sdk/... and tm2/pkg/bft/types/... suites exit 0 with it.
Evidence: judge-1-nil-behind-pointer.go at head a74212699: '*[]int: copy nil=true', '*map: copy nil=false', '**int: PANIC unsupported type invalid', '*any: PANIC reflect: call of reflect.Value.Type on zero Value'; finder probe at merge base 7916d1dd6: the same four plus 'sdk.Context{}.BlockHeader(): PANIC'; with judge-1-nil-guard.patch: all four 'copy nil=true', BlockHeader() returns <nil>
Artifact: tests/judge-1-nil-behind-pointer.go

## tm2/pkg/amino/deep_copy.go:136 [gh](https://github.com/gnolang/gno/blob/a742126999443305c6bfcf6fd6364ea57fba51d4/tm2/pkg/amino/deep_copy.go#L136) · Suggestion
State: CONFIRMED, band: Suggestion, angle: removed
TL;DR: the Map case stores each source value as is, so a map of slices or pointers still aliases its source after DeepCopy
Check: go test ./tm2/pkg/bft/consensus/types -run 'TestSoloSiblings|TestSoloTypeWalk' -v -count=1 with the probe copied in; expect 'map value aliased: false', observe true | go test ./tm2/pkg/bft/consensus/types -run 'TestSoloSiblings|TestSoloTypeWalk' -v -count=1 with the probe copied in; expect 'map value aliased: false', observe true
Details: deep_copy.go:136-137 read val := src.MapIndex(key) then cpy.SetMapIndex(key, val) with no deepCopy of val, the same write-through the diff removes from the Slice case. Predates the diff. No current caller reaches a map: the type walk over RoundState, Header, ConsensusParams and ConsensusConfig prints no MAP line. Copying each value through deepCopy into reflect.New(src.Type().Elem()).Elem() makes the probe print 'map value aliased: false' and keeps tm2/pkg/amino/... green.
Evidence: judge-2-map-values.go at head a74212699 and at 7916d1dd6: 'map value aliased: true'; with judge-2-map-values.patch: 'map value aliased: false', go test ./tm2/pkg/amino/... ok
Artifact: tests/judge-2-map-values.go
