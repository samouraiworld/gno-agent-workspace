# Review: [#6110](https://github.com/gnolang/gno/pull/6110)
Event: REQUEST_CHANGES
Verdict: REQUEST CHANGES. A `[]byte` call argument now reaches the realm with a capacity set by line breaks in its base64 encoding, and a `map[any]` literal with two typed-nil pointer keys is rejected at preprocess; the branch introduces both.
Model: claude-opus-5-5, standard review
Commit: a5370a166b459689a3febf222f3a5edd8739058e
Overview: [overview.md](../overview.md)
Open the code: [head worktree](../../../../../.worktrees/gno-review-6110)
Round: 3. The head moved from c800a571a to a5370a166 with three commits and a merge of master, whose one conflict sits in a gas txtar outside this diff. Round 2's Critical, its map-literal Warning and its `convert.go` Suggestion each reran closed, and its ADR Nit read closed; both new Warnings sit on the lines those answers changed. 2 finders, a reflector pass by the judge, 6 candidates (5 from the finders, 1 from the reflector), each Warning rerun at head and merge base by an agent that was not its finder, the Nit judged by read; 1 refuted.

## Body

## gno.land/pkg/sdk/vm/convert.go:216 [gh](https://github.com/gnolang/gno/blob/a5370a166b459689a3febf222f3a5edd8739058e/gno.land/pkg/sdk/vm/convert.go#L216) · [↗](../../../../../.worktrees/gno-review-6110/gno.land/pkg/sdk/vm/convert.go#L216) · Warning
`alloc.NewSliceFromData(bz)` keeps the whole decode buffer, so line breaks the base64 decoder ignores grow `cap(b)`, the stored array and the allocation charge.

```suggestion
			tv.V = alloc.NewSliceFromData(bz[:len(bz):len(bz)])
```

<details><summary>repro</summary>

The realm stores `b` and returns `len(b), cap(b)` for three encodings of the same three bytes.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6110 -R gnolang/gno
curl -fsSL -o gno.land/pkg/sdk/vm/zz_sf2cap_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6110-vm-owned-string-backing-id/3-a5370a166/tests/solo-finder-2-bytes-arg-cap-malleable_test.go
go test ./gno.land/pkg/sdk/vm -run TestSF2BytesArgCapMalleable -count=1 -v -timeout 2m
rm gno.land/pkg/sdk/vm/zz_sf2cap_test.go
```

The length stays 3 while the capacity follows the padding, 3, 6 and 15; every case returns `(3 int)\n(3 int)` with the suggestion applied:

```text
zz_sf2cap_test.go:29: arg="AAAA" -> "(3 int)\n(3 int)\n\n"
zz_sf2cap_test.go:29: arg="AAAA\n\n\n\n" -> "(3 int)\n(6 int)\n\n"
zz_sf2cap_test.go:29: arg="AAAA\r\n\r\n\r\n\r\n\r\n\r\n\r\n\r\n" -> "(3 int)\n(15 int)\n\n"
```

The string to `[]byte` conversion, the other caller of `NewSliceFromData`, clips its buffer the same way at [`values_conversions.go:1043`](https://github.com/gnolang/gno/blob/a5370a166b459689a3febf222f3a5edd8739058e/gnovm/pkg/gnolang/values_conversions.go#L1043).

</details>

## gnovm/pkg/gnolang/preprocess.go:2503 [gh](https://github.com/gnolang/gno/blob/a5370a166b459689a3febf222f3a5edd8739058e/gnovm/pkg/gnolang/preprocess.go#L2503) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/preprocess.go#L2503) · Warning
The guard admits a typed-nil pointer key, which is no Go constant, so preprocessing rejects `map[any]int{(*int)(nil): 1, (*int)(nil): 2}` as a duplicate. Restrict the check to bool, string and numeric keys, which also accepts `map[*int]int{nil: 1, nil: 2}` and keeps the duplicate-key filetests green.

<details><summary>repro</summary>

The file must preprocess and print `1`, as `go run` does.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6110 -R gnolang/gno
curl -fsSL -o gnovm/tests/files/zz_finder1_nk1.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6110-vm-owned-string-backing-id/3-a5370a166/tests/solo-finder-1-nil-pointer-map-key.gno
go test ./gnovm/pkg/gnolang -run 'TestFiles/zz_finder1_nk1.gno$' -count=1 -v -timeout 2m
rm gnovm/tests/files/zz_finder1_nk1.gno
```

The preprocessor rejects the literal before `main` runs:

```text
files_test.go:135: unexpected panic: main/zz_finder1_nk1.gno:14:7-50: duplicate key (nil *int) in map literal
--- FAIL: TestFiles/zz_finder1_nk1.gno (0.00s)
```

The same file passes on the base branch. `map[*int]int{nil: 1, nil: 2}` fails on both, with the same panic. The `cx.T == nil` disjunct never holds, since `IsUndefined` returns `tv.T == nil` at [`values.go:1339`](https://github.com/gnolang/gno/blob/a5370a166b459689a3febf222f3a5edd8739058e/gnovm/pkg/gnolang/values.go#L1339).

</details>
