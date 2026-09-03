# Review: [#6110](https://github.com/gnolang/gno/pull/6110)
Event: REQUEST_CHANGES

## Body
The mint partition holds for a stored string, a slice of it, a string-keyed map and a package literal kept live across a forced GC, each measuring the same offset from master warm, simulate-only, cold after a node restart and warm again, and `gnokey_gasfee` reports 1024118 alone and under six concurrent in-process nodes sharing the counter. The wire is mint-independent too, byte-identical to master across six cases including a nonzero ID and Extent and the `s[0:0]` shape.

## gnovm/pkg/gnolang/realm.go:1895 [gh](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/realm.go#L1895) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/realm.go#L1895)
Critical: this makes `fillTypesOfValue` allocate, and `loadObjectSafe` calls it four lines after `ds.cacheObjects[oid] = oo` and before any caller has wired the object in, so a GC triggered by the charge finds the object unstamped, `GarbageCollectObjectCache` deletes it, and the next `RefValue` with that `ObjectID` loads a second independent copy while the transaction commits normally. Minting before the cache insert closes the window, as does charging on a path that cannot collect, which is what the "Allocate atomically" comment twenty-five lines above `store.go:599` already protects against.

<details><summary>repro</summary>

Two package-level pointers to one persisted object; after a cold load a write through `A` must be visible through `B`.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6110 -R gnolang/gno
curl -fsSL -o gnovm/pkg/gnolang/string_load_alias_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6110-vm-owned-string-backing-id/1-4236ef9c6/tests/string_load_alias_test.go
go test -v -run TestZZAliasSurvivesLoad ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/string_load_alias_test.go
```

`broken` counts allocator caps where the write through `A` was invisible through `B`; `aborted` counts caps too tight to finish, so the 56 are completed transactions.

```
ZZ cold-load charge = 5448 bytes
ZZ ok=3746 broken=56 aborted=2199
ZZ broken caps: 5168..5223
--- FAIL: TestZZAliasSurvivesLoad
```

The same file on master, same sweep:

```
ZZ ok=3746 broken=0 aborted=2255
--- PASS: TestZZAliasSurvivesLoad
```

The trigger is a pure function of the transaction, so every node commits the same divergence rather than forking, and the 48-byte header charge is what crosses the cap, so any realm object carrying a string reaches it whatever the string's length.
</details>

## gnovm/pkg/gnolang/preprocess.go:2510 [gh](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/preprocess.go#L2510) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/preprocess.go#L2510)
Critical: `MapKey` compares types by `TypeID` string where the old `TypedValue` key compared by interface identity, so `map[any]int{(*int)(nil): 1, (*int)(nil): 2}` is now a duplicate, and one `maketx addpkg` carrying that literal today stops every upgraded node from starting, since `VMKeeper.Initialize` re-preprocesses every stored mempackage with no recover on the path. Restricting the check to keys Go treats as constants keeps it, since a typed nil is not one.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6110 -R gnolang/gno
cat > gnovm/tests/files/zz_nilptr.gno <<'EOF'
// PKGPATH: gno.land/r/test
package test

var m = map[any]int{(*int)(nil): 1, (*int)(nil): 2}

func main() { println(len(m)) }

// Output:
// 1
EOF
go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zz_nilptr.gno$' -count=1
rm gnovm/tests/files/zz_nilptr.gno
```

The golden is what master and `go run` both produce, so the failure is the finding:

```
--- FAIL: TestFiles/zz_nilptr.gno (0.00s)
    files_test.go:135: unexpected panic: gno.land/r/test/zz_nilptr.gno:4:9-52: duplicate key (nil *int) in map literal
```

Master: `ok github.com/gnolang/gno/gnovm/pkg/gnolang 1.814s`. `(**int)(nil)`, `(*T)(nil)` and a mixed literal behave the same way. The control that isolates the mechanism is `map[*int]int{nil: 1, nil: 2}`, whose keys share the map's own single `*PointerType`: that one is refused on both versions. Nothing in the tree carries the shape today, across 21 interface-keyed map literals and no typed-nil key among them, which is what makes it cheap to create rather than reassuring.
</details>

