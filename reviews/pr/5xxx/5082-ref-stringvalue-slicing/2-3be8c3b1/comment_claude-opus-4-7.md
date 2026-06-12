# Review: PR #5082
Event: REQUEST_CHANGES

## Body
The reference-StringValue optimization is sound in shape, but two things block and the in-memory accounting model needs to be stated explicitly rather than left implicit.

- benchstore no longer builds behind `-tags=genproto2`: `StringValue` is now a struct, so the `gno.StringValue(s)` conversion at [`gnovm/cmd/benchstore/values.go:44`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/cmd/benchstore/values.go#L44) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/cmd/benchstore/values.go#L44) is invalid. The migration was applied everywhere else but missed here, and no CI job builds with that tag, so the break is latent. Verified locally on the current head (3be8c3b1). Fix: `gno.NewStringValue(s)`.
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
- The one red check (`main / test`) is the `stdlib_restart_compare` gas assertion: `Convert` reports `GAS USED: 1974481` but the test asserts `EXACT_GAS=1974482` ([CI job 24844592434](https://github.com/gnolang/gno/actions/runs/24844592434/job/72727940545), txtar:20). The branch is also behind master, which has since recalibrated this constant to 2235646, so recalibrate it to the value this PR's code produces after merging master.
- Relative to #4885 (ltzmaxwell's comment on this PR suggests overlapping goals): land this as a partial step and finish the accounting in #4885, or does one supersede the other?
- Every check is at the Go layer; a `.gno` filetest pinning the slice alloc/gas cost and an in-tree test for the GC re-count and amino round-trip trade-off would lock the behavior in.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5082-ref-stringvalue-slicing/2-3be8c3b1/claude-opus-4-7_davd-gzl.md · [↗](claude-opus-4-7_davd-gzl.md)

*(AI Agent)*

## gnovm/pkg/gnolang/alloc.go:648-653 [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc.go#L648)
A reference-mode StringValue reports a flat 48 bytes from GetShallowSize(), but Go keeps the parent string's full backing array alive for as long as the slice points into it. After GC re-counts the live set, a 1-byte slice of a 100KB parent is accounted as 48 bytes while the host still holds 100KB, so a transaction can stay under maxAllocTx (500M) on paper while pinning far more real memory. Either charge allocStringRef + len(data) in ref mode, or stop keeping refs alive across persistence, and state the chosen accounting model in the PR.

*(AI Agent)*

## gnovm/pkg/gnolang/alloc.go:88-90 [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc.go#L88)
StringValue is now a 24-byte struct (16-byte string header + 1-byte bool + 7 padding), but allocString and allocStringRef are both still _allocHeap + 16, so every StringValue under-charges by 8 bytes. There is also no check() line for StringValue in the init() self-check that catches this drift for the other value types. Bump both constants to _allocHeap + 24 (or alias allocStringRef = allocString so future size fixes hit both) and add a self-check entry.

*(AI Agent)*

## gnovm/pkg/gnolang/values.go:123-133 [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/values.go#L123)
MarshalAmino writes only data and UnmarshalAmino always sets ref=false, so a ref-mode StringValue that GetShallowSize() reports as 48 bytes before persistence reports 48 + len(data) after a restart. Because the GC re-count pass rebuilds the allocator's byte counter from GetShallowSize, the same string contributes a different amount to the budget before vs after restart. Either don't keep ref mode across persistence, or document the asymmetry and why it can't drift gas in a realistic workload.

*(AI Agent)*

## gnovm/pkg/gnolang/alloc.go:392-398 [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc.go#L392)
NewString (copy-producing paths) charges 48 + len while NewStringRef (slicing) charges a flat 48, but nothing records that reference mode must only ever be produced by GetSlice. Add a note on the StringValue type so a future contributor doesn't introduce a new ref producer and widen the GC under-counting above.

*(AI Agent)*

## gnovm/pkg/gnolang/pb3_gen.go:247-250 [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/pb3_gen.go#L247)
This file is generated and marked DO NOT EDIT, but the MarshalAmino/UnmarshalAmino dispatch here was hand-edited. It matches what genproto2 emits for an amino-marshaler type, so it is correct today; re-running genproto2 so the diff reads as regenerated avoids a confusing no-op diff for the next person who regenerates.

*(AI Agent)*

## gnovm/pkg/gnolang/values.go:92-95 [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/values.go#L92)
Every StringValue now carries an 8-byte ref bool that only GetSlice ever sets, so every literal and concatenation result pays for it. Worth weighing a separate ref type, or a bit in the unused TypedValue.N word, against the simplicity of the current struct.

*(AI Agent)*

## gnovm/pkg/gnolang/values.go:97 [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/values.go#L97)
The new exported doc comments (NewStringValue here, plus NewStringValueRef, Value, IsRef, Len, MarshalAmino, and UnmarshalAmino through line 133) are missing terminating periods.

*(AI Agent)*

## gnovm/pkg/gnolang/alloc_test.go:87 [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc_test.go#L87)
`_ = result.GetString()` here, and `_ = s3.GetString()` at line 113, discard the value, so they assert nothing. Replace with a content check or drop them.

*(AI Agent)*

## gnovm/pkg/gnolang/bench_test.go:52-53 [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/bench_test.go#L52)
NewAllocator(1024*1024) is constructed inside the b.N loop, so allocator setup is counted by ReportAllocs() and pollutes the allocs/op number this benchmark reports. Hoist it and call alloc.Reset() per iteration.

*(AI Agent)*
