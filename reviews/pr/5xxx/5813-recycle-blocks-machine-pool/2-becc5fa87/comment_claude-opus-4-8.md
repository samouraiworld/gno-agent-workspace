# Review: PR #5813
Event: REQUEST_CHANGES

## Body
The recycle/allocate gas split is deterministic: the per-machine pool starts empty each run, so the split values appear only on a pool hit and reverting the split restores the old uniform goldens.

The stale-golden failures reproduce on a fresh clone and build cache under the pinned Go 1.25.9, yet `ci / gnovm` reports green on becc5fa87, so the passing check does not reflect this head.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5813-recycle-blocks-machine-pool/2-becc5fa87/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/tests/files/gas/string_eql_diff_len.gno:17 [↗](../../../../../.worktrees/gno-review-5813/gnovm/tests/files/gas/string_eql_diff_len.gno#L17)
This golden and 18 others are stale, so `TestFiles` fails on a clean checkout. Nine `gas/*` files (`compute_map_key_big_bytes`, `compute_map_key_small_bytes`, `large_array_string_eql`, `large_string_cmp`, `slice_alloc`, `small_string_cmp`, `string_eql_diff_len`, `string_struct_eql`, `switch_case_eql`) and ten `alloc_*` files (`alloc_0`, `alloc_1`, `alloc_3`, `alloc_4`, `alloc_5`, `alloc_6`, `alloc_6a`, `alloc_7`, `alloc_7a`, `alloc_10c`) carry pre-split values; regenerate them.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5813 -R gnolang/gno
cd gnovm
# gas/* are per-test deterministic; alloc_* read shared-store MemStats, so run one isolated.
go test ./pkg/gnolang/ -run 'Files$/^gas' -test.short -count=1 -timeout=300s 2>&1 | grep -E '^\s*--- FAIL: TestFiles/gas/' | sort
go test ./pkg/gnolang/ -run 'Files$/^alloc_7\.gno$' -test.short -count=1 2>&1 | grep -A4 'FAIL: TestFiles/alloc_7'
```

```
    --- FAIL: TestFiles/gas/compute_map_key_big_bytes.gno (0.08s)
    --- FAIL: TestFiles/gas/compute_map_key_small_bytes.gno (0.00s)
    --- FAIL: TestFiles/gas/large_array_string_eql.gno (0.00s)
    --- FAIL: TestFiles/gas/large_string_cmp.gno (0.00s)
    --- FAIL: TestFiles/gas/slice_alloc.gno (0.47s)
    --- FAIL: TestFiles/gas/small_string_cmp.gno (0.01s)
    --- FAIL: TestFiles/gas/string_eql_diff_len.gno (0.00s)
    --- FAIL: TestFiles/gas/string_struct_eql.gno (0.00s)
    --- FAIL: TestFiles/gas/switch_case_eql.gno (0.00s)
    --- FAIL: TestFiles/alloc_7.gno (0.00s)
        files_test.go:129: Output diff:
            -MemStats:  Allocator{maxBytes:100000000, bytes:5747}
            +MemStats:  Allocator{maxBytes:100000000, bytes:6355}
```
</details>

## SKIP gnovm/pkg/gnolang/op_call.go:678 [↗](../../../../../.worktrees/gno-review-5813/gnovm/pkg/gnolang/op_call.go#L678)
The no-recycle exclusion for a defer-origin block hangs on this one `setNoRecycle()`. `Defer.Parent` feeds only the recount GC visitor, never defer execution, so a future defer path that omits this call would recycle a still-referenced block in production, where the `debugAssert` guard is compiled out. Acceptable here; [#5856](https://github.com/gnolang/gno/pull/5856) drops `Defer.Parent`, removing the flag and the field.

## gnovm/pkg/gnolang/values.go:2487-2488 [↗](../../../../../.worktrees/gno-review-5813/gnovm/pkg/gnolang/values.go#L2487)
The comment says allocation accounting "is by numNames, independent of capacity," but `newPooledBlock` charges `AllocateBlock(max(numNames, 14))`, by capacity. Update the sentence to match.