## gnovm/pkg/gnolang/alloc.go:824-826 [gh](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/alloc.go#L824-L826) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/alloc.go#L824-L826)
The header charge returned here is [`allocString`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/alloc.go#L110-L112), still `_allocHeap + 16` under a comment reading "StringValue is a Go string (16 bytes, by value)", and the struct this branch introduces measures 32, so every live string value is billed 48 where the model wants 64, here and in `AllocateString`.

<details><summary>repro</summary>

Every other by-value type in that const block derives its size from `unsafe.Sizeof` and is asserted against it by the `init()` guard at `alloc.go:155-174`, which carries thirteen entries and none for `StringValue`.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6110 -R gnolang/gno
cat > gnovm/pkg/gnolang/zz_sz_test.go <<'EOF'
package gnolang

import (
	"testing"
	"unsafe"
)

func TestZZStringValueHeaderCharge(t *testing.T) {
	actual := int64(unsafe.Sizeof(StringValue{}))
	if want := int64(_allocHeap) + actual; allocString != want {
		t.Fatalf("allocString = %d; StringValue is %d bytes, so the model wants _allocHeap + %d = %d (short by %d)",
			allocString, actual, actual, want, want-allocString)
	}
}
EOF
go test ./gnovm/pkg/gnolang/ -run TestZZStringValueHeaderCharge -count=1
rm gnovm/pkg/gnolang/zz_sz_test.go
```

```
--- FAIL: TestZZStringValueHeaderCharge (0.00s)
    zz_sz_test.go:14: allocString = 48; StringValue is 32 bytes, so the model wants _allocHeap + 32 = 64 (short by 16)
```

The same test passes on master, where the type is 16 bytes. A program holding 1000 live string values recounts 105682 bytes at 48 and 121682 at 64.
</details>

## gnovm/pkg/gnolang/garbage_collector.go:240 [gh](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/garbage_collector.go#L240) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/garbage_collector.go#L240)
An untracked string recounts at 48 bytes whatever its length, because the byte charge is gated on `ID != 0` and `GetShallowSize` dropped to the header alone, so a retained 8445-byte panic value costs 48 here and 8493 on master.

<details><summary>repro</summary>

The three production constructors that leave `ID` zero are `typedString`, `typedRuntimeError` and the copy in `gnovm/stdlibs/chain/runtime/native.go:64`, and gno `recover()` hands all three to user code, which may store them in a realm global. That is 97 times short on the one class the ADR excludes by design, and it is a regression rather than a known gap: master's `GetShallowSize` was `allocString + allocStringByte*int64(len(sv))`.

Not a memory denial of service, and I checked before writing this: retaining an untracked error charges the same 28534 gas at both heads, and a tracked string of the same size costs 21656 gas for 21210 charged bytes, so the untracked path buys fewer real bytes per gas than the tracked one. Gas binds it; the accounting does not.

Charging `len(sv.Str)` when `ID` is zero restores the floor.
</details>

## gnovm/pkg/gnolang/values.go:2708 [gh](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/values.go#L2708) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/values.go#L2708)
Inheriting the source's `Extent` charges the whole backing for a slice that outlives it, which is what Go's retain semantics call for, so a transaction holding a one-byte slice of a large dead backing now needs the full backing's headroom against `maxAllocTx`. No fixture asserts that headroom and no line in the body names it.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6110 -R gnolang/gno
curl -fsSL -o gnovm/tests/files/zz_maxalloc.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6110-vm-owned-string-backing-id/1-4236ef9c6/tests/slice_extent_maxalloc.gno
go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zz_maxalloc.gno$' -count=1 -v
rm gnovm/tests/files/zz_maxalloc.gno
```

A 100000-byte string sliced to one byte with the source dropped, collected, then rebuilt, under `MAXALLOC: 260000`:

```
master:  ok 1 100000 Allocator{maxBytes:260000, bytes:209521}
head:    unexpected panic: allocation limit exceeded
```

The same shape without the second build reports 114872 allocator bytes here against 14873 on master, a difference of 99999. Exposure resets at the transaction boundary, since amino persists `Str` alone and the load path re-mints with `Extent = len(Str)`.
</details>

## gnovm/pkg/gnolang/values.go:2704 [gh](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/values.go#L2704) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/values.go#L2704)
Missing test: dropping the per-byte charge here takes 73.1 per cent off the gas for string slicing, and no golden in `gno.land/pkg/integration/testdata/` or `gnovm/tests/files/gas/` slices a string, so the branch's largest consensus-visible movement is the one nothing measures.

<details><summary>test case</summary>

`gnovm/tests/files/gas/slice.gno`, 100 slices of 900000 bytes off one backing:

```go
// MAXALLOC: 500000000
package main

import "strings"

func main() {
	s := strings.Repeat("a", 1000000)
	n := 0
	for i := 0; i < 100; i++ {
		t := s[i : i+900000]
		n += len(t)
	}
	_ = n
}

// Gas:
// 5958729
```

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6110 -R gnolang/gno
go test ./gnovm/pkg/gnolang/ -run 'TestFiles/gas/slice.gno$' -count=1 -v
```

The committed goldens that did move all sit between 0.0001 and 0.033 per cent, and every one of them is the new load charge. Master reports 22155232 on the fixture above.
</details>

## gnovm/adr/pr6110_string_backing_id.md:112-113 [gh](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/adr/pr6110_string_backing_id.md?plain=1#L112-L113) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/adr/pr6110_string_backing_id.md#L112-L113)
"gas txtars unchanged" is what a reviewer reads to decide the branch is not gas-affecting, and it moves nine gas assertions across six files, as re-minting every loaded string must.

<details><summary>the six</summary>

| Fixture | before | after | delta |
|---|---|---|---|
| `gc.txtar` | 151131783 | 151131933 | +150 |
| `gnokey_gasfee.txtar` | 1023935 | 1024118 | +183 |
| `stdlib_ibc_crypto_determinism.txtar` | 2547040 | 2547225 | +185 |
| `compute_map_key_restore_gas.txtar` | 2268351 | 2268663 | +312 |
| `stdlib_restart_compare.txtar` | 1985075 | 1985712 | +637 |
| `simulate_gas.txtar` | 958ugnot | 959ugnot | three fee assertions |

Roughly 57 gas per loaded string. The neighbouring sentence, "`alloc_0/1/7` +26/+16/+11 B", is measured against #4885 as well: `alloc_1.gno` and `alloc_7.gno` are not in this diff and pass unmodified, and master's `alloc_0.gno` body run at this head gives 8091 against its committed 8094.
</details>

## gnovm/pkg/gnolang/values.go:100-101 [gh](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/values.go#L100-L101) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/values.go#L100-L101)
Nothing enforces this note: `.github/golangci.yml` is `default: none`, none of the eighteen enabled linters inspects an operand's type, and `nodes.go:2404`'s `if tv.V != old.V` in `StaticBlock.Define2` already compares the mint where it compared content.

<details><summary>the walk, and a test that would hold the line</summary>

A typed walk over the 50 packages under `gnovm/` finds five `==` or `!=` sites on a string-carrying type. Four cannot receive a `StringValue`: three sit in `isEql` branches the kind dispatch reaches only for Map, Slice, Func and Pointer values, and the fourth is `seenValues.IndexOf`, whose two callers pass only array, struct, map, pointer and slice values.

`nodes.go:2404` is the fifth and I could not reach it: a 30-case redefinition battery matches master exactly, and instrumenting the site to report instead of panic gave zero hits over the whole `TestFiles` corpus. So it is latent, and the missing guard rather than the site is the point.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6110 -R gnolang/gno
curl -fsSL -o gnovm/pkg/gnolang/stringvalue_equality_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6110-vm-owned-string-backing-id/1-4236ef9c6/tests/stringvalue_equality_test.go
go test -v -run TestNoDirectStringValueEquality ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/stringvalue_equality_test.go
```

It keys its allow-list on `basename:line` with a stated reason per entry, uses `golang.org/x/tools/go/packages` which is already a direct require, and fails on a planted `sv.Fields[realmFieldPkgPath] == sv.Fields[realmFieldAddr]`.
</details>

## gnovm/tests/files/alloc_13b.gno:14-18 [gh](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/tests/files/alloc_13b.gno#L14-L18) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/tests/files/alloc_13b.gno#L14-L18)
Missing test: `s1 := s` keeps a full-length value alive carrying the source's own `Extent`, so the slice's `Extent` is never load-bearing and the zero-count direction this header names goes uncaught across all 27 `alloc_*` filetests.

<details><summary>test case</summary>

`alloc_13b` with `s1` removed, so only the ten-byte slice survives:

```go
// MAXALLOC: 100000
// PKGPATH: gno.land/r/test

package test

import (
	"runtime"
)

func main() {
	base := `<the same 512 bytes as alloc_13b>`
	s := base + base // 1024B fresh backing: charged and tracked
	s2 := s[1:11]    // slice shares the backing; the only survivor
	s = ""           // source dies
	base = ""        // literal ref dies too
	runtime.GC()
	println("after GC: ", runtime.MemStats())
	println(len(s2))
}

// Output:
// after GC:  Allocator{maxBytes:100000, bytes:8872}
// 10
```

Full file at [`tests/alloc_13c.gno`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6110-vm-owned-string-backing-id/1-4236ef9c6/tests/alloc_13c.gno). Green at this head; setting `Extent` to `high - low` moves it to 7858 and dropping identity moves it to 7848, while `alloc_13b` and the other 26 stay green under both.
</details>

## gnovm/pkg/gnolang/parity_test.go:88-89 [gh](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/parity_test.go#L88-L89) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/parity_test.go#L88-L89)
Missing test: both `StringValue` cases carry `ID` zero and `Extent` zero, so neither can distinguish a codec that persists the mint from one that does not, and mint-independence is what keeps two nodes with different restart histories in agreement.

<details><summary>test case</summary>

`AssertCodecParity` cannot be extended to cover it, because its roundtrip-fidelity invariant is deliberately false for this type and the next person to add a minted case gets a red test with no explanation. Two tests instead, at [`tests/stringvalue_wire_test.go`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6110-vm-owned-string-backing-id/1-4236ef9c6/tests/stringvalue_wire_test.go): one asserting the encoded bytes are identical across a mint, a second mint of the same content, a slice, an untracked value and a zero-length slice of a tracked source; one asserting the object hash matches for equal content under two different mints and straight off the wire.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6110 -R gnolang/gno
curl -fsSL -o gnovm/pkg/gnolang/stringvalue_wire_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6110-vm-owned-string-backing-id/1-4236ef9c6/tests/stringvalue_wire_test.go
go test -v -run 'TestStringValueWireIsMintIndependent|TestObjectHashIsMintIndependent' ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/stringvalue_wire_test.go
```
</details>
