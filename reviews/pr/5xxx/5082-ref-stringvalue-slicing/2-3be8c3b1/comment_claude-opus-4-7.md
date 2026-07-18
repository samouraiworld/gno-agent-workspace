# Review: PR [#5082](https://github.com/gnolang/gno/pull/5082)
Posted: https://github.com/gnolang/gno/pull/5082#pullrequestreview-4726542178
Event: COMMENT

## Body
Verified on 3be8c3b1: regenerating [`pb3_gen.go`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/pb3_gen.go) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/pb3_gen.go) with [`misc/genproto2`](https://github.com/gnolang/gno/blob/3be8c3b1/misc/genproto2/genproto2.go) · [↗](../../../../../.worktrees/gno-review-5082/misc/genproto2/genproto2.go) reproduces it byte for byte, so the edit to the generated file is what the generator emits.

- benchstore no longer builds behind `-tags=genproto2`: the `gno.StringValue(s)` conversion at [`gnovm/cmd/benchstore/values.go:44`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/cmd/benchstore/values.go#L44) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/cmd/benchstore/values.go#L44) does not compile. No CI job builds with that tag.
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
- The red `main / test` check is this branch's own gas change: [`stdlib_restart_compare.txtar:7`](https://github.com/gnolang/gno/blob/3be8c3b1/gno.land/pkg/integration/testdata/stdlib_restart_compare.txtar#L7) · [↗](../../../../../.worktrees/gno-review-5082/gno.land/pkg/integration/testdata/stdlib_restart_compare.txtar#L7) pins `EXACT_GAS=1974482` from the merge base, and the branch produces 1974481. Restart parity still holds, so only the constant needs recalibrating.
  <details><summary>repro</summary>

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5082 -R gnolang/gno
  go test -run 'TestTestdata/stdlib_restart_compare' ./gno.land/pkg/integration/
  ```

  ```
  --- FAIL: TestTestdata (0.01s)
      --- FAIL: TestTestdata/stdlib_restart_compare (3.65s)
              GAS USED:   1974481
              > stdout 'GAS USED:\s+'${EXACT_GAS}
              FAIL: testdata/stdlib_restart_compare.txtar:20: no match for `GAS USED:\s+1974482` found in stdout
  FAIL	github.com/gnolang/gno/gno.land/pkg/integration	3.682s
  ```
  </details>

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5082-ref-stringvalue-slicing/2-3be8c3b1/claude-opus-4-7_davd-gzl.md · [↗](claude-opus-4-7_davd-gzl.md)

## gnovm/pkg/gnolang/alloc.go:88-90 [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc.go#L88) [posted](https://github.com/gnolang/gno/pull/5082#discussion_r3606563553)
`allocStringRef` and [`allocString`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/alloc.go#L86) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc.go#L86) are the pre-PR `_allocHeap + 16`, but `StringValue` is a 24-byte struct now, so both are 8 bytes low and the metering undercharges every string. The undercount is uniform and deterministic, so it does not diverge across nodes, but the constant is wrong against the rule that it equal `unsafe.Sizeof`, and the [`init()` self-check](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/alloc.go#L132-L151) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc.go#L132-L151) that catches this drift on the other struct-backed types has no `StringValue` line. Bump both to `_allocHeap + 24` and add a check line.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5082 -R gnolang/gno
cat > gnovm/pkg/gnolang/zz_size_test.go <<'EOF'
package gnolang

import (
	"testing"
	"unsafe"
)

func TestStringValueAllocConstantDrift(t *testing.T) {
	want := int64(_allocHeap) + int64(unsafe.Sizeof(StringValue{}))
	t.Logf("unsafe.Sizeof(StringValue{}) = %d", unsafe.Sizeof(StringValue{}))
	if allocString != want {
		t.Fatalf("allocString = %d, _allocHeap+sizeof = %d", allocString, want)
	}
	if allocStringRef != want {
		t.Fatalf("allocStringRef = %d, _allocHeap+sizeof = %d", allocStringRef, want)
	}
}
EOF
go test -v -run TestStringValueAllocConstantDrift ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/zz_size_test.go
```

```
=== RUN   TestStringValueAllocConstantDrift
    zz_size_test.go:10: unsafe.Sizeof(StringValue{}) = 24
    zz_size_test.go:12: allocString = 48, _allocHeap+sizeof = 56
--- FAIL: TestStringValueAllocConstantDrift (0.00s)
FAIL	github.com/gnolang/gno/gnovm/pkg/gnolang	0.011s
```
</details>

## SKIP gnovm/pkg/gnolang/alloc.go:392-398 [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc.go#L392)
Suggestion: `NewStringRef` charges a flat 48 and skips the per-byte cost on the assumption its argument shares an existing backing array, but nothing enforces that. Only [`GetSlice`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/values.go#L2233-L2240) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/values.go#L2233-L2240) calls it today and always passes a reslice, so this is latent; documenting the shared-backing precondition as load-bearing would keep a future caller from silently under-metering.

## gnovm/pkg/gnolang/values.go:92-95 [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/values.go#L92) [posted](https://github.com/gnolang/gno/pull/5082#discussion_r3606563558)
Suggestion: the `ref` bool grows `StringValue` from 16 to 24 bytes, so every string pays for a field only [`GetSlice`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/values.go#L2233-L2240) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/values.go#L2233-L2240) sets. A separate reference type would avoid it.

## gnovm/pkg/gnolang/alloc_test.go:87 [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc_test.go#L87) [posted](https://github.com/gnolang/gno/pull/5082#discussion_r3606563565)
Nit: `_ = result.GetString()` here and `_ = s3.GetString()` at line 113 discard the value, so they assert nothing.

## SKIP gnovm/pkg/gnolang/bench_test.go:52-53 [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/bench_test.go#L52)
Nit: `NewAllocator(1024*1024)` sits inside the `b.N` loop, so `ReportAllocs` counts allocator setup in the allocs/op this benchmark reports.
