# Claims: gnolang/gno#6110 round 3, a5370a166, claude-opus-5-5, standard review

Round shape: solo round, 2 finders and one judge and writer, every Critical and Warning run, the rest read

## Findings

| Line | # | Band | State | Merge base | Draft |
| --- | --- | --- | --- | --- | --- |
| [`convert.go:216`](https://github.com/gnolang/gno/blob/a5370a166/gno.land/pkg/sdk/vm/convert.go#L216) | 1 | Warning | CONFIRMED | differs | posts |
| [`convert.go:216`](https://github.com/gnolang/gno/blob/a5370a166/gno.land/pkg/sdk/vm/convert.go#L216) | 1001 | Warning | CONFIRMED | differs | with row 1 |
| [`preprocess.go:2503`](https://github.com/gnolang/gno/blob/a5370a166/gnovm/pkg/gnolang/preprocess.go#L2503) | 2 | Warning | CONFIRMED | differs | posts |
| [`preprocess.go:2503`](https://github.com/gnolang/gno/blob/a5370a166/gnovm/pkg/gnolang/preprocess.go#L2503) | 1002 | Warning | CONFIRMED | differs | with row 2 |
| [`preprocess.go:2503`](https://github.com/gnolang/gno/blob/a5370a166/gnovm/pkg/gnolang/preprocess.go#L2503) | 3 | Nit | CONFIRMED | same | with row 2 |
| [`values_conversions.go:1044`](https://github.com/gnolang/gno/blob/a5370a166/gnovm/pkg/gnolang/values_conversions.go#L1044) | 2001 | Warning | REFUTED |  |  |

### 1 · Warning · CONFIRMED · `gno.land/pkg/sdk/vm/convert.go:216`

A []byte call argument reaches the realm with cap = base64 DecodedLen(len(arg)), so padding the encoding with newlines grows cap, the persisted array and the alloc charge

- Check: cp tests/solo-finder-1-slice-arg-cap.go gno.land/pkg/sdk/vm/zz_finder1_capcall_test.go && go test ./gno.land/pkg/sdk/vm -run TestFinder1SliceArgCapCall -count=1 -v -timeout 2m at a5370a166 and 6c49f5420
- Observed: head: arg len 4 -> cap(b) = (3 int); arg len 404 -> cap(b) = (303 int). base: (1 int) for both. Head with tests/judge-fixes.patch (bz[:len(bz):len(bz)]): (1 int) for both.
- Test: [`tests/solo-finder-1-slice-arg-cap.go`](tests/solo-finder-1-slice-arg-cap.go)

### 1001 · Warning · CONFIRMED · `gno.land/pkg/sdk/vm/convert.go:216`

One decoded byte string arrives with a cap chosen by ignored \r\n in its encoding, and a realm that stores it persists the slack

- Check: cp tests/solo-finder-2-bytes-arg-cap-malleable_test.go gno.land/pkg/sdk/vm/zz_sf2cap_test.go && go test ./gno.land/pkg/sdk/vm -run TestSF2BytesArgCapMalleable -count=1 -v -timeout 2m at a5370a166 and 6c49f5420
- Observed: head: "AAAA" -> (3 int)(3 int); "AAAA\n\n\n\n" -> (3 int)(6 int); "AAAA"+8x"\r\n" -> (3 int)(15 int). base: (3 int)(3 int) for all three. Head with tests/judge-fixes.patch: (3 int)(3 int) for all three.
- Test: [`tests/solo-finder-2-bytes-arg-cap-malleable_test.go`](tests/solo-finder-2-bytes-arg-cap-malleable_test.go)

### 2 · Warning · CONFIRMED · `gnovm/pkg/gnolang/preprocess.go:2503`

map[any]int{(*int)(nil): 1, (*int)(nil): 2} fails preprocess with a duplicate-key panic; Go and the merge base accept it

- Check: cp tests/solo-finder-1-nil-pointer-map-key.gno gnovm/tests/files/zz_finder1_nk1.gno && go test ./gnovm/pkg/gnolang -run 'TestFiles/zz_finder1_nk1.gno$' -count=1 -v -timeout 2m at a5370a166 and 6c49f5420
- Observed: head: unexpected panic: main/zz_finder1_nk1.gno:14:7-50: duplicate key (nil *int) in map literal. base: PASS. go1.25.9 go run of the same literal: 1. Head with tests/judge-fixes.patch: ok, 25 filetests PASS.
- Test: [`tests/solo-finder-1-nil-pointer-map-key.gno`](tests/solo-finder-1-nil-pointer-map-key.gno)

### 1002 · Warning · CONFIRMED · `gnovm/pkg/gnolang/preprocess.go:2503`

The typed-nil pointer member of the class the guard sweeps is still deduplicated; map[*int]int{nil: 1, nil: 2} fails at both head and base

- Check: cp tests/solo-finder-2-maplit-nil-f.gno gnovm/tests/files/sf2_maplit_nil_f.gno; cp tests/solo-finder-2-maplit-nil-b.gno gnovm/tests/files/sf2_maplit_nil_b.gno; go test ./gnovm/pkg/gnolang -run 'TestFiles/(sf2_maplit_nil_f\|sf2_maplit_nil_b).gno$' -count=1 -v -timeout 2m at a5370a166 and 6c49f5420
- Observed: head: sf2_maplit_nil_b and sf2_maplit_nil_f FAIL, duplicate key (nil *int) in map literal. base: sf2_maplit_nil_f PASS, sf2_maplit_nil_b FAIL with the same panic. Head with tests/judge-fixes.patch: both PASS.
- Test: [`tests/solo-finder-2-maplit-nil-f.gno`](tests/solo-finder-2-maplit-nil-f.gno)

### 3 · Nit · CONFIRMED · `gnovm/pkg/gnolang/preprocess.go:2503`

The cx.T == nil disjunct never holds: IsUndefined returns tv.T == nil, so !IsUndefined() already excludes it

- Check: sed -n '1328,1340p' gnovm/pkg/gnolang/values.go
- Observed: values.go:1339 return tv.T == nil

### 2001 · Warning · REFUTED · `gnovm/pkg/gnolang/values_conversions.go:1044`

The string to []byte conversion clips cap to len before NewSliceFromData

- Check: grep -rn 'NewSliceFromData(' --include='*.go' gno.land gnovm tm2; filetest printing len and cap of []byte("abcde")
- Refuted by: values_conversions.go:1043 data = data[:len(data):len(data)] // defensive: force cap = len (not guaranteed by Go spec)

## Cleared by the finder

No verifier ran these: the finder's own read or run settled each.

| Line | Angle | Suspected | Cleared by |
| --- | --- | --- | --- |
| [`preprocess.go:4643`](https://github.com/gnolang/gno/blob/a5370a166/gnovm/pkg/gnolang/preprocess.go#L4643) | lines | round-2 Critical: folded string const backing differs between a long-running and a restarted node | round-2 repro b2-catalog-const-backing-restart_test.go at a5370a166: `long-running node: ... share a backing = false, recount = 2096` and `restarted node: ... share a backing = false, recount = 2096`, PASS. Block nodes are never amino-loaded (store.go setBlockNode backend write commented out), and PrepareNewValues copies static values only for new names, so no other path carries a cached static string into a later tx. |
| [`preprocess.go:2503`](https://github.com/gnolang/gno/blob/a5370a166/gnovm/pkg/gnolang/preprocess.go#L2503) | lines | round-2 Warning: map[any] literal with a typed-nil slice key rejected at preprocess | round-2 repro b2-lines-maplit-unhashable-const-key.gno as gnovm/tests/files/b2lines_maplit_a.gno at a5370a166: `ok github.com/gnolang/gno/gnovm/pkg/gnolang` |
| [`alloc.go:600`](https://github.com/gnolang/gno/blob/a5370a166/gnovm/pkg/gnolang/alloc.go#L600) | claims | mintString godoc claims exactly three callers | grep -rn 'mintString(' gnovm/pkg gno.land/pkg gnovm/stdlibs minus tests: realm.go:1899, preprocess.go:4643, alloc.go:591 |
| [`values.go:117`](https://github.com/gnolang/gno/blob/a5370a166/gnovm/pkg/gnolang/values.go#L117) | claims | invariant comment claims ClearObjectCache runs per message and resets alloc.bytes and the object cache | keeper.go:401 getGnoTransactionStore calls txStore.ClearObjectCache(); store.go:1505 ClearObjectCache does ds.alloc.Reset() and ds.cacheObjects = make(...); alloc.go:363 Reset sets alloc.bytes = 0 |
| [`convert.go:216`](https://github.com/gnolang/gno/blob/a5370a166/gno.land/pkg/sdk/vm/convert.go#L216) | reach | NewSliceFromData now stamps the arg's base array with alloc.currentRealmID, which Machine.Release never clears, so a second MsgCall in one tx might stamp a []byte arg with the first message's realm and persist it as a foreign object | go test ./gno.land/pkg/sdk/vm -run TestSF2BytesArgStaleRealmStamp -v (Call r/aa.Bump then r/bb.Set(b) in one tx store, then Mut in the next): 'multi=true Stored base oid=06a9901f...:8 ownedByAA=false ownedByBB=true', 'multi=true Mut err=<nil>' |
| [`convert.go:37`](https://github.com/gnolang/gno/blob/a5370a166/gno.land/pkg/sdk/vm/convert.go#L37) | reach | nil allocator on query/render paths reaching the new NewArrayFromData/NewSliceFromData charge | grep -rn 'convertArgToGno' gno.land gnovm tm2 (non-test): only keeper.go:1366 convertArgToGno(m.Alloc, ...) in Call; alloc.go:403 Allocate and alloc.go:582 stampPkgID both return early on alloc == nil |
| [`stdlib_ibc_crypto_determinism.txtar:41`](https://github.com/gnolang/gno/blob/a5370a166/gno.land/pkg/integration/testdata/stdlib_ibc_crypto_determinism.txtar#L41) | removed | merge conflict resolution of EXACT_GAS, and txtars pinning gas against the new []byte arg charge | git diff 525d94291^2 a5370a166 -- the txtar: master's 1850271 -> 1850456, the same +185 the branch carried; every CI check run is green on a5370a166, so any EXACT_GAS fixture the new charge moved is caught by CI |

## Record

<details><summary>One row per candidate, the table the round tools parse</summary>

| # | State | Band | file:line | Check | Tier | Mechanism |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Warning | gno.land/pkg/sdk/vm/convert.go:216 | cp tests/solo-finder-1-slice-arg-cap.go gno.land/pkg/sdk/vm/zz_finder1_capcall_test.go && go test ./gno.land/pkg/sdk/vm -run TestFinder1SliceArgCapCall -count=1 -v -timeout 2m at a5370a166 and 6c49f5420 | hot | bytes-arg-cap-malleable |
| 2 | CONFIRMED | Warning | gnovm/pkg/gnolang/preprocess.go:2503 | cp tests/solo-finder-1-nil-pointer-map-key.gno gnovm/tests/files/zz_finder1_nk1.gno && go test ./gnovm/pkg/gnolang -run 'TestFiles/zz_finder1_nk1.gno$' -count=1 -v -timeout 2m at a5370a166 and 6c49f5420 | hot | maplit-typed-nil-key-dedup |
| 3 | CONFIRMED | Nit | gnovm/pkg/gnolang/preprocess.go:2503 | sed -n '1328,1340p' gnovm/pkg/gnolang/values.go | hot |  |
| 1001 | CONFIRMED | Warning | gno.land/pkg/sdk/vm/convert.go:216 | cp tests/solo-finder-2-bytes-arg-cap-malleable_test.go gno.land/pkg/sdk/vm/zz_sf2cap_test.go && go test ./gno.land/pkg/sdk/vm -run TestSF2BytesArgCapMalleable -count=1 -v -timeout 2m at a5370a166 and 6c49f5420 | hot | bytes-arg-cap-malleable |
| 1002 | CONFIRMED | Warning | gnovm/pkg/gnolang/preprocess.go:2503 | cp tests/solo-finder-2-maplit-nil-f.gno gnovm/tests/files/sf2_maplit_nil_f.gno; cp tests/solo-finder-2-maplit-nil-b.gno gnovm/tests/files/sf2_maplit_nil_b.gno; go test ./gnovm/pkg/gnolang -run 'TestFiles/(sf2_maplit_nil_f\|sf2_maplit_nil_b).gno$' -count=1 -v -timeout 2m at a5370a166 and 6c49f5420 | hot | maplit-typed-nil-key-dedup |
| 2001 | REFUTED | Warning | gnovm/pkg/gnolang/values_conversions.go:1044 | grep -rn 'NewSliceFromData(' --include='*.go' gno.land gnovm tm2; filetest printing len and cap of []byte("abcde") |  |  |

</details>

Hit rate per tier, from the record: hot 5/5 confirmed over 4 files, warm 0/0 confirmed over 1 files, cold 0/0 confirmed over 3 files.

## Completeness

- Round 2's findings, rerun at a5370a166: the Critical's artifact `b2-catalog-const-backing-restart_test.go` prints `share a backing = false, recount = 2096` for both node histories, PASS, closed by the `mintString` in `evalConst`; the Warning's `b2-lines-maplit-unhashable-const-key.gno` passes, closed by the `isComparable` guard. The convert.go Suggestion is closed by `NewSliceFromData`, which opened row 1. The ADR Nit was read, not rerun.
- Catalog: determinism and gas (row 1, cap and charge follow base64 padding), VM semantics vs Go and type-check (row 2), realm state safety (the round 2 Critical rerun). Global state, coin, storage deposit, caller identity, twin input paths, older stored data and VM-fault recoverability: the diff touches none.
- Sibling sweep by shape: `NewSliceFromData` has one other caller, `values_conversions.go:1044`, which clips cap to len at line 1043 (row 2001, refuted; filetest `[]byte("abcde")` prints `5 5` at head and base). The map-literal guard's class swept by kind: a typed-nil chan key is refused earlier (`channels are not permitted`), so the pointer kind is the one comparable non-constant member left.
- Both fixes in `tests/judge-fixes.patch` were applied at head and run: the two cap tests print cap = len, and 25 filetests (`maplit*`, `map0`, `map42`, the nil-key fixtures) pass.
- Repros in the draft were run from the `tests/` files with `cp`, not through the `curl` line, and no txtar was written for row 1: the budget went to the base runs.
- Parent rerun before the post, go1.25.9: rows 1 and 2 reproduce at a5370a166 and pass at 6c49f5420; with `tests/judge-fixes.patch` the three encodings print cap 3 and 73 `TestFiles/(zz_finder1_nk1|maplit*|map*)` filetests pass. `go run` of `map[any]int{(*int)(nil): 1, (*int)(nil): 2}` and `map[*int]int{nil: 1, nil: 2}` prints `1 1`; the patched head runs the second as a filetest with `// Output: 1`, ok.
