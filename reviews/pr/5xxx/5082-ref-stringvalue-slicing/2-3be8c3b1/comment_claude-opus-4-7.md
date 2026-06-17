# Review: PR #5082
Event: REQUEST_CHANGES

## Body
The reference-StringValue optimization is sound in shape but blocks on the two unanchored findings below, and the ref-accounting model (a ref slice counted as a flat 48 bytes while pinning the parent's full backing array) should be stated in the PR, not left implicit. Reproduced on 3be8c3b1.

- benchstore no longer builds behind `-tags=genproto2`: `StringValue` is now a struct, so the `gno.StringValue(s)` conversion at [`gnovm/cmd/benchstore/values.go:44`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/cmd/benchstore/values.go#L44) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/cmd/benchstore/values.go#L44) no longer compiles. The migration landed everywhere else but missed this file, and no CI job builds with that tag, so the break is latent. Fix: `gno.NewStringValue(s)`.
  <details><summary>repro</summary>

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5082 -R gnolang/gno
  go build -tags=genproto2 ./gnovm/cmd/benchstore/
  ```

  ```
  # github.com/gnolang/gno/gnovm/cmd/benchstore
  gnovm/cmd/benchstore/values.go:44:62: cannot convert s (variable of type string) to type gnolang.StringValue
  ```
  </details>
- The red `main / test` check is the `stdlib_restart_compare` gas assertion: `Convert` reports `GAS USED: 1974481` but `EXACT_GAS=1974482` (txtar:7) ([CI job 24844592434](https://github.com/gnolang/gno/actions/runs/24844592434/job/72727940545)). The branch is behind master, which recalibrated this constant to 2235646, so set it to the value this branch produces after merging master, not 1974481.
- Relative to [#4885](https://github.com/gnolang/gno/issues/4885) (ltzmaxwell's comment on this PR points at overlapping goals): is the plan to land this as a partial step and finish the accounting in #4885, or does one supersede the other?
- No `.gno` filetest pins the slice alloc/gas cost, and there's no in-tree test for the GC re-count or amino round-trip trade-off; both would lock the behavior in.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5082-ref-stringvalue-slicing/2-3be8c3b1/claude-opus-4-7_davd-gzl.md · [↗](claude-opus-4-7_davd-gzl.md)

## gnovm/pkg/gnolang/alloc.go:648-653 [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc.go#L648)
A 1-byte ref slice of a 100KB parent reports a flat 48 bytes from GetShallowSize(), yet Go keeps the parent's full backing array alive while the slice points into it, so after the GC re-count a transaction can stay under maxAllocTx (500M) on paper while pinning far more real memory. Either charge `allocStringRef + len(data)` in ref mode, or stop keeping refs alive across persistence, and state the chosen model in the PR.

## gnovm/pkg/gnolang/alloc.go:88-90 [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc.go#L88)
allocString and allocStringRef are both still `_allocHeap + 16`, but StringValue is now a 24-byte struct (16-byte string header + 1-byte bool + 7 padding), so every StringValue under-charges by 8 bytes. With no StringValue line in the init() self-check, this drift goes uncaught the way it would for the other value types. Bump both constants to `_allocHeap + 24` and add a self-check entry.

## gnovm/pkg/gnolang/values.go:123-133 [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/values.go#L123)
A ref-mode StringValue that GetShallowSize() reports as 48 bytes before persistence reports 48 + len(data) after a restart, because MarshalAmino writes only data and UnmarshalAmino sets ref=false. Since the GC re-count rebuilds the allocator's byte counter from GetShallowSize, the same string contributes a different amount to the budget before vs after restart. Either don't keep ref mode across persistence, or document the asymmetry and why it can't drift gas in a realistic workload.

## gnovm/pkg/gnolang/alloc.go:392-398 [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc.go#L392)
Nothing records that reference mode must only ever be produced by GetSlice, so a future contributor can add a new NewStringRef call site and widen the GC under-counting that ref mode already introduces. Add a note on the StringValue type pinning that contract.

## gnovm/pkg/gnolang/pb3_gen.go:247-250 [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/pb3_gen.go#L247)
This DO-NOT-EDIT generated file was hand-edited here; the MarshalAmino/UnmarshalAmino dispatch matches what genproto2 emits for an amino-marshaler type, so it is correct today. Re-run genproto2 so the diff reads as regenerated and doesn't confuse whoever regenerates next.

## gnovm/pkg/gnolang/values.go:92-95 [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/values.go#L92)
Every StringValue now carries an 8-byte ref bool that only GetSlice ever sets, so every literal and concatenation result pays for it. Worth weighing a separate ref type, or a bit in the unused TypedValue.N word, against the simplicity of the current struct.

## gnovm/pkg/gnolang/values.go:97 [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/values.go#L97)
The new exported doc comments (NewStringValue, NewStringValueRef, Value, IsRef, Len, MarshalAmino, UnmarshalAmino, through line 133) lack terminating periods.

## gnovm/pkg/gnolang/alloc_test.go:87 [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc_test.go#L87)
`_ = result.GetString()` here, and `_ = s3.GetString()` at line 113, discard the value, so they assert nothing. Replace with a content check or drop them.

## gnovm/pkg/gnolang/bench_test.go:52-53 [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/bench_test.go#L52)
NewAllocator(1024*1024) is constructed inside the b.N loop, so allocator setup is counted by ReportAllocs() and pollutes the allocs/op number this benchmark reports. Hoist it and call alloc.Reset() per iteration.
