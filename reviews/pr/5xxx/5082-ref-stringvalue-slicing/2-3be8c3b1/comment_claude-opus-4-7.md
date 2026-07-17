# Review: PR [#5082](https://github.com/gnolang/gno/pull/5082)
Event: REQUEST_CHANGES

## Body
Verified on 3be8c3b1: regenerating [`gnovm/pkg/gnolang/pb3_gen.go`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/pb3_gen.go) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/pb3_gen.go) through [`misc/genproto2`](https://github.com/gnolang/gno/blob/3be8c3b1/misc/genproto2/genproto2.go) · [↗](../../../../../.worktrees/gno-review-5082/misc/genproto2/genproto2.go) reproduces the committed file byte for byte, so the edit to the DO-NOT-EDIT file is what the generator emits.

- benchstore no longer builds behind `-tags=genproto2`: `StringValue` is a struct now, so the `gno.StringValue(s)` conversion at [`gnovm/cmd/benchstore/values.go:44`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/cmd/benchstore/values.go#L44) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/cmd/benchstore/values.go#L44) does not compile. It is the only call site left unmigrated, and no CI job builds with that tag, so nothing catches it.
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
- The red `main / test` check is this branch's own gas change: [`stdlib_restart_compare.txtar:7`](https://github.com/gnolang/gno/blob/3be8c3b1/gno.land/pkg/integration/testdata/stdlib_restart_compare.txtar#L7) · [↗](../../../../../.worktrees/gno-review-5082/gno.land/pkg/integration/testdata/stdlib_restart_compare.txtar#L7) pins `EXACT_GAS=1974482` from the merge base, and the branch produces 1974481. The restart parity the test guards still holds, so only the constant needs recalibrating on current master.
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
- ltzmaxwell's [question from April](https://github.com/gnolang/gno/pull/5082#issuecomment-4251302565) is still open, and the recount he calls necessary there is the piece this PR leaves out: [#4885](https://github.com/gnolang/gno/pull/4885) tracks each string backing as a range and charges it once per GC cycle, so a slice inherits its source's bytes instead of escaping the budget. Does this land as a partial step with the counting finished there, or does one supersede the other?

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5082-ref-stringvalue-slicing/2-3be8c3b1/claude-opus-4-7_davd-gzl.md · [↗](claude-opus-4-7_davd-gzl.md)

## gnovm/pkg/gnolang/alloc.go:648-653 [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc.go#L648)
A ref-mode slice reports a flat 48 bytes, so after the [GC re-count](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/garbage_collector.go#L177-L188) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/garbage_collector.go#L177-L188) a 1-byte slice of a 64 MiB parent is charged 48 bytes while Go still holds the whole parent array. [`maxAllocTx = 500_000_000`](https://github.com/gnolang/gno/blob/3be8c3b1/gno.land/pkg/sdk/vm/keeper.go#L50) · [↗](../../../../../.worktrees/gno-review-5082/gno.land/pkg/sdk/vm/keeper.go#L50) bounds real memory pressure per transaction, so a transaction that slices large strings stays far under the cap while pinning much more. Either charge the retained length in ref mode, or stop letting a ref outlive its parent.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5082 -R gnolang/gno
cat > gnovm/pkg/gnolang/zz_ref_test.go <<'EOF'
package gnolang

import (
	"runtime"
	"strings"
	"testing"
)

func TestRefSliceUndercountsRetainedBytes(t *testing.T) {
	const parentLen = 64 << 20
	alloc := NewAllocator(1 << 30)
	tv := TypedValue{T: StringType, V: alloc.NewString(strings.Repeat("a", parentLen))}
	slice := tv.GetSlice(alloc, 0, 1)
	tv = TypedValue{}

	// The GC re-walks the live set and rebuilds the counter from GetShallowSize.
	alloc.Reset()
	alloc.Recount(slice.V.(StringValue).GetShallowSize())
	_, accounted := alloc.Status()

	var m runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m)
	t.Logf("VM accounts %d bytes; Go heap holds %d MiB with only the 1-byte slice live",
		accounted, m.HeapAlloc>>20)
	runtime.KeepAlive(slice)
}
EOF
go test -v -run TestRefSliceUndercountsRetainedBytes ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/zz_ref_test.go
```

```
=== RUN   TestRefSliceUndercountsRetainedBytes
    zz_ref_test.go:24: VM accounts 48 bytes; Go heap holds 70 MiB with only the 1-byte slice live
--- PASS: TestRefSliceUndercountsRetainedBytes (0.02s)
ok  	github.com/gnolang/gno/gnovm/pkg/gnolang	0.034s
```
</details>

## gnovm/pkg/gnolang/alloc.go:88-90 [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc.go#L88)
`allocStringRef` and [`allocString`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/alloc.go#L86) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc.go#L86) are both `_allocHeap + 16`, but `StringValue` is a 24-byte struct now, so both under-charge by 8 bytes. Every other value type has a line in the [`init()` self-check](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/alloc.go#L132-L151) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc.go#L132-L151) that panics when a constant drifts from `unsafe.Sizeof`, and `StringValue` has none.

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

## gnovm/pkg/gnolang/values.go:123-133 [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/values.go#L123)
`MarshalAmino` writes only `data` and `UnmarshalAmino` sets `ref=false`, so a ref-mode `StringValue` that [`GetShallowSize`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/alloc.go#L647-L653) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc.go#L647-L653) reports as 48 bytes reports 54 after a restart. The GC re-count rebuilds the allocator's byte counter from that method, so the same string charges the budget differently on either side of a restart.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5082 -R gnolang/gno
cat > gnovm/pkg/gnolang/zz_roundtrip_test.go <<'EOF'
package gnolang

import "testing"

func TestRefRoundTripChangesShallowSize(t *testing.T) {
	ref := NewStringValueRef("abcdef")
	repr, _ := ref.MarshalAmino()
	var after StringValue
	_ = after.UnmarshalAmino(repr)
	t.Logf("GetShallowSize before persistence = %d, after = %d",
		ref.GetShallowSize(), after.GetShallowSize())
}
EOF
go test -v -run TestRefRoundTripChangesShallowSize ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/zz_roundtrip_test.go
```

```
=== RUN   TestRefRoundTripChangesShallowSize
    zz_roundtrip_test.go:10: GetShallowSize before persistence = 48, after = 54
--- PASS: TestRefRoundTripChangesShallowSize (0.00s)
ok  	github.com/gnolang/gno/gnovm/pkg/gnolang	0.011s
```
</details>

## gnovm/pkg/gnolang/alloc.go:392-398 [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc.go#L392)
The doc comment says reference mode is for slicing, but not that [`GetSlice`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/values.go#L2233-L2240) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/values.go#L2233-L2240) must stay its only producer, which is what bounds the under-counted bytes to slices today. A second `NewStringRef` call site would widen that silently.

## gnovm/pkg/gnolang/alloc_test.go:51 [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc_test.go#L51)
Missing test: the new cases all assert the charge at allocation time, so nothing pins what a ref costs after the GC re-count or after an amino round-trip, which is where the two accounting rules diverge.

<details><summary>test cases</summary>

```go
func TestStringRefSizeAfterGCRecount(t *testing.T) {
	const parentLen = 100_000
	alloc := NewAllocator(1024 * 1024 * 1024)
	tv := TypedValue{T: StringType, V: alloc.NewString(strings.Repeat("a", parentLen))}
	slice := tv.GetSlice(alloc, 0, 1)

	// The GC re-walks the live set and rebuilds the counter from GetShallowSize.
	alloc.Reset()
	alloc.Recount(slice.V.(StringValue).GetShallowSize())
	_, accounted := alloc.Status()

	// The 1-byte slice still pins the parent's whole backing array.
	if want := int64(allocStringRef); accounted != want {
		t.Fatalf("post-GC bytes: got %d, want %d", accounted, want)
	}
}

func TestStringRefSizeAcrossAminoRoundTrip(t *testing.T) {
	ref := NewStringValueRef("abcdef")
	repr, err := ref.MarshalAmino()
	if err != nil {
		t.Fatalf("MarshalAmino: %v", err)
	}
	var after StringValue
	if err := after.UnmarshalAmino(repr); err != nil {
		t.Fatalf("UnmarshalAmino: %v", err)
	}

	// Ref in, owner out: the same value reports two sizes.
	if got, want := ref.GetShallowSize(), int64(allocStringRef); got != want {
		t.Fatalf("pre-round-trip: got %d, want %d", got, want)
	}
	if got, want := after.GetShallowSize(), allocString+allocStringByte*int64(len("abcdef")); got != want {
		t.Fatalf("post-round-trip: got %d, want %d", got, want)
	}
}
```
</details>

## gnovm/pkg/gnolang/values.go:92-95 [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/values.go#L92)
Suggestion: every `StringValue` now carries a `ref` bool that only [`GetSlice`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/values.go#L2233-L2240) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/values.go#L2233-L2240) ever sets, and padding takes the struct from 16 to 24 bytes, so every literal and concatenation result pays for a field it never uses. A separate ref type, or a bit in the unused `TypedValue.N` word, would avoid that: deliberate trade for the simpler struct?

## gnovm/pkg/gnolang/values.go:97 [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/values.go#L97)
Nit: the new exported doc comments lack terminating periods, from `NewStringValue` here through `UnmarshalAmino` at line 128.

## gnovm/pkg/gnolang/alloc_test.go:87 [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc_test.go#L87)
Nit: `_ = result.GetString()` here and `_ = s3.GetString()` at line 113 discard the value, so they assert nothing about the sliced content.

## gnovm/pkg/gnolang/bench_test.go:52-53 [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/bench_test.go#L52)
Nit: `NewAllocator(1024*1024)` is constructed inside the `b.N` loop, so `ReportAllocs` counts allocator setup in the allocs/op this benchmark reports.
