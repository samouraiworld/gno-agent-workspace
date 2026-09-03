# PR [#6110](https://github.com/gnolang/gno/pull/6110): feat(gnovm)!: VM-owned string backing identity — mint IDs in StringValue (alt. to #4885)

URL: https://github.com/gnolang/gno/pull/6110
Author: ltzmaxwell | Base: master | Files: 45 | +727 -113
Reviewed by: davd-gzl | Model: claude-opus-5 (deep) | Commit: `4236ef9c6` (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6110 4236ef9c6`
Overview: [visual overview](../overview.html)

## Overview

The GnoVM caps how much memory one transaction may hold and rebuilds that total from scratch at each garbage collection by walking every reachable value. Strings broke the rebuild in three directions at once: a string loaded from the store was never counted, two strings sharing one Go byte array were each billed for all of it, and slicing charged for a copy that never happened. This branch answers the question the rebuild needs, which is whether a given byte array has already been counted, by giving every string a serial number the VM issues itself. Copies and slices inherit the serial and the full length of the array behind them, so the collector charges each array once per cycle whichever value it reaches first. The competing branch, [#4885](https://github.com/gnolang/gno/pull/4885), answers the same question with Go heap addresses and pays for a treap, a clone rule and a pin to keep those addresses stable; this one carries the answer inside the value and deletes all of that machinery.

**Verdict: REQUEST CHANGES** — charging for a string on the object-load path put an allocation, and therefore a collection, inside the window where a loaded object is in the store cache and not yet reachable, which silently splits one on-chain object into two and commits the result (2 critical, 6 warnings, 4 missing tests, 4 suggestions, 5 nits).

## Verify first

- [`gnovm/pkg/gnolang/realm.go:1895`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/realm.go#L1895) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/realm.go#L1895) — this line makes `fillTypesOfValue` allocate, and [`store.go:603`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/store.go#L603) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/store.go#L603) calls it four lines after the object goes into `cacheObjects` and before any caller has wired it into the graph. Run `tests/string_load_alias_test.go` and confirm every allocator cap keeps two pointers to one object pointing at one object.
- [`gnovm/pkg/gnolang/preprocess.go:2510`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/preprocess.go#L2510) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/preprocess.go#L2510) — the duplicate-key set changed from Go equality on `TypedValue`, which compares `T` by interface identity, to `MapKey`, which compares `T` by `TypeID` string. Deploy `var m = map[any]int{(*int)(nil): 1, (*int)(nil): 2}` on master, then boot this branch over that store.
- [`gnovm/pkg/gnolang/alloc.go:112`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/alloc.go#L112) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/alloc.go#L112) — `allocString` still charges a 16-byte value. Print `unsafe.Sizeof(StringValue{})` at this head and decide whether the 32-byte struct should cost 32, since bumping it moves every gas golden.

## Summary

`StringValue` becomes `{Str string; ID uint64; Extent int64}` at [`values.go:102-107`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/values.go#L102-L107) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/values.go#L102-L107); `NewString` stamps a serial from a process-global atomic at [`alloc.go:526`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/alloc.go#L526) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/alloc.go#L526), and the collector adds `allocStringByte * Extent` once per serial per walk from a visitor-local set at [`garbage_collector.go:240-245`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/garbage_collector.go#L240-L245) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/garbage_collector.go#L240-L245). The claim the whole design rests on, that the numeric serial never reaches consensus and only the grouping it induces matters, holds under every path I could reach. What the branch does not carry is a matching correction to the cost model, and one of its two new charge sites lands somewhere the allocator was never allowed to run before.

Reading order: [`values.go:102-116`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/values.go#L102-L116) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/values.go#L102-L116) for the type and its amino conversion, then [`alloc.go:519-527`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/alloc.go#L519-L527) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/alloc.go#L519-L527) for the mint, then the collector, then [`values.go:2698-2710`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/values.go#L2698-L2710) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/values.go#L2698-L2710) for slicing, then [`realm.go:1894-1895`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/realm.go#L1894-L1895) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/realm.go#L1894-L1895) for the load path, and last [`preprocess.go:2505-2517`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/preprocess.go#L2505-L2517) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/preprocess.go#L2505-L2517), which exists only because the struct lost value equality.

## Benchmarks / Numbers

Gas. The first five rows are the branch's own committed goldens, re-measured at `2ed70a202`; the last three come from fixtures in [`tests/`](./tests/).

| Measurement | base | head | delta | percent |
|---|---|---|---|---|
| `gc.txtar` `Alloc` | 151131783 | 151131933 | +150 | +0.0001 |
| `gnokey_gasfee` `Hello` | 1023935 | 1024118 | +183 | +0.0179 |
| `stdlib_ibc_crypto_determinism` | 2547040 | 2547225 | +185 | +0.0073 |
| `compute_map_key_restore_gas` `Insert` | 2268351 | 2268663 | +312 | +0.0138 |
| `stdlib_restart_compare` `Convert` | 1985075 | 1985712 | +637 | +0.0321 |
| [`slice_gas.gno`](./tests/slice_gas.gno), 100 slices of 900000 bytes off one backing | 22155232 | 5958729 | **−16196503** | **−73.1** |
| [`load_gas.txtar`](./tests/load_gas.txtar), 2000 stored strings cold-loaded in one call | 21447206 | 21561348 | +114142 | +0.53 |
| [`mint_partition.txtar`](./tests/mint_partition.txtar), forced-collection realm call | 76971588 | 76971988 | +400 | +0.0005 |

`simulate_gas.txtar` moved too, from `gas fee: 958ugnot` to `959ugnot` and two more fee assertions; it asserts a fee rather than a gas figure, so it is not in the table.

Allocator bytes, which feed `maxAllocTx` and collection timing rather than gas:

| Shape | base | head |
|---|---|---|
| [`slice_extent_recount.gno`](./tests/slice_extent_recount.gno), one-byte slice of a 100000-byte backing, source dead, after `runtime.GC()` | 14873 | 114872 |
| [`slice_extent_maxalloc.gno`](./tests/slice_extent_maxalloc.gno), the same shape twice under `MAXALLOC: 260000` | 209521 | aborts |
| 1000 live string values, with `allocString` at 48 and at 64 | 105682 | 121682 |

## Critical (must fix)

- **[one on-chain object becomes two, and the transaction commits]** [`realm.go:1895`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/realm.go#L1895) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/realm.go#L1895) — charging for a loaded string puts a collection inside the window where the object is in the store cache and nothing else references it, so the collector evicts it and the next reference loads an independent copy.
  <details><summary>details</summary>

  `loadObjectSafe` writes `ds.cacheObjects[oid] = oo` at [`store.go:599`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/store.go#L599) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/store.go#L599) and calls `fillTypesOfValue` at [`store.go:603`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/store.go#L603) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/store.go#L603). At `2ed70a202` that call reaches the allocator on no path. This branch adds `store.GetAllocator().NewString(cv.Str)`, which reaches `alloc.collect()` at [`alloc.go:345`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/alloc.go#L345) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/alloc.go#L345) whenever the charge crosses the cap.

  A collection ends by running `GarbageCollectObjectCache` at [`garbage_collector.go:77`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/garbage_collector.go#L77) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/garbage_collector.go#L77), which deletes every cache entry the walk did not stamp, at [`store.go:1442`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/store.go#L1442) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/store.go#L1442). The object being loaded is stamped by nothing, because callers wire it in only after `GetObject` returns, as at [`values.go:328-329`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/values.go#L328-L329) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/values.go#L328-L329). So it is evicted, `loadObjectSafe` returns it anyway, and the next `RefValue` carrying that `ObjectID` misses the cache and loads a second copy. Two Go objects, one `ObjectID`, both live in one transaction.

  The file already knows this hazard. Twenty-five lines above the eviction window, [`store.go:574-576`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/store.go#L574-L576) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/store.go#L574-L576) reads "Allocate atomically: one Allocate call prevents GC from intercepting between shallow-size and RefValue-size accounting". That guard covers the allocation before the cache insert and not the one this branch adds after it.

  Nothing aborts. The write through one reference is finalized and committed while the other reference reads the old value, and the trigger is a pure function of the transaction, so every honest node commits the same corrupt state rather than forking. Any realm object carrying a string reaches this line, whatever the string's length, because it is the 48-byte header charge that crosses the cap.

  **Repro:** two package-level pointers to one persisted `*Node`, then a cold call writing through the first and reading through the second, swept across allocator caps around the load's own cost. Full fixture at [`tests/string_load_alias_test.go`](./tests/string_load_alias_test.go).

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 6110 -R gnolang/gno
  curl -fsSL -o gnovm/pkg/gnolang/string_load_alias_test.go \
    https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6110-vm-owned-string-backing-id/1-4236ef9c6/tests/string_load_alias_test.go
  go test -v -run TestZZAliasSurvivesLoad ./gnovm/pkg/gnolang/
  rm gnovm/pkg/gnolang/string_load_alias_test.go
  ```

  The failure is the finding: `broken` counts caps where a write through `A` was invisible through `B`, and `aborted` counts caps too tight to finish at all, so these are completed transactions.

  ```
  ZZ cold-load charge = 5448 bytes
  ZZ ok=3746 broken=56 aborted=2199
  ZZ broken caps: 5168..5223
  --- FAIL: TestZZAliasSurvivesLoad
  ```

  The same file at `2ed70a202`, same sweep:

  ```
  ZZ cold-load charge = 5392 bytes
  ZZ ok=3746 broken=0 aborted=2255
  --- PASS: TestZZAliasSurvivesLoad
  ```

  Fix: keep the load-path mint out of `Allocate`. Either run `fillTypesOfValue` before the cache insert, or charge it on this route through a path that cannot collect, which is what the atomic-allocate comment twenty-five lines above is already protecting.
  </details>

- **[one permissionless transaction, sent today, stops every node from starting once this ships]** [`preprocess.go:2510`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/preprocess.go#L2510) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/preprocess.go#L2510) — `map[any]int{(*int)(nil): 1, (*int)(nil): 2}` deploys on master, and this branch refuses it inside the boot-time preprocess of every stored package.
  <details><summary>details</summary>

  Master keyed the duplicate set on `TypedValue`, whose Go equality compares `T` by interface identity. `MapKey` compares `T` by its `TypeID` string. Two `(*int)(nil)` conversions build two separate `*PointerType` instances carrying one `TypeID`, and under an interface-typed map key `checkOrConvertType` leaves each key at its own conversion-site type, so master saw two keys where this head sees one. The control that isolates the mechanism is `map[*int]int{nil: 1, nil: 2}`, whose keys share the map's own single `*PointerType`: that one is refused on both versions. Go compiles the literal and collapses the two keys at run time, so this narrows the accepted language.

  The refusal runs at boot. [`machine.go:338-353`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/machine.go#L338-L353) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/machine.go#L338-L353) re-preprocesses every stored mempackage, because [`store.go:942-955`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/store.go#L942-L955) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/store.go#L942-L955) has its backend write commented out and a BlockNode therefore exists only because preprocess built it. `IterMemPackage` walks the whole backend package index, user deployments included. That loop sits inside `VMKeeper.Initialize` at [`keeper.go:203`](https://github.com/gnolang/gno/blob/4236ef9c6/gno.land/pkg/sdk/vm/keeper.go#L203) · [↗](../../../../../.worktrees/gno-review-6110/gno.land/pkg/sdk/vm/keeper.go#L203), which runs during app construction on every boot with no recover on the path.

  Nothing in the tree carries the shape today: 21 interface-keyed map literals, none with a typed-nil key. That is the point rather than the reassurance. The literal costs one `maketx addpkg` from any account, it passes the Go type checker, and it is inert until this branch merges, at which point every node that restarts fails to start and no upgrade short of a code patch recovers them.

  **Repro:**

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

  The golden is what master and Go both produce, so the failure is the finding:

  ```
  --- FAIL: TestFiles/zz_nilptr.gno (0.00s)
      files_test.go:135: unexpected panic: gno.land/r/test/zz_nilptr.gno:4:9-52: duplicate key (nil *int) in map literal
  ```

  Same file at `2ed70a202`: `ok github.com/gnolang/gno/gnovm/pkg/gnolang 1.814s`. Under go1.25.9 the literal has length 1, and `(**int)(nil)`, `(*T)(nil)` and a mixed literal behave the same way.

  Fix: run the duplicate check only for keys Go itself treats as constants, since a typed nil is not one and master's type-identity comparison was what let it through.
  </details>

## Warnings (should fix)

- **[the value grew to 32 bytes and its charge stayed at 16]** [`alloc.go:110-112`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/alloc.go#L110-L112) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/alloc.go#L110-L112) — every live string value is billed 48 where the file's own arithmetic wants 64, which is 15.1 per cent of the recount on a string-heavy program.
  <details><summary>details</summary>

  The constant reads `allocString = _allocHeap + 16` under the comment "StringValue is a Go string (16 bytes, by value)". At `2ed70a202` that was exact: `type StringValue string` measures 16. This branch makes it a three-field struct measuring 32 and leaves both the constant and the comment alone. Every other by-value entry in that block derives its size from `unsafe.Sizeof` and is asserted against it by the `init()` guard at [`alloc.go:155-174`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/alloc.go#L155-L174) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/alloc.go#L155-L174), which carries thirteen entries and none for `StringValue`. The constant feeds `AllocateString` as well as `GetShallowSize`, so the drift is on the charge path and not only the recount. A program holding 1000 live string values recounts 105682 bytes at 48 and 121682 at 64.

  **Repro:**

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

  The same test passes at `2ed70a202`. Fix: raise the constant to `_allocHeap + 32` and add a `_allocStringValue` entry to the `init()` guard, or say in the ADR that the header charge is deliberately held at the old size.
  </details>

- **[VM-internal text lost the byte charge master gave it]** [`alloc.go:824-826`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/alloc.go#L824-L826) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/alloc.go#L824-L826) — a retained 8445-byte panic string recounts as 48 bytes here and as 8493 at master.
  <details><summary>details</summary>

  `GetShallowSize` returned `allocString + allocStringByte*len(sv)` on master, so the collector charged every string in full whether or not the allocator had minted it. Here it returns the header alone and the bytes come back only when `sv.ID != 0` at [`garbage_collector.go:240`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/garbage_collector.go#L240) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/garbage_collector.go#L240). Every VM panic string is built by `typedString` at [`values.go:3347`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/values.go#L3347) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/values.go#L3347), by `typedRuntimeError` at [`values.go:3363`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/values.go#L3363) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/values.go#L3363) and by the copy in [`chain/runtime/native.go:64`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/stdlibs/chain/runtime/native.go#L64) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/stdlibs/chain/runtime/native.go#L64), and gno `recover()` hands all three to user code, which may retain them. The marginal charge per retained error falls from 8493 to 48, which is 97 times short on a path the ADR calls a bounded undercount.

  It is not a memory denial of service and I checked: on a metered machine the same programs charge the same 28534 gas per retained error at both heads, and a tracked string of the same size costs 21656 gas for 21210 charged bytes, so untracked retention buys fewer real bytes per gas than tracked retention. Gas binds the vector, not the allocator. What is left is that the invariant the branch exists to establish is 97 times wrong on the one class it deliberately excludes, and that master was right here.

  Fix: keep `GetShallowSize` charging `len(sv.Str)` when `ID` is zero, so an untracked value costs at least what it occupies.
  </details>

- **[a small slice of a large dead backing now needs the whole backing's headroom]** [`values.go:2708`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/values.go#L2708) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/values.go#L2708) — a fixture that completes under `MAXALLOC` at master aborts here with `allocation limit exceeded`.
  <details><summary>details</summary>

  The slice inherits the source's `Extent`, and the collector charges `allocStringByte * Extent` per live serial per walk. When only a small slice survives a large backing, the recount charges the whole array where master charged the visible window. [`slice_extent_recount.gno`](./tests/slice_extent_recount.gno) reports 114872 allocator bytes here against 14873 at `2ed70a202`, a difference of 99999 for a one-byte slice of a 100000-byte backing.

  The accounting is right and Go agrees with it, since a string slice pins its whole backing, and this is the correction the design set out to make. The finding is the blast radius: any realm keeping a small slice of a large string live across a collection inside one transaction now consumes the full backing against `maxAllocTx`, which is 500000000 at [`keeper.go:50`](https://github.com/gnolang/gno/blob/4236ef9c6/gno.land/pkg/sdk/vm/keeper.go#L50) · [↗](../../../../../.worktrees/gno-review-6110/gno.land/pkg/sdk/vm/keeper.go#L50). Exposure resets at the transaction boundary, because amino persists `Str` alone and `fillTypesOfValue` re-mints with `Extent = len(Str)`; within one transaction it is a hard abort. No fixture asserts this direction and the pull request body does not name it.

  **Repro:** [`slice_extent_maxalloc.gno`](./tests/slice_extent_maxalloc.gno), a 100000-byte string built by concatenation, sliced to one byte, source dropped, collected, then rebuilt, under `MAXALLOC: 260000`.

  ```
  2ed70a202:  ok 1 100000 Allocator{maxBytes:260000, bytes:209521}
  4236ef9c6:  unexpected panic: allocation limit exceeded
  ```

  Fix: ship a filetest of that shape and say in the body that a transaction holding a small slice of a large dead backing now needs the full backing's headroom.
  </details>

- **[the largest gas movement in the branch is pinned by no golden]** [`values.go:2704`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/values.go#L2704) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/values.go#L2704) — string slicing costs 73.1 per cent less gas and nothing in the tree measures it.
  <details><summary>details</summary>

  Master charged `AllocateString(high-low)`, a header plus the sliced bytes; this head charges `alloc.Allocate(allocString)`, the header alone. [`slice_gas.gno`](./tests/slice_gas.gno), which takes 100 slices of 900000 bytes off one backing, reports 22155232 gas at `2ed70a202` and 5958729 here.

  Every gas golden in `gno.land/pkg/integration/testdata/` moved by between 0.0001 and 0.033 per cent, all of it the new load charge, and none of them slices a string. `gnovm/tests/files/gas/` has no slicing fixture either. So the one consensus-visible change large enough for a validator to notice is invisible to every job in CI.

  Fix: add a `gnovm/tests/files/gas/` fixture that slices a string, so the movement is pinned rather than free-floating.
  </details>

- **[the ADR tells a maintainer that gas did not move]** [`pr6110_string_backing_id.md:112-113`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/adr/pr6110_string_backing_id.md?plain=1#L112-L113) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/adr/pr6110_string_backing_id.md#L112-L113) — "gas txtars unchanged", in a diff that changes six of them.
  <details><summary>details</summary>

  The branch moves nine gas assertions across six files under `gno.land/pkg/integration/testdata/`, and `simulate_gas.txtar` carries three of them plus two comment lines. The diff is right and the sentence is wrong: re-minting every loaded string at [`realm.go:1895`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/realm.go#L1895) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/realm.go#L1895) must move gas, and it does, at roughly 57 gas per loaded string.

  This is the one line a reviewer reads to decide the change is not gas-affecting, in the document that becomes the merged record of a consensus change and outlives the pull request page where the six changed files sit beside it. Fix: replace it with the measured table.
  </details>

- **[nothing enforces the equality rule the design depends on]** [`values.go:100-101`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/values.go#L100-L101) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/values.go#L100-L101) — the prohibition on comparing `StringValue` by `==` is prose, and one existing site already violates it.
  <details><summary>details</summary>

  The type doc says to compare by `Str` and the ADR says new direct `==` on a `TypedValue` or `.V` holding strings must not be introduced. [`.github/golangci.yml`](https://github.com/gnolang/gno/blob/4236ef9c6/.github/golangci.yml?plain=1#L13) · [↗](../../../../../.worktrees/gno-review-6110/.github/golangci.yml#L13) is `default: none` and none of its eighteen enabled linters inspects an operand's type; `gocritic`, whose ruleguard could express this, has settings but is absent from the enable list, so that block is dead.

  A typed walk over the 50 packages under `gnovm/` finds five `==` or `!=` sites on a string-carrying type. Four cannot receive a `StringValue`: three sit in `isEql` branches the kind dispatch reaches only for Map, Slice, Func and Pointer values, and the fourth is `seenValues.IndexOf`, whose two callers pass only array, struct, map, pointer and slice values. The fifth is [`nodes.go:2404`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/nodes.go#L2404) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/nodes.go#L2404), `if tv.V != old.V` in `StaticBlock.Define2`, which compared content on master and compares the serial here, since a literal mints afresh on every evaluation. I could not reach it: a 30-case redefinition battery matches master exactly, and instrumenting the site to report instead of panic gave zero hits over the whole `TestFiles` corpus. So the site is latent, and the missing guard is the finding.

  Fix: land [`tests/stringvalue_equality_test.go`](./tests/stringvalue_equality_test.go), which walks the package for `==` on a string-carrying type against an allow-list carrying a traced reason per entry, with `nodes.go:2404` either comparing `Str` or listed.
  </details>

## Missing Tests

- **[the fixture named as the design's proof cannot fail in the direction it names]** [`alloc_13b.gno:14-18`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/tests/files/alloc_13b.gno#L14-L18) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/tests/files/alloc_13b.gno#L14-L18) — it keeps a full copy beside the slice, so the slice's own `Extent` is never load-bearing.
  <details><summary>details</summary>

  The header says counting zero times is the ref-flag undercount that sinks a #5082-style design, and the pull request body cites this fixture as what pins it. It does catch the double-count direction: making `GetSlice` mint a fresh serial moves the golden from 8920 to 9944. The other direction is uncovered by the whole suite, not only by this fixture. Setting `Extent` to the slice length, and separately dropping identity altogether, each leave all 27 `alloc_*` filetests green.

  [`tests/alloc_13c.gno`](./tests/alloc_13c.gno) closes it: `alloc_13b` with `s1` removed so only the ten-byte slice survives. It is green at this head at 8872, and moves to 7858 and 7848 under the two breakages that leave everything else green.
  </details>

- **[no codec case carries a live mint, so nothing proves the wire is mint-independent]** [`parity_test.go:88-89`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/parity_test.go#L88-L89) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/parity_test.go#L88-L89) — both `StringValue` cases are `ID` zero and `Extent` zero.
  <details><summary>details</summary>

  The property that keeps two nodes agreeing is that the serial never reaches the store, and object hashes are taken over `amino.MustMarshalAny` output at [`store.go:656-663`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/store.go#L656-L663) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/store.go#L656-L663). Neither committed case can distinguish a codec that persists the mint from one that does not, because neither carries one, and the counter is a process-global that survives across transactions, so a node that restarted mints differently from one that did not. The project's own gate cannot be extended to cover it either: `AssertCodecParity`'s roundtrip-fidelity invariant is deliberately false for this type, so the next person to add a minted case gets a red test with no explanation.

  [`tests/stringvalue_wire_test.go`](./tests/stringvalue_wire_test.go) closes it with two tests, one asserting the encoded bytes are identical across a mint, a second mint of the same content, a slice, an untracked value and a zero-length slice of a tracked source, and one asserting the object hash is the same for equal content under two different mints and straight off the wire.
  </details>

- **[the capture carve-out the new comment claims has no fixture]** [`realm.go:1928-1930`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/realm.go#L1928-L1930) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/realm.go#L1928-L1930) — the comment's claim measures out at 1052 bytes, and nothing would notice if it stopped holding.
  <details><summary>details</summary>

  The comment says `FuncValue` captures need no walking because each is a `*HeapItemValue` persisted as its own object. With the re-mint reverted, a closure-captured 1052-byte string collapses by exactly 1052 bytes, 9596 to 8544, against a four-byte control that collapses by 4, so the capture does travel through `fillTypesOfValue`. [`tests/alloc_15c.gno`](./tests/alloc_15c.gno), [`alloc_15d.gno`](./tests/alloc_15d.gno) and [`alloc_15e.gno`](./tests/alloc_15e.gno) state it, the last two being the pair whose difference is the assertion.
  </details>

- **[untracked propagation and the two incompatible empty strings]** [`values.go:2708`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/values.go#L2708) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/values.go#L2708) — slicing an untracked string yields an untracked slice, and two zero-length strings behave differently.
  <details><summary>details</summary>

  `NewString("")` gives `ID` zero and `Extent` zero at [`alloc.go:521-522`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/alloc.go#L521-L522) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/alloc.go#L521-L522), while `s[5:5]` on a tracked megabyte gives a serial and an `Extent` of a million, so one surviving zero-length slice pins the whole backing in the recount. Go retains the backing for a zero-length slice too, so this is the conservative answer rather than a defect, but it is an unstated invariant and precisely what a later optimisation of empty strings would break. [`tests/stringvalue_untracked_test.go`](./tests/stringvalue_untracked_test.go) states it in three subtests.
  </details>

## Suggestions

- **[a second direct equality site is missing from the ADR's audit]** [`nodes.go:2404`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/nodes.go#L2404) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/nodes.go#L2404) — the ADR names `kset` as the one site; `StaticBlock.Define2` is another. Unreached over the whole `TestFiles` corpus, so latent only. Comparing `Str` there is one line and closes the class.
- **[user code can hold an untracked string, which the ADR does not say]** [`pr6110_string_backing_id.md:42-44`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/adr/pr6110_string_backing_id.md?plain=1#L42-L44) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/adr/pr6110_string_backing_id.md#L42-L44) — the ADR describes `ID == 0` as VM panic text and uverse init. `recover()` hands that text to user code, which can store it in a realm global. The undercount is bounded by gas rather than by anything in the accounting, and the ADR reads as though it were bounded by the design.
- **[the ADR names two untracked constructors and there are three]** [`chain/runtime/native.go:64`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/stdlibs/chain/runtime/native.go#L64) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/stdlibs/chain/runtime/native.go#L64) — a third production `typedString`, reached by `AssertOriginCall`'s panic and recoverable like the other two.
- **[`alloc_14a` gates nothing `alloc_14` does not]** [`alloc_14a.gno:4-6`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/tests/files/alloc_14a.gno#L4-L6) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/tests/files/alloc_14a.gno#L4-L6) — with the collector's dedup disabled, `alloc_14`'s golden becomes exactly 9502, which is `alloc_14a`'s own value, and both pass when the accessors are reverted to untracked copies. It is a reference point rather than an assertion, and its header reads as the latter.

## Nits

- [`uverse.go:268-269`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/uverse.go#L268-L269) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/uverse.go#L268-L269) — "pkgPath usually shares the realm path's backing, so `NewString` clones it to keep ranges disjoint" describes #4885's `trackString` clone-on-overlap. This branch has no ranges and `NewString` at [`alloc.go:519-527`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/alloc.go#L519-L527) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/alloc.go#L519-L527) clones nothing. Commit 63555bc swept for stale references of this class and missed this one. Not posted: a comment changes no behaviour.
- [`pr6110_string_backing_id.md:38`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/adr/pr6110_string_backing_id.md?plain=1#L38) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/adr/pr6110_string_backing_id.md#L38) — "pre-sized by the allocator's mint count" against a bare `make(map[uint64]struct{})` at [`garbage_collector.go:200`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/garbage_collector.go#L200) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/garbage_collector.go#L200). Not posted: no behaviour rests on it.
- [`pr6110_string_backing_id.md:50-53`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/adr/pr6110_string_backing_id.md?plain=1#L50-L53) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/adr/pr6110_string_backing_id.md#L50-L53) — "the treap, `trackString` clone-on-overlap, the pin, `CleanupTrackedStrings` and `clearStringTracking` are deleted" describes a delta against #4885's branch; none of them exists at `2ed70a202`, so this diff deletes none of it. The pull request body gets this right. The same bullet's "the allocator keeps no string state" is true of the struct and passes over the new process-global at [`alloc.go:50`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/alloc.go#L50) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/alloc.go#L50). Not posted: the ADR's gas sentence is the one that changes a decision.
- [`pr6110_string_backing_id.md:72`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/adr/pr6110_string_backing_id.md?plain=1#L72) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/adr/pr6110_string_backing_id.md#L72) — "`alloc_0/1/7` +26/+16/+11 B" is measured against #4885. `alloc_1.gno` and `alloc_7.gno` are not in the diff and pass unmodified here; `alloc_0.gno` is the only golden that moved, and its commit also rewrote the fixture body. Master's `alloc_0.gno` body run at this head gives 8091 against the committed 8094, a delta of −3 rather than +26. Not posted: folded into the gas sentence, which is the load-bearing one.
- [`garbage_collector.go:243`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/garbage_collector.go#L243) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/garbage_collector.go#L243) — `size += allocStringByte * sv.Extent` where the sibling `AllocateString` at [`alloc.go:370`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/alloc.go#L370) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/alloc.go#L370) writes the same arithmetic through `overflow.Addp` and `overflow.Mulp`. Unreachable, since `allocStringByte` is 1 and `Extent` is always some in-process `len`. Not posted: a convention break with no failure behind it.

## Verified

- The design's central claim holds under every path I could reach. [`mint_partition.txtar`](./tests/mint_partition.txtar) keeps a stored string, a slice of it, a string-keyed map and a package literal live across a forced collection, and measures head minus base at exactly +400 in every configuration it runs: warm, simulate-only, after the simulation, cold after a node restart, and warm again. Object caches are rebuilt per message at [`store.go:259`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/store.go#L259) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/store.go#L259) and cleared from [`keeper.go:401`](https://github.com/gnolang/gno/blob/4236ef9c6/gno.land/pkg/sdk/vm/keeper.go#L401) · [↗](../../../../../.worktrees/gno-review-6110/gno.land/pkg/sdk/vm/keeper.go#L401), and the stdlib byte cache, the strongest candidate for a bypass, still reaches `fillTypesOfValue` unconditionally at [`store.go:603`](https://github.com/gnolang/gno/blob/4236ef9c6/gnovm/pkg/gnolang/store.go#L603) · [↗](../../../../../.worktrees/gno-review-6110/gnovm/pkg/gnolang/store.go#L603).
- The process-global counter is inert to scheduling: `gnokey_gasfee` reports 1024118 run alone and 1024118 under six concurrent in-process nodes sharing it.
- The wire format is byte-identical to master across six cases including values carrying a nonzero serial and extent, and including the zero-length slice of a tracked source. `MarshalAny` on the value, on a `TypedValue` wrapping it and on a struct holding it, and `MarshalJSON`, all match.
- The committed `pb3_gen.go` is what the generator emits: deleting it and re-running `misc/genproto2` leaves `git status --porcelain` empty.
- `TestAppHashCrossrealm38` passes at this head, and no `.gno` file under `gnovm/stdlibs/**` changed, so the pin should not move.
- `gnovm/tests/files/gas/slice_alloc.gno` is untouched and its filtered-run figure is 70971209 at both base and head, the same offset from the committed 70970748 that clean master shows, so no golden was regenerated from a filtered run.
- Slicing preserves gno-to-Go parity: 25 assertions over content, length, re-slicing, byte and rune conversion, concatenation, all six comparisons, map-key use, rendering and struct, interface and array equality produce byte-identical output under go1.25.9, master and this head.
- The `.grealm` accessors are unchanged in behaviour: 27 assertions covering `Address`, `PkgPath`, `String`, `Subpath`, `IsValid`, `Previous` on the empty-pkgpath user realm, `Sub`, and `runtimeError.Error` match master line for line, the only difference being 698 more gas.
- The duplicate-map-key rewrite is load-bearing rather than cleanup. Built with this head's struct and master's `map[TypedValue]` set, the preprocessor's duplicate check stops firing on `map[string]int{"a": 1, "a": 2}`, because preprocess mints each literal its own serial and struct equality then sees two keys. The literal is still refused, by the Go type checker rather than by the VM.
- Content identity for strings is densely covered already: turning `isEql`'s `StringKind` branch into Go struct equality reddens at least 76 filetests, and carrying the serial into `MapKeyBytes` reddens at least 18.
- Every raw construction site the ADR claims to have migrated is migrated. The count of removed non-test conversions is 20, against the ADR's "~21", and none remains anywhere in the tree.
- Filetests `alloc_0`, `alloc_13`, `alloc_13a`, `alloc_13b`, `alloc_14`, `alloc_14a`, `alloc_14b` and the seven new unit tests are green at this head, and `go test ./gnovm/pkg/gnolang/ -run TestFiles` passes with zero failing subtests.

## Open questions

- Post-restart gas for a genesis-`loadpkg` realm drifts by +29 then +14 across the first two calls **on master as well**, 76971588 to 76971617 to 76971602. So `stdlib_restart_compare`-shaped equality does not hold for every realm shape at the base either. Not posted: pre-existing, and the diff neither sweeps that class nor freezes it.
- The Critical on the load path was found with the store allocator and the machine allocator being the same object, which is how the four transaction paths in `gno.land/pkg/sdk/vm` build them. `withQueryEvalMachine` passes a fresh allocator instead, so the query path may hold the object cache differently. Not posted: I did not build the query-path repro, and the transaction path is enough to block.
- The ADR's benchmark table is stated against #4885's branch rather than master, so none of its figures is checkable from this diff alone. Not posted: the numbers are offered as a comparison between the two proposals, which is what they are.
