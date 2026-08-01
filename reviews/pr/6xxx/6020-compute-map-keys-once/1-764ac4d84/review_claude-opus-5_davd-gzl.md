# PR [#6020](https://github.com/gnolang/gno/pull/6020): perf(gnovm): compute map keys once, and skip the redundant TypeID prefix for concrete key types

URL: https://github.com/gnolang/gno/pull/6020
Author: Villaquiranm | Base: master | Files: 21 | +1447 -48
Reviewed by: davd-gzl | Model: claude-opus-5 | Commit: 764ac4d84 (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6020 764ac4d84`

**TL;DR:** Every read, write and delete on a Gno map turns the key into a string first, and the VM charges gas for each byte of that string. This PR stops building the string twice on every write, and stops prefixing it with a type name that is the same for every key in the map. Both make maps cheaper.

**Verdict: APPROVE** — the encoding is in-memory only, the build-and-probe flag comes from one derivation per call site, and the gas change is a verified reduction; `delete` is the one write path still computing the key twice, and two of the new tests do not cover the case they name (2 Nits, 2 Missing tests, 2 Suggestions).

## Verify first

- [`gnovm/pkg/gnolang/values.go:1031-1042`](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/pkg/gnolang/values.go#L1031-L1042) · [↗](../../../../../.worktrees/gno-review-6020/gnovm/pkg/gnolang/values.go#L1031-L1042) — the whole change rests on the index build and every later probe passing the same `omitKeyType`. Confirm nothing indexes `vmap` outside that one build and the three accessors: `grep -rn '\.vmap\[' --include='*.go' gnovm/` returns five lines, at values.go 1039, 1061, 1076, 1093 and 1111.
- [`gnovm/tests/files/gas/compute_map_key_concrete_key.gno:22`](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/tests/files/gas/compute_map_key_concrete_key.gno#L22) · [↗](../../../../../.worktrees/gno-review-6020/gnovm/tests/files/gas/compute_map_key_concrete_key.gno#L22) — the merge-base number for this new golden is 135249, not the 125449 the summary table reports. Copy the file onto d1a33f574 and run the full `go test ./gnovm/pkg/gnolang/ -run Files -test.short`; the golden fails with `+135249`.
- [`gnovm/pkg/gnolang/realm.go:1901-1922`](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/pkg/gnolang/realm.go#L1901-L1922) · [↗](../../../../../.worktrees/gno-review-6020/gnovm/pkg/gnolang/realm.go#L1901-L1922) — this walk replaces the ref resolution `ComputeMapKey` used to do as a side effect, and every divergence between the two is an observable change. Read it against [`ComputeMapKey`'s array and struct branches](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/pkg/gnolang/values.go#L1951-L1999) · [↗](../../../../../.worktrees/gno-review-6020/gnovm/pkg/gnolang/values.go#L1951-L1999) and confirm the NaN early return is the only one.

## Summary

`ComputeMapKey` serializes a map key into the string that indexes `MapValue.vmap`, charging [80 gas per call plus 4/10 per byte](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/pkg/gnolang/machine.go#L1661-L1674) · [↗](../../../../../.worktrees/gno-review-6020/gnovm/pkg/gnolang/machine.go#L1661-L1674). Two costs are removed. First, every map write ran it twice: once in [`GetPointerAtIndex`](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/pkg/gnolang/values.go#L2452-L2468) · [↗](../../../../../.worktrees/gno-review-6020/gnovm/pkg/gnolang/values.go#L2452-L2468) to find the key it was about to displace, then again inside `GetPointerForKey`; the displaced key's object is now returned from the accessor instead. Second, for a map whose static key type is not an interface the leading `TypeID` prefix is the same for every key, so [`mapKeyOmitType`](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/pkg/gnolang/values.go#L1871-L1872) · [↗](../../../../../.worktrees/gno-review-6020/gnovm/pkg/gnolang/values.go#L1871-L1872) drops it. On the `map[address]T` shape the two together take 135249 gas down to 124849, of which 9800 is the deduplication and 600 the prefix.

The flag has to be identical between the index build and every probe, and the load path has no static key type in scope, so `vmap` is no longer built at load: [`fillTypesOfValue`](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/pkg/gnolang/realm.go#L1972-L1983) · [↗](../../../../../.worktrees/gno-review-6020/gnovm/pkg/gnolang/realm.go#L1972-L1983) drops it and [`ensureVmap`](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/pkg/gnolang/values.go#L1031-L1042) · [↗](../../../../../.worktrees/gno-review-6020/gnovm/pkg/gnolang/values.go#L1031-L1042) builds it on first keyed access with the caller's flag. That removal is what makes the rest of the diff big: the old build was also the deep ref-resolver for composite keys, so [`fillMapKeyRefs`](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/pkg/gnolang/realm.go#L1901-L1922) · [↗](../../../../../.worktrees/gno-review-6020/gnovm/pkg/gnolang/realm.go#L1901-L1922) takes that job over, and it has to run before the key is copied at both insert sites.

Nothing persisted changes. [`MapKey`](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/pkg/gnolang/values.go#L919) · [↗](../../../../../.worktrees/gno-review-6020/gnovm/pkg/gnolang/values.go#L919) exists only as the `vmap` index type, and [`copyValueWithRefs` writes a `MapValue` from `List` alone](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/pkg/gnolang/realm.go#L1722-L1732) · [↗](../../../../../.worktrees/gno-review-6020/gnovm/pkg/gnolang/realm.go#L1722-L1732), so the encoding never reaches the amino image. What is consensus-visible is the gas, and one output change for a composite key holding a NaN before an object-bearing field, pinned by [`zrealm_map10.gno`](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/tests/files/zrealm_map10.gno#L44-L45) · [↗](../../../../../.worktrees/gno-review-6020/gnovm/tests/files/zrealm_map10.gno#L44-L45).

Reading order: `values.go` (predicate, `ensureVmap`, the three accessors, `GetPointerAtIndex`), then `realm.go` (`fillMapKeyRefs`, the load path), then the three call sites in `op_expressions.go` and `uverse.go`, then the filetests.

## Diagram

```
master, one map write                  this PR
---------------------                  -------
GetPointerAtIndex                      GetPointerAtIndex
  iv.ComputeMapKey  ── charged           fillMapKeyRefs(iv)   ── resolve, explicit
  vmap[key] -> oldObject                 ivk := iv.Copy
  ivk := iv.Copy                         GetPointerForKey(ivk, omitKeyType)
  GetPointerForKey(ivk)                    ensureVmap        ── build if absent
    ComputeMapKey ── charged AGAIN         ComputeMapKey     ── charged once
    vmap[key] -> mli                       vmap[key] -> mli, oldKeyObject
                                                            ── returned to caller

load (fillTypesOfValue)                load (fillTypesOfValue)
  per key: ComputeMapKey                 per key: fillMapKeyRefs
    -> builds vmap                       vmap: not built
    -> resolves refs (side effect)
```

## Fix

Three accessors gain an `omitKeyType` parameter and open by ensuring the index exists, so one derived value feeds both the build and the probe and they cannot disagree. `GetPointerForKey` additionally returns the object of the stored key it displaces ([`values.go:1056-1082`](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/pkg/gnolang/values.go#L1056-L1082) · [↗](../../../../../.worktrees/gno-review-6020/gnovm/pkg/gnolang/values.go#L1056-L1082)), which is the only thing the caller's second computation was for. The load-bearing constraint is ordering: moving the surviving `ComputeMapKey` inside the accessor moves it after `iv.Copy`, and `Copy` shallow-copies an unresolved `RefValue` child, so an explicit `fillMapKeyRefs` has to run before the copy at [`values.go:2461`](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/pkg/gnolang/values.go#L2461) · [↗](../../../../../.worktrees/gno-review-6020/gnovm/pkg/gnolang/values.go#L2461) and at [`op_expressions.go:587`](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/pkg/gnolang/op_expressions.go#L587) · [↗](../../../../../.worktrees/gno-review-6020/gnovm/pkg/gnolang/op_expressions.go#L587) or the stored key aliases the caller's variable.

## Benchmarks / Numbers

Full `go test ./gnovm/pkg/gnolang/ -run Files -test.short` at merge base d1a33f574 and at 764ac4d84, go1.25.9. The new golden was copied onto the merge base to get its baseline; every test body is byte-identical across the two trees, only the `// Gas:` line moves.

| golden | d1a33f574 | 764ac4d84 | Δ |
|---|---:|---:|---:|
| `compute_map_key_big_bytes` | 36827018 | 23405151 | −36.4% |
| `compute_map_key_big_struct` | 186401382 | 120708884 | −35.2% |
| `compute_map_key_small_bytes` | 7974 | 7873 | −1.3% |
| `compute_map_key_small_struct` | 6109 | 5677 | −7.1% |
| `compute_map_key_concrete_key` | 135249 | 124849 | −7.7% |

The last row's 10400 reconciles exactly against the 100 writes the golden does. Before, each write paid two `ComputeMapKey` calls of `80 + 45*4/10 = 98`; after, one call of `80 + 32*4/10 = 92`. So 9800 is the dropped second call and 600 is the 13-byte `main.Address:` prefix on the surviving one. The PR summary reports this row as `125449 → 124849`, −0.5%; 125449 is the value after the deduplication commit, not the merge base.

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- **[a test comment that names the wrong guarantee]** [`gnovm/pkg/gnolang/values_test.go:433`](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/pkg/gnolang/values_test.go#L433) · [↗](../../../../../.worktrees/gno-review-6020/gnovm/pkg/gnolang/values_test.go#L433) — says `baseOf` must unwrap the declared key type or the realm case misses the win, but `DeclaredType.Kind()` already returns the base's kind, so the call is a no-op.
  <details><summary>details</summary>

  [`(*DeclaredType).Kind`](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/pkg/gnolang/types.go#L1515-L1517) · [↗](../../../../../.worktrees/gno-review-6020/gnovm/pkg/gnolang/types.go#L1515-L1517) returns `dt.Base.Kind()`, so `baseOf(mt.Key).Kind()` and `mt.Key.Kind()` agree for every input, declared or not. Rewriting [the predicate](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/pkg/gnolang/values.go#L1871-L1872) · [↗](../../../../../.worktrees/gno-review-6020/gnovm/pkg/gnolang/values.go#L1871-L1872) to `mt.Key.Kind() != InterfaceKind` changes no result: `TestMapKeyOmitType`, `TestMapKeyOmitType_declaredKey`, `TestComputeMapKey`, `TestMapValue_vmapBuiltLazily`, `map51` through `map54` and `zrealm_map6` through `zrealm_map10` stay green, and the five `gas/compute_map_key_*` goldens report byte-identical numbers with and without it. The `baseOf` is worth keeping for symmetry with the nested rule, but the comment claims a behavioral dependency the type system does not have. Fix: say the call is defensive, or point the comment at the `mt.Key` versus `mt.Elem()` distinction, which is the one the test really pins.
  </details>

- **[a comment that promises a numeric guarantee no assertion holds]** [`gnovm/pkg/gnolang/values.go:1013-1024`](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/pkg/gnolang/values.go#L1013-L1024) · [↗](../../../../../.worktrees/gno-review-6020/gnovm/pkg/gnolang/values.go#L1013-L1024) — the comment states that narrowing `fillMapKeyRefs` would silently reintroduce first-touch store gas here, but nothing in the tree would catch that.
  <details><summary>details</summary>

  The claim is correct: [`GetObjectSafe` returns from `cacheObjects` without charging](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/pkg/gnolang/store.go#L514-L526) · [↗](../../../../../.worktrees/gno-review-6020/gnovm/pkg/gnolang/store.go#L514-L526), so the lazy build is free only because [the load path already walked every key](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/pkg/gnolang/realm.go#L1972-L1983) · [↗](../../../../../.worktrees/gno-review-6020/gnovm/pkg/gnolang/realm.go#L1972-L1983). No filetest asserts a gas number for a realm map with composite keys, and `// Gas:` is only used by `package main` filetests, so the invariant lives in prose alone. Fix: none available inside this diff; flagging for whoever adds a realm-level gas harness.
  </details>

## Missing Tests

- **[the one key-replacement case that is observable is not exercised]** [`gnovm/tests/files/map51.gno:3-6`](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/tests/files/map51.gno#L3-L6) · [↗](../../../../../.worktrees/gno-review-6020/gnovm/tests/files/map51.gno#L3-L6) — the file says it pins the displaced-key handoff, but its output does not change when the handoff is removed, and its `-0.0` is the constant `0`.
  <details><summary>details</summary>

  `println(m[0])` and `println(m[-0.0])` both read the stored value, which is `"negative"` whichever key the entry holds; the struct half reads a value too. Deleting `mli.Key = key` at [`values.go:1066`](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/pkg/gnolang/values.go#L1066) · [↗](../../../../../.worktrees/gno-review-6020/gnovm/pkg/gnolang/values.go#L1066) leaves `map51` green, while `zrealm_map7` and `zrealm_map8` turn red on their object-identity goldens, so the line is covered but the float case named in the comment is not. It cannot be: the untyped constant `-0.0` is `+0`, which [`float9.gno`](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/tests/files/float9.gno#L1-L3) · [↗](../../../../../.worktrees/gno-review-6020/gnovm/tests/files/float9.gno#L1-L3) already documents, so the map never receives a negative zero. `math.Signbit` on the ranged key at 764ac4d84 confirms both entries are `+0`. Fix: build the negative zero with `math.Copysign(0, -1)` and range the map, asserting the sign of the stored key; [`tests/map55.gno`](tests/map55.gno) does that, is green at 764ac4d84, and turns red on the same revert that leaves `map51` green.
  </details>

- **[the interface branch of the predicate is untested across a store round trip]** [`gnovm/pkg/gnolang/values.go:1031-1042`](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/pkg/gnolang/values.go#L1031-L1042) · [↗](../../../../../.worktrees/gno-review-6020/gnovm/pkg/gnolang/values.go#L1031-L1042) — no filetest persists an interface-keyed map, so the shape where the lazy build must keep the prefix never reaches the load path.
  <details><summary>details</summary>

  [`map52.gno`](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/tests/files/map52.gno#L3-L4) · [↗](../../../../../.worktrees/gno-review-6020/gnovm/tests/files/map52.gno#L3-L4) covers `map[any]string` in one execution, and [`zrealm_map6.gno`](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/tests/files/zrealm_map6.gno#L4-L6) · [↗](../../../../../.worktrees/gno-review-6020/gnovm/tests/files/zrealm_map6.gno#L4-L6) covers the round trip for a concrete key, but nothing covers both. Scanning every `// PKGPATH:` filetest for an interface-kinded key type returns nothing; the only hit, [`zrealm_launder_rdata_iface.gno:46`](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/tests/files/zrealm_launder_rdata_iface.gno#L46) · [↗](../../../../../.worktrees/gno-review-6020/gnovm/tests/files/zrealm_launder_rdata_iface.gno#L46), is `map[string]any` built inside `main`. The failure it would catch is total: a build that drops the prefix merges `int(1)` and `int64(1)` into one entry and every probe misses. [`tests/zrealm_map11.gno`](tests/zrealm_map11.gno) builds the map in `init`, reads, inserts and deletes in `main`, and is green at 764ac4d84.
  </details>

## Suggestions

- **[the sweep's own class, missed at one site]** [`gnovm/pkg/gnolang/uverse.go:1249`](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/pkg/gnolang/uverse.go#L1249) · [↗](../../../../../.worktrees/gno-review-6020/gnovm/pkg/gnolang/uverse.go#L1249) — `delete` still computes the map key twice, once in `GetValueForKey` and once in `DeleteForKey`, which is one whole `ComputeMapKey` charge per delete.
  <details><summary>details</summary>

  Measured at 764ac4d84 on a filetest that inserts and then deletes one `[1<<18]byte` key: 527876 gas. Making `delete` skip the `GetValueForKey` probe drops it to 422938, a difference of 104938, which is exactly `80 + 262146*4/10`, the charge for one call over the bracketed 256 KiB key. That is 20% of the whole program's gas in that filetest. [`DeleteForKey`](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/pkg/gnolang/values.go#L1104-L1117) · [↗](../../../../../.worktrees/gno-review-6020/gnovm/pkg/gnolang/values.go#L1104-L1117) already returns nil when the key is absent, which is the only thing the probe decides; the value it also needs for the second `DidUpdate` is `mli.Value`, which the same lookup has in hand. The same doubling was raised against the original metering PR ([#5127](https://github.com/gnolang/gno/pull/5127)) and is still open. Fix: have `DeleteForKey` hand back the removed entry's value alongside its key, and drop the probe.
  </details>

- **[two open PRs move the same charge in opposite directions]** [`gnovm/pkg/gnolang/values.go:1037`](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/pkg/gnolang/values.go#L1037) · [↗](../../../../../.worktrees/gno-review-6020/gnovm/pkg/gnolang/values.go#L1037) — the lazy build passes a nil `*Machine` to suppress its own gas; [#5710](https://github.com/gnolang/gno/pull/5710) removes that parameter and reads the meter off the `Store`, at which point the suppression stops working and the build becomes metered at first keyed access.
  <details><summary>details</summary>

  [#5710](https://github.com/gnolang/gno/pull/5710) is open against the same function and drops `*Machine` from `ComputeMapKey` entirely, so `ComputeMapKey(nil, store, omitKeyType)` here would start charging the tx that first touches the map rather than nothing. That is not wrong on its own, but it converts a build that is invisible today into a first-touch spike whose size is the whole map, and it lands silently because the nil argument still compiles. Whichever merges second has to make the choice explicitly. Fix: nothing in this diff; worth a note to the other author so the merge order is deliberate.
  </details>

## Verified

- Each of the three new realm guards fails when its own line is removed, and its siblings stay green: dropping `mv.ensureVmap` from `GetPointerForKey` fails `zrealm_map7` while `zrealm_map6` and `map51` pass; dropping `fillMapKeyRefs(store, iv)` from `GetPointerAtIndex` fails `zrealm_map8` while `zrealm_map7` and `map39b` pass; dropping `fillMapKeyRefs` from `doOpMapLit` fails `zrealm_map9` while `zrealm_map8` passes.
- `zrealm_map9` and `zrealm_map10` both reproduce at merge base d1a33f574: master prints `stored key: 1 99 value: 5` for the map-literal aliasing case and `ref(08c90256b8c3292fbe757844cfc1de009fb75c14:14)` for the NaN key. `zrealm_map7`, `zrealm_map8` and `map51` pass unchanged at the merge base, so those three pin behavior this branch had to preserve rather than behavior it repairs.
- The gas table above is measured, not taken from the PR: the merge-base run has the new golden copied in and reports 135249, and the 10400 delta decomposes into 9800 for the dropped per-write call and 600 for the prefix, matching the per-access arithmetic exactly.
- `delete` pays one redundant `ComputeMapKey`: 527876 gas for insert-then-delete of a `[1<<18]byte` key, 422938 with the `GetValueForKey` probe removed, a difference equal to the single-call charge for that key size to the unit.
- Lazily writing `vmap` on a read path does not create a shared-state race: every query entry point in `gno.land/pkg/sdk/vm/keeper.go` builds its own store through `newGnoTransactionStore`, and [`BeginTransaction` gives each one a fresh `cacheObjects`](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/pkg/gnolang/store.go#L247-L256) · [↗](../../../../../.worktrees/gno-review-6020/gnovm/pkg/gnolang/store.go#L247-L256), so no `*MapValue` is reachable from two goroutines.
- A realm map that is persisted empty still works with the deferred build: an `init`-time `map[string]int{}` read, written and deleted from in a later transaction round-trips green, so the amino image never leaves `List` nil for `ensureVmap` to dereference.
- Green at 764ac4d84: the full `-run Files -test.short` suite, and `go test ./gno.land/pkg/sdk/vm/ -run Gas`. At merge base d1a33f574 on go1.25.9 the same suite reports exactly one failure, the new gas golden copied in for the baseline, so the ten pre-existing failures the PR describes are a local toolchain artifact and not a property of master.

## Open questions

- The NaN-shaped store-gas increase is pinned as an output change by `zrealm_map10` but not as a number, and it cannot be: `// Gas:` is only read for `package main` filetests, and this shape needs a realm round trip. Not posted: the harness would have to grow first, which is out of this PR's scope.
- `doOpMapLit` discards the displaced key object that `GetPointerForKey` now returns. Safe today because a map literal always inserts a fresh `Copy`, so the displaced key is never a real object, but the discard is silent. Not posted: no reachable defect, and naming it would read as a request to change working code.
