# Findings in posting order, from round assemble: 2 to post, 0 SKIP, 1 refuted kept out

## gno.land/pkg/sdk/vm/convert.go:216 [gh](https://github.com/gnolang/gno/blob/a5370a166b459689a3febf222f3a5edd8739058e/gno.land/pkg/sdk/vm/convert.go#L216) · Warning
State: CONFIRMED, band: Warning, angle: lines
TL;DR: a []byte call argument reaches the realm with cap = base64 DecodedLen(len(arg)), so padding the encoding with newlines grows cap, the persisted array and the alloc charge
Check: cp tests/solo-finder-1-slice-arg-cap.go gno.land/pkg/sdk/vm/zz_finder1_capcall_test.go && go test ./gno.land/pkg/sdk/vm -run TestFinder1SliceArgCapCall -count=1 -v -timeout 2m at a5370a166 and 6c49f5420
Details: base64.StdEncoding.DecodeString returns dbuf[:n] over a DecodedLen-sized buffer; NewSliceFromData (alloc.go:679) charges cap(data), takes data[:cap(data)] as the base array and sets Maxcap = cap(data). The other caller, values_conversions.go:1043, clips data[:len(data):len(data)] before the call; convert.go:216 does not. Clipping bz the same way brings cap back to len at head.
Evidence: head: arg len 4 -> cap(b) = (3 int); arg len 404 -> cap(b) = (303 int). base: (1 int) for both. Head with tests/judge-fixes.patch (bz[:len(bz):len(bz)]): (1 int) for both.
Artifact: tests/solo-finder-1-slice-arg-cap.go
State: CONFIRMED, band: Warning, angle: reach
TL;DR: one decoded byte string arrives with a cap chosen by ignored \r\n in its encoding, and a realm that stores it persists the slack
Check: cp tests/solo-finder-2-bytes-arg-cap-malleable_test.go gno.land/pkg/sdk/vm/zz_sf2cap_test.go && go test ./gno.land/pkg/sdk/vm -run TestSF2BytesArgCapMalleable -count=1 -v -timeout 2m at a5370a166 and 6c49f5420
Details: Same line and same edit as row 1, measured through a realm that stores the slice: len stays 3 while cap moves 3, 6, 15 with 0, 4 and 16 bytes of line breaks. The merge base built the SliceValue with Maxcap: len(bz).
Evidence: head: "AAAA" -> (3 int)(3 int); "AAAA\n\n\n\n" -> (3 int)(6 int); "AAAA"+8x"\r\n" -> (3 int)(15 int). base: (3 int)(3 int) for all three. Head with tests/judge-fixes.patch: (3 int)(3 int) for all three.
Artifact: tests/solo-finder-2-bytes-arg-cap-malleable_test.go

## gnovm/pkg/gnolang/preprocess.go:2503 [gh](https://github.com/gnolang/gno/blob/a5370a166b459689a3febf222f3a5edd8739058e/gnovm/pkg/gnolang/preprocess.go#L2503) · Warning
State: CONFIRMED, band: Warning, angle: catalog
TL;DR: map[any]int{(*int)(nil): 1, (*int)(nil): 2} fails preprocess with a duplicate-key panic; Go and the merge base accept it
Check: cp tests/solo-finder-1-nil-pointer-map-key.gno gnovm/tests/files/zz_finder1_nk1.gno && go test ./gnovm/pkg/gnolang -run 'TestFiles/zz_finder1_nk1.gno$' -count=1 -v -timeout 2m at a5370a166 and 6c49f5420
Details: The guard lets through every comparable typed nil, and a nil pointer folds to a ConstExpr; ComputeMapKey maps both keys to one MapKey, so the kset check panics. The merge base keyed kset by TypedValue with Go ==, and the two *int PointerType values differ, so it never matched. A typed nil is no Go constant, so Go never compares it. Restricting the check to keys whose base type is a PrimitiveType passes nk1, f, b, maplit*, map0, map42 and maplit4/5 (25 filetests).
Evidence: head: unexpected panic: main/zz_finder1_nk1.gno:14:7-50: duplicate key (nil *int) in map literal. base: PASS. go1.25.9 go run of the same literal: 1. Head with tests/judge-fixes.patch: ok, 25 filetests PASS.
Artifact: tests/solo-finder-1-nil-pointer-map-key.gno
State: CONFIRMED, band: Warning, angle: removed
TL;DR: the typed-nil pointer member of the class the guard sweeps is still deduplicated; map[*int]int{nil: 1, nil: 2} fails at both head and base
Check: cp tests/solo-finder-2-maplit-nil-f.gno gnovm/tests/files/sf2_maplit_nil_f.gno; cp tests/solo-finder-2-maplit-nil-b.gno gnovm/tests/files/sf2_maplit_nil_b.gno; go test ./gnovm/pkg/gnolang -run 'TestFiles/(sf2_maplit_nil_f|sf2_maplit_nil_b).gno$' -count=1 -v -timeout 2m at a5370a166 and 6c49f5420
Details: Same line as row 2. Variant f (map[any]) passes at the base and fails at head; variant b (map[*int]) fails at both, a pre-existing member of the class this guard sweeps. The chan member is refused earlier (channels are not permitted), so the pointer kind is the one left. The PrimitiveType restriction in tests/judge-fixes.patch passes both.
Evidence: head: sf2_maplit_nil_b and sf2_maplit_nil_f FAIL, duplicate key (nil *int) in map literal. base: sf2_maplit_nil_f PASS, sf2_maplit_nil_b FAIL with the same panic. Head with tests/judge-fixes.patch: both PASS.
Artifact: tests/solo-finder-2-maplit-nil-f.gno
State: CONFIRMED, band: Nit, angle: lines
TL;DR: the cx.T == nil disjunct never holds: IsUndefined returns tv.T == nil, so !IsUndefined() already excludes it
Check: sed -n '1328,1340p' gnovm/pkg/gnolang/values.go
Details: values.go:1339 `return tv.T == nil` closes IsUndefined; the guard tests !cx.TypedValue.IsUndefined() before (cx.T == nil || ...), so the left disjunct is dead. The PrimitiveType restriction of rows 2 and 1002 drops it.
Evidence: values.go:1339 return tv.T == nil
