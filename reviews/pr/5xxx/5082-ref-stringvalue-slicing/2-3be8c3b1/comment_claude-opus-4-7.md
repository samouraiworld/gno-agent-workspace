# Review: PR [#5082](https://github.com/gnolang/gno/pull/5082)
Event: REQUEST_CHANGES

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

## gnovm/pkg/gnolang/values.go:2234-2239 [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/values.go#L2234)
A 32 MiB slice of a 64 MiB parent is charged 48 bytes here, against 33,554,480 on master, and the slice keeps the whole parent alive either way. Nothing tracks the retained backing, so the per-byte charge is what currently bounds how much a transaction can retain by slicing, and dropping it leaves that unbounded.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5082 -R gnolang/gno
cat > /tmp/zz_sc_test.go <<'EOF'
package gnolang

import (
	"strings"
	"testing"
)

func TestSliceChargeVsRetention(t *testing.T) {
	const parentLen = 64 << 20
	alloc := NewAllocator(1 << 30)
	tv := TypedValue{T: StringType, V: alloc.NewString(strings.Repeat("a", parentLen))}
	_, before := alloc.Status()

	// Half the parent. No bytes are copied on either revision: Go substrings
	// share the backing array, so the parent stays alive via the slice.
	slice := tv.GetSlice(alloc, 0, 32<<20)
	_, after := alloc.Status()

	t.Logf("32 MiB slice of a 64 MiB parent: charged %d bytes", after-before)
	_ = slice
}
EOF

# on the PR head:
cp /tmp/zz_sc_test.go gnovm/pkg/gnolang/zz_sc_test.go
go test -v -run TestSliceChargeVsRetention ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/zz_sc_test.go

# the same test on master:
git checkout origin/master
cp /tmp/zz_sc_test.go gnovm/pkg/gnolang/zz_sc_test.go
go test -v -run TestSliceChargeVsRetention ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/zz_sc_test.go
```

```
# PR head 3be8c3b1:
    zz_sc_test.go:19: 32 MiB slice of a 64 MiB parent: charged 48 bytes

# master 27b5b8e24:
    zz_sc_test.go:19: 32 MiB slice of a 64 MiB parent: charged 33554480 bytes
```
</details>

## gnovm/pkg/gnolang/alloc.go:88-90 [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc.go#L88)
`allocStringRef` and [`allocString`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/alloc.go#L86) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc.go#L86) are both `_allocHeap + 16`, but `StringValue` is a 24-byte struct now, so every string under-charges by 8 bytes. Unlike every other value type, it has no line in the [`init()` self-check](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/alloc.go#L132-L151) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc.go#L132-L151) to catch the drift.

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
`UnmarshalAmino` sets `ref=false`, so the same string [reports](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/alloc.go#L647-L653) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc.go#L647-L653) 48 bytes before a restart and 54 after. The GC rebuilds the allocator's counter from those numbers.

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
Nothing says [`GetSlice`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/values.go#L2233-L2240) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/values.go#L2233-L2240) must stay the only producer of reference mode. Reference mode is only free because slicing shares a backing, so a call site that passes a freshly built string would under-charge it by its whole length.

## gnovm/pkg/gnolang/alloc_test.go:51 [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc_test.go#L51)
Missing test: nothing pins the round-trip, where a ref reports 48 bytes going in and 54 coming back out as an owner.

<details><summary>test cases</summary>

```go
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
Suggestion: the `ref` bool takes `StringValue` from 16 to 24 bytes, so every literal and concatenation pays for a field only [`GetSlice`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/values.go#L2233-L2240) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/values.go#L2233-L2240) sets. A separate ref type would avoid that: deliberate trade for the simpler struct?

## gnovm/pkg/gnolang/values.go:97 [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/values.go#L97)
Nit: the new doc comments lack terminating periods, from `NewStringValue` here through `UnmarshalAmino` at line 128.

## gnovm/pkg/gnolang/alloc_test.go:87 [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc_test.go#L87)
Nit: `_ = result.GetString()` here and `_ = s3.GetString()` at line 113 discard the value, so they assert nothing.

## gnovm/pkg/gnolang/bench_test.go:52-53 [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/bench_test.go#L52)
Nit: `NewAllocator(1024*1024)` sits inside the `b.N` loop, so `ReportAllocs` counts allocator setup in the allocs/op this benchmark reports.
