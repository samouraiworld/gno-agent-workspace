# PR [#5960](https://github.com/gnolang/gno/pull/5960): perf(gnovm): speed up DidUpdate per-write ownership hook

URL: https://github.com/gnolang/gno/pull/5960
Author: omarsy | Base: master | Files: 7 | +343 -43
Reviewed by: davd-gzl | Model: claude-opus-4-8 (xhigh, deep) | Commit: 75f126bf1 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5960 75f126bf1`

**TL;DR:** Every time a realm's stored data is written, the VM runs a bookkeeping hook that records who owns what. This PR makes that hook cheaper by comparing package identifiers word-by-word instead of byte-by-byte, and by looking up each object's bookkeeping header once instead of on every field read.

**Verdict: REQUEST CHANGES** — correct and behavior-preserving, and the attach/swap paths do improve. But on amd64, which is what validators run, the most common write shape does not improve at all: `RealPrimitive` measures p=0.855 against a claimed −43.3%. `PkgID.eq` takes its operands by value, so each call site copies 20 bytes four times, and on this architecture that costs about what the change saves. A pointer receiver is one line and takes the same benchmarks to −46.58%.

## Summary

`Realm.DidUpdate` runs after every mutation of realm-owned state (11 `op_assign` sites, inc/dec, map/slice/append, pointer writes). The PR attacks two costs in it: `PkgID` equality on a 20-byte `Hashlet`, which Go lowers to a `runtime.memequal` call rather than unrolling; and interface dispatch through `Object` for each flag and refcount read. It adds `PkgID.eq` (word compares), rewrites `Hashlet.IsZero` the same way, and devirtualizes `DidUpdate` by fetching each object's ObjectInfo once and splitting every `MarkX(oo)` into a wrapper over `markX(oo, oi)`.

Both premises are true, and the devirtualization is genuinely behavior-preserving. The problem is arithmetic, not correctness: on amd64, `eq`'s value receiver costs about what the `memequal` call it removes cost, so the two comparison changes net out to roughly nothing there and the measured win comes from the devirtualization alone.

## Glossary

- ObjectInfo: the per-object ownership and persistence header; 20 concrete types satisfy `Object` through it, and `*ObjectInfo` is the only type implementing the accessor set.
- borrow rule: the three rules in `PushFrameCall` deciding whose realm authority a non-crossing call runs under.
- new-real: an object reachable from the realm graph but not yet assigned a persisted ObjectID.
- GnoVM: the Gno virtual machine (`gnovm/pkg/gnolang`).

## Fix

The devirtualization is the load-bearing half and should stay: fetching `*ObjectInfo` once per object collapses three-to-four dynamic calls per `MarkX` into one, which is where the real −14%/−11% on the attach and swap paths comes from. The comparison half needs one change before it earns its place: [`realm.go:127`](https://github.com/gnolang/gno/blob/75f126bf1/gnovm/pkg/gnolang/realm.go#L127) · [↗](../../../../../.worktrees/gno-review-5960/gnovm/pkg/gnolang/realm.go#L127) declares `func (pid PkgID) eq(o PkgID) bool`, and because the body slices `pid.Hashlet[0:8]` the operands must be addressable, so the inlined body materializes stack copies at every call site instead of comparing in registers. Switching to a pointer receiver removes the copies and turns a −5.84% change into a −46.58% one.

## Benchmarks / Numbers

Interleaved, core-pinned (`taskset -c 2`), alternating binaries, `benchstat` n=8, Zen4 / go1.26.5 linux/amd64. Same benchmark file on every side (it compiles unmodified against the merge-base `f99caf537`).

| scenario | base | head, as merged | pointer receiver | PR claims |
|---|---|---|---|---|
| `DidUpdate_NilRealm` | 2.140n | ~ (p=0.959) | ~ | −12.8% (p=0.000) |
| `DidUpdate_Unreal` | 3.568n | ~ (p=0.645) | ~ | −21.6% (p=0.000) |
| `DidUpdate_RealPrimitive` | 15.235n | **~ (p=0.855)** | **−74.20%** | **−43.3%** |
| `DidUpdate_RealAttach` | 24.525n | −14.09% (p=0.002) | −62.40% | −57.2% |
| `DidUpdate_RealSwap` | 34.44n | −11.34% (p=0.007) | −52.59% | −57.0% |
| geomean | 9.964n | **−5.84%** | **−46.58%** | **−41%** |

0 B/op and 0 allocs/op on every side, all samples equal.

## Critical (must fix)

None.

## Warnings (should fix)

- **[most common write shape does not speed up on amd64]** `gnovm/pkg/gnolang/realm.go:127` — `eq` takes both operands by value, so each call site copies 20 bytes four times; on amd64 that costs about what the change saves, and `RealPrimitive` measures p=0.855 against a claimed −43.3%.
  <details><summary>details</summary>

  `eq` slices `pid.Hashlet[0:8]`, which forces both operands addressable, so the inlined body cannot compare in registers. Disassembling the real call site rather than the standalone symbol shows it: at [`realm.go:341`](https://github.com/gnolang/gno/blob/75f126bf1/gnovm/pkg/gnolang/realm.go#L341) · [↗](../../../../../.worktrees/gno-review-5960/gnovm/pkg/gnolang/realm.go#L341) the inlined body emits four 20-byte copies into `0x58(SP)`, `0x1d4(SP)`, `0x184(SP)`, `0x10c(SP)` (sixteen `MOVUPS`: two loads and two stores each) before a single PkgID comparison. The copies are stored as overlapping 16-byte writes at `[0..16)` and `[4..20)`, then read back as words at `[0..8)`, `[8..16)`, `[16..20)`, so each load straddles a store boundary and pays store-to-load forwarding penalties.

  `RealPrimitive` isolates this: the fixture is pre-dirtied so `markDirty` early-returns at [`realm.go:471`](https://github.com/gnolang/gno/blob/75f126bf1/gnovm/pkg/gnolang/realm.go#L471) · [↗](../../../../../.worktrees/gno-review-5960/gnovm/pkg/gnolang/realm.go#L471), leaving one `GetObjectInfo` call plus the compare. Against base that path drops two interface calls and a `memequal` call, and still measures flat (15.235n → 15.155n, p=0.855) — the copies eat exactly what devirtualization saves. `-gcflags=-S` prints the standalone `PkgID.eq` body (`size=43, locals=0x0`) as three clean MOV/CMP pairs, which is what makes this easy to miss: that body is copy-free only because the ABI already passes both 20-byte structs on the stack, and no `eq` symbol survives linking at all, so it never executes.

  Scope of this claim: it is an amd64 result. `GOARCH=arm64 GOOS=darwin` emits the same store-then-reload copies (`LDP`/`STP` into `autotmp`, `pid`, `o`), but emitting them is not the same as paying for them — the cost here is a store-to-load forwarding stall, which is a property of a given core. Apple's cores forward stores far more aggressively than Zen4, so the copies may be near-free on an M1 while the savings side stays real, which would reconcile both tables. I have no M1 to settle it. What holds regardless: on amd64 this form gets ~nothing on the primitive path, and amd64 is what validators run.

  Fix: give `eq` a pointer receiver and pointer argument. That alone measures −24.90% geomean; also comparing straight off `poi.ID.PkgID` instead of binding the intermediate `poPkgID` local reaches −46.58%, with `RealPrimitive` at −74.20%. Built and run, tests green — this is not a suggestion on paper.
  </details>

- **[benchmark cannot attribute its own numbers]** `gnovm/pkg/gnolang/realm_didupdate_bench_test.go:39-47` — `BenchmarkDidUpdate_NilRealm` executes no line this PR changes, so its reported −12.8% at p=0.000 measures code layout, not the optimization.
  <details><summary>details</summary>

  `benchMachine()` never sets `Stage`, and `Stage` is a string type ([`context.go:3-9`](https://github.com/gnolang/gno/blob/75f126bf1/gnovm/pkg/gnolang/context.go#L3-L9) · [↗](../../../../../.worktrees/gno-review-5960/gnovm/pkg/gnolang/context.go#L3-L9)), so it is `""`. `DidUpdate` therefore enters the nil-realm block, fails `m.Stage == StageRun` at [`realm.go:295`](https://github.com/gnolang/gno/blob/75f126bf1/gnovm/pkg/gnolang/realm.go#L295) · [↗](../../../../../.worktrees/gno-review-5960/gnovm/pkg/gnolang/realm.go#L295), and returns. The first changed line inside `DidUpdate` is 331 (`git diff -U0` hunk header `@@ -318 +331,8 @@`); the nil-realm block spans 288-315.

  `NilRealm` is the clean case: zero changed lines. `Unreal` is milder but related, reaching the rewritten early-return, which swaps one interface call (`po.GetIsReal()`) for another (`po.GetObjectInfo()`), so there is little to win there either. Measured, both are noise: p=0.959 and p=0.645 against base, against claimed −12.8% and −21.6% at p=0.000. Both still swing: reverting only `eq`→`==` moved them 2.16→5.02n and 3.36→7.39n, though neither path ever runs an `eq`, and the pointer-receiver variant moves `NilRealm` +8.41% (p=0.028). Two of the five rows in the claimed geomean cannot carry a speedup, which is why the −41% does not survive a controlled A/B.

  Because `Stage` is `""`, this benchmark also skips the `/p/`-immutability gate entirely, so it does not exercise the nil-realm shape a real transaction takes.

  Fix: drop the two scenarios that cannot move, or set `Stage` so the nil-realm gate is actually exercised, and quote a geomean over paths the diff reaches.
  </details>

## Nits

- **[rewrite not shown to pay at its call site]** `gnovm/pkg/gnolang/hash_image.go:54` — the `memequal` premise holds here, but the win does not show up through `ObjectID.IsZero`, the shape the comment names: I measure 9.768n before versus 10.170n after. That is ~0.4n on a 9.8n benchmark, which is inside this package's layout noise, so read it as "no demonstrated gain" rather than a regression. `ObjectID.IsZero` reaches `Hashlet.IsZero` through the same value-receiver chain, so the copies apply here too and a pointer receiver would be the same fix.
- **[dead exported code]** `gnovm/pkg/gnolang/realm.go:485` — `MarkNewDeleted` has no callers left anywhere in the repo; base called it at `realm.go:388`, which is now `markNewDeleted(xo, xoi)`. The other three wrappers keep real callers (`machine.go:872,878`, `realm.go:721,736,744,815,873,946`). Exported, so no linter flags it.
- **[eq narrows silently if PkgID gains a field]** `gnovm/pkg/gnolang/realm.go:61-63` — `eq` reads exactly 20 bytes, so it is `==` only while `PkgID` is nothing but its `Hashlet`. `HashSize` drift is covered: the tests catch growth and shrinkage is already a compile error. Field addition is covered by nothing. Adding a `Version uint8` first fails to build, but only at the unkeyed literal `&PkgID{HashBytes(...)}` (`realm.go:101`); key it, as the compiler forces, and the build is clean while `a == b` is false and `a.eq(b)` is true for two distinct PkgIDs. `TestPkgIDEq` and `TestHashletIsZero` both still pass, and `alloc.go`'s `unsafe.Sizeof` init guard stays silent because the `uint8` hides in `ObjectID`'s existing padding. Since `eq` gates realm identity in `IsReadonlyBy` and `isExternalRealm`, that direction fails open. The package already has the idiom (`alloc.go:146`); four lines pin it at compile time, catching both drifts at once:

  ```go
  // PkgID.eq and Hashlet.IsZero hard-code an 8+8+4 layout covering all of PkgID.
  var (
  	_ [unsafe.Sizeof(PkgID{}) - 20]struct{}
  	_ [20 - unsafe.Sizeof(PkgID{})]struct{}
  )
  ```
- **[same read, two spellings]** `gnovm/pkg/gnolang/realm.go:434` — `markNewReal` reads `pv.GetOwner()`/`pv.GetRefCount()` in the `*PackageValue` branch but `oi.GetOwner()` in the `else`. Same values, but the split invites a reader to think they differ.
- **[fixtures drift across iterations]** `gnovm/pkg/gnolang/realm_didupdate_bench_test.go:85-93` — after 1000 `RealSwap` iterations `co.RefCount` is 1001 and `xo.RefCount` is −999, both starting at 1. Timing impact is nil once branch outcomes stabilize, but `xo` stays real, which is exactly `DecRefCount`'s `if debug { if oi.GetIsReal() { panic("should not happen") } }` arm.

## Missing Tests

- **[load-bearing invariant unpinned]** `gnovm/pkg/gnolang/realm.go:337` — nothing catches an `Object` implementation overriding an `ObjectInfo` accessor, which is the assumption the whole devirtualization rests on.
  <details><summary>details</summary>

  `DidUpdate` and the `markX` bodies now read and write exclusively through `*ObjectInfo`. If any implementation ever overrides one of these accessors, `DidUpdate` would silently bypass the override while every other caller keeps seeing it. Today the invariant holds — I scanned all 32 `Object` methods and the only override is `RefValue.GetObjectID()` ([`values.go:2738`](https://github.com/gnolang/gno/blob/75f126bf1/gnovm/pkg/gnolang/values.go#L2738) · [↗](../../../../../.worktrees/gno-review-5960/gnovm/pkg/gnolang/values.go#L2738)), and `RefValue` has no embedded ObjectInfo so it does not satisfy `Object` at all — but it is enforced only by a grep, and a grep does not run in CI.

  Confirmed by mutation: adding `func (sv *StructValue) GetIsReal() bool { return !sv.ObjectInfo.GetIsReal() }` leaves the PR's own tests green. The test below fails on it:

  ```
  --- FAIL: TestObjectInfoAccessorsAreNotOverridden (0.00s)
      *gnolang.StructValue: GetIsReal() via Object = false, via *ObjectInfo = true
  ```

  Ready to add, green on the PR as-is: [`tests/realm_devirt_test.go`](tests/realm_devirt_test.go). Note the surface is wider than the `var _ Object = ...` block suggests: a `go/types` sweep finds 20 concrete implementations, not 8 — the `BlockNode` family (`FileNode`, `FuncDecl`, `IfStmt`, ...) reaches `ObjectInfo` through `StaticBlock` → `Block`. The test covers all 20 and catches an injected override on `*FileNode` as well as on `*StructValue`.
  </details>

## Suggestions

- **[mismatched pair corrupts bookkeeping silently]** `gnovm/pkg/gnolang/realm.go:430` — the four `markX(oo, oi)` bodies require `oi == oo.GetObjectInfo()` and nothing checks it, including under `-tags debugAssert`.
  <details><summary>details</summary>

  `markDirty` sets the flag on `oi` but appends `oo` to `rlm.updated`, so a mismatched pair flags one object and enqueues another. Demonstrated with `rlm.markDirty(a, b.GetObjectInfo())`: `a.GetIsDirty()` stays false while `b.GetIsDirty()` becomes true, `rlm.updated` contains `a`, and a subsequent correct call appends `a` twice. Silent under `-tags debugAssert` too.

  Latent today — the forms are unexported and only `DidUpdate` and the wrappers call them — but `debugAssert` is a build-tag const, so the guard costs nothing in production. Verified: it fires on the mismatch and produces zero false positives across `-run Files -test.short -tags debugAssert`.
  </details>

- **[sibling check left on the slow path]** `gnovm/pkg/gnolang/machine.go:2408` — borrow rule #3 does the same PkgID compare as borrow rule #2 thirty lines above, in the same function, and keeps `!=`.
  <details><summary>details</summary>

  [`machine.go:2377`](https://github.com/gnolang/gno/blob/75f126bf1/gnovm/pkg/gnolang/machine.go#L2377) · [↗](../../../../../.worktrees/gno-review-5960/gnovm/pkg/gnolang/machine.go#L2377) was converted to `!recvOID.PkgID.eq(m.Realm.ID)`; [`machine.go:2408`](https://github.com/gnolang/gno/blob/75f126bf1/gnovm/pkg/gnolang/machine.go#L2408) · [↗](../../../../../.worktrees/gno-review-5960/gnovm/pkg/gnolang/machine.go#L2408) still reads `pid != m.Realm.ID`. Both run per call.

  Two more expressions of the same shape lines 382 and 404 converted stay on `!=` at [`realm.go:710`](https://github.com/gnolang/gno/blob/75f126bf1/gnovm/pkg/gnolang/realm.go#L710) · [↗](../../../../../.worktrees/gno-review-5960/gnovm/pkg/gnolang/realm.go#L710) and [`realm.go:807`](https://github.com/gnolang/gno/blob/75f126bf1/gnovm/pkg/gnolang/realm.go#L807) · [↗](../../../../../.worktrees/gno-review-5960/gnovm/pkg/gnolang/realm.go#L807), plus `realm.go:1125,1169,1223,2031,2039,2057` and `store.go:448`. Those are finalize-time, so leaving them is defensible scoping; the concern is that two idioms for one comparison now coexist with nothing saying which to reach for.
  </details>

## Verified

- The `memequal` premise holds, contrary to what a reader might assume from Go's small-array unrolling: on go1.26.5/amd64 a `[20]byte` `==` written inline, with no `//go:noinline`, emits `CALL runtime.memequal(SB)`. Same for `h == Hashlet{}`.
- `eq` and `IsZero` inline (cost 46 and 29, under the 80 budget) and neither they nor their callers escape to the heap; the real `(*Realm).DidUpdate` at head contains zero `memequal` calls. The mechanism works; only its cost model is wrong.
- The devirtualization is behavior-preserving: `GetObjectID()` is `return oi.ID` and `GetObjectInfo()` is `return oi`, both pure, and no `Object` implementation overrides any accessor. The one override in the package, `RefValue.GetObjectID()`, is on a type that does not implement `Object` — confirmed by type assertion, not by reading.
- Reverting only `eq`→`==` at the three `DidUpdate` sites moves `NilRealm` and `Unreal` by ~2x each, on paths that execute no `eq` — the harness measures code layout at this granularity.
- The pointer-receiver variant is not just a theory: built, `TestPkgIDEq`/`TestHashletIsZero` green, zero `MOVUPS` at the call site, −46.58% geomean.
- Clean master merge: `git show 1d008f7f2 --cc` prints no conflict hunks, so nothing was authored in the merge.
- Suites run at 75f126bf1: `sdk/vm -run Gas` ok; `integration -run TestTestdata` ok (66s); `gnolang -run Files -test.short` fails 10 tests, and the failing set is character-for-character identical on base and head, so they are pre-existing on master rather than introduced here.

## Open questions

- `Hashlet.IsZero` is reached from many more guards than `DidUpdate`'s compare (`ownership.go:281,530`, `machine.go:2376,2746`, `store.go:609,1200`), so a pointer-receiver `IsZero` may be worth more than the `DidUpdate` work — but it is a method-set change on an exported type, so it needs its own PR rather than a note here.
